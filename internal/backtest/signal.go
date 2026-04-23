package backtest

import (
	"sort"
)

// SignalStrategy 通用策略接口，只负责信号生成
type SignalStrategy interface {
	ProcessKLine(kline KLineWithContract) []TradeSignal
	Position() *SignalPosition
	SetPosition(pos *SignalPosition)
	SetCurrentSymbol(symbol string)
	UpdateStateOnly(kline KLineWithContract)
}

// RolloverHandler 换月处理接口
type RolloverHandler interface {
	CheckAndExecute(currentSymbol, previousSymbol string, newKline KLineWithContract, oldKline KLineWithContract, date string, newSymbolKlines []KLineWithContract) []TradeSignal
}

type pendingRolloverInfo struct {
	fromSymbol string
	toSymbol   string
}

// SignalEngine 信号引擎，负责生成交易信号
type SignalEngine struct {
	klines          []KLineWithContract
	dominantMap     map[string]string
	strategy        SignalStrategy
	rollover        RolloverHandler
	klineIndex      map[string]KLineWithContract
	klinesByDate    map[string][]KLineWithContract
	klinesBySymbol  map[string][]KLineWithContract
	pendingRollover *pendingRolloverInfo
	stateRecorder   StateRecorder
}

func NewSignalEngine(
	klines []KLineWithContract,
	dominantMap map[string]string,
	strategy SignalStrategy,
	rollover RolloverHandler,
) *SignalEngine {
	sorted := make([]KLineWithContract, len(klines))
	copy(sorted, klines)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date < sorted[j].Date
	})

	index := make(map[string]KLineWithContract, len(sorted))
	klinesByDate := make(map[string][]KLineWithContract)
	klinesBySymbol := make(map[string][]KLineWithContract)
	for _, kl := range sorted {
		key := kl.Date + "|" + kl.Symbol
		index[key] = kl
		klinesByDate[kl.Date] = append(klinesByDate[kl.Date], kl)
		klinesBySymbol[kl.Symbol] = append(klinesBySymbol[kl.Symbol], kl)
	}

	return &SignalEngine{
		klines:         sorted,
		dominantMap:    dominantMap,
		strategy:       strategy,
		rollover:       rollover,
		klineIndex:     index,
		klinesByDate:   klinesByDate,
		klinesBySymbol: klinesBySymbol,
	}
}

func (e *SignalEngine) SetStateRecorder(recorder StateRecorder) {
	e.stateRecorder = recorder
}

func (e *SignalEngine) Calculate() ([]TradeSignal, error) {
	var allSignals []TradeSignal
	previousSymbol := ""

	for _, kl := range e.klines {
		dominant, ok := e.dominantMap[kl.Date]
		if !ok || dominant != kl.Symbol {
			continue
		}

		if !isValidPrice(kl.Close) {
			continue
		}

		if dateKlines, ok := e.klinesByDate[kl.Date]; ok {
			for _, dateKl := range dateKlines {
				if dateKl.Symbol != kl.Symbol {
					e.strategy.UpdateStateOnly(dateKl)
				}
			}
		}

		if previousSymbol != "" && previousSymbol != kl.Symbol {
			oldKlineKey := kl.Date + "|" + previousSymbol
			var oldKline KLineWithContract
			var hasOldKline bool
			if k, ok := e.klineIndex[oldKlineKey]; ok {
				oldKline = k
				hasOldKline = true
				signals := e.strategy.ProcessKLine(oldKline)
				if len(signals) > 0 {
					allSignals = append(allSignals, signals...)
				}
			}

			e.pendingRollover = &pendingRolloverInfo{
				fromSymbol: previousSymbol,
				toSymbol:   kl.Symbol,
			}
			e.strategy.SetCurrentSymbol(kl.Symbol)

			var newSymbolKlines []KLineWithContract
			if allKlines, ok := e.klinesBySymbol[kl.Symbol]; ok {
				for _, k := range allKlines {
					if k.Date < kl.Date {
						newSymbolKlines = append(newSymbolKlines, k)
					}
				}
			}

			if e.stateRecorder != nil && hasOldKline {
				e.stateRecorder.RecordState(kl.Date, kl, e.strategy.Position())
			}
			previousSymbol = kl.Symbol
			continue
		}

		if e.pendingRollover != nil {
			oldKlineKey := kl.Date + "|" + e.pendingRollover.fromSymbol
			oldKline, ok := e.klineIndex[oldKlineKey]
			if !ok {
				oldKline = kl
			}

			var newSymbolKlines []KLineWithContract
			if allKlines, ok := e.klinesBySymbol[e.pendingRollover.toSymbol]; ok {
				for _, k := range allKlines {
					if k.Date < kl.Date {
						newSymbolKlines = append(newSymbolKlines, k)
					}
				}
			}

			rolloverSignals := e.rollover.CheckAndExecute(
				e.pendingRollover.toSymbol,
				e.pendingRollover.fromSymbol,
				kl,
				oldKline,
				kl.Date,
				newSymbolKlines,
			)
			if len(rolloverSignals) > 0 {
				allSignals = append(allSignals, rolloverSignals...)
			}
			e.pendingRollover = nil
		}

		signals := e.strategy.ProcessKLine(kl)
		if len(signals) > 0 {
			allSignals = append(allSignals, signals...)
		}

		if e.stateRecorder != nil {
			e.stateRecorder.RecordState(kl.Date, kl, e.strategy.Position())
		}

		previousSymbol = kl.Symbol
	}

	return allSignals, nil
}

func isValidPrice(price float64) bool {
	return price > 0
}

// StateRecorder 状态记录器接口（可选）
type StateRecorder interface {
	RecordState(date string, kline KLineWithContract, position *SignalPosition)
	GetStateHistory() []StateRecord
}

type DefaultStateRecorder struct {
	records []StateRecord
}

func NewDefaultStateRecorder() *DefaultStateRecorder {
	return &DefaultStateRecorder{
		records: make([]StateRecord, 0),
	}
}

func (r *DefaultStateRecorder) RecordState(date string, kline KLineWithContract, position *SignalPosition) {
	var posDesc string
	if position == nil {
		posDesc = "无持仓"
	} else {
		posDesc = position.Direction.String() + " " + position.Symbol + "@" + position.OpenPrice.StringFixed(2)
	}
	r.records = append(r.records, StateRecord{
		Date:       date,
		Symbol:     kline.Symbol,
		Position:   posDesc,
		ClosePrice: kline.Close,
	})
}

func (r *DefaultStateRecorder) GetStateHistory() []StateRecord {
	return r.records
}
