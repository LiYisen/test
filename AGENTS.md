# AGENTS.md - AI Agent 开发指南

## 项目概述

期货回测系统，Go 实现核心逻辑，Python 负责数据抓取。采用信号-资金分离架构：

```
K线数据 → [信号层] → TradeSignal[] → [资金层] → DailyRecord[] / TradeRecord[]
```

技术栈: Go 1.21+ / Python 3.x + akshare / Gin / SPA (HTML + JS + Chart.js)

## 核心架构

- **信号层** (`internal/strategy/`): 生成交易信号，不涉及资金。已注册策略：yinyang（阴阳线突破）、ma（双均线交叉）
- **资金层** (`internal/backtest/portfolio.go`): 接收信号，计算保证金、盈亏、资金曲线
- **基金引擎** (`internal/fund/`): 多品种组合回测，基于净值计算，并发执行，配置在 `config/funds.json`
- **Web服务** (`internal/web/`): REST API + SPA前端，基金回测采用异步任务+进度轮询

## 添加新策略

1. 在 `internal/strategy/` 下创建包
2. 实现 `SignalStrategy` 接口（ProcessKLine/Position/SetPosition等）
3. 创建适配器实现类型转换
4. 实现 `StrategyFactory` 接口（Create/Name/GetParams/CreateRolloverHandler等）
5. 在 `factory.go` 的 `init()` 中注册
6. 更新 `config/strategies.json`

## 常见陷阱

### 移仓换月时序
- T日收盘后：检测主力合约切换，记录待移仓信息
- T+1日开盘：执行移仓换月

### 临时阴阳集合（TempState）
- 生成时机：逆势开仓时（做多+阴线，或做空+阳线）
- 仅影响下一根K线的信号价格计算，使用后自动标记为已使用
- 信号价格计算优先级：未使用的临时集合 > 当前K线实际方向

### MA策略信号延迟
- 交叉信号在T日收盘后检测，T+1日开盘价执行

## 基线验证

每次代码改动后必须验证基线，防止回归：

```bash
go run cmd/web/main.go                    # 启动服务
.\scripts\verify_baselines.ps1            # Windows验证
```

验证内容：Signals一致性 + Statistics一致性

## 常用命令

```bash
go run cmd/main.go -symbol RB -start 20240101 -end 20241231 -leverage 3  # 回测
go run cmd/web/main.go                                                     # Web服务
go test ./internal/strategy/yinyang/... ./internal/strategy/ma/... -v     # 测试
```

## 文档

- [设计文档](docs/DESIGN.md): 系统架构和模块设计
- [阴阳线策略](docs/STRATEGY.md): 阴阳线策略详细说明
- [双均线策略](docs/MA_STRATEGY.md): 双均线交叉策略说明
- [需求文档](docs/REQUIREMENTS.md): 系统需求说明
