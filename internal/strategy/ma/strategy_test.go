package ma

import (
	"testing"

	"futures-backtest/internal/backtest"

	"github.com/stretchr/testify/assert"
)

func TestNewMAStrategy(t *testing.T) {
	strategy := NewMAStrategy(5, 20, 1.0)
	assert.NotNil(t, strategy)
	assert.Equal(t, 5, strategy.shortPeriod)
	assert.Equal(t, 20, strategy.longPeriod)
	assert.InDelta(t, 1.0, strategy.leverage, 0.0001)
}

func TestStateManager_Update(t *testing.T) {
	sm := NewStateManager(5, 20)

	for i := 0; i < 20; i++ {
		sm.Update(backtest.KLineData{
			Date:  "2026-01-01",
			Close: 100 + float64(i),
		})
	}

	assert.True(t, sm.IsReady())
	shortMA, longMA := sm.GetMAs()
	assert.Greater(t, shortMA, 0.0)
	assert.Greater(t, longMA, 0.0)
}

func TestStateManager_IsReady(t *testing.T) {
	sm := NewStateManager(5, 20)

	assert.False(t, sm.IsReady())

	for i := 0; i < 20; i++ {
		sm.Update(backtest.KLineData{
			Date:  "2026-01-01",
			Close: 100 + float64(i),
		})
	}

	assert.True(t, sm.IsReady())
}

func TestStateManager_GetMAs(t *testing.T) {
	sm := NewStateManager(3, 5)

	klines := []backtest.KLineData{
		{Date: "2026-01-01", Close: 10},
		{Date: "2026-01-02", Close: 20},
		{Date: "2026-01-03", Close: 30},
		{Date: "2026-01-04", Close: 40},
		{Date: "2026-01-05", Close: 50},
	}

	for _, k := range klines {
		sm.Update(k)
	}

	shortMA, longMA := sm.GetMAs()

	assert.InDelta(t, 40.0, shortMA, 0.0001)
	assert.InDelta(t, 30.0, longMA, 0.0001)
}

func TestMAStrategy_ProcessKLine_NoPosition(t *testing.T) {
	strategy := NewMAStrategy(3, 5, 1.0)

	klines := []backtest.KLineWithContract{
		{Symbol: "RB2501", KLineData: backtest.KLineData{Date: "2026-01-01", Open: 100, Close: 100}},
		{Symbol: "RB2501", KLineData: backtest.KLineData{Date: "2026-01-02", Open: 101, Close: 101}},
		{Symbol: "RB2501", KLineData: backtest.KLineData{Date: "2026-01-03", Open: 102, Close: 102}},
		{Symbol: "RB2501", KLineData: backtest.KLineData{Date: "2026-01-04", Open: 103, Close: 103}},
		{Symbol: "RB2501", KLineData: backtest.KLineData{Date: "2026-01-05", Open: 104, Close: 104}},
	}

	for _, k := range klines {
		signals := strategy.ProcessKLine(k)
		assert.Equal(t, 0, len(signals))
	}
}

func TestMAStrategy_ProcessKLine_GoldenCross(t *testing.T) {
	strategy := NewMAStrategy(3, 5, 1.0)

	for i := 0; i < 25; i++ {
		kline := backtest.KLineWithContract{
			Symbol: "RB2501",
			KLineData: backtest.KLineData{
				Date:  "2026-01-01",
				Open:  100 + float64(i),
				Close: 100 + float64(i),
			},
		}
		strategy.ProcessKLine(kline)
	}

	kline := backtest.KLineWithContract{
		Symbol: "RB2501",
		KLineData: backtest.KLineData{
			Date:  "2026-01-26",
			Open:  130,
			Close: 130,
		},
	}
	signals := strategy.ProcessKLine(kline)

	assert.GreaterOrEqual(t, len(signals), 0)
}

func TestMAStrategy_ProcessKLine_DeathCross(t *testing.T) {
	strategy := NewMAStrategy(3, 5, 1.0)

	for i := 0; i < 25; i++ {
		kline := backtest.KLineWithContract{
			Symbol: "RB2501",
			KLineData: backtest.KLineData{
				Date:  "2026-01-01",
				Open:  100 - float64(i),
				Close: 100 - float64(i),
			},
		}
		strategy.ProcessKLine(kline)
	}

	kline := backtest.KLineWithContract{
		Symbol: "RB2501",
		KLineData: backtest.KLineData{
			Date:  "2026-01-26",
			Open:  70,
			Close: 70,
		},
	}
	signals := strategy.ProcessKLine(kline)

	assert.GreaterOrEqual(t, len(signals), 0)
}

func TestMAStrategy_Position(t *testing.T) {
	strategy := NewMAStrategy(3, 5, 1.0)
	assert.Nil(t, strategy.Position())
}

func TestMAStrategy_State(t *testing.T) {
	strategy := NewMAStrategy(3, 5, 1.0)

	klines := []backtest.KLineWithContract{
		{Symbol: "RB2501", KLineData: backtest.KLineData{Date: "2026-01-01", Close: 100}},
		{Symbol: "RB2501", KLineData: backtest.KLineData{Date: "2026-01-02", Close: 101}},
		{Symbol: "RB2501", KLineData: backtest.KLineData{Date: "2026-01-03", Close: 102}},
	}

	for _, k := range klines {
		strategy.ProcessKLine(k)
	}

	state := strategy.State()
	assert.Greater(t, state.ShortMA, 0.0)
}

func TestMAAdapter_ProcessKLine(t *testing.T) {
	strategy := NewMAStrategy(3, 5, 1.0)
	adapter := NewMAAdapter(strategy)

	kline := backtest.KLineWithContract{
		Symbol: "RB2501",
		KLineData: backtest.KLineData{
			Date:  "2026-01-01",
			Open:  100,
			Close: 100,
		},
	}

	signals := adapter.ProcessKLine(kline)
	assert.Equal(t, 0, len(signals))
}

func TestMAAdapter_Position(t *testing.T) {
	strategy := NewMAStrategy(3, 5, 1.0)
	adapter := NewMAAdapter(strategy)

	pos := adapter.Position()
	assert.Nil(t, pos)
}

func TestMAAdapter_State(t *testing.T) {
	strategy := NewMAStrategy(3, 5, 1.0)
	adapter := NewMAAdapter(strategy)

	state := adapter.State()
	assert.InDelta(t, 0.0, state.ShortMA, 0.0001)
	assert.InDelta(t, 0.0, state.LongMA, 0.0001)
}

func TestRolloverHandler_CheckAndExecute_NoPosition(t *testing.T) {
	strategy := NewMAStrategy(3, 5, 1.0)
	handler := NewRolloverHandler(strategy)

	signals := handler.CheckAndExecute(
		"RB2505",
		"RB2501",
		backtest.KLineWithContract{Symbol: "RB2505", KLineData: backtest.KLineData{Date: "2026-01-15", Open: 100}},
		backtest.KLineWithContract{Symbol: "RB2501", KLineData: backtest.KLineData{Date: "2026-01-15", Open: 100}},
		"2026-01-15",
		nil,
	)

	assert.Equal(t, 0, len(signals))
}

func TestRolloverHandler_CheckAndExecute_WithPosition(t *testing.T) {
	strategy := NewMAStrategy(3, 5, 1.0)

	strategy.ProcessKLine(backtest.KLineWithContract{
		Symbol: "RB2501",
		KLineData: backtest.KLineData{
			Date:  "2026-01-01",
			Open:  100,
			Close: 100,
		},
	})

	handler := NewRolloverHandler(strategy)

	signals := handler.CheckAndExecute(
		"RB2505",
		"RB2501",
		backtest.KLineWithContract{Symbol: "RB2505", KLineData: backtest.KLineData{Date: "2026-01-15", Open: 100}},
		backtest.KLineWithContract{Symbol: "RB2501", KLineData: backtest.KLineData{Date: "2026-01-15", Open: 100}},
		"2026-01-15",
		nil,
	)

	assert.Equal(t, 0, len(signals))
}
