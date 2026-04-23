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
│   └── main.go                 # 命令行入口
├── config/                     # 配置文件
│   └── strategies.json         # 策略配置
├── docs/                       # 文档
│   ├── DESIGN.md               # 设计文档
│   └── STRATEGY.md             # 策略说明文档
├── internal/                   # 内部包
│   ├── backtest/               # 回测核心
│   │   ├── signal.go           # 信号引擎
│   │   ├── portfolio.go        # 资金引擎
│   │   ├── data.go             # 数据管理
│   │   └── dominant.go         # 主力合约识别
│   ├── strategy/               # 策略层
│   │   ├── interface.go        # 策略接口定义
│   │   ├── factory.go          # 策略工厂
│   │   ├── yinyang_wrapper.go  # 阴阳策略包装器
│   │   └── yinyang/            # 阴阳线策略实现
│   │       ├── strategy.go     # 策略主逻辑
│   │       ├── state.go        # 状态管理
│   │       ├── signal.go       # 信号计算
│   │       └── rollover.go     # 移仓换月
│   └── web/                    # Web服务
│       ├── server.go           # HTTP服务器
│       └── config.go           # 配置加载
├── pkg/                        # 公共包
│   └── pyexec/                 # Python执行器
├── scripts/                    # Python脚本
│   └── get_futures_data.py     # 数据抓取脚本
├── web/                        # Web前端
│   └── templates/
│       └── index.html          # 主页面模板
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
  - 输出 `DailyRecord` 和 `TradeRecord`

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
3. 创建对应的 `StrategyFactory`
4. 在 `factory.go` 的 `init()` 中注册
5. 更新 `config/strategies.json`

## 4. 关键数据结构

### 4.1 K线数据

```go
type KLineWithContract struct {
    Date     time.Time
    Open     decimal.Decimal
    High     decimal.Decimal
    Low      decimal.Decimal
    Close    decimal.Decimal
    Volume   int64
    Hold     int64
    Symbol   string
}
```

### 4.2 交易信号

```go
type TradeSignal struct {
    Date     time.Time
    Symbol   string
    Action   string      // "Buy", "Sell", "CloseLong", "CloseShort"
    Price    decimal.Decimal
    Leverage decimal.Decimal
    Type     string      // "yinyang"
}
```

### 4.3 阴阳线状态

```go
type YinYangState struct {
    Yang1    YinYangElement  // 最近阳线元素
    Yin1     YinYangElement  // 最近阴线元素
    Yang2    YinYangElement  // 次近阳线元素
    Yin2     YinYangElement  // 次近阴线元素
    IsYang   bool            // 当前方向（true=阳）
}
```

### 4.4 临时阴阳集合

```go
type TempYinYangState struct {
    State    YinYangState    // 临时状态
    IsYang   bool            // 临时状态方向（关键！）
}
```

**重要**: 临时阴阳集合有自己的状态方向（IsYang），与当前K线实际方向不同。

## 5. 修改代码指南

### 5.1 修改策略逻辑

**位置**: `internal/strategy/yinyang/`

**关键文件**:

- `strategy.go`: 策略主逻辑，处理K线生成信号
- `state.go`: 阴阳集合状态管理
- `signal.go`: 信号价格计算
- `rollover.go`: 移仓换月处理

**注意事项**:

1. 修改信号价格计算时，确保区分持仓状态
2. 临时阴阳集合的状态方向（IsYang）必须正确设置
3. 移仓换月必须延迟执行

### 5.2 修改Web服务

**位置**: `internal/web/`

**关键文件**:

- `server.go`: HTTP服务器，API端点
- `config.go`: 配置加载

**添加新API时**:

1. 在 `server.go` 中添加路由
2. 实现处理函数
3. 更新前端 `web/templates/index.html`

### 5.3 修改数据层

**位置**: `internal/backtest/data.go`, `internal/backtest/dominant.go`

**注意事项**:

1. 主力合约识别逻辑在 `dominant.go`
2. 数据缓存机制在 `data.go`
3. Python交互通过 `pkg/pyexec`

### 5.4 添加新策略

**步骤**:

1. 创建 `internal/strategy/newstrategy/` 目录
2. 实现 `SignalStrategy` 接口
3. 创建 `StrategyFactory` 实现
4. 在 `factory.go` 注册:
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
isYang := kline.Close.GreaterThan(kline.Open)
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
```

## 8. 代码风格

### 8.1 Go代码规范

- 使用 `gofmt` 格式化代码
- 遵循 Go 官方代码规范
- 使用 `decimal.Decimal` 处理价格计算，避免浮点误差
- 错误处理使用 `error` 返回值

### 8.2 命名约定

- 接口: `SignalStrategy`, `StrategyFactory`
- 实现: `YinYangStrategy`, `YinYangFactory`
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
- [策略配置](config/strategies.json): 策略参数配置

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

1. 修改 `internal/backtest/dominant.go`
2. 更新 `docs/STRATEGY.md` 中的移仓换月说明

### 11.4 添加新的API端点

1. 在 `internal/web/server.go` 添加路由
2. 实现处理函数
3. 更新前端页面

## 12. 调试技巧

### 12.1 打印阴阳集合状态

```go
state := strategy.State()
fmt.Printf("阳1: High=%s, Low=%s\n", state.Yang1.High, state.Yang1.Low)
fmt.Printf("阴1: High=%s, Low=%s\n", state.Yin1.High, state.Yin1.Low)
```

### 12.2 追踪信号价格计算

```go
longPrice, shortPrice := strategy.SignalPrices()
fmt.Printf("做多信号价: %s, 做空信号价: %s\n", longPrice, shortPrice)
```

### 12.3 检查临时状态

```go
if tempState := strategy.TempState(); tempState != nil {
    fmt.Printf("临时状态方向: %v\n", tempState.IsYang)
}
```

## 13. 性能优化

### 13.1 数据缓存

- K线数据缓存在内存中
- 主力合约映射只计算一次

### 13.2 并发安全

- 策略工厂使用读写锁保护
- 状态管理器按合约隔离

## 14. 安全注意事项

- 不要在日志中输出敏感信息
- API参数验证使用 Gin 的 binding 标签
- 文件路径使用绝对路径

---

**最后更新**: 2026-04-23
