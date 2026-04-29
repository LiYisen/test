package yinyang

import (
	"math"

	"futures-backtest/internal/backtest"
)

type RolloverHandler struct {
	strategy *YinYangStrategy
}

func NewRolloverHandler(strategy *YinYangStrategy) *RolloverHandler {
	return &RolloverHandler{
		strategy: strategy,
	}
}

func (h *RolloverHandler) CheckAndExecute(
	currentSymbol string,
	previousSymbol string,
	newKline backtest.KLineWithContract,
	oldKline backtest.KLineWithContract,
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

	oldOpenPrice := oldKline.Open
	if oldOpenPrice <= 0 {
		oldOpenPrice = newKline.Open
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

	newOpenPrice := newKline.Open

	newState := h.strategy.StateForSymbol(currentSymbol)

	var newDir backtest.Direction
	var leverage float64

	simPosition := h.strategy.SimulateTrading(newSymbolKlines)
	if simPosition != nil {
		newDir = simPosition.Direction
	} else {
		sm := h.strategy.getOrCreateStateManager(currentSymbol)
		currentIsYang := sm.CurrentIsYang()

		var longPrice, shortPrice float64

		if currentIsYang {
			if newState.Yang2.IsValid {
				longPrice = math.Max(newState.Yin1.High, newState.Yang2.High)
			} else {
				longPrice = newState.Yin1.High
			}
			shortPrice = math.Min(newState.Yin1.Low, newState.Yang1.Low)
		} else {
			longPrice = math.Max(newState.Yin1.High, newState.Yang1.High)
			if newState.Yin2.IsValid {
				shortPrice = math.Min(newState.Yang1.Low, newState.Yin2.Low)
			} else {
				shortPrice = newState.Yang1.Low
			}
		}

		longDistance := math.Abs(newOpenPrice - longPrice)
		shortDistance := math.Abs(newOpenPrice - shortPrice)

		if longDistance <= shortDistance {
			newDir = backtest.Buy
		} else {
			newDir = backtest.Sell
		}
	}

	if newDir == backtest.Buy {
		leverage = h.strategy.calcLongLeverage(newState, newOpenPrice)
	} else {
		leverage = h.strategy.calcShortLeverage(newState, newOpenPrice)
	}

	signals = append(signals, backtest.TradeSignal{
		SignalDate: date,
		Price:      newOpenPrice,
		Direction:  newDir,
		Leverage:   leverage,
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
		Leverage:  leverage,
	})

	h.strategy.UpdateReverseSignalPrice()

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
