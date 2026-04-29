package yinyang

import (
	"futures-backtest/internal/backtest"
)

type YinYangAdapter struct {
	strategy *YinYangStrategy
}

func NewYinYangAdapter(strategy *YinYangStrategy) *YinYangAdapter {
	return &YinYangAdapter{strategy: strategy}
}

func (a *YinYangAdapter) ProcessKLine(kline backtest.KLineWithContract) []backtest.TradeSignal {
	return a.strategy.ProcessKLine(kline)
}

func (a *YinYangAdapter) Position() *backtest.SignalPosition {
	return a.strategy.Position()
}

func (a *YinYangAdapter) SetPosition(pos *backtest.SignalPosition) {
	a.strategy.SetPosition(pos)
}

func (a *YinYangAdapter) SetCurrentSymbol(symbol string) {
	a.strategy.SetCurrentSymbol(symbol)
}

func (a *YinYangAdapter) UpdateStateOnly(kline backtest.KLineWithContract) {
	a.strategy.UpdateStateOnly(kline)
}

func (a *YinYangAdapter) SignalPrices() (long, short float64) {
	return a.strategy.SignalPrices()
}

func (a *YinYangAdapter) TempState() (YinYangState, bool) {
	return a.strategy.TempState()
}

func (a *YinYangAdapter) State() YinYangState {
	return a.strategy.State()
}

func (a *YinYangAdapter) StateForSymbol(symbol string) YinYangState {
	return a.strategy.StateForSymbol(symbol)
}

func (a *YinYangAdapter) SimulateTrading(klines []backtest.KLineWithContract) *backtest.SignalPosition {
	return a.strategy.SimulateTrading(klines)
}

func (a *YinYangAdapter) ReverseSignalPriceForSymbol(symbol string, position *backtest.SignalPosition) float64 {
	return a.strategy.ReverseSignalPriceForSymbol(symbol, position)
}

func (a *YinYangAdapter) SignalPricesForSymbol(symbol string) (long, short float64) {
	return a.strategy.SignalPricesForSymbol(symbol)
}

func (a *YinYangAdapter) GetStrategy() *YinYangStrategy {
	return a.strategy
}

var _ backtest.SignalStrategy = (*YinYangAdapter)(nil)
