package strategy

import (
	"futures-backtest/internal/backtest"
)

type SignalStrategy interface {
	ProcessKLine(kline backtest.KLineWithContract) []backtest.TradeSignal
	Position() *backtest.SignalPosition
	SetPosition(pos *backtest.SignalPosition)
	SetCurrentSymbol(symbol string)
	UpdateStateOnly(kline backtest.KLineWithContract)
}

type RolloverHandler interface {
	CheckAndExecute(currentSymbol, previousSymbol string, newKline backtest.KLineWithContract, oldKline backtest.KLineWithContract, date string, newSymbolKlines []KLineWithContract) []backtest.TradeSignal
}

type StrategyFactory interface {
	Create(params map[string]interface{}) SignalStrategy
	Name() string
	Description() string
}

type KLineWithContract = backtest.KLineWithContract
