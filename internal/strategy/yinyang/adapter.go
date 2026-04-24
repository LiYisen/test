package yinyang

import (
	"futures-backtest/internal/backtest"

	"github.com/shopspring/decimal"
)

// YinYangAdapter 将阴阳线策略适配为通用策略接口
type YinYangAdapter struct {
	strategy *YinYangStrategy
}

// NewYinYangAdapter 创建适配器
func NewYinYangAdapter(strategy *YinYangStrategy) *YinYangAdapter {
	return &YinYangAdapter{strategy: strategy}
}

// ProcessKLine 处理K线数据，返回交易信号
func (a *YinYangAdapter) ProcessKLine(kline backtest.KLineWithContract) []backtest.TradeSignal {
	return a.strategy.ProcessKLine(kline)
}

// Position 返回当前持仓状态
func (a *YinYangAdapter) Position() *backtest.SignalPosition {
	return a.strategy.Position()
}

// SetPosition 设置持仓状态
func (a *YinYangAdapter) SetPosition(pos *backtest.SignalPosition) {
	a.strategy.SetPosition(pos)
}

// SetCurrentSymbol 设置当前交易品种
func (a *YinYangAdapter) SetCurrentSymbol(symbol string) {
	a.strategy.SetCurrentSymbol(symbol)
}

// UpdateStateOnly 仅更新状态（不生成信号）
func (a *YinYangAdapter) UpdateStateOnly(kline backtest.KLineWithContract) {
	a.strategy.UpdateStateOnly(kline)
}

// SignalPrices 获取当前信号价格（阴阳线特有）
func (a *YinYangAdapter) SignalPrices() (long, short decimal.Decimal) {
	return a.strategy.SignalPrices()
}

// TempState 获取临时状态（阴阳线特有）
func (a *YinYangAdapter) TempState() (YinYangState, bool) {
	return a.strategy.TempState()
}

// State 获取当前状态（阴阳线特有）
func (a *YinYangAdapter) State() YinYangState {
	return a.strategy.State()
}

// StateForSymbol 获取指定品种的状态（阴阳线特有）
func (a *YinYangAdapter) StateForSymbol(symbol string) YinYangState {
	return a.strategy.StateForSymbol(symbol)
}

// SimulateTrading 模拟交易（阴阳线特有）
func (a *YinYangAdapter) SimulateTrading(klines []backtest.KLineWithContract) *backtest.SignalPosition {
	return a.strategy.SimulateTrading(klines)
}

// ReverseSignalPriceForSymbol 获取反向信号价格（阴阳线特有）
func (a *YinYangAdapter) ReverseSignalPriceForSymbol(symbol string, position *backtest.SignalPosition) decimal.Decimal {
	return a.strategy.ReverseSignalPriceForSymbol(symbol, position)
}

// SignalPricesForSymbol 获取指定品种的信号价格（阴阳线特有）
func (a *YinYangAdapter) SignalPricesForSymbol(symbol string) (long, short decimal.Decimal) {
	return a.strategy.SignalPricesForSymbol(symbol)
}

// GetStrategy 获取底层策略实例（用于换月处理等特有逻辑）
func (a *YinYangAdapter) GetStrategy() *YinYangStrategy {
	return a.strategy
}

// Ensure YinYangAdapter implements backtest.SignalStrategy
var _ backtest.SignalStrategy = (*YinYangAdapter)(nil)
