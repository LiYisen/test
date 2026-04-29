package yinyang

import (
	"math"

	"futures-backtest/internal/backtest"
)

type YinYangStrategy struct {
	stateManagers  map[string]*StateManager
	leverageFactor float64

	position *backtest.SignalPosition

	longSignalPrice  float64
	shortSignalPrice float64

	reverseSignalPrice float64

	ready               bool
	currentSymbol       string
	hasEverHeldPosition bool
}

func NewYinYangStrategy(leverageFactor float64) *YinYangStrategy {
	return &YinYangStrategy{
		stateManagers:  make(map[string]*StateManager),
		leverageFactor: leverageFactor / 100.0,
	}
}

func (s *YinYangStrategy) getOrCreateStateManager(symbol string) *StateManager {
	if sm, ok := s.stateManagers[symbol]; ok {
		return sm
	}
	sm := NewStateManager()
	s.stateManagers[symbol] = sm
	return sm
}

func (s *YinYangStrategy) Position() *backtest.SignalPosition {
	return s.position
}

func (s *YinYangStrategy) SetPosition(pos *backtest.SignalPosition) {
	s.position = pos
	if pos != nil {
		s.currentSymbol = pos.Symbol
	}
}

func (s *YinYangStrategy) SetCurrentSymbol(symbol string) {
	s.currentSymbol = symbol
}

func (s *YinYangStrategy) State() YinYangState {
	if s.currentSymbol == "" {
		return YinYangState{}
	}
	return s.stateManagers[s.currentSymbol].State()
}

func (s *YinYangStrategy) StateForSymbol(symbol string) YinYangState {
	return s.getOrCreateStateManager(symbol).State()
}

func (s *YinYangStrategy) TempState() (YinYangState, bool) {
	if s.currentSymbol == "" {
		return YinYangState{}, false
	}
	return s.stateManagers[s.currentSymbol].GetTempState()
}

func (s *YinYangStrategy) UpdateStateOnly(kline backtest.KLineWithContract) {
	sm := s.getOrCreateStateManager(kline.Symbol)
	sm.Update(kline.KLineData)
}

func (s *YinYangStrategy) UpdateAllStates(kline backtest.KLineWithContract) {
	sm := s.getOrCreateStateManager(kline.Symbol)
	sm.Update(kline.KLineData)
}

func (s *YinYangStrategy) ProcessKLine(kline backtest.KLineWithContract) []backtest.TradeSignal {
	var signals []backtest.TradeSignal

	sm := s.getOrCreateStateManager(kline.Symbol)
	s.currentSymbol = kline.Symbol

	if s.position != nil {
		if sm.HasTempState() {
			s.UpdateReverseSignalPrice()
		}

		triggered := false
		if s.position.Direction == backtest.Buy {
			if kline.Low <= s.reverseSignalPrice {
				triggered = true
			}
		} else {
			if kline.High >= s.reverseSignalPrice {
				triggered = true
			}
		}

		if sm.HasTempState() {
			sm.MarkTempUsed()
		}

		sm.Update(kline.KLineData)

		if triggered {
			if s.position.Direction == backtest.Buy {
				signals = s.closeAndReverse(kline, backtest.CloseLong)
			} else {
				signals = s.closeAndReverse(kline, backtest.CloseShort)
			}
		} else {
			s.UpdateReverseSignalPrice()
		}

		return signals
	}

	if !s.ready {
		sm.Update(kline.KLineData)
		state := sm.State()
		if state.Yin1.IsValid && state.Yang1.IsValid {
			s.ready = true
			s.updateNoPositionSignalPrices()
		}
		return nil
	}

	if sm.HasTempState() {
		s.updateNoPositionSignalPrices()
	}

	longTriggered := kline.High >= s.longSignalPrice
	shortTriggered := kline.Low <= s.shortSignalPrice

	if sm.HasTempState() {
		sm.MarkTempUsed()
	}

	if longTriggered {
		signals = s.executeOpenSignal(kline, backtest.Buy, s.longSignalPrice)
		sm.Update(kline.KLineData)
		s.tryGenerateTempState(kline, backtest.Buy)
		s.UpdateReverseSignalPrice()
		return signals
	}

	if shortTriggered {
		signals = s.executeOpenSignal(kline, backtest.Sell, s.shortSignalPrice)
		sm.Update(kline.KLineData)
		s.tryGenerateTempState(kline, backtest.Sell)
		s.UpdateReverseSignalPrice()
		return signals
	}

	sm.Update(kline.KLineData)
	s.updateNoPositionSignalPrices()
	return nil
}

func (s *YinYangStrategy) executeOpenSignal(kline backtest.KLineWithContract, dir backtest.Direction, price float64) []backtest.TradeSignal {
	sm := s.getOrCreateStateManager(kline.Symbol)
	state := sm.State()
	var leverage float64
	if dir == backtest.Buy {
		leverage = s.calcLongLeverage(state, price)
	} else {
		leverage = s.calcShortLeverage(state, price)
	}

	s.position = &backtest.SignalPosition{
		Symbol:    kline.Symbol,
		Direction: dir,
		OpenPrice: price,
		OpenDate:  kline.Date,
		Leverage:  leverage,
	}
	s.hasEverHeldPosition = true

	return []backtest.TradeSignal{{
		SignalDate: kline.Date,
		Price:      price,
		Direction:  dir,
		Leverage:   leverage,
		SignalType: "yinyang",
		Symbol:     kline.Symbol,
		OpenPrice:  price,
		OpenDate:   kline.Date,
	}}
}

func (s *YinYangStrategy) updateNoPositionSignalPrices() {
	sm := s.getOrCreateStateManager(s.currentSymbol)
	var state YinYangState
	var currentIsYang bool
	if tempState, ok := sm.GetTempState(); ok {
		state = tempState
		currentIsYang = tempState.IsYang
	} else {
		state = sm.State()
		currentIsYang = sm.CurrentIsYang()
	}

	if state.Yin1.IsValid && state.Yang1.IsValid {
		if !s.hasEverHeldPosition {
			s.longSignalPrice = math.Max(state.Yin1.High, state.Yang1.High)
			s.shortSignalPrice = math.Min(state.Yin1.Low, state.Yang1.Low)
		} else {
			if currentIsYang {
				s.longSignalPrice = math.Max(state.Yin1.High, state.Yang1.High)
				if state.Yin2.IsValid {
					s.shortSignalPrice = math.Min(state.Yang1.Low, state.Yin2.Low)
				} else {
					s.shortSignalPrice = state.Yang1.Low
				}
			} else {
				if state.Yang2.IsValid {
					s.longSignalPrice = math.Max(state.Yin1.High, state.Yang2.High)
				} else {
					s.longSignalPrice = state.Yin1.High
				}
				s.shortSignalPrice = math.Min(state.Yin1.Low, state.Yang1.Low)
			}
		}
	}
}

func (s *YinYangStrategy) UpdateReverseSignalPrice() {
	if s.position == nil {
		return
	}

	sm := s.getOrCreateStateManager(s.currentSymbol)
	var state YinYangState
	var currentIsYang bool
	if tempState, ok := sm.GetTempState(); ok {
		state = tempState
		currentIsYang = tempState.IsYang
	} else {
		state = sm.State()
		currentIsYang = sm.CurrentIsYang()
	}

	if s.position.Direction == backtest.Buy {
		if currentIsYang {
			s.reverseSignalPrice = math.Min(state.Yin1.Low, state.Yang1.Low)
		} else {
			if state.Yin2.IsValid {
				s.reverseSignalPrice = math.Min(state.Yang1.Low, state.Yin2.Low)
			} else {
				s.reverseSignalPrice = state.Yang1.Low
			}
		}
	} else if s.position.Direction == backtest.Sell {
		if currentIsYang {
			if state.Yang2.IsValid {
				s.reverseSignalPrice = math.Max(state.Yin1.High, state.Yang2.High)
			} else {
				s.reverseSignalPrice = state.Yin1.High
			}
		} else {
			s.reverseSignalPrice = math.Max(state.Yin1.High, state.Yang1.High)
		}
	}
}

func (s *YinYangStrategy) closeAndReverse(kline backtest.KLineWithContract, closeDir backtest.Direction) []backtest.TradeSignal {
	var signals []backtest.TradeSignal
	execPrice := s.reverseSignalPrice

	signals = append(signals, backtest.TradeSignal{
		SignalDate: kline.Date,
		Price:      execPrice,
		Direction:  closeDir,
		Leverage:   s.position.Leverage,
		SignalType: "yinyang",
		Symbol:     kline.Symbol,
		OpenPrice:  s.position.OpenPrice,
		OpenDate:   s.position.OpenDate,
	})

	sm := s.getOrCreateStateManager(kline.Symbol)
	state := sm.State()

	if closeDir == backtest.CloseLong {
		leverage := s.calcShortLeverage(state, execPrice)
		s.position = &backtest.SignalPosition{
			Symbol:    kline.Symbol,
			Direction: backtest.Sell,
			OpenPrice: execPrice,
			OpenDate:  kline.Date,
			Leverage:  leverage,
		}

		signals = append(signals, backtest.TradeSignal{
			SignalDate: kline.Date,
			Price:      execPrice,
			Direction:  backtest.Sell,
			Leverage:   leverage,
			SignalType: "yinyang",
			Symbol:     kline.Symbol,
			OpenPrice:  execPrice,
			OpenDate:   kline.Date,
		})

		s.tryGenerateTempState(kline, backtest.Sell)
		s.UpdateReverseSignalPrice()
	} else {
		leverage := s.calcLongLeverage(state, execPrice)
		s.position = &backtest.SignalPosition{
			Symbol:    kline.Symbol,
			Direction: backtest.Buy,
			OpenPrice: execPrice,
			OpenDate:  kline.Date,
			Leverage:  leverage,
		}

		signals = append(signals, backtest.TradeSignal{
			SignalDate: kline.Date,
			Price:      execPrice,
			Direction:  backtest.Buy,
			Leverage:   leverage,
			SignalType: "yinyang",
			Symbol:     kline.Symbol,
			OpenPrice:  execPrice,
			OpenDate:   kline.Date,
		})

		s.tryGenerateTempState(kline, backtest.Buy)
		s.UpdateReverseSignalPrice()
	}

	return signals
}

func (s *YinYangStrategy) SignalPrices() (long, short float64) {
	if s.position == nil {
		if s.ready {
			return s.longSignalPrice, s.shortSignalPrice
		}
		return 0, 0
	}

	if s.position.Direction == backtest.Buy {
		return 0, s.reverseSignalPrice
	} else {
		return s.reverseSignalPrice, 0
	}
}

func (s *YinYangStrategy) SignalPricesForSymbol(symbol string) (long, short float64) {
	sm := s.getOrCreateStateManager(symbol)
	state := sm.State()

	if !state.Yin1.IsValid || !state.Yang1.IsValid {
		return 0, 0
	}

	longPrice := math.Max(state.Yin1.High, state.Yang1.High)
	shortPrice := math.Min(state.Yin1.Low, state.Yang1.Low)

	return longPrice, shortPrice
}

func (s *YinYangStrategy) ReverseSignalPriceForSymbol(symbol string, position *backtest.SignalPosition) float64 {
	if position == nil {
		return 0
	}

	sm := s.getOrCreateStateManager(symbol)
	state := sm.State()
	currentIsYang := sm.CurrentIsYang()

	if position.Direction == backtest.Buy {
		if currentIsYang {
			return math.Min(state.Yin1.Low, state.Yang1.Low)
		} else {
			if state.Yin2.IsValid {
				return math.Min(state.Yang1.Low, state.Yin2.Low)
			} else {
				return state.Yang1.Low
			}
		}
	} else {
		if currentIsYang {
			if state.Yang2.IsValid {
				return math.Max(state.Yin1.High, state.Yang2.High)
			} else {
				return state.Yin1.High
			}
		} else {
			return math.Max(state.Yin1.High, state.Yang1.High)
		}
	}
}

var maxLeverage = 6.0

func (s *YinYangStrategy) calcLongLeverage(state YinYangState, openPrice float64) float64 {
	minLow := math.Min(state.Yin1.Low, state.Yang1.Low)
	denominator := openPrice - minLow
	if denominator <= 0 {
		return 1
	}
	lev := s.leverageFactor * openPrice / denominator
	if lev > maxLeverage {
		return maxLeverage
	}
	return lev
}

func (s *YinYangStrategy) calcShortLeverage(state YinYangState, openPrice float64) float64 {
	maxHigh := math.Max(state.Yin1.High, state.Yang1.High)
	denominator := maxHigh - openPrice
	if denominator <= 0 {
		return 1
	}
	lev := s.leverageFactor * openPrice / denominator
	if lev > maxLeverage {
		return maxLeverage
	}
	return lev
}

func (s *YinYangStrategy) isKLineYang(kline backtest.KLineData) bool {
	return kline.Close > kline.Open
}

func (s *YinYangStrategy) isKLineDoji(kline backtest.KLineData) bool {
	return kline.Close == kline.Open
}

func (s *YinYangStrategy) tryGenerateTempState(kline backtest.KLineWithContract, dir backtest.Direction) {
	sm := s.getOrCreateStateManager(kline.Symbol)
	isDoji := s.isKLineDoji(kline.KLineData)
	isYang := s.isKLineYang(kline.KLineData)

	if isDoji {
		return
	}

	if dir == backtest.Buy && !isYang {
		sm.GenerateTempState(true, kline.High, kline.Low)
	} else if dir == backtest.Sell && isYang {
		sm.GenerateTempState(false, kline.High, kline.Low)
	}
}

func (s *YinYangStrategy) SimulateTrading(klines []backtest.KLineWithContract) *backtest.SignalPosition {
	simStrategy := NewYinYangStrategy(s.leverageFactor * 100)

	for _, kl := range klines {
		simStrategy.ProcessKLine(kl)
	}

	return simStrategy.Position()
}
