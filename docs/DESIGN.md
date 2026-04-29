# 期货回测系统设计文档

## 1. 架构概览

本项目采用 Go 语言作为核心逻辑层，利用 Python (`akshare` 库) 负责期货行情数据的抓取，两者通过 JSON 格式交换数据。

系统采用**信号生成与资金计算分离**的两层架构：
- **信号层**: 策略引擎逐根 K 线计算交易信号，不涉及资金。可直接对接实盘交易系统。
- **资金层**: 接收交易信号作为输入，独立计算保证金、盈亏、每日资金曲线和统计指标。

```
K线数据 → [信号层] → TradeSignal[] → [资金层] → DailyRecord[] / TradeRecord[]
                ↑                           ↑
          SignalPosition                Position
         (信号层持仓)                (资金层持仓)
```

## 2. 目录结构

```
d:\code\local\test\
├── cmd/                        # 入口程序
│   ├── main.go                 # 命令行回测入口
│   ├── dbcli/                  # 数据库命令行工具
│   │   └── main.go             # 查询/导出/迁移工具
│   └── web/                    # Web服务入口
│       └── main.go             # Web服务启动文件
├── config/                     # 配置文件（兼容保留）
│   ├── funds.json              # 基金配置（已迁移至数据库）
│   ├── strategies.json         # 策略配置（已迁移至数据库）
│   └── symbols.json            # 品种配置（已迁移至数据库）
├── db/                         # SQLite数据库文件
│   └── futures.db              # 主数据库（WAL模式）
├── docs/                       # 文档
│   ├── DESIGN.md               # 设计文档（本文档）
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
│   ├── fund/                   # 基金模式
│   │   ├── types.go            # 基金类型定义
│   │   ├── config.go           # 基金配置管理（从数据库加载）
│   │   ├── engine.go           # 基金回测引擎
│   │   ├── storage.go          # 基金结果存储（数据库+文件双写）
│   │   └── task.go             # 异步任务管理
│   ├── db/                     # 数据库访问层
│   │   ├── database.go         # 数据库初始化与连接管理
│   │   ├── symbols.go          # 品种CRUD与搜索
│   │   ├── strategies.go       # 策略CRUD与参数管理
│   │   ├── funds.go            # 基金CRUD与持仓管理
│   │   ├── results.go          # 回测结果/基金结果CRUD与导出
│   │   └── migrate.go          # JSON→数据库迁移工具
│   ├── strategy/               # 策略层
│   │   ├── interface.go        # 策略接口定义
│   │   ├── factory.go          # 策略工厂与注册
│   │   ├── ma/                 # 双均线策略实现
│   │   │   ├── strategy.go     # 策略主逻辑
│   │   │   ├── state.go        # 状态管理
│   │   │   ├── adapter.go      # 策略适配器
│   │   │   └── rollover.go     # 移仓换月
│   │   └── yinyang/            # 阴阳线策略实现
│   │       ├── strategy.go     # 策略主逻辑
│   │       ├── state.go        # 状态管理
│   │       ├── types.go        # 类型定义
│   │       ├── adapter.go      # 策略适配器
│   │       └── rollover.go     # 移仓换月
│   └── web/                    # Web服务
│       ├── server.go           # HTTP服务器与路由
│       ├── config.go           # 配置加载（数据库优先，自动同步）
│       ├── storage.go          # 结果存储（数据库优先+文件双写）
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
├── testret/                    # 基线测试结果目录
├── go.mod                      # Go模块定义
├── go.sum                      # 依赖版本锁定
└── AGENTS.md                   # AI开发指南
```

## 3. 核心接口

### 3.1 SignalStrategy 接口

```go
type SignalStrategy interface {
    ProcessKLine(kline KLineWithContract) []TradeSignal
    Position() *SignalPosition
    SetPosition(pos *SignalPosition)
    SetCurrentSymbol(symbol string)
    UpdateStateOnly(kline KLineWithContract)
}
```

### 3.2 StrategyFactory 接口

```go
type StrategyFactory interface {
    Create(params map[string]interface{}) SignalStrategy
    Name() string
    Description() string
    DisplayName() string
    GetParams() []StrategyParamConfig
    CreateRolloverHandler(strategy SignalStrategy) RolloverHandler
    CreateStateRecorder() StateRecorder
}
```

### 3.3 StateRecorder 接口

```go
type StateRecorder interface {
    RecordState(date string, kline KLineWithContract, position *SignalPosition)
    GetStateHistory() []StateRecord
}
```

### 3.4 RolloverHandler 接口

```go
type RolloverHandler interface {
    CheckAndExecute(currentSymbol, previousSymbol string, newKline, oldKline KLineWithContract, 
                    date string, newSymbolKlines []KLineWithContract) []TradeSignal
}
```

## 4. 策略工厂模式

系统采用策略工厂模式，支持动态注册和选择不同的交易策略。

### 4.1 工厂注册表

```go
type FactoryRegistry struct {
    factories map[string]StrategyFactory
    mu        sync.RWMutex
}

func init() {
    DefaultRegistry.Register(NewYinYangFactory())
    DefaultRegistry.Register(NewMAFactory())
}
```

### 4.2 已注册策略

| 策略名称 | 显示名称 | 描述 |
|---------|---------|------|
| yinyang | 阴阳线突破策略 | 基于阴阳线形态的趋势跟踪策略 |
| ma | 双均线交叉策略 | 基于双均线交叉的趋势跟踪策略 |

### 4.3 添加新策略步骤

1. 在 `internal/strategy/` 下创建新包
2. 实现 `SignalStrategy` 接口
3. 创建策略特有类型（如需要）
4. 创建适配器实现类型转换
5. 实现 `StrategyFactory` 接口的所有方法
6. 在 `factory.go` 的 `init()` 中注册
7. 更新 `config/strategies.json`

## 5. 策略实现

### 5.1 阴阳线突破策略 (yinyang)

**位置**: `internal/strategy/yinyang/`

**核心文件**:
- `strategy.go`: 策略主逻辑，处理K线生成信号
- `state.go`: 阴阳集合状态管理
- `types.go`: 策略特有类型定义
- `adapter.go`: 策略适配器
- `rollover.go`: 移仓换月处理

**核心类型**:
```go
type YinYangElement struct {
    High    float64
    Low     float64
    IsValid bool
}

type YinYangState struct {
    IsYang bool
    Yang1  YinYangElement
    Yin1   YinYangElement
    Yang2  YinYangElement
    Yin2   YinYangElement
}
```

**信号价格计算逻辑**:
- 无持仓（从未开仓）: 使用阴1阳1
- 无持仓（有持仓历史）: 使用阴1阳1和阴2阳2
- 持有多头: 根据当前K线方向选择反向信号价
- 持有空头: 根据当前K线方向选择反向信号价

**杠杆计算**:
- 开多: `杠杆 = leverageFactor * 开仓价格 / (开仓价格 - 止损价)`
- 开空: `杠杆 = leverageFactor * 开仓价格 / (止损价 - 开仓价格)`
- 最大杠杆限制: 6.0

### 5.2 双均线交叉策略 (ma)

**位置**: `internal/strategy/ma/`

**核心文件**:
- `strategy.go`: 策略主逻辑
- `state.go`: 均线状态管理
- `adapter.go`: 策略适配器
- `rollover.go`: 移仓换月处理

**核心逻辑**:
- 金叉（短期均线上穿长期均线）: 做多信号
- 死叉（短期均线下穿长期均线）: 做空信号
- 信号延迟执行: 在下一根K线开盘价执行

**参数**:
- `short_period`: 短期均线周期（默认5）
- `long_period`: 长期均线周期（默认20）
- `leverage`: 杠杆系数（默认1.0）

## 6. 数据层

### 6.1 DataManager

**位置**: `internal/data/futures_data.go`

**职责**:
- 封装对 Python 脚本的调用
- 维护 K 线数据的内存缓存
- 按合约代码分组存储 K 线数据

### 6.2 DominantContractIdentifier

**位置**: `internal/data/dominant.go`

**职责**: 识别主力合约，跟踪主力合约切换

**核心逻辑**:
1. 初始主力合约: 回测第一天，选择持仓量最大的合约
2. 单向切换: 合约只往后续月份切换，不往回切换
3. 切换条件: 后续月份合约的持仓量和成交量**同时**超过当前主力合约
4. 保持机制: 如果只有一项指标超过，保持当前主力合约不变

## 7. 回测引擎

### 7.1 SignalEngine

**位置**: `internal/backtest/signal.go`

**职责**: 核心信号处理流水线，协调策略执行和状态记录

**处理流程**:
```
for each date in sorted dates:
    1. 检查是否有待执行的移仓换月
    2. 检查主力合约是否发生变化
    3. 调用策略处理K线
    4. 记录状态历史
```

**时序保证**:
- K线按日期升序处理
- 移仓换月延迟执行
- 状态更新在信号判断之后

### 7.2 PortfolioEngine

**位置**: `internal/backtest/portfolio.go`

**职责**: 独立模块，接收交易信号进行资金计算

**核心计算**:
- 保证金: `保证金 = 开仓价 * 手数 * 合约乘数 / 杠杆`
- 盈亏: `盈亏 = (平仓价 - 开仓价) * 手数 * 合约乘数`
- 日收益率: `日收益率 = 当日盈亏 / 账户净值`

## 8. 核心数据结构

### 8.1 K线数据

```go
type KLineData struct {
    Date   string  `json:"date"`
    Open   float64 `json:"open"`
    High   float64 `json:"high"`
    Low    float64 `json:"low"`
    Close  float64 `json:"close"`
    Volume float64 `json:"volume"`
    Amount float64 `json:"amount"`
    Hold   float64 `json:"hold"`
    Settle float64 `json:"settle"`
}

type KLineWithContract struct {
    Symbol string
    KLineData
}
```

### 8.2 交易信号

```go
type Direction int

const (
    Buy Direction = iota       // 开多
    Sell                       // 开空
    Close                      // 平仓
    CloseShort                 // 平空
    CloseLong                  // 平多
)

type TradeSignal struct {
    SignalDate string
    Price      float64
    Direction  Direction
    Leverage   float64
    Quantity   float64
    SignalType string
    Symbol     string
    OpenPrice  float64
    OpenDate   string
}
```

### 8.3 持仓状态

```go
type SignalPosition struct {
    Symbol    string
    Direction Direction
    OpenPrice float64
    OpenDate  string
    Leverage  float64
}

type Position struct {
    Symbol       string
    Direction    Direction
    OpenPrice    float64
    OpenDate     string
    Quantity     float64
    Leverage     float64
    CurrentPrice float64
}
```

### 8.4 每日记录

```go
type DailyRecord struct {
    Date        string
    Position    float64
    Cash        float64
    TotalValue  float64
    PnL         float64
    DailyReturn float64
}

type PositionReturn struct {
    OpenDate   string
    CloseDate  string
    Symbol     string
    Direction  Direction
    OpenPrice  float64
    ClosePrice float64
    Leverage   float64
    Return     float64
}
```

## 9. Web API

### 9.1 回测相关

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/backtest | 执行回测 |
| GET | /api/results | 列出所有回测结果 |
| GET | /api/results/:id | 获取结果概要 |
| GET | /api/results/:id/data | 获取结果详细数据 |
| DELETE | /api/results/:id | 删除回测结果 |

### 9.2 策略与品种

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | /api/strategies | 获取策略列表 |
| GET | /api/symbols | 搜索品种 |

### 9.3 组合分析

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | /api/portfolio | 执行组合分析 |

## 10. 基金模式架构

### 10.1 概述

基金模式支持将多个品种按照指定权重组合为一个基金进行回测。基金回测基于净值计算，不依赖初始资金。

### 10.2 核心模块

**位置**: `internal/fund/`

| 文件 | 职责 |
|------|------|
| types.go | 基金配置、回测结果、统计指标等类型定义 |
| config.go | 基金配置的加载、保存、查询，使用 `sync.Once` 保证线程安全 |
| engine.go | 基金回测引擎，并发执行品种回测并合并结果 |
| storage.go | 基金回测结果的文件存储与加载 |

### 10.3 数据流

```
FundConfig → [FundEngine.RunBacktest] → FundResult
                    │
                    ├── 并发执行各品种回测（通过 channel 收集结果）
                    ├── 按权重加权合并各品种净值
                    └── 计算基金整体统计指标
```

### 10.4 基金配置结构

```go
type FundConfig struct {
    ID          string           `json:"id"`
    Name        string           `json:"name"`
    Description string           `json:"description"`
    StartDate   string           `json:"start_date"`
    EndDate     string           `json:"end_date"`
    Positions   []FundPosition   `json:"positions"`
}

type FundPosition struct {
    Symbol   string                 `json:"symbol"`
    Strategy string                 `json:"strategy"`
    Weight   float64                `json:"weight"`
    Params   map[string]interface{} `json:"params"`
}
```

### 10.5 结果存储结构

```
ret/
├── funding/                        # 基金回测结果（独立目录）
│   └── {fund_id}/                  # 按基金ID分组
│       └── {result_id}/            # 按回测结果ID分组
│           ├── fund_result.json    # 基金整体结果
│           └── positions/          # 各品种详细结果
│               ├── {symbol1}.json
│               └── {symbol2}.json
├── {backtest_id}.json              # 单品种回测结果
└── ...
```

### 10.6 并发安全

- 品种回测使用 goroutine 并发执行，结果通过 buffered channel 收集。
- 基金配置加载使用 `sync.Once` 确保只加载一次。
- 配置读写使用 `sync.RWMutex` 保护。
- 年化收益计算使用 `math.Pow` 进行幂运算。

## 11. Web前端架构

### 11.1 单页应用（SPA）

系统前端采用 SPA 架构，所有功能模块集成在 `web/templates/index.html` 中。

### 11.2 页面结构

```
┌──────────────────────────────────────────────┐
│ 侧边栏 (固定)  │  内容区域                    │
│ ┌────────────┐ │ ┌──────────────────────────┐ │
│ │ FT 期货回测 │ │ │ 页面标题 + 描述          │ │
│ ├────────────┤ │ ├──────────────────────────┤ │
│ │ ▶ 回测     │ │ │                          │ │
│ │   组合分析 │ │ │  功能卡片区域             │ │
│ │   基金     │ │ │  (表单/图表/表格)         │ │
│ ├────────────┤ │ │                          │ │
│ │ ▶ 收起     │ │ │                          │ │
│ └────────────┘ │ └──────────────────────────┘ │
└──────────────────────────────────────────────┘
```

### 11.3 导航切换

- 点击导航栏选项时，通过 `switchPage()` 函数切换显示对应的 `page-section`。
- 使用 CSS `display: none/block` 控制页面区域可见性。
- 切换时自动触发目标模块的数据加载（如组合分析加载结果列表，基金加载基金列表）。

### 11.4 技术栈

- **Chart.js 4.x**: 图表渲染（资金曲线、回撤、日收益、品种对比）
- **Fetch API**: 与后端 REST API 通信
- **CSS Custom Properties**: 主题色彩系统（深色主题）
- **CSS Grid/Flexbox**: 响应式布局

### 11.5 图表配置

所有图表使用双Y轴配置：
- 左Y轴：净值/资金曲线
- 右Y轴：回撤百分比（无 `max: 0` 限制，允许正方向显示回撤值）

API 返回的数值为字符串格式（如 `"0.2195"`），前端使用 `safeParseFloat()` 函数安全解析。

## 12. 数据库层

### 12.1 概述

系统使用 SQLite 数据库存储配置和回测结果，替代原有的 JSON 文件存储方式。数据库驱动采用纯 Go 实现的 `modernc.org/sqlite`，无需 CGO 依赖。

### 12.2 数据库文件

- 默认路径: `db/futures.db`
- WAL 模式: 提升并发读写性能
- 忙等待: 5000ms

### 12.3 表结构

| 表名 | 用途 | 替代的JSON文件 |
|------|------|---------------|
| `symbols` | 交易品种配置 | config/symbols.json |
| `strategies` | 策略配置 | config/strategies.json |
| `strategy_params` | 策略参数（子表） | strategies.json中的params |
| `funds` | 基金配置 | config/funds.json |
| `fund_positions` | 基金持仓（子表） | funds.json中的positions |
| `backtest_results` | 单品种回测结果 | ret/*.json |
| `fund_results` | 基金回测结果 | ret/funding/目录 |
| `config_meta` | 元数据键值对 | 默认策略等配置 |

### 12.4 数据访问层

**位置**: `internal/db/`

| 文件 | 职责 |
|------|------|
| database.go | 数据库初始化、建表、连接管理（InitDB/ResetDB/CloseDB） |
| symbols.go | 品种CRUD、批量Upsert、搜索（支持代码/名称/拼音/交易所） |
| strategies.go | 策略CRUD、参数管理、配置元数据 |
| funds.go | 基金CRUD、持仓管理、权重验证 |
| results.go | 回测结果/基金结果CRUD、JSON/CSV导出 |
| migrate.go | JSON→数据库迁移工具 |

### 12.5 存储策略

采用**数据库优先+文件双写**的过渡策略：

- **写入**: 同时写入数据库和文件（文件写入失败不影响主流程）
- **读取**: 优先从数据库读取，数据库无数据时回退到文件
- **列表**: 优先从数据库查询，数据库为空时回退到文件系统

### 12.6 命令行工具

**位置**: `cmd/dbcli/main.go`

```bash
go run ./cmd/dbcli/... tables                    # 列出所有表
go run ./cmd/dbcli/... symbols                   # 列出品种
go run ./cmd/dbcli/... strategies                # 列出策略
go run ./cmd/dbcli/... funds                     # 列出基金
go run ./cmd/dbcli/... results                   # 列出回测结果
go run ./cmd/dbcli/... export <表名> json        # 导出为JSON
go run ./cmd/dbcli/... export <表名> csv         # 导出为CSV
go run ./cmd/dbcli/... migrate                   # 从JSON迁移数据
go run ./cmd/dbcli/... delete <表名> <ID>        # 删除记录
```

## 13. 关键算法

### 13.1 K线处理时序

```
处理K线T时：
  Step 1: 使用T-1日计算的信号价格
  Step 2: 判断T日是否触发交易
  Step 3: 更新状态（使用T日数据）
  Step 4: 计算T+1日的信号价格
```

**禁止事项**:
- 在触发判断前更新状态
- 使用T+1日的数据影响T日的决策
- 在移仓检测日立即执行移仓

### 13.2 移仓换月延迟执行

```
T日收盘后：
  1. 检测主力合约切换
  2. 记录待移仓信息
  3. 更新所有合约状态

T+1日开盘：
  1. 执行移仓换月
  2. 继续正常交易逻辑
```

---

**最后更新**: 2026-04-30
