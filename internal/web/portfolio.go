package web

import (
	"fmt"
	"math"
	"sort"

	"futures-backtest/internal/backtest"
)

type PortfolioResult struct {
	IDs             []string                  `json:"ids"`
	Symbols         []string                  `json:"symbols"`
	DailyRecords    []PortfolioDailyRecord    `json:"daily_records"`
	PositionReturns []PortfolioPositionReturn `json:"position_returns"`
	Statistics      PortfolioStatistics       `json:"statistics"`
}

type PortfolioDailyRecord struct {
	Date        string            `json:"date"`
	TotalValue  float64           `json:"total_value"`
	DailyReturn float64           `json:"daily_return"`
	PnL         float64           `json:"pnl"`
	Components  map[string]float64 `json:"components"`
}

type PortfolioPositionReturn struct {
	OpenDate   string  `json:"open_date"`
	CloseDate  string  `json:"close_date"`
	Symbol     string  `json:"symbol"`
	Direction  string  `json:"direction"`
	OpenPrice  float64 `json:"open_price"`
	ClosePrice float64 `json:"close_price"`
	Leverage   float64 `json:"leverage"`
	Return     float64 `json:"return"`
	Weight     float64 `json:"weight"`
}

type PortfolioStatistics struct {
	TotalReturn      float64 `json:"total_return"`
	AnnualReturn     float64 `json:"annual_return"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	MaxDrawdownRatio float64 `json:"max_drawdown_ratio"`
	WinRate          float64 `json:"win_rate"`
	ProfitLossRatio  float64 `json:"profit_loss_ratio"`
	WinningTrades    int     `json:"winning_trades"`
	LosingTrades     int     `json:"losing_trades"`
	TotalTrades      int     `json:"total_trades"`
	TotalWin         float64 `json:"total_win"`
	TotalLoss        float64 `json:"total_loss"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	CalmarRatio      float64 `json:"calmar_ratio"`
	TradingDays      int     `json:"trading_days"`
	FinalValue       float64 `json:"final_value"`
}

func CalculatePortfolio(results []*ResultData) (*PortfolioResult, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("至少需要选择一个回测结果")
	}

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
			baseValues[i] = 1.0
		}
	}

	weight := 1.0 / float64(len(results))

	var portfolioRecords []PortfolioDailyRecord
	var cumulativeValue = 1.0

	for _, date := range dates {
		var dailyReturnSum float64
		components := make(map[string]float64)
		validCount := 0

		for i, r := range results {
			if dr, ok := dailyMaps[i][date]; ok {
				dailyReturnSum += dr.DailyReturn
				if baseValues[i] != 0 {
					components[r.Request.Symbol] = dr.TotalValue / baseValues[i]
				} else {
					components[r.Request.Symbol] = dr.TotalValue
				}
				validCount++
			}
		}

		if validCount == 0 {
			continue
		}

		avgDailyReturn := dailyReturnSum / float64(validCount)
		cumulativeValue = cumulativeValue * (1 + avgDailyReturn)

		record := PortfolioDailyRecord{
			Date:        date,
			TotalValue:  cumulativeValue,
			DailyReturn: avgDailyReturn,
			PnL:         cumulativeValue - 1,
			Components:  components,
		}
		portfolioRecords = append(portfolioRecords, record)
	}

	var allReturns []PortfolioPositionReturn
	for _, r := range results {
		for _, pr := range r.PositionReturns {
			allReturns = append(allReturns, PortfolioPositionReturn{
				OpenDate:   pr.OpenDate,
				CloseDate:  pr.CloseDate,
				Symbol:     pr.Symbol,
				Direction:  pr.Direction.String(),
				OpenPrice:  pr.OpenPrice,
				ClosePrice: pr.ClosePrice,
				Leverage:   pr.Leverage,
				Return:     pr.Return,
				Weight:     weight,
			})
		}
	}

	stats := calculatePortfolioStatistics(portfolioRecords, allReturns)

	var ids, symbols []string
	for _, r := range results {
		ids = append(ids, r.ID)
		symbols = append(symbols, r.Request.Symbol)
	}

	return &PortfolioResult{
		IDs:             ids,
		Symbols:         symbols,
		DailyRecords:    portfolioRecords,
		PositionReturns: allReturns,
		Statistics:      stats,
	}, nil
}

func calculatePortfolioStatistics(records []PortfolioDailyRecord, returns []PortfolioPositionReturn) PortfolioStatistics {
	stats := PortfolioStatistics{
		TradingDays: len(records),
	}

	if len(records) == 0 {
		return stats
	}

	stats.FinalValue = records[len(records)-1].TotalValue
	stats.TotalReturn = stats.FinalValue - 1

	if len(records) > 1 {
		years := float64(len(records)) / 250.0
		if years > 0 {
			stats.AnnualReturn = math.Pow(stats.FinalValue, 1/years) - 1
		}
	}

	stats.MaxDrawdown, stats.MaxDrawdownRatio = calculatePortfolioMaxDrawdown(records)

	winCount, lossCount, totalWin, totalLoss := calculatePortfolioPositionStats(returns)
	stats.WinningTrades = winCount
	stats.LosingTrades = lossCount
	stats.TotalTrades = len(returns)
	stats.TotalWin = totalWin
	stats.TotalLoss = totalLoss

	if stats.TotalTrades > 0 {
		stats.WinRate = float64(winCount) / float64(stats.TotalTrades)
	}

	if winCount > 0 && lossCount > 0 {
		avgWin := totalWin / float64(winCount)
		avgLoss := totalLoss / float64(lossCount)
		if avgLoss != 0 {
			stats.ProfitLossRatio = avgWin / avgLoss
		}
	}

	stats.SharpeRatio = calculatePortfolioSharpeRatio(records)

	if stats.MaxDrawdownRatio != 0 {
		stats.CalmarRatio = stats.AnnualReturn / stats.MaxDrawdownRatio
	}

	return stats
}

func calculatePortfolioMaxDrawdown(records []PortfolioDailyRecord) (float64, float64) {
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

func calculatePortfolioPositionStats(returns []PortfolioPositionReturn) (winCount, lossCount int, totalWin, totalLoss float64) {
	for _, pr := range returns {
		pnl := pr.Return * pr.Leverage * pr.Weight
		if pnl > 0 {
			winCount++
			totalWin += pnl
		} else if pnl < 0 {
			lossCount++
			totalLoss += math.Abs(pnl)
		}
	}
	return
}

func calculatePortfolioSharpeRatio(records []PortfolioDailyRecord) float64 {
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

func CheckTimeOverlap(results []*ResultData) (bool, string) {
	if len(results) < 2 {
		return true, ""
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			r1, r2 := results[i], results[j]

			if r1.Request.StartDate == r2.Request.StartDate &&
				r1.Request.EndDate == r2.Request.EndDate {
				continue
			}

			start1, end1 := r1.Request.StartDate, r1.Request.EndDate
			start2, end2 := r2.Request.StartDate, r2.Request.EndDate

			if !isDateOverlap(start1, end1, start2, end2) {
				return false, fmt.Sprintf("%s(%s~%s) 与 %s(%s~%s) 时间段不重叠",
					r1.Request.Symbol, start1, end1,
					r2.Request.Symbol, start2, end2)
			}
		}
	}

	return true, ""
}

func isDateOverlap(start1, end1, start2, end2 string) bool {
	return start1 <= end2 && start2 <= end1
}

func GetCommonDateRange(results []*ResultData) (string, string) {
	if len(results) == 0 {
		return "", ""
	}

	commonStart := results[0].Request.StartDate
	commonEnd := results[0].Request.EndDate

	for _, r := range results[1:] {
		if r.Request.StartDate > commonStart {
			commonStart = r.Request.StartDate
		}
		if r.Request.EndDate < commonEnd {
			commonEnd = r.Request.EndDate
		}
	}

	return commonStart, commonEnd
}

func FilterResultsByDateRange(results []*ResultData, startDate, endDate string) []*ResultData {
	startFormatted := formatDateForComparison(startDate)
	endFormatted := formatDateForComparison(endDate)

	var filtered []*ResultData

	for _, r := range results {
		filteredResult := &ResultData{
			ID:              r.ID,
			Request:         r.Request,
			Signals:         filterSignalsByDateRange(r.Signals, startFormatted, endFormatted),
			DailyRecords:    filterDailyRecordsByDateRange(r.DailyRecords, startFormatted, endFormatted),
			PositionReturns: filterPositionReturnsByDateRange(r.PositionReturns, startFormatted, endFormatted),
			Statistics:      r.Statistics,
			StateHistory:    r.StateHistory,
			DominantMap:     r.DominantMap,
			Klines:          r.Klines,
		}
		filtered = append(filtered, filteredResult)
	}

	return filtered
}

func formatDateForComparison(date string) string {
	if len(date) == 8 {
		return date[:4] + "-" + date[4:6] + "-" + date[6:8]
	}
	return date
}

func filterSignalsByDateRange(signals []backtest.TradeSignal, startDate, endDate string) []backtest.TradeSignal {
	var filtered []backtest.TradeSignal
	for _, s := range signals {
		if s.SignalDate >= startDate && s.SignalDate <= endDate {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func filterDailyRecordsByDateRange(records []backtest.DailyRecord, startDate, endDate string) []backtest.DailyRecord {
	var filtered []backtest.DailyRecord
	for _, r := range records {
		if r.Date >= startDate && r.Date <= endDate {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func filterPositionReturnsByDateRange(returns []backtest.PositionReturn, startDate, endDate string) []backtest.PositionReturn {
	var filtered []backtest.PositionReturn
	for _, r := range returns {
		if r.OpenDate >= startDate && r.CloseDate <= endDate {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
