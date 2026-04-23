# AGENTS.md - AI Agent 开发指南

本文档为 AI 编程助手提供项目开发指南，确保代码修改符合项目架构和规范。

## 1. 项目概述

期货回测系统，采用 Go 语言实现核心回测逻辑，Python 负责数据抓取。系统采用**信号生成与资金计算分离**的两层架构。

### 1.1 核心架构

```
K线数据 → [信号层] → TradeSignal[] → [资金层] → DailyRecord[] / TradeRecord[]
```

### 1.2 技术栈

- **核心语言**: Go 1.21+
- **数据抓取**: Python 3.x + akshare
- **Web服务**: Gin框架
- **前端**: HTML + JavaScript (原生)
- **数据交换**: JSON

## 2. 目录结构

```
d:\code\local\test\
├── cmd/                        # 入口程序
│   ├── main.go                 # 命令行回测入口
│   └── web/                    # Web服务入口
│       └── main.go             # Web服务启动文件
├── config/                     # 配置文件
│   ├── strategies.json         # 策略配置
│   └── symbols.json            # 品种配置
├── docs/                       # 文档
│   ├── DESIGN.md               # 设计文档
│   ├── STRATEGY.md             # 策略说明文档
│   └── REQUIREMENTS.md         # 需求文档
├── internal/                   # 内部包
│   ├── backtest/               # 回测核心
│   │   ├── signal.go           # 信号引擎
│   │   ├── portfolio.go        # 资金引擎
│   │   ├── types.go            # 类型定义
│   │   ├── stats.go            # 统计计算
│   │   └── reporter.go         # 报告生成
│   ├── data/                   # 数据管理
│   │   ├── futures_data.go     # 期货数据获取
│   │   └── dominant.go         # 主力合约识别
│   ├── strategy/               # 策略层
│   │   ├── interface.go        # 策略接口定义
│   │   ├── factory.go          # 策略工厂与注册
│   │   ├── yinyang_wrapper.go  # 阴阳策略包装器
│   │   └── yinyang/            # 阴阳线策略实现
│   │       ├── strategy.go     # 策略主逻辑
│   │       ├── state.go        # 状态管理
│   │       ├── adapter.go      # 策略适配器
│   │       └── rollover.go     # 移仓换月
│   └── web/                    # Web服务
│       ├── server.go           # HTTP服务器
│       ├── config.go           # 配置加载
│       ├── storage.go          # 结果存储
│       └── portfolio.go        # 组合分析
├── pkg/                        # 公共包
│   └── pyexec/                 # Python执行器
│       └── pyexec.go           # Python调用封装
├── scripts/                    # Python脚本
│   ├── get_futures_data.py     # 数据抓取脚本
│   └── get_symbols.py          # 品种获取脚本
├── web/                        # Web前端
│   ├── static/                 # 静态资源
│   └── templates/              # HTML模板
│       ├── index.html          # 主页面模板
│       └── portfolio.html      # 组合分析页面
├── ret/                        # 回测结果存储目录
├── go.mod                      # Go模块定义
├── go.sum                      # 依赖版本锁定
└── AGENTS.md                   # 本文档
```

## 3. 核心设计原则

### 3.1 信号-资金分离

**重要**: 系统严格区分信号层和资金层，修改代码时必须遵守：

- **信号层** (`internal/strategy/`, `internal/backtest/signal.go`):
  - 只负责生成交易信号（开多/开空/平多/平空）
  - 不涉及资金计算、保证金、盈亏
  - 使用 `SignalPosition` 表示持仓状态
  - 输出 `TradeSignal` 列表

- **资金层** (`internal/backtest/portfolio.go`):
  - 接收 `TradeSignal` 作为输入
  - 计算保证金、盈亏、资金曲线
  - 使用 `Position` 表示实际持仓
  - 输出 `DailyRecord` 和 `PositionReturn`

### 3.2 时序性要求

**关键**: 禁止使用未来函数，严格遵循时序：

```
处理K线T时：
  Step 1: 使用T-1日计算的信号价格
  Step 2: 判断T日是否触发交易
  Step 3: 更新阴阳集合状态（使用T日数据）
  Step 4: 计算T+1日的信号价格
```

**禁止事项**:

- ❌ 在触发判断前更新阴阳集合状态
- ❌ 使用T+1日的数据影响T日的决策
- ❌ 在移仓检测日立即执行移仓（必须延迟到T+1日）

### 3.3 策略工厂模式

系统使用策略工厂模式支持多策略：

```go
type StrategyFactory interface {
    Create(params map[string]interface{}) SignalStrategy
    Name() string
    Description() string
}
```

**添加新策略时**:

1. 在 `internal/strategy/` 下创建新包
2. 实现 `SignalStrategy` 接口
3. 创建对应的 `StrategyFactory` 实现
4. 在 `factory.go` 的 `init()` 中注册
5. 更新 `config/strategies.json`

## 4. 关键数据结构

### 4.1 K线数据

```go
type KLineData struct {
    Date   string  `json:"date"`    // 日期，格式 YYYY-MM-DD
    Open   float64 `json:"open"`    // 开盘价
    High   float64 `json:"high"`    // 最高价
    Low    float64 `json:"low"`     // 最低价
    Close  float64 `json:"close"`   // 收盘价
    Volume float64 `json:"volume"`  // 成交量
    Amount float64 `json:"amount"`  // 成交额
    Hold   float64 `json:"hold"`    // 持仓量
    Settle float64 `json:"settle"`  // 结算价
}

type KLineWithContract struct {
    Symbol string `json:"symbol"`  // 合约代码
    KLineData
}
```

### 4.2 交易信号

```go
type Direction int

const (
    Buy Direction = iota       // 开多
    Sell                       // 开空
    Close                      // 平仓（通用）
    CloseShort                 // 平空
    CloseLong                  // 平多
)

type TradeSignal struct {
    SignalDate string          `json:"signal_date"`  // 信号日期
    Price      decimal.Decimal `json:"price"`        // 信号价格
    Direction  Direction       `json:"direction"`    // 交易方向
    Leverage   decimal.Decimal `json:"leverage"`     // 杠杆系数
    Quantity   decimal.Decimal `json:"quantity"`     // 交易数量
    SignalType string          `json:"signal_type"`  // 信号类型
    Symbol     string          `json:"symbol"`       // 合约代码
    OpenPrice  decimal.Decimal `json:"open_price"`   // 开仓价格
    OpenDate   string          `json:"open_date"`    // 开仓日期
}
```

### 4.3 持仓状态

```go
// 信号层持仓状态
type SignalPosition struct {
    Symbol    string          `json:"symbol"`     // 合约代码
    Direction Direction       `json:"direction"`  // 持仓方向
    OpenPrice decimal.Decimal `json:"open_price"` // 开仓价格
    OpenDate  string          `json:"open_date"`  // 开仓日期
    Leverage  decimal.Decimal `json:"leverage"`   // 杠杆系数
}

// 资金层持仓状态
type Position struct {
    Symbol       string          `json:"symbol"`        // 合约代码
    Direction    Direction       `json:"direction"`     // 持仓方向
    OpenPrice    decimal.Decimal `json:"open_price"`    // 开仓价格
    OpenDate     string          `json:"open_date"`     // 开仓日期
    Quantity     decimal.Decimal `json:"quantity"`      // 持仓数量
    Leverage     decimal.Decimal `json:"leverage"`      // 杠杆系数
    CurrentPrice decimal.Decimal `json:"current_price"` // 当前价格
}
```

### 4.4 阴阳线状态

```go
type YinYangElement struct {
    High    decimal.Decimal `json:"high"`     // 最高价
    Low     decimal.Decimal `json:"low"`      // 最低价
    IsValid bool            `json:"is_valid"` // 是否有效
}

type YinYangState struct {
    IsYang bool           `json:"is_yang"` // 当前方向（true=阳）
    Yang1  YinYangElement `json:"yang1"`   // 最近阳线元素
    Yin1   YinYangElement `json:"yin1"`    // 最近阴线元素
    Yang2  YinYangElement `json:"yang2"`   // 次近阳线元素
    Yin2   YinYangElement `json:"yin2"`    // 次近阴线元素
}
```

**重要**: 临时阴阳集合通过 `StateManager.GetTempState()` 方法获取，返回 `(YinYangState, bool)`，不是独立的结构体。

### 4.5 每日记录

```go
type DailyRecord struct {
    Date        string          `json:"date"`         // 日期
    Position    decimal.Decimal `json:"position"`     // 持仓市值
    Cash        decimal.Decimal `json:"cash"`         // 现金
    TotalValue  decimal.Decimal `json:"total_value"`  // 总资产
    PnL         decimal.Decimal `json:"pnl"`          // 当日盈亏
    DailyReturn decimal.Decimal `json:"daily_return"` // 日收益率
}

type PositionReturn struct {
    OpenDate   string          `json:"open_date"`   // 开仓日期
    CloseDate  string          `json:"close_date"`  // 平仓日期
    Symbol     string          `json:"symbol"`      // 合约代码
    Direction  Direction       `json:"direction"`   // 交易方向
    OpenPrice  decimal.Decimal `json:"open_price"`  // 开仓价格
    ClosePrice decimal.Decimal `json:"close_price"` // 平仓价格
    Leverage   decimal.Decimal `json:"leverage"`    // 杠杆系数
    Return     decimal.Decimal `json:"return"`      // 收益率
}
```

## 5. 修改代码指南

### 5.1 修改策略逻辑

**位置**: `internal/strategy/yinyang/`

**关键文件**:

- `strategy.go`: 策略主逻辑，处理K线生成信号，包含信号价格计算
- `state.go`: 阴阳集合状态管理，临时状态生成
- `adapter.go`: 策略适配器，将阴阳策略适配为通用接口
- `rollover.go`: 移仓换月处理

**注意事项**:

1. 修改信号价格计算时，确保区分持仓状态
2. 临时阴阳集合通过 `StateManager.GetTempState()` 获取
3. 移仓换月必须延迟执行

### 5.2 修改Web服务

**位置**: `internal/web/`

**关键文件**:

- `server.go`: HTTP服务器，API端点
- `config.go`: 配置加载
- `storage.go`: 结果存储与加载
- `portfolio.go`: 组合分析逻辑

**添加新API时**:

1. 在 `server.go` 中添加路由
2. 实现处理函数
3. 更新前端 `web/templates/` 下对应页面

### 5.3 修改数据层

**位置**: `internal/data/`

**关键文件**:

- `futures_data.go`: 期货数据获取与缓存
- `dominant.go`: 主力合约识别逻辑

**注意事项**:

1. 主力合约识别逻辑在 `dominant.go`
2. 数据缓存机制在 `futures_data.go`
3. Python交互通过 `pkg/pyexec`

### 5.4 添加新策略

**步骤**:

1. 创建 `internal/strategy/newstrategy/` 目录
2. 实现 `SignalStrategy` 接口
3. 创建 `StrategyFactory` 实现
4. 在 `factory.go` 的 `init()` 中注册:
   ```go
   func init() {
       DefaultRegistry.Register(&NewStrategyFactory{})
   }
   ```
5. 更新 `config/strategies.json`

## 6. 常见问题与陷阱

### 6.1 临时阴阳集合的状态方向

**问题**: 临时阴阳集合的 IsYang 与当前K线方向不同

**正确做法**:

```go
// 做多开仓 + 阴线 → 临时集合IsYang = true
// 做空开仓 + 阳线 → 临时集合IsYang = false
```

**错误做法**:

```go
// 错误：使用当前K线方向
isYang := kline.Close > kline.Open
```

### 6.2 无持仓的两种情况

**问题**: 无持仓有两种情况，信号价格计算不同

**正确做法**:

```go
if !hasEverHeldPosition {
    // 真正的无持仓：只使用阴1阳1
    longSignalPrice = max(阴1.High, 阳1.High)
    shortSignalPrice = min(阴1.Low, 阳1.Low)
} else {
    // 有持仓历史：使用阴1阳1和阴2阳2
    if currentIsYang {
        shortSignalPrice = min(阳1.Low, 阴2.Low)
    } else {
        shortSignalPrice = min(阳1.Low, 阴1.Low)
    }
}
```

### 6.3 移仓换月时序

**问题**: 移仓换月必须延迟执行

**正确做法**:

```
T日收盘后：
  1. 检测主力合约切换
  2. 记录待移仓信息
  3. 更新所有合约状态

T+1日开盘：
  1. 执行移仓换月
  2. 继续正常交易逻辑
```

**错误做法**:

```go
// 错误：在检测当天立即执行移仓
if dominantChanged {
    executeRollover()  // ❌ 违反时序
}
```

### 6.4 止损价 vs 反向信号价

**问题**: 止损价和反向信号价计算方式不同

**止损价**（开仓时确定）:

```go
// 开多
stopLossPrice = min(阴1.Low, 阳1.Low)
// 开空
stopLossPrice = max(阴1.High, 阳1.High)
```

**反向信号价**（持仓期间动态更新）:

```go
// 持有多头 + 阳线
reverseSignalPrice = min(阴1.Low, 阳1.Low)
// 持有多头 + 阴线
reverseSignalPrice = min(阳1.Low, 阴2.Low)
```

## 7. 测试与验证

### 7.1 运行回测

```bash
go run cmd/main.go -symbol RB -start 20260101 -end 20260421 -leverage 4
```

### 7.2 启动Web服务

```bash
go run cmd/web/main.go
```

访问: http://localhost:8080

### 7.3 API测试

```bash
# 执行回测
curl -X POST http://localhost:8080/api/backtest \
  -H "Content-Type: application/json" \
  -d '{
    "symbol": "RB",
    "start_date": "20260101",
    "end_date": "20260421",
    "leverage": 3.0,
    "strategy": "yinyang"
  }'

# 获取策略列表
curl http://localhost:8080/api/strategies

# 获取回测结果列表
curl http://localhost:8080/api/results

# 组合分析
curl -X POST http://localhost:8080/api/portfolio \
  -H "Content-Type: application/json" \
  -d '{
    "ids": ["RB_yinyang_20260101_20260421_3_1234567890", "JM_yinyang_20260101_20260421_3_1234567891"]
  }'
```

## 8. 代码风格

### 8.1 Go代码规范

- 使用 `gofmt` 格式化代码
- 遵循 Go 官方代码规范
- 使用 `decimal.Decimal` 处理价格计算，避免浮点误差
- 错误处理使用 `error` 返回值

### 8.2 命名约定

- 接口: `SignalStrategy`, `StrategyFactory`, `RolloverHandler`
- 实现: `YinYangStrategy`, `YinYangFactory`, `YinYangAdapter`
- 状态: `YinYangState`, `StateManager`
- 方法: 驼峰命名，首字母大写表示公开

### 8.3 注释规范

- 包注释: 在 `package` 语句前
- 函数注释: 在函数定义前
- 复杂逻辑: 行内注释说明意图

## 9. 依赖管理

### 9.1 Go依赖

```bash
go mod tidy
go mod download
```

### 9.2 Python依赖

```bash
pip install akshare
```

## 10. 文档资源

- [设计文档](docs/DESIGN.md): 系统架构和模块设计
- [策略说明](docs/STRATEGY.md): 阴阳线策略详细说明
- [需求文档](docs/REQUIREMENTS.md): 系统需求说明
- [策略配置](config/strategies.json): 策略参数配置
- [品种配置](config/symbols.json): 期货品种配置

## 11. 常用修改场景

### 11.1 修改策略参数

1. 更新 `config/strategies.json` 中的参数定义
2. 更新策略实现中的参数处理逻辑
3. 更新 `docs/STRATEGY.md` 文档

### 11.2 添加新的技术指标

1. 在 `internal/strategy/yinyang/state.go` 中添加指标计算
2. 更新 `YinYangState` 结构体
3. 修改信号价格计算逻辑

### 11.3 修改主力合约识别规则

1. 修改 `internal/data/dominant.go`
2. 更新 `docs/STRATEGY.md` 中的移仓换月说明

### 11.4 添加新的API端点

1. 在 `internal/web/server.go` 添加路由
2. 实现处理函数
3. 更新前端页面

### 11.5 添加新的期货品种

1. 更新 `config/symbols.json` 添加品种配置
2. 运行 `scripts/get_symbols.py` 验证品种代码
3. 测试数据获取和回测

## 12. 调试技巧

### 12.1 打印阴阳集合状态

```go
// 通过适配器访问策略特有方法
adapter := strategy.(*yinyang.YinYangAdapter)
state := adapter.State()
fmt.Printf("阳1: High=%s, Low=%s\n", state.Yang1.High, state.Yang1.Low)
fmt.Printf("阴1: High=%s, Low=%s\n", state.Yin1.High, state.Yin1.Low)
fmt.Printf("当前方向: %v\n", state.IsYang)
```

### 12.2 追踪信号价格计算

```go
// 通过适配器访问信号价格
adapter := strategy.(*yinyang.YinYangAdapter)
longPrice, shortPrice := adapter.SignalPrices()
fmt.Printf("做多信号价: %s, 做空信号价: %s\n", longPrice, shortPrice)
```

### 12.3 检查临时状态

```go
// 临时状态返回值是 (YinYangState, bool)
adapter := strategy.(*yinyang.YinYangAdapter)
if tempState, ok := adapter.TempState(); ok {
    fmt.Printf("临时状态方向: %v\n", tempState.IsYang)
    fmt.Printf("临时阳1: High=%s, Low=%s\n", tempState.Yang1.High, tempState.Yang1.Low)
}
```

### 12.4 查看持仓状态

```go
pos := strategy.Position()
if pos != nil {
    fmt.Printf("持仓: %s %s@%s 开仓日期: %s\n", 
        pos.Direction.String(), pos.Symbol, pos.OpenPrice.StringFixed(2), pos.OpenDate)
} else {
    fmt.Println("无持仓")
}
```

## 13. 性能优化

### 13.1 数据缓存

- K线数据缓存在内存中（`internal/data/futures_data.go`）
- 主力合约映射只计算一次
- 回测结果持久化到 `ret/` 目录

### 13.2 并发安全

- 策略工厂使用读写锁保护（`internal/strategy/factory.go`）
- 状态管理器按合约隔离（`internal/strategy/yinyang/state.go`）

## 14. 安全注意事项

- 不要在日志中输出敏感信息
- API参数验证使用 Gin 的 binding 标签
- 文件路径使用绝对路径
- Python脚本执行通过 `pkg/pyexec` 封装，避免命令注入

## 15. Web API 端点

### 15.1 回测相关

- `POST /api/backtest` - 执行回测
- `GET /api/results` - 列出所有回测结果
- `GET /api/results/:id` - 获取结果概要
- `GET /api/results/:id/data?type=...` - 获取结果详细数据
- `DELETE /api/results/:id` - 删除回测结果

### 15.2 策略与品种

- `GET /api/strategies` - 获取策略列表
- `GET /api/symbols?q=...` - 搜索品种

### 15.3 组合分析

- `POST /api/portfolio` - 执行组合分析

### 15.4 页面路由

- `GET /` - 回测主页
- `GET /portfolio` - 组合分析页面

---

**最后更新**: 2026-04-23
