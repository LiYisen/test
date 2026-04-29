package ma

import (
	"futures-backtest/internal/backtest"
)

type MAAdapter struct {
	strategy *MAStrategy
}

func NewMAAdapter(strategy *MAStrategy) *MAAdapter {
	return &MAAdapter{strategy: strategy}
}

func (a *MAAdapter) ProcessKLine(kline backtest.KLineWithContract) []backtest.TradeSignal {
	return a.strategy.ProcessKLine(kline)
}

func (a *MAAdapter) Position() *backtest.SignalPosition {
	return a.strategy.Position()
}

func (a *MAAdapter) SetPosition(pos *backtest.SignalPosition) {
	a.strategy.SetPosition(pos)
}

func (a *MAAdapter) SetCurrentSymbol(symbol string) {
	a.strategy.SetCurrentSymbol(symbol)
}

func (a *MAAdapter) UpdateStateOnly(kline backtest.KLineWithContract) {
	a.strategy.UpdateStateOnly(kline)
}

func (a *MAAdapter) State() MAState {
	return a.strategy.State()
}

func (a *MAAdapter) StateForSymbol(symbol string) MAState {
	return a.strategy.StateForSymbol(symbol)
}

func (a *MAAdapter) GetMAs() (shortMA, longMA float64) {
	return a.strategy.GetMAs()
}

func (a *MAAdapter) GetMAsForSymbol(symbol string) (shortMA, longMA float64) {
	return a.strategy.GetMAsForSymbol(symbol)
}

func (a *MAAdapter) GetStrategy() *MAStrategy {
	return a.strategy
}

var _ backtest.SignalStrategy = (*MAAdapter)(nil)
