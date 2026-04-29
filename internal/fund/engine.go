package fund

import (
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"futures-backtest/internal/backtest"
	"futures-backtest/internal/data"
	"futures-backtest/internal/strategy"
)

type ProgressFunc func(progress int, step string)

type FundEngine struct {
	dataManager *data.FuturesDataManager
	onProgress  ProgressFunc
}

func NewFundEngine(dataManager *data.FuturesDataManager) *FundEngine {
	return &FundEngine{
		dataManager: dataManager,
	}
}

func (e *FundEngine) SetProgressCallback(fn ProgressFunc) {
	e.onProgress = fn
}

func (e *FundEngine) reportProgress(progress int, step string) {
	if e.onProgress != nil {
		e.onProgress(progress, step)
	}
}

type positionBacktestResult struct {
	Symbol       string
	Strategy     string
	Weight       float64
	Signals      []backtest.TradeSignal
	DailyRecords []backtest.DailyRecord
	Statistics   backtest.Statistics
	Error        error
}

func (e *FundEngine) RunBacktest(config FundConfig, startDate, endDate string) (*FundResult, error) {
	log.Printf("[基金] RunBacktest开始: fund=%s, start=%s, end=%s", config.Name, startDate, endDate)
	e.reportProgress(5, "验证配置")

	if err := ValidateFundConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	if startDate == "" {
		startDate = config.StartDate
	}
	if endDate == "" {
		endDate = config.EndDate
	}

	totalPositions := len(config.Positions)
	e.reportProgress(10, fmt.Sprintf("开始回测 %d 个品种", totalPositions))

	var mu sync.Mutex
	completedCount := 0

	var wg sync.WaitGroup
	resultsChan := make(chan *positionBacktestResult, len(config.Positions))

	for i, pos := range config.Positions {
		wg.Add(1)
		go func(posConfig PositionConfig, index int) {
			defer wg.Done()
			e.reportProgress(10+int(float64(index)/float64(totalPositions)*60), fmt.Sprintf("回测品种 %s/%s", posConfig.Symbol, posConfig.Strategy))
			result := e.runPositionBacktest(posConfig, startDate, endDate)
			resultsChan <- result

			mu.Lock()
			completedCount++
			progress := 10 + int(float64(completedCount)/float64(totalPositions)*60)
			e.reportProgress(progress, fmt.Sprintf("完成品种 %s (%d/%d)", posConfig.Symbol, completedCount, totalPositions))
			mu.Unlock()
		}(pos, i)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var results []*positionBacktestResult
	for r := range resultsChan {
		results = append(results, r)
	}

	for _, r := range results {
		if r.Error != nil {
			return nil, fmt.Errorf("品种 %s 回测失败: %w", r.Symbol, r.Error)
		}
	}

	e.reportProgress(75, "合并每日记录")

	positionResults := make(map[string]*PositionResult)
	for _, r := range results {
		positionResults[r.Symbol] = &PositionResult{
			Symbol:       r.Symbol,
			Strategy:     r.Strategy,
			Weight:       r.Weight,
			Signals:      r.Signals,
			DailyRecords: r.DailyRecords,
			Statistics:   r.Statistics,
		}
	}

	fundDailyRecords := e.mergeDailyRecords(results)

	e.reportProgress(85, "计算基金统计")

	fundStats := e.calculateFundStatistics(fundDailyRecords, results)

	e.reportProgress(95, "生成结果")

	resultID := fmt.Sprintf("%s_%s_%s_%d", config.ID, startDate, endDate, time.Now().Unix())

	e.reportProgress(100, "回测完成")

	return &FundResult{
		ID:              resultID,
		FundID:          config.ID,
		FundName:        config.Name,
		Timestamp:       time.Now().Unix(),
		StartDate:       startDate,
		EndDate:         endDate,
		Statistics:      fundStats,
		DailyRecords:    fundDailyRecords,
		PositionResults: positionResults,
	}, nil
}

func (e *FundEngine) runPositionBacktest(posConfig PositionConfig, startDate, endDate string) *positionBacktestResult {
	log.Printf("[基金] 开始品种回测: %s/%s", posConfig.Symbol, posConfig.Strategy)
	result := &positionBacktestResult{
		Symbol:   posConfig.Symbol,
		Strategy: posConfig.Strategy,
		Weight:   posConfig.Weight,
	}

	factory, err := strategy.DefaultRegistry.Get(posConfig.Strategy)
	if err != nil {
		result.Error = fmt.Errorf("获取策略失败: %w", err)
		return result
	}

	params := make(map[string]interface{})
	if posConfig.Params != nil {
		for k, v := range posConfig.Params {
			params[k] = v
		}
	}

	warmupDays := factory.GetWarmupDays(params)

	var warmupStartDate string
	var backtestStartDateFormatted string
	if warmupDays > 0 {
		start, err := time.Parse("20060102", startDate)
		if err != nil {
			result.Error = fmt.Errorf("解析开始日期失败: %w", err)
			return result
		}

		requiredTradingDays := warmupDays + 5
		calendar, err := e.dataManager.GetTradeCalendar("20000101", startDate)
		if err == nil && len(calendar) > 0 {
			warmupStartDate = calculateWarmupStartDate(calendar, startDate, requiredTradingDays)
		} else {
			warmupStart := start.AddDate(0, 0, -requiredTradingDays*2)
			warmupStartDate = warmupStart.Format("20060102")
		}

		backtestStartDateFormatted = start.Format("2006-01-02")
	} else {
		warmupStartDate = startDate
	}

	contractSymbols := generateContractSymbols(posConfig.Symbol, warmupStartDate, endDate)

	var allKlines []backtest.KLineWithContract
	for _, cs := range contractSymbols {
		klines, err := e.dataManager.GetFuturesKLine(cs, warmupStartDate, endDate)
		if err != nil {
			continue
		}
		for _, kl := range klines {
			allKlines = append(allKlines, backtest.KLineWithContract{
				Symbol: cs,
				KLineData: backtest.KLineData{
					Date:   kl.Date,
					Open:   kl.Open,
					High:   kl.High,
					Low:    kl.Low,
					Close:  kl.Close,
					Volume: kl.Volume,
					Amount: kl.Amount,
					Hold:   kl.Hold,
					Settle: kl.Settle,
				},
			})
		}
	}

	if len(allKlines) == 0 {
		result.Error = fmt.Errorf("未获取到任何K线数据")
		return result
	}

	identifier := data.NewDominantContractIdentifier(e.dataManager)
	dominantResult, err := identifier.Identify(posConfig.Symbol, allKlines, warmupStartDate, endDate)
	if err != nil {
		result.Error = fmt.Errorf("识别主力合约失败: %w", err)
		return result
	}

	dominantMap := make(map[string]string, len(dominantResult))
	for t, sym := range dominantResult {
		dateStr := t.Format("2006-01-02")
		dominantMap[dateStr] = sym
	}

	sigStrategy := factory.Create(params)
	rollover := factory.CreateRolloverHandler(sigStrategy)
	stateRecorder := factory.CreateStateRecorder()

	signalEngine := backtest.NewSignalEngine(allKlines, dominantMap, sigStrategy, rollover)
	signalEngine.SetStateRecorder(stateRecorder)
	signalEngine.SetWarmupDays(warmupDays, backtestStartDateFormatted)

	signals, err := signalEngine.Calculate()
	if err != nil {
		result.Error = fmt.Errorf("计算交易信号失败: %w", err)
		return result
	}

	dominantKlines := filterDominantKlines(allKlines, dominantMap)
	portfolioEngine := backtest.NewPortfolioEngine()
	dailyRecords, positionReturns, err := portfolioEngine.Calculate(signals, dominantKlines)
	if err != nil {
		result.Error = fmt.Errorf("计算资金收益失败: %w", err)
		return result
	}

	stats := backtest.CalculateStatistics(dailyRecords, positionReturns)

	if warmupDays > 0 {
		dailyRecords = filterDailyRecordsByDate(dailyRecords, startDate)
	}

	result.Signals = signals
	result.DailyRecords = dailyRecords
	result.Statistics = stats

	return result
}

func (e *FundEngine) mergeDailyRecords(results []*positionBacktestResult) []FundDailyRecord {
	dateSet := make(map[string]bool)
	for _, r := range results {
		for _, dr := range r.DailyRecords {
			dateSet[dr.Date] = true
		}
	}

	var dates []string
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	dailyMaps := make([]map[string]backtest.DailyRecord, len(results))
	baseValues := make([]float64, len(results))
	for i, r := range results {
		dailyMaps[i] = make(map[string]backtest.DailyRecord)
		for _, dr := range r.DailyRecords {
			dailyMaps[i][dr.Date] = dr
		}
		if len(r.DailyRecords) > 0 {
			baseValues[i] = r.DailyRecords[0].TotalValue
		} else {
			baseValues[i] = 1
		}
	}

	var fundRecords []FundDailyRecord
	cumulativeValue := 1.0

	for _, date := range dates {
		var weightedReturn float64
		components := make(map[string]float64)
		totalWeight := 0.0

		for i, r := range results {
			if dr, ok := dailyMaps[i][date]; ok {
				weightedReturn += dr.DailyReturn * r.Weight
				if baseValues[i] != 0 {
					components[r.Symbol] = dr.TotalValue / baseValues[i]
				} else {
					components[r.Symbol] = dr.TotalValue
				}
				totalWeight += r.Weight
			}
		}

		if totalWeight == 0 {
			continue
		}

		cumulativeValue = cumulativeValue * (1 + weightedReturn)

		record := FundDailyRecord{
			Date:        date,
			TotalValue:  cumulativeValue,
			DailyReturn: weightedReturn,
			PnL:         cumulativeValue - 1,
			Components:  components,
		}
		fundRecords = append(fundRecords, record)
	}

	return fundRecords
}

func (e *FundEngine) calculateFundStatistics(records []FundDailyRecord, results []*positionBacktestResult) FundStatistics {
	stats := FundStatistics{
		TradingDays: len(records),
	}

	if len(records) == 0 {
		return stats
	}

	stats.TotalReturn = records[len(records)-1].TotalValue - 1

	if len(records) > 1 {
		tradingDays := float64(len(records))
		years := tradingDays / 250.0
		if years > 0 {
			stats.AnnualReturn = math.Pow(records[len(records)-1].TotalValue, 1/years) - 1
		}
	}

	stats.MaxDrawdown, stats.MaxDrawdownRatio = calculateFundMaxDrawdown(records)

	winCount, lossCount := 0, 0
	for _, r := range results {
		winCount += r.Statistics.WinningTrades
		lossCount += r.Statistics.LosingTrades
	}
	stats.WinningTrades = winCount
	stats.LosingTrades = lossCount
	stats.TotalTrades = winCount + lossCount

	if stats.TotalTrades > 0 {
		stats.WinRate = float64(winCount) / float64(stats.TotalTrades)
	}

	stats.SharpeRatio = calculateFundSharpeRatio(records)

	if stats.MaxDrawdownRatio != 0 {
		stats.CalmarRatio = stats.AnnualReturn / stats.MaxDrawdownRatio
	}

	return stats
}

func calculateFundMaxDrawdown(records []FundDailyRecord) (float64, float64) {
	if len(records) == 0 {
		return 0, 0
	}

	peak := 1.0
	maxDD := 0.0
	maxDDPercent := 0.0

	for _, rec := range records {
		if rec.TotalValue > peak {
			peak = rec.TotalValue
		}
		drawdown := peak - rec.TotalValue
		drawdownPercent := drawdown / peak

		if drawdown > maxDD {
			maxDD = drawdown
			maxDDPercent = drawdownPercent
		}
	}

	return maxDD, maxDDPercent
}

func calculateFundSharpeRatio(records []FundDailyRecord) float64 {
	if len(records) < 2 {
		return 0
	}

	var sum, sqSum float64
	for _, rec := range records {
		r := rec.DailyReturn
		sum += r
		sqSum += r * r
	}

	mean := sum / float64(len(records))
	variance := sqSum/float64(len(records)) - mean*mean

	if variance <= 0 {
		return 0
	}

	stdDev := math.Sqrt(variance)
	if stdDev == 0 {
		return 0
	}

	sharpe := mean / stdDev
	if math.IsInf(sharpe, 0) || math.IsNaN(sharpe) {
		return 0
	}

	return sharpe * math.Sqrt(250)
}

func calculateWarmupStartDate(calendar []data.TradeDate, startDate string, requiredDays int) string {
	startDateFormatted := formatDateForComparison(startDate)

	var tradingDates []string
	for _, td := range calendar {
		if td.IsTradingDay && td.Date < startDateFormatted {
			tradingDates = append(tradingDates, td.Date)
		}
	}
	sort.Strings(tradingDates)

	if len(tradingDates) < requiredDays {
		return startDate
	}

	return tradingDates[len(tradingDates)-requiredDays]
}

func generateContractSymbols(product, startDate, endDate string) []string {
	start, err := time.Parse("20060102", startDate)
	if err != nil {
		return []string{product}
	}
	end, err := time.Parse("20060102", endDate)
	if err != nil {
		return []string{product}
	}

	limitDate := end.AddDate(1, 0, 0)
	limitYear := limitDate.Year()
	limitMonth := int(limitDate.Month())

	startYear := start.Year()
	startMonth := int(start.Month()) + 1
	if startMonth > 12 {
		startMonth = 1
		startYear++
	}

	seen := make(map[string]bool)
	var symbols []string

	year := startYear
	month := startMonth
	for {
		if year > limitYear || (year == limitYear && month > limitMonth) {
			return symbols
		}
		sym := fmt.Sprintf("%s%02d%02d", product, year%100, month)
		if !seen[sym] {
			seen[sym] = true
			symbols = append(symbols, sym)
		}
		month++
		if month > 12 {
			month = 1
			year++
		}
	}
}

func filterDominantKlines(klines []backtest.KLineWithContract, dominantMap map[string]string) []backtest.KLineWithContract {
	var result []backtest.KLineWithContract
	for _, kl := range klines {
		if dominant, ok := dominantMap[kl.Date]; ok && dominant == kl.Symbol {
			result = append(result, kl)
		}
	}
	return result
}

func filterDailyRecordsByDate(records []backtest.DailyRecord, startDate string) []backtest.DailyRecord {
	var result []backtest.DailyRecord
	formattedStart := formatDateForComparison(startDate)
	for _, r := range records {
		if r.Date >= formattedStart {
			result = append(result, r)
		}
	}
	return result
}

func formatDateForComparison(date string) string {
	if len(date) == 8 {
		return date[:4] + "-" + date[4:6] + "-" + date[6:8]
	}
	return date
}
