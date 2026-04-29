package yinyang

import (
	"fmt"

	"futures-backtest/internal/backtest"
)

type StateManager struct {
	state   YinYangState
	lastDir bool
	hasData bool

	tempState    YinYangState
	hasTempState bool
	tempUsed     bool

	prevState     YinYangState
	prevDir       bool
	currentIsYang bool
}

func NewStateManager() *StateManager {
	return &StateManager{}
}

func (m *StateManager) State() YinYangState {
	return m.state
}

type YinYangStateRecorder struct {
	records []backtest.StateRecord
}

func NewYinYangStateRecorder() *YinYangStateRecorder {
	return &YinYangStateRecorder{}
}

func (r *YinYangStateRecorder) RecordState(date string, kline backtest.KLineWithContract, position *backtest.SignalPosition) {
	var posDesc string
	if position == nil {
		posDesc = "无持仓"
	} else {
		posDesc = fmt.Sprintf("%s %s@%.2f", position.Direction.String(), position.Symbol, position.OpenPrice)
	}

	r.records = append(r.records, backtest.StateRecord{
		Date:       date,
		Symbol:     kline.Symbol,
		Position:   posDesc,
		ClosePrice: kline.Close,
	})
}

func (r *YinYangStateRecorder) GetStateHistory() []backtest.StateRecord {
	return r.records
}

func (m *StateManager) Update(kline backtest.KLineData) {
	isYang := m.determineDirection(kline)
	m.updateState(isYang, kline.High, kline.Low)
}

func (m *StateManager) determineDirection(kline backtest.KLineData) bool {
	if kline.Close > kline.Open {
		return true
	}
	if kline.Close < kline.Open {
		return false
	}
	if !m.hasData {
		return true
	}
	return m.lastDir
}

func (m *StateManager) updateState(isYang bool, high, low float64) {
	m.prevState = m.state
	m.prevDir = m.lastDir
	m.currentIsYang = isYang

	if !m.hasData {
		m.hasData = true
		m.lastDir = isYang
		m.state.IsYang = isYang
		if isYang {
			m.state.Yang1 = YinYangElement{High: high, Low: low, IsValid: true}
		} else {
			m.state.Yin1 = YinYangElement{High: high, Low: low, IsValid: true}
		}
		return
	}

	if isYang == m.lastDir {
		if isYang {
			mergeElement(&m.state.Yang1, high, low)
		} else {
			mergeElement(&m.state.Yin1, high, low)
		}
	} else {
		if isYang {
			m.state.Yang2 = m.state.Yang1
			m.state.Yang1 = YinYangElement{High: high, Low: low, IsValid: true}
		} else {
			m.state.Yin2 = m.state.Yin1
			m.state.Yin1 = YinYangElement{High: high, Low: low, IsValid: true}
		}
		m.state.IsYang = isYang
	}

	m.lastDir = isYang
}

func mergeElement(elem *YinYangElement, high, low float64) {
	if !elem.IsValid {
		elem.High = high
		elem.Low = low
		elem.IsValid = true
		return
	}
	if high > elem.High {
		elem.High = high
	}
	if low < elem.Low {
		elem.Low = low
	}
}

func (m *StateManager) GenerateTempState(isYangOverride bool, high, low float64) {
	m.tempState = m.state

	if m.currentIsYang != m.prevDir {
		if m.currentIsYang {
			m.tempState.Yang1 = m.prevState.Yang1
			m.tempState.Yang2 = m.prevState.Yang2
		} else {
			m.tempState.Yin1 = m.prevState.Yin1
			m.tempState.Yin2 = m.prevState.Yin2
		}
	} else {
		if m.currentIsYang {
			m.tempState.Yang1 = m.prevState.Yang1
		} else {
			m.tempState.Yin1 = m.prevState.Yin1
		}
	}

	if isYangOverride == m.prevDir {
		if isYangOverride {
			mergeElement(&m.tempState.Yang1, high, low)
		} else {
			mergeElement(&m.tempState.Yin1, high, low)
		}
	} else {
		if isYangOverride {
			m.tempState.Yang2 = m.tempState.Yang1
			m.tempState.Yang1 = YinYangElement{High: high, Low: low, IsValid: true}
		} else {
			m.tempState.Yin2 = m.tempState.Yin1
			m.tempState.Yin1 = YinYangElement{High: high, Low: low, IsValid: true}
		}
		m.tempState.IsYang = isYangOverride
	}

	m.hasTempState = true
	m.tempUsed = false
}

func (m *StateManager) GetTempState() (YinYangState, bool) {
	if m.hasTempState && !m.tempUsed {
		return m.tempState, true
	}
	return YinYangState{}, false
}

func (m *StateManager) MarkTempUsed() {
	m.tempUsed = true
}

func (m *StateManager) ClearTempState() {
	m.hasTempState = false
	m.tempUsed = false
	m.tempState = YinYangState{}
}

func (m *StateManager) HasTempState() bool {
	return m.hasTempState && !m.tempUsed
}

func (m *StateManager) CurrentIsYang() bool {
	return m.currentIsYang
}
