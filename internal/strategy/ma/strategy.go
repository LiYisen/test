package ma

import (
	"fmt"

	"futures-backtest/internal/backtest"
)

type MAStrategy struct {
	stateManagers map[string]*StateManager
	shortPeriod   int
	longPeriod    int
	leverage      float64

	position *backtest.SignalPosition

	pendingSignal *PendingSignal
	currentSymbol string
}

type PendingSignal struct {
	Direction backtest.Direction
	Date      string
}

func NewMAStrategy(shortPeriod, longPeriod int, leverage float64) *MAStrategy {
	if shortPeriod >= longPeriod {
		panic("短期均线周期必须小于长期均线周期")
	}

	return &MAStrategy{
		stateManagers: make(map[string]*StateManager),
		shortPeriod:   shortPeriod,
		longPeriod:    longPeriod,
		leverage:      leverage,
	}
}

func (s *MAStrategy) getOrCreateStateManager(symbol string) *StateManager {
	if sm, ok := s.stateManagers[symbol]; ok {
		return sm
	}
	sm := NewStateManager(s.shortPeriod, s.longPeriod)
	s.stateManagers[symbol] = sm
	return sm
}

func (s *MAStrategy) Position() *backtest.SignalPosition {
	return s.position
}

func (s *MAStrategy) SetPosition(pos *backtest.SignalPosition) {
	s.position = pos
	if pos != nil {
		s.currentSymbol = pos.Symbol
	}
}

func (s *MAStrategy) SetCurrentSymbol(symbol string) {
	s.currentSymbol = symbol
}

func (s *MAStrategy) UpdateStateOnly(kline backtest.KLineWithContract) {
	sm := s.getOrCreateStateManager(kline.Symbol)
	sm.Update(kline.KLineData)
}

func (s *MAStrategy) ProcessKLine(kline backtest.KLineWithContract) []backtest.TradeSignal {
	var signals []backtest.TradeSignal

	sm := s.getOrCreateStateManager(kline.Symbol)
	s.currentSymbol = kline.Symbol

	if s.pendingSignal != nil && s.pendingSignal.Date != kline.Date {
		if s.position != nil {
			var closeDir backtest.Direction
			if s.position.Direction == backtest.Buy {
				closeDir = backtest.CloseLong
			} else {
				closeDir = backtest.CloseShort
			}

			closeSignal := backtest.TradeSignal{
				SignalDate: kline.Date,
				Symbol:     kline.Symbol,
				Direction:  closeDir,
				Price:      kline.Open,
				Leverage:   s.leverage,
				SignalType: "ma",
				OpenPrice:  s.position.OpenPrice,
				OpenDate:   s.position.OpenDate,
			}
			signals = append(signals, closeSignal)
			s.position = nil
		}

		openSignal := backtest.TradeSignal{
			SignalDate: kline.Date,
			Symbol:     kline.Symbol,
			Direction:  s.pendingSignal.Direction,
			Price:      kline.Open,
			Leverage:   s.leverage,
			SignalType: "ma",
		}
		signals = append(signals, openSignal)

		s.position = &backtest.SignalPosition{
			Symbol:    kline.Symbol,
			Direction: s.pendingSignal.Direction,
			OpenPrice: kline.Open,
			OpenDate:  kline.Date,
			Leverage:  s.leverage,
		}

		s.pendingSignal = nil
	}

	sm.Update(kline.KLineData)

	if !sm.IsReady() {
		return signals
	}

	shortMA, longMA := sm.GetMAs()
	prevShortMA, prevLongMA := sm.GetPrevMAs()

	if prevShortMA > 0 && prevLongMA > 0 && shortMA > 0 && longMA > 0 {
		goldenCross := prevShortMA <= prevLongMA && shortMA > longMA
		deathCross := prevShortMA >= prevLongMA && shortMA < longMA

		if goldenCross {
			s.pendingSignal = &PendingSignal{
				Direction: backtest.Buy,
				Date:      kline.Date,
			}
		} else if deathCross {
			s.pendingSignal = &PendingSignal{
				Direction: backtest.Sell,
				Date:      kline.Date,
			}
		}
	}

	return signals
}

func (s *MAStrategy) State() MAState {
	if s.currentSymbol == "" {
		return MAState{}
	}
	return s.stateManagers[s.currentSymbol].State()
}

func (s *MAStrategy) StateForSymbol(symbol string) MAState {
	return s.getOrCreateStateManager(symbol).State()
}

func (s *MAStrategy) GetMAs() (shortMA, longMA float64) {
	if s.currentSymbol == "" {
		return 0, 0
	}
	return s.stateManagers[s.currentSymbol].GetMAs()
}

func (s *MAStrategy) GetMAsForSymbol(symbol string) (shortMA, longMA float64) {
	return s.getOrCreateStateManager(symbol).GetMAs()
}

func (s *MAStrategy) SimulateTrading(klines []backtest.KLineWithContract) *backtest.SignalPosition {
	simStrategy := NewMAStrategy(s.shortPeriod, s.longPeriod, s.leverage)

	for _, kl := range klines {
		simStrategy.ProcessKLine(kl)
	}

	return simStrategy.Position()
}

func (s *MAStrategy) String() string {
	return fmt.Sprintf("MA策略(短期=%d, 长期=%d, 杠杆=%.2f)", s.shortPeriod, s.longPeriod, s.leverage)
}
