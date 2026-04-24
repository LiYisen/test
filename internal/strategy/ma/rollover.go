package ma

import (
	"futures-backtest/internal/backtest"

	"github.com/shopspring/decimal"
)

type RolloverHandler struct {
	strategy *MAStrategy
}

func NewRolloverHandler(strategy *MAStrategy) *RolloverHandler {
	return &RolloverHandler{strategy: strategy}
}

func (h *RolloverHandler) CheckAndExecute(
	currentSymbol, previousSymbol string,
	newKline, oldKline backtest.KLineWithContract,
	date string,
	newSymbolKlines []backtest.KLineWithContract,
) []backtest.TradeSignal {
	if currentSymbol == previousSymbol || previousSymbol == "" {
		return nil
	}

	position := h.strategy.Position()
	if position == nil {
		return nil
	}

	var signals []backtest.TradeSignal

	oldOpenPrice := decimal.NewFromFloat(oldKline.Open)
	if !oldOpenPrice.IsPositive() {
		oldOpenPrice = decimal.NewFromFloat(newKline.Open)
	}

	closeDir := closeDirection(position.Direction)
	signals = append(signals, backtest.TradeSignal{
		SignalDate: date,
		Price:      oldOpenPrice,
		Direction:  closeDir,
		Leverage:   position.Leverage,
		SignalType: "rollover",
		Symbol:     previousSymbol,
		OpenPrice:  position.OpenPrice,
		OpenDate:   position.OpenDate,
	})

	newOpenPrice := decimal.NewFromFloat(newKline.Open)

	var newDir backtest.Direction

	simPosition := h.strategy.SimulateTrading(newSymbolKlines)
	if simPosition != nil {
		newDir = simPosition.Direction
	} else {
		newDir = position.Direction
	}

	signals = append(signals, backtest.TradeSignal{
		SignalDate: date,
		Price:      newOpenPrice,
		Direction:  newDir,
		Leverage:   position.Leverage,
		SignalType: "rollover",
		Symbol:     currentSymbol,
		OpenPrice:  newOpenPrice,
		OpenDate:   date,
	})

	h.strategy.SetPosition(&backtest.SignalPosition{
		Symbol:    currentSymbol,
		Direction: newDir,
		OpenPrice: newOpenPrice,
		OpenDate:  date,
		Leverage:  position.Leverage,
	})

	return signals
}

func closeDirection(posDir backtest.Direction) backtest.Direction {
	switch posDir {
	case backtest.Buy:
		return backtest.CloseLong
	case backtest.Sell:
		return backtest.CloseShort
	default:
		return backtest.Close
	}
}

var _ backtest.RolloverHandler = (*RolloverHandler)(nil)
