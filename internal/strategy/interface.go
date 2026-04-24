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

type StrategyFactory interface {
	Create(params map[string]interface{}) SignalStrategy
	Name() string
	Description() string
	DisplayName() string
	GetParams() []StrategyParamConfig
	CreateRolloverHandler(strategy SignalStrategy) backtest.RolloverHandler
	CreateStateRecorder() backtest.StateRecorder
}

type KLineWithContract = backtest.KLineWithContract
