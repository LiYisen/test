package yinyang

import (
	"futures-backtest/internal/backtest"

	"github.com/shopspring/decimal"
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

	newState := h.strategy.StateForSymbol(currentSymbol)

	var newDir backtest.Direction
	var leverage decimal.Decimal

	simPosition := h.strategy.SimulateTrading(newSymbolKlines)
	if simPosition != nil {
		newDir = simPosition.Direction
	} else {
		sm := h.strategy.getOrCreateStateManager(currentSymbol)
		currentIsYang := sm.CurrentIsYang()

		var longPrice, shortPrice decimal.Decimal

		if currentIsYang {
			if newState.Yang2.IsValid {
				longPrice = decimal.Max(newState.Yin1.High, newState.Yang2.High)
			} else {
				longPrice = newState.Yin1.High
			}
			shortPrice = decimal.Min(newState.Yin1.Low, newState.Yang1.Low)
		} else {
			longPrice = decimal.Max(newState.Yin1.High, newState.Yang1.High)
			if newState.Yin2.IsValid {
				shortPrice = decimal.Min(newState.Yang1.Low, newState.Yin2.Low)
			} else {
				shortPrice = newState.Yang1.Low
			}
		}

		longDistance := newOpenPrice.Sub(longPrice).Abs()
		shortDistance := newOpenPrice.Sub(shortPrice).Abs()

		if longDistance.LessThanOrEqual(shortDistance) {
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
