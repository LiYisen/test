# AGENTS.md - AI Agent 开发指南

本文档为 AI 编程助手提供项目开发指南，确保代码修改符合项目架构和规范。

## 1. 项目概述

期货回测系统，采用 Go 语言实现核心回测逻辑，Python 负责数据抓取。系统采用**信号生成与资金计算分离**的两层架构。

```
K线数据 → [信号层] → TradeSignal[] → [资金层] → DailyRecord[] / TradeRecord[]
```

**技术栈**: Go 1.21+ / Python 3.x + akshare / Gin框架 / HTML + JavaScript

## 2. 核心设计原则

### 2.1 信号-资金分离

- **信号层** (`internal/strategy/`): 只负责生成交易信号，不涉及资金计算
- **资金层** (`internal/backtest/portfolio.go`): 接收信号，计算保证金、盈亏、资金曲线

### 2.2 时序性要求

**禁止使用未来函数**:

```
处理K线T时：
  Step 1: 使用T-1日计算的信号价格
  Step 2: 判断T日是否触发交易
  Step 3: 更新状态（使用T日数据）
  Step 4: 计算T+1日的信号价格
```

### 2.3 策略工厂模式

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

## 3. 添加新策略步骤

1. 在 `internal/strategy/` 下创建新包
2. 实现 `SignalStrategy` 接口
3. 创建策略特有类型（如需要）
4. 创建适配器实现类型转换
5. 实现 `StrategyFactory` 接口的所有方法
6. 在 `factory.go` 的 `init()` 中注册
7. 更新 `config/strategies.json`

## 4. 常见陷阱

### 4.1 移仓换月时序

移仓换月必须延迟执行：

- T日收盘后：检测主力合约切换，记录待移仓信息
- T+1日开盘：执行移仓换月

### 4.2 无持仓的两种情况

- `hasEverHeldPosition = false`: 真正的无持仓，只使用阴1阳1
- `hasEverHeldPosition = true`: 有持仓历史，使用阴1阳1和阴2阳2

### 4.3 止损价 vs 反向信号价

- **止损价**: 开仓时确定，固定为阴1阳1的高低点
- **反向信号价**: 持仓期间动态更新

## 5. 基线验证规则

**重要**: 每次代码改动后，必须验证基线结果，防止改动引发回归问题。

### 5.1 基线验证

项目已集成基线验证skill，详细说明请参考 [.trae/skills/baseline-verifier/SKILL.md](.trae/skills/baseline-verifier/SKILL.md)。

**快速验证**：

1. 启动Web服务：

```bash
go run cmd/web/main.go
```

2. 执行验证脚本：

```powershell
# Windows
.\scripts\verify_baselines.ps1

# Linux/macOS
./scripts/verify_baselines.sh
```

### 5.2 验证内容

- **Signals验证**: 检查交易信号是否一致，验证策略逻辑
- **Statistics验证**: 检查统计数据是否一致，验证计算方法

### 5.3 更新基线

如果是有意的修改导致结果变化，更新基线文件：

```powershell
# Windows
$symbol = "RB"; $strategy = "yinyang"; $startDate = "20240101"; $endDate = "20241231"
$pattern = "${symbol}_${strategy}_${startDate}_${endDate}_*_*.json"
$latestFile = Get-ChildItem -Path "ret\$pattern" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
Copy-Item $latestFile.FullName -Destination "baselines\${symbol}_${strategy}_${startDate}_${endDate}_baseline.json"
```

## 6. 测试与验证

```bash
# 运行回测
go run cmd/main.go -symbol RB -start 20240101 -end 20241231 -leverage 3

# 启动Web服务
go run cmd/web/main.go

# 运行测试
go test ./internal/strategy/yinyang/... ./internal/strategy/ma/... -v
```

## 7. 文档资源

- [设计文档](docs/DESIGN.md): 系统架构和模块设计
- [策略说明](docs/STRATEGY.md): 阴阳线策略详细说明
- [需求文档](docs/REQUIREMENTS.md): 系统需求说明

---

**最后更新**: 2026-04-24
