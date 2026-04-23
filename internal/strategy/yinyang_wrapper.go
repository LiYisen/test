package strategy

import (
	"futures-backtest/internal/backtest"
	"futures-backtest/internal/strategy/yinyang"
)

type YinYangRolloverHandler struct {
	handler *yinyang.RolloverHandler
}

func NewYinYangRolloverHandler(strategy *yinyang.YinYangStrategy) *YinYangRolloverHandler {
	return &YinYangRolloverHandler{
		handler: yinyang.NewRolloverHandler(strategy),
	}
}

func (h *YinYangRolloverHandler) CheckAndExecute(
	currentSymbol, previousSymbol string,
	newKline, oldKline backtest.KLineWithContract,
	date string,
	newSymbolKlines []backtest.KLineWithContract,
) []backtest.TradeSignal {
	return h.handler.CheckAndExecute(currentSymbol, previousSymbol, newKline, oldKline, date, newSymbolKlines)
}

type YinYangStateRecorder struct {
	recorder *yinyang.YinYangStateRecorder
}

func NewYinYangStateRecorder() *YinYangStateRecorder {
	return &YinYangStateRecorder{
		recorder: yinyang.NewYinYangStateRecorder(),
	}
}

func (r *YinYangStateRecorder) RecordState(date string, kline backtest.KLineWithContract, position *backtest.SignalPosition) {
	r.recorder.RecordState(date, kline, position)
}

func (r *YinYangStateRecorder) GetStateHistory() []backtest.StateRecord {
	return r.recorder.GetStateHistory()
}

type YinYangAdapter = yinyang.YinYangAdapter
