# 期货回测系统

一个专业的期货量化回测框架，采用信号生成与资金计算分离的两层架构设计，支持多种交易策略和组合分析。

## ✨ 功能特性

- 🎯 **信号-资金分离架构**：策略逻辑与资金管理解耦，便于策略开发和维护
- 📊 **多策略支持**：内置阴阳线突破策略、双均线交叉策略，支持自定义策略扩展
- 🔄 **移仓换月**：自动处理主力合约切换，确保回测连续性
- 📈 **组合分析**：支持多品种组合回测，计算组合收益和风险指标
- 🖥️ **Web界面**：提供直观的Web界面，可视化回测结果
- 📉 **丰富的图表**：资金曲线、回撤分析、日收益、交易记录等可视化展示
- ⚡ **高性能**：Go语言实现核心逻辑，回测速度快
- 🛡️ **安全性**：前端XSS防护，数据验证，防止注入攻击

## 🛠️ 技术栈

- **后端**: Go 1.21+ / Gin框架
- **前端**: HTML + JavaScript + Chart.js
- **数据源**: Python 3.x + akshare
- **数据库**: JSON文件存储

## 📦 安装

### 前置要求

- Go 1.21 或更高版本
- Python 3.x（用于数据抓取）

### 安装步骤

1. **克隆仓库**

```bash
git clone https://github.com/LiYisen/test.git
cd test
```

2. **安装Go依赖**

```bash
go mod download
```

3. **安装Python依赖**（可选，用于数据抓取）

```bash
pip install akshare
```

## 🚀 快速开始

### 1. 启动Web服务

```bash
go run cmd/web/main.go
```

服务将在 `http://localhost:8080` 启动。

### 2. 运行回测

**Web界面方式**：

1. 打开浏览器访问 `http://localhost:8080`
2. 选择品种、日期范围、策略和参数
3. 点击"运行回测"按钮

**命令行方式**：

```bash
go run cmd/main.go -symbol RB -start 20240101 -end 20241231 -strategy yinyang -leverage 3
```

### 3. 查看结果

- **统计指标**：总收益率、年化收益、最大回撤、夏普比率、胜率等
- **图表分析**：资金曲线与回撤、日收益分布
- **交易记录**：详细的开仓、平仓记录

## 📊 内置策略

### 1. 阴阳线突破策略 (YinYang)

基于价格突破的顺势策略，通过识别关键价格点位生成交易信号。

**参数**：

- 无需参数

**特点**：

- 适合趋势行情
- 自动处理移仓换月
- 风险可控

### 2. 双均线交叉策略 (MA)

基于移动平均线的趋势跟踪策略，通过金叉死叉判断买卖时机。

**参数**：

- `short_period`: 短期均线周期（默认：5）
- `long_period`: 长期均线周期（默认：20）

**特点**：

- 避免盘中假信号
- 适合趋势跟踪
- 参数可调

## 🏗️ 项目结构

```
.
├── cmd/                    # 命令行入口
│   ├── main.go            # 命令行回测
│   └── web/               # Web服务
├── internal/              # 内部包
│   ├── backtest/          # 回测核心逻辑
│   │   ├── portfolio.go   # 资金管理
│   │   ├── engine.go      # 回测引擎
│   │   └── types.go       # 类型定义
│   ├── strategy/          # 策略实现
│   │   ├── yinyang/       # 阴阳线策略
│   │   ├── ma/            # 双均线策略
│   │   └── factory.go     # 策略工厂
│   └── web/               # Web服务
│       └── server.go      # HTTP服务器
├── web/                   # 前端资源
│   └── templates/         # HTML模板
├── data/                  # 数据目录
│   └── klines/            # K线数据
├── ret/                   # 回测结果
├── baselines/             # 基线结果
├── config/                # 配置文件
│   └── strategies.json    # 策略配置
└── docs/                  # 文档
    ├── DESIGN.md          # 设计文档
    ├── STRATEGY.md        # 策略说明
    └── REQUIREMENTS.md    # 需求文档
```

## 🎨 核心设计

### 信号-资金分离架构

```
K线数据 → [信号层] → TradeSignal[] → [资金层] → DailyRecord[] / TradeRecord[]
```

**信号层**：

- 只负责生成交易信号
- 不涉及资金计算
- 支持多种策略

**资金层**：

- 接收交易信号
- 计算保证金、盈亏
- 生成资金曲线

### 时序性保证

**禁止使用未来函数**，确保回测的真实性：

```
处理K线T时：
  Step 1: 使用T-1日计算的信号价格
  Step 2: 判断T日是否触发交易
  Step 3: 更新状态（使用T日数据）
  Step 4: 计算T+1日的信号价格
```

## 🔧 开发指南

### 添加新策略

1. 在 `internal/strategy/` 下创建新包
2. 实现 `SignalStrategy` 接口
3. 创建策略特有类型（如需要）
4. 实现 `StrategyFactory` 接口
5. 在 `factory.go` 的 `init()` 中注册
6. 更新 `config/strategies.json`

详细说明请参考 [AGENTS.md](AGENTS.md)

### 运行测试

```bash
go test ./internal/strategy/yinyang/... ./internal/strategy/ma/... -v
```

### 基线验证

每次代码改动后，建议验证基线结果，防止引入回归问题：

```bash
# Windows
.\scripts\verify_baselines.ps1

# Linux/macOS
./scripts/verify_baselines.sh
```

## 📈 性能指标

系统计算以下关键指标：

- **收益指标**：总收益率、年化收益率
- **风险指标**：最大回撤、最大回撤比例
- **风险调整收益**：夏普比率、卡玛比率
- **交易统计**：胜率、盈亏比、交易次数

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 开发流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📝 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

- [akshare](https://github.com/akfamily/akshare) - 数据源
- [Gin](https://github.com/gin-gonic/gin) - Web框架
- [Chart.js](https://www.chartjs.org/) - 图表库

## 📧 联系方式

如有问题或建议，请提交 Issue 或联系项目维护者。

---

**最后更新**: 2026-04-24
