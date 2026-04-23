# 期货回测系统设计文档 (Design)

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

## 2. 模块设计

### 2.1 `cmd/main.go` (入口)
- **职责**: 接收命令行参数（-symbol, -start, -end, -leverage）。
- **流程**:
    1. 初始化 `DataManager` 获取所有 K 线和主力合约。
    2. 创建 `SignalEngine` 调用策略生成交易信号。
    3. 调用 `Reporter` 输出交易信号和阴阳线状态历史。
    4. （可选）创建 `PortfolioEngine` 根据信号模拟资金运作。

### 2.2 `internal/data/` (数据层)

#### DataManager
- 封装对 Python 脚本的调用，维护 K 线数据的内存缓存。
- 按合约代码分组存储 K 线数据。
- 提供按日期范围获取 K 线的接口。

#### DominantContractIdentifier
**职责**: 识别主力合约，跟踪主力合约切换。

**核心逻辑**：
1. **初始主力合约**：回测第一天，选择持仓量最大的合约
2. **单向切换**：合约只往后续月份切换，不往回切换
3. **切换条件**：当后续月份合约的持仓量和成交量**同时**超过当前主力合约时，切换
4. **保持机制**：如果只有一项指标超过，保持当前主力合约不变

**核心方法**：
- `Identify(product, allKlines, startDate, endDate)`: 识别主力合约
- `findInitialDominantContract(klines)`: 找初始主力合约（持仓量最大）
- `findSwitchDominantContract(klines, currentSymbol, currentVolume, currentHold)`: 检查切换条件
- `isLaterContract(newSymbol, currentSymbol)`: 判断是否为后续月份合约
- `extractYearMonth(symbol)`: 从合约代码提取年月

**输出格式**：`map[time.Time]string`，key 为日期，value 为主力合约代码。

#### Python 交互
- 通过 `pkg/pyexec` 执行 `scripts/get_futures_data.py`。
- JSON 格式数据交换，确保跨语言数据一致性。

### 2.3 `internal/strategy/` (策略层 — 信号层)

#### 策略工厂模式
系统采用策略工厂模式，支持动态注册和选择不同的交易策略。

**核心接口**：
```go
type StrategyFactory interface {
    Create(params map[string]interface{}) SignalStrategy
    Name() string
    Description() string
}
```

**策略注册表**：
```go
type FactoryRegistry struct {
    factories map[string]StrategyFactory
}
```

**已注册策略**：
| 策略名称 | 描述 |
|---------|------|
| yinyang | 阴阳线突破策略 |

**策略配置文件**: `config/strategies.json`
```json
{
  "strategies": [
    {
      "name": "yinyang",
      "display_name": "阴阳线突破策略",
      "description": "基于阴阳线形态的趋势跟踪策略",
      "params": [...],
      "enabled": true
    }
  ],
  "default_strategy": "yinyang"
}
```

#### SignalStrategy 接口
```go
type SignalStrategy interface {
    ProcessKLine(kline KLineWithContract) []TradeSignal
    Position() *SignalPosition
    SetPosition(pos *SignalPosition)
    SetCurrentSymbol(symbol string)
    UpdateStateOnly(kline KLineWithContract)
}
```

#### StateManager (阴阳线策略)
**职责**: 维护单个合约的阴阳线元素状态。

**核心字段**：
```go
type StateManager struct {
    state          YinYangState   // 当前阴阳集合状态
    tempState      YinYangState   // 临时阴阳集合状态
    hasTempState   bool           // 是否存在临时状态
    tempUsed       bool           // 临时状态是否已使用
    prevState      YinYangState   // 更新前的状态快照
    prevDir        bool           // 更新前的方向（true=阳）
    currentIsYang  bool           // 当前 K 线的实际方向
}
```

**核心方法**：
- `Update(kline)`: 更新阴阳集合状态
- `State()`: 获取当前阴阳集合状态
- `GetTempState()`: 获取临时阴阳集合状态
- `GenerateTempState(isYangOverride, high, low)`: 生成临时阴阳集合
- `MarkTempUsed()`: 标记临时状态已使用
- `CurrentIsYang()`: 获取当前K线方向

**临时阴阳集合的状态方向（关键）**：
临时阴阳集合有自己的状态方向（IsYang），与当前K线的实际方向不同：

| 场景 | 当前K线实际方向 | 临时集合状态方向（IsYang） |
|------|----------------|--------------------------|
| 做多 + 阴线 | 阴线（false） | 阳线（true） |
| 做空 + 阳线 | 阳线（true） | 阴线（false） |

**信号价格计算时**：
- 如果存在临时阴阳集合，使用临时集合的状态方向（IsYang）
- 如果不存在临时阴阳集合，使用当前K线的实际方向

#### YinYangStrategy
**职责**: 实现阴阳线突破交易策略。

**核心字段**：
```go
type YinYangStrategy struct {
    stateManagers      map[string]*StateManager  // 多合约状态管理
    currentSymbol      string                    // 当前主力合约
    position           *SignalPosition           // 当前持仓
    longSignalPrice    decimal.Decimal           // 做多信号价
    shortSignalPrice   decimal.Decimal           // 做空信号价
    reverseSignalPrice decimal.Decimal          // 反向信号价
    leverageFactor     decimal.Decimal           // 杠杆系数
    hasEverHeldPosition bool                     // 是否曾经开过仓（关键）
}
```

**hasEverHeldPosition 字段说明**：
- 用于区分"真正的无持仓"和"有持仓历史但当前无持仓"
- 当回测开始从未开过仓时，此字段为 false
- 一旦开仓成功，此字段设置为 true
- 影响无持仓时的信号价格计算方式

**核心方法**：
- `ProcessKLine(kline)`: 处理单根K线，返回交易信号
- `UpdateStateOnly(kline)`: 仅更新状态，不生成信号
- `State()`: 获取当前合约的阴阳集合状态
- `StateForSymbol(symbol)`: 获取指定合约的阴阳集合状态
- `SetCurrentSymbol(symbol)`: 切换当前主力合约
- `SignalPrices()`: 获取当前合约的信号价格
- `SignalPricesForSymbol(symbol)`: 获取指定合约的信号价格
- `TempState()`: 获取临时阴阳集合状态
- `SimulateTrading(klines)`: 模拟交易，用于移仓换月时确定新合约方向

**信号价格计算逻辑**：
```
无持仓时（需要区分两种情况）：

情况1：真正的无持仓（hasEverHeldPosition = false）
  做多信号价 = max(阴1.High, 阳1.High)
  做空信号价 = min(阴1.Low, 阳1.Low)
  // 只使用阴1阳1，不使用阴2阳2

情况2：有持仓历史但当前无持仓（hasEverHeldPosition = true）
  如果当前K线为阳线：
    做多信号价 = max(阴1.High, 阳1.High)
    做空信号价 = min(阳1.Low, 阴2.Low)  // 使用阴2
  如果当前K线为阴线：
    做多信号价 = max(阴1.High, 阳2.High)  // 使用阳2
    做空信号价 = min(阴1.Low, 阳1.Low)

持有多头时（做空信号价）：
  当前K线为阳线 → min(阳1.Low, 阴1.Low)
  当前K线为阴线 → min(阳1.Low, 阴2.Low)

持有空头时（做多信号价）：
  当前K线为阳线 → max(阴1.High, 阳2.High)
  当前K线为阴线 → max(阴1.High, 阳1.High)
```

**杠杆计算逻辑**：
```
开多时：
  止损价 = min(阴1.Low, 阳1.Low)
  杠杆 = leverageFactor * 开仓价格 / (开仓价格 - 止损价)

开空时：
  止损价 = max(阴1.High, 阳1.High)
  杠杆 = leverageFactor * 开仓价格 / (止损价 - 开仓价格)

限制：最大杠杆 = 6.0
```

**注意**：杠杆计算使用的止损价固定为阴1阳1的高低点，与持仓期间动态更新的反向信号价不同。

#### RolloverHandler
**职责**: 处理主力合约切换日的移仓换月逻辑。

**核心方法**：
- `CheckAndExecute(currentSymbol, previousSymbol, newKline, oldKline, date, newSymbolKlines)`: 
  检查并执行移仓换月

**移仓换月流程**：
```
1. 使用旧合约开盘价平仓
2. 获取新合约历史K线
3. 调用SimulateTrading模拟交易
4. 根据模拟结果确定新合约方向
5. 使用新合约开盘价开仓
6. 更新策略持仓状态
```

#### YinYangStateRecorder
**职责**: 记录每日阴阳线状态快照。

**核心方法**：
- `RecordState()`: 记录普通状态
- `RecordRolloverState()`: 记录移仓换月时的状态（包含新旧合约信息）
- `GetStateHistory()`: 获取状态历史记录

### 2.4 `internal/backtest/` (回测引擎层)

#### 信号引擎 (SignalEngine)
**职责**: 核心信号处理流水线，协调策略执行和状态记录。

**核心字段**：
```go
type SignalEngine struct {
    strategy         SignalStrategy    // 策略实例
    rolloverHandler  RolloverHandler   // 移仓换月处理器
    dominantMap      map[string]string // 日期->主力合约映射
    klinesBySymbol   map[string][]KLineWithContract  // 合约->K线映射
    pendingRollover  *pendingRolloverInfo            // 待执行移仓信息
    stateRecorder    StateRecorder                   // 状态记录器
}
```

**核心方法**：
- `Calculate()`: 计算所有交易信号
- `processKLine()`: 处理单根K线

**处理流程**：
```
for each date in sorted dates:
    1. 获取当日主力合约K线
    2. 检查是否有待执行的移仓换月
       - 如果有，执行移仓换月
       - 记录移仓换月状态
    3. 检查主力合约是否发生变化
       - 如果变化，记录待移仓信息
       - 更新所有合约状态
       - 记录状态历史
    4. 调用策略处理K线
    5. 记录状态历史
```

**时序保证**：
- K线按日期升序处理
- 移仓换月延迟执行
- 状态更新在信号判断之后

#### 资金引擎 (PortfolioEngine)
**职责**: 独立模块，接收交易信号进行资金计算。

**核心计算**：
- 保证金: `保证金 = 开仓价 * 手数 * 合约乘数 / 杠杆`
- 盈亏: `盈亏 = (平仓价 - 开仓价) * 手数 * 合约乘数`
- 日收益率: `日收益率 = 当日盈亏 / 账户净值`

#### 规则引擎 (RuleManager)
**职责**: 支持规则化的回测约束校验。

**内置规则**：
- 最大回撤限制
- 杠杆限制
- 资金下限
- 持仓上限

#### 结果展示 (Reporter)
**职责**: 输出交易信号和阴阳线状态历史。

**输出内容**：
- 交易信号列表
- 阴阳线状态历史
- 临时阴阳集合记录
- 主力切换信息

## 3. 核心数据结构

### 3.1 SignalPosition (信号层持仓)
```go
type SignalPosition struct {
    Symbol    string          // 合约代码
    Direction Direction       // 持仓方向 (Buy/Sell)
    OpenPrice decimal.Decimal // 开仓价格
    OpenDate  string          // 开仓日期
    Leverage  decimal.Decimal // 杠杆倍数
}
```

### 3.2 Position (资金层持仓)
```go
type Position struct {
    Symbol       string    // 合约代码
    Direction    Direction // 持仓方向
    OpenPrice    float64   // 开仓价格
    OpenDate     string    // 开仓日期
    Quantity     float64   // 持仓数量
    Leverage     float64   // 杠杆倍数
    CurrentPrice float64   // 当前价格
}
```

### 3.3 TradeSignal (交易信号)
```go
type TradeSignal struct {
    SignalDate string          // 信号日期
    Price      decimal.Decimal // 信号价格
    Direction  Direction       // 交易方向
    Leverage   decimal.Decimal // 杠杆倍数
    Quantity   float64         // 数量
    SignalType string          // 信号类型 (yinyang / rollover)
    Symbol     string          // 合约代码
    OpenPrice  decimal.Decimal // 开仓价格
    OpenDate   string          // 开仓日期
}
```

### 3.4 YinYangState (阴阳线状态)
```go
type YinYangState struct {
    Yang1  YinYangElement // 阳1元素
    Yin1   YinYangElement // 阴1元素
    Yang2  YinYangElement // 阳2元素
    Yin2   YinYangElement // 阴2元素
    IsYang bool           // 当前是否为阳线
}

type YinYangElement struct {
    High    decimal.Decimal // 最高价
    Low     decimal.Decimal // 最低价
    IsValid bool            // 是否有效
}
```

### 3.5 StateRecord (状态记录)
```go
type StateRecord struct {
    Date          string         // 日期
    Symbol        string         // 合约
    Yang1         YinYangElement // 阳1
    Yin1          YinYangElement // 阴1
    Yang2         YinYangElement // 阳2
    Yin2          YinYangElement // 阴2
    LongPrice     decimal.Decimal // 做多信号价
    ShortPrice    decimal.Decimal // 做空信号价
    Position      string         // 持仓描述
    HasTemp       bool           // 是否有临时阴阳集合
    TempYang1     YinYangElement // 临时阳1
    TempYin1      YinYangElement // 临时阴1
    TempYang2     YinYangElement // 临时阳2
    TempYin2      YinYangElement // 临时阴2
    IsRollover    bool           // 是否为移仓换月记录
    OldSymbol     string         // 旧合约代码
    OldYang1      YinYangElement // 旧合约阳1
    OldYin1       YinYangElement // 旧合约阴1
    OldYang2      YinYangElement // 旧合约阳2
    OldYin2       YinYangElement // 旧合约阴2
    OldLongPrice  decimal.Decimal // 旧合约做多信号价
    OldShortPrice decimal.Decimal // 旧合约做空信号价
}
```

### 3.6 StrategyConfig (策略配置)
```go
type StrategyConfig struct {
    Name        string                `json:"name"`
    DisplayName string                `json:"display_name"`
    Description string                `json:"description"`
    Params      []StrategyParamConfig `json:"params"`
    Enabled     bool                  `json:"enabled"`
}

type StrategyParamConfig struct {
    Name        string  `json:"name"`
    DisplayName string  `json:"display_name"`
    Type        string  `json:"type"`
    Default     float64 `json:"default"`
    Min         float64 `json:"min"`
    Max         float64 `json:"max"`
    Description string  `json:"description"`
}
```

## 4. 核心接口 (Interfaces)

### 4.1 `SignalStrategy` (信号层策略接口)
```go
type SignalStrategy interface {
    ProcessKLine(kline KLineWithContract) []TradeSignal
    Position() *SignalPosition
    SetPosition(pos *SignalPosition)
    SetCurrentSymbol(symbol string)
    UpdateStateOnly(kline KLineWithContract)
}
```

### 4.2 `StrategyFactory` (策略工厂接口)
```go
type StrategyFactory interface {
    Create(params map[string]interface{}) SignalStrategy
    Name() string
    Description() string
}
```

### 4.3 `StateRecorder` (状态记录器接口)
```go
type StateRecorder interface {
    RecordState(date string, kline KLineWithContract, position *SignalPosition)
    GetStateHistory() []StateRecord
}
```

### 4.4 `RolloverHandler` (移仓换月处理器接口)
```go
type RolloverHandler interface {
    CheckAndExecute(currentSymbol, previousSymbol string, newKline, oldKline KLineWithContract, 
                    date string, newSymbolKlines []KLineWithContract) []TradeSignal
}
```

## 5. 关键算法与时序

### 5.1 K线处理时序（核心）
```
输入: KLine_T (第T天的K线数据)

Step 1: 处理临时阴阳集合（如果有）
  - 如果存在上一根K线生成的临时阴阳集合
  - 使用临时集合的状态方向（IsYang）计算信号价格
  - 注意：不是使用当前K线的实际方向

Step 2: 触发判断
  - 使用Step 1计算的信号价格
  - 判断KLine_T是否触发成交
  - 判断逻辑：High >= 做多信号价 或 Low <= 做空信号价

Step 3: 标记临时状态
  - 如果存在临时阴阳集合，标记为已使用（tempUsed = true）
  - 注意：必须在触发判断之后才能标记

Step 4: 更新阴阳集合状态
  - 使用KLine_T的数据更新阴阳集合
  - 判断K线方向（阳线/阴线/十字星）
  - 执行元素合并或降级

Step 5: 成交后处理
  如果成交：
    - 生成交易信号
    - 判断是否生成临时阴阳集合
    - 计算反向信号价
  如果不成交：
    - 更新信号价格
```

### 5.2 阴阳集合更新算法
```
Update(kline):
  1. 判断K线方向：
     - Close > Open → 阳线
     - Close < Open → 阴线
     - Close == Open → 十字星（沿用上一方向）

  2. 保存当前状态快照：
     prevState = state
     prevDir = currentIsYang

  3. 方向变化处理：
     如果方向发生变化（阳→阴 或 阴→阳）：
       - 旧的Yang1降级为Yang2
       - 旧的Yin1降级为Yin2
       - 新K线成为新的Yang1或Yin1
     
     如果方向相同：
       - 合并到对应的Yang1或Yin1
       - 更新最高价和最低价
```

### 5.3 临时阴阳集合生成算法
```
GenerateTempState(isYangOverride, high, low):
  1. tempState = state（使用更新后的状态）

  2. 撤销当前K线对正常阴阳集合的影响：
     如果当前K线方向与之前方向不同：
       - 恢复prevState的对应元素
     如果当前K线方向与之前方向相同：
       - 从当前元素中移除当前K线的影响

  3. 把当前K线当成相反方向处理：
     如果isYangOverride == prevDir：
       - 合入对应元素（更新High/Low）
     如果isYangOverride != prevDir：
       - 创建新元素
```

### 5.4 移仓换月延迟执行
```
T日收盘后：
  1. 检测主力合约是否切换
     dominant(T) != dominant(T-1)
  
  2. 如果切换：
     - 记录待移仓信息（pendingRollover）
     - 更新所有合约的阴阳集合状态
     - 记录状态历史（包含新旧合约信息）

T+1日开盘：
  1. 检查是否有待执行的移仓换月
  
  2. 如果有：
     a. 使用旧合约开盘价平仓
     b. 获取新合约历史K线
     c. 调用SimulateTrading模拟交易
     d. 根据模拟结果确定新合约方向
     e. 使用新合约开盘价开仓
     f. 更新策略持仓状态
  
  3. 继续处理T+1日的正常交易逻辑
```

### 5.5 模拟交易算法
```
SimulateTrading(klines):
  1. 创建新的策略实例（独立状态）
  
  2. 逐根处理历史K线：
     for each kline in klines:
       signals = strategy.ProcessKLine(kline)
       if len(signals) > 0:
         执行交易信号
         更新持仓状态
  
  3. 返回最终持仓状态：
     - Direction: 持仓方向
     - OpenPrice: 开仓价格
     - OpenDate: 开仓日期
```

### 5.6 信号-资金分离流程
```
1. 信号生成阶段：
   SignalEngine.Calculate() → []TradeSignal
   - 只涉及信号价格计算
   - 不涉及资金计算
   - 输出纯交易信号

2. 资金计算阶段（独立）：
   PortfolioEngine.Calculate(signals, klines, initialCash)
   - 接收交易信号作为输入
   - 计算保证金、盈亏
   - 生成每日资金记录

3. 结果输出阶段：
   Reporter输出信号和状态
   资金层输出每日记录和统计
```

## 6. 文件结构
```
cmd/
  main.go                                # 命令行入口
  web/main.go                            # Web 服务入口
config/
  symbols.json                           # 品种配置
  strategies.json                        # 策略配置
internal/
  backtest/
    types.go                             # 核心数据结构
    signal.go                            # SignalEngine, StateRecorder 接口
    portfolio.go                         # PortfolioEngine 资金计算引擎
    stats.go                             # 统计指标计算
    reporter.go                          # Reporter 结果展示
  strategy/
    interface.go                         # 策略接入标准接口
    factory.go                           # 策略工厂注册表
    yinyang_wrapper.go                   # 阴阳线策略包装器
    yinyang/
      strategy.go                        # YinYangStrategy 阴阳线突破策略
      state.go                           # StateManager 阴阳线状态管理
      rollover.go                        # RolloverHandler 移仓换月处理器
      adapter.go                         # YinYangAdapter 策略适配器
  data/
    futures_data.go                      # DataManager 数据管理
    dominant.go                          # DominantContractIdentifier 主力合约识别
  web/
    server.go                            # Web 服务器和 API 端点
    config.go                            # 配置管理
    storage.go                           # 回测结果存储
    portfolio.go                         # 组合分析引擎
web/
  templates/
    index.html                           # 主页 UI
    portfolio.html                       # 组合分析 UI
scripts/
  get_futures_data.py                    # Python数据获取脚本
  get_symbols.py                         # Python品种列表获取脚本
pkg/
  pyexec/                                # Python执行封装
ret/                                     # 回测结果存储目录
```

## 7. 开发与部署
- **语言版本**: Go 1.20+, Python 3.8+
- **依赖库**: `akshare` (Python), `shopspring/decimal` (Go), `gin-gonic/gin` (Go)
- **运行方式**: 
  - 命令行: `go run cmd/main.go -symbol RB -start 20260101 -end 20260415 -leverage 4`
  - Web服务: `go run cmd/web/main.go`

## 8. Web 服务架构

### 8.1 服务概述
系统提供 Web 服务接口，支持通过 UI 进行回测操作和结果可视化。

**入口**: `cmd/web/main.go`
**端口**: 8080

### 8.2 API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/` | GET | 主页 UI |
| `/portfolio` | GET | 组合分析 UI |
| `/api/symbols` | GET | 获取品种列表 |
| `/api/strategies` | GET | 获取策略列表 |
| `/api/backtest` | POST | 执行回测 |
| `/api/results` | GET | 列出所有结果 |
| `/api/results/:id` | GET | 获取结果概要 |
| `/api/results/:id/data` | GET | 获取结果详细数据 |
| `/api/results/:id` | DELETE | 删除结果 |
| `/api/portfolio` | POST | 组合分析 |

### 8.3 回测请求参数
```json
{
  "symbol": "RB",
  "start_date": "20260101",
  "end_date": "20260421",
  "leverage": 3.0,
  "strategy": "yinyang"
}
```

### 8.4 回测结果存储
- 存储目录: `ret/`
- 文件格式: JSON
- 文件命名: `{symbol}_{strategy}_{startDate}_{endDate}_{leverage}_{timestamp}.json`

### 8.5 组合分析功能

**职责**: 支持选择多个同时段的回测结果，计算组合收益和风险指标。

## 9. 扩展策略

### 9.1 添加新策略
1. 实现 `SignalStrategy` 接口
2. 创建策略工厂实现 `StrategyFactory` 接口
3. 在 `init()` 中注册到 `DefaultRegistry`
4. 更新 `config/strategies.json` 配置文件

### 9.2 策略示例
```go
type MyStrategy struct {
    // 策略字段
}

func (s *MyStrategy) ProcessKLine(kline KLineWithContract) []TradeSignal {
    // 实现信号生成逻辑
}

type MyStrategyFactory struct{}

func (f *MyStrategyFactory) Name() string {
    return "mystrategy"
}

func (f *MyStrategyFactory) Create(params map[string]interface{}) SignalStrategy {
    return &MyStrategy{}
}

func init() {
    strategy.DefaultRegistry.Register(&MyStrategyFactory{})
}
```
