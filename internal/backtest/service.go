package backtest

import (
	"time"
)

type BacktestResult struct {
	Signals           []TradeSignal
	DailyRecords      []DailyRecord
	PositionReturns   []PositionReturn
	Statistics        Statistics
	StateHistory      []StateRecord
	DominantMap       map[string]string
	Klines            []KLineWithContract
	WarmupDays        int
	BacktestStartDate string
}

type BacktestService struct{}

func NewBacktestService() *BacktestService {
	return &BacktestService{}
}

type BacktestInput struct {
	Klines            []KLineWithContract
	DominantMap       map[string]string
	Symbol            string
	StartDate         string
	EndDate           string
	Strategy          SignalStrategy
	Rollover          RolloverHandler
	StateRecorder     StateRecorder
	WarmupDays        int
	BacktestStartDate string
}

func (s *BacktestService) Run(input BacktestInput) (*BacktestResult, error) {
	signalEngine := NewSignalEngine(input.Klines, input.DominantMap, input.Strategy, input.Rollover)
	signalEngine.SetStateRecorder(input.StateRecorder)
	signalEngine.SetWarmupDays(input.WarmupDays, input.BacktestStartDate)

	signals, err := signalEngine.Calculate()
	if err != nil {
		return nil, err
	}

	dominantKlines := FilterDominantKlines(input.Klines, input.DominantMap)

	portfolioEngine := NewPortfolioEngine()
	dailyRecords, positionReturns, err := portfolioEngine.Calculate(signals, dominantKlines)
	if err != nil {
		return nil, err
	}

	stats := CalculateStatistics(dailyRecords, positionReturns)

	var filteredKlines []KLineWithContract
	var filteredDailyRecords []DailyRecord
	if input.WarmupDays > 0 {
		filteredKlines = FilterKlinesByDate(dominantKlines, input.StartDate)
		filteredDailyRecords = FilterDailyRecordsByDate(dailyRecords, input.StartDate)
	} else {
		filteredKlines = dominantKlines
		filteredDailyRecords = dailyRecords
	}

	var stateHistory []StateRecord
	if input.StateRecorder != nil {
		stateHistory = input.StateRecorder.GetStateHistory()
	}

	return &BacktestResult{
		Signals:           signals,
		DailyRecords:      filteredDailyRecords,
		PositionReturns:   positionReturns,
		Statistics:        stats,
		StateHistory:      stateHistory,
		DominantMap:       input.DominantMap,
		Klines:            filteredKlines,
		WarmupDays:        input.WarmupDays,
		BacktestStartDate: input.BacktestStartDate,
	}, nil
}

func GenerateResultID(symbol, strategy, startDate, endDate string, leverage float64) string {
	return symbol + "_" + strategy + "_" + startDate + "_" + endDate + "_" + time.Now().Format("20060102150405")
}
