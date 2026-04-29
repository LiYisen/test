package ma

import (
	"futures-backtest/internal/backtest"
)

type MAState struct {
	ShortMA float64 `json:"short_ma"`
	LongMA  float64 `json:"long_ma"`
}

type StateManager struct {
	shortPeriod int
	longPeriod  int

	prices []float64

	shortMA float64
	longMA  float64

	prevShortMA float64
	prevLongMA  float64
}

func NewStateManager(shortPeriod, longPeriod int) *StateManager {
	return &StateManager{
		shortPeriod: shortPeriod,
		longPeriod:  longPeriod,
		prices:      make([]float64, 0),
	}
}

func (m *StateManager) Update(kline backtest.KLineData) {
	close := kline.Close

	m.prevShortMA = m.shortMA
	m.prevLongMA = m.longMA

	m.prices = append(m.prices, close)

	if len(m.prices) >= m.shortPeriod {
		var sum float64
		for i := len(m.prices) - m.shortPeriod; i < len(m.prices); i++ {
			sum += m.prices[i]
		}
		m.shortMA = sum / float64(m.shortPeriod)
	}

	if len(m.prices) >= m.longPeriod {
		var sum float64
		for i := len(m.prices) - m.longPeriod; i < len(m.prices); i++ {
			sum += m.prices[i]
		}
		m.longMA = sum / float64(m.longPeriod)
	}

	if len(m.prices) > m.longPeriod*2 {
		m.prices = m.prices[len(m.prices)-m.longPeriod*2:]
	}
}

func (m *StateManager) IsReady() bool {
	return len(m.prices) >= m.longPeriod
}

func (m *StateManager) GetMAs() (shortMA, longMA float64) {
	return m.shortMA, m.longMA
}

func (m *StateManager) GetPrevMAs() (shortMA, longMA float64) {
	return m.prevShortMA, m.prevLongMA
}

func (m *StateManager) State() MAState {
	return MAState{
		ShortMA: m.shortMA,
		LongMA:  m.longMA,
	}
}
