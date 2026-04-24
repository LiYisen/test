package ma

import (
	"futures-backtest/internal/backtest"

	"github.com/shopspring/decimal"
)

type MAState struct {
	ShortMA decimal.Decimal `json:"short_ma"`
	LongMA  decimal.Decimal `json:"long_ma"`
}

type StateManager struct {
	shortPeriod int
	longPeriod  int

	prices []decimal.Decimal

	shortMA decimal.Decimal
	longMA  decimal.Decimal

	prevShortMA decimal.Decimal
	prevLongMA  decimal.Decimal
}

func NewStateManager(shortPeriod, longPeriod int) *StateManager {
	return &StateManager{
		shortPeriod: shortPeriod,
		longPeriod:  longPeriod,
		prices:      make([]decimal.Decimal, 0),
	}
}

func (m *StateManager) Update(kline backtest.KLineData) {
	close := decimal.NewFromFloat(kline.Close)

	m.prevShortMA = m.shortMA
	m.prevLongMA = m.longMA

	m.prices = append(m.prices, close)

	if len(m.prices) >= m.shortPeriod {
		var sum decimal.Decimal
		for i := len(m.prices) - m.shortPeriod; i < len(m.prices); i++ {
			sum = sum.Add(m.prices[i])
		}
		m.shortMA = sum.Div(decimal.NewFromInt(int64(m.shortPeriod)))
	}

	if len(m.prices) >= m.longPeriod {
		var sum decimal.Decimal
		for i := len(m.prices) - m.longPeriod; i < len(m.prices); i++ {
			sum = sum.Add(m.prices[i])
		}
		m.longMA = sum.Div(decimal.NewFromInt(int64(m.longPeriod)))
	}

	if len(m.prices) > m.longPeriod*2 {
		m.prices = m.prices[len(m.prices)-m.longPeriod*2:]
	}
}

func (m *StateManager) IsReady() bool {
	return len(m.prices) >= m.longPeriod
}

func (m *StateManager) GetMAs() (shortMA, longMA decimal.Decimal) {
	return m.shortMA, m.longMA
}

func (m *StateManager) GetPrevMAs() (shortMA, longMA decimal.Decimal) {
	return m.prevShortMA, m.prevLongMA
}

func (m *StateManager) State() MAState {
	return MAState{
		ShortMA: m.shortMA,
		LongMA:  m.longMA,
	}
}
