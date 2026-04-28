# AGENTS.md - AI Agent 开发指南

## 1. 项目概述

期货回测系统，采用 Go 语言实现核心回测逻辑，Python 负责数据抓取。系统采用**信号生成与资金计算分离**的两层架构。

```
K线数据 → [信号层] → TradeSignal[] → [资金层] → DailyRecord[] / TradeRecord[]
```

**技术栈**: Go 1.21+ / Python 3.x + akshare / Gin框架 / SPA (HTML + JavaScript + Chart.js)

## 2. 核心设计原则

### 2.1 信号-资金分离

- **信号层** (`internal/strategy/`): 只负责生成交易信号，不涉及资金计算
- **资金层** (`internal/backtest/portfolio.go`): 接收信号，计算保证金、盈亏、资金曲线

### 2.2 基金模式

- **基金引擎** (`internal/fund/`): 多品种组合回测，基于净值计算，并发执行品种回测
- **配置管理**: 基金配置存储在 `config/funds.json`，使用 `sync.Once` 保证线程安全
- **结果存储**: 基金回测结果独立存储在 `ret/funding/` 目录下

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

### 4.5 临时阴阳集合（TempState）

**生成时机**：逆势开仓时（做多+阴线，或做空+阳线）

**使用规则**：

- 仅影响下一根K线的信号价格计算
- 下一根K线使用后自动标记为已使用（tempUsed = true）
- 不影响正常阴阳集合的状态更新

**信号价格计算优先级**：

1. 如果存在未使用的临时集合 → 使用临时集合的状态方向(IsYang)
2. 如果不存在临时集合 → 使用当前K线的实际方向(currentIsYang)

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

## 6. 测试与验证

```bash
# 运行回测
go run cmd/main.go -symbol RB -start 20240101 -end 20241231 -leverage 3

# 启动Web服务
go run cmd/web/main.go

# 运行测试
go test ./internal/strategy/yinyang/... ./internal/strategy/ma/... -v
```

## 7. Web界面说明

系统采用 SPA 架构，所有功能模块集成在 `web/templates/index.html` 中。

- **左侧导航栏**: 包含"回测"、"组合分析"、"基金"三个功能入口，支持收起/展开
- **无刷新切换**: 通过客户端 JavaScript 控制页面区域显示/隐藏
- **响应式设计**: 768px 以下屏幕自动折叠导航栏
- **API数据格式**: 后端返回的数值为字符串格式，前端使用 `safeParseFloat()` 安全解析

## 8. 文档资源

- [设计文档](docs/DESIGN.md): 系统架构和模块设计
- [策略说明](docs/STRATEGY.md): 阴阳线策略详细说明
- [需求文档](docs/REQUIREMENTS.md): 系统需求说明

---

**最后更新**: 2026-04-28
