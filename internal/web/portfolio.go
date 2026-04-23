package web

import (
	"fmt"
	"math"
	"sort"

	"futures-backtest/internal/backtest"

	"github.com/shopspring/decimal"
)

// PortfolioResult 组合回测结果
type PortfolioResult struct {
	IDs             []string                  `json:"ids"`
	Symbols         []string                  `json:"symbols"`
	DailyRecords    []PortfolioDailyRecord    `json:"daily_records"`
	PositionReturns []PortfolioPositionReturn `json:"position_returns"`
	Statistics      PortfolioStatistics       `json:"statistics"`
}

// PortfolioDailyRecord 组合每日记录
type PortfolioDailyRecord struct {
	Date        string                     `json:"date"`
	TotalValue  decimal.Decimal            `json:"total_value"`
	DailyReturn decimal.Decimal            `json:"daily_return"`
	PnL         decimal.Decimal            `json:"pnl"`
	Components  map[string]decimal.Decimal `json:"components"`
}

// PortfolioPositionReturn 组合持仓收益
type PortfolioPositionReturn struct {
	OpenDate   string          `json:"open_date"`
	CloseDate  string          `json:"close_date"`
	Symbol     string          `json:"symbol"`
	Direction  string          `json:"direction"`
	OpenPrice  decimal.Decimal `json:"open_price"`
	ClosePrice decimal.Decimal `json:"close_price"`
	Leverage   decimal.Decimal `json:"leverage"`
	Return     decimal.Decimal `json:"return"`
	Weight     decimal.Decimal `json:"weight"`
}

// PortfolioStatistics 组合统计指标
type PortfolioStatistics struct {
	TotalReturn      decimal.Decimal `json:"total_return"`
	AnnualReturn     decimal.Decimal `json:"annual_return"`
	MaxDrawdown      decimal.Decimal `json:"max_drawdown"`
	MaxDrawdownRatio decimal.Decimal `json:"max_drawdown_ratio"`
	WinRate          decimal.Decimal `json:"win_rate"`
	ProfitLossRatio  decimal.Decimal `json:"profit_loss_ratio"`
	WinningTrades    int             `json:"winning_trades"`
	LosingTrades     int             `json:"losing_trades"`
	TotalTrades      int             `json:"total_trades"`
	TotalWin         decimal.Decimal `json:"total_win"`
	TotalLoss        decimal.Decimal `json:"total_loss"`
	SharpeRatio      decimal.Decimal `json:"sharpe_ratio"`
	CalmarRatio      decimal.Decimal `json:"calmar_ratio"`
	TradingDays      int             `json:"trading_days"`
	FinalValue       decimal.Decimal `json:"final_value"`
}

// CalculatePortfolio 计算多品种组合回测
// 假设各品种等权重分配资金，独立计算后再汇总
// 注意：调用此函数前应已通过 FilterResultsByDateRange 过滤到共同日期范围
func CalculatePortfolio(results []*ResultData) (*PortfolioResult, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("至少需要选择一个回测结果")
	}

	// 收集所有日期
	dateSet := make(map[string]bool)
	for _, r := range results {
		for _, dr := range r.DailyRecords {
			dateSet[dr.Date] = true
		}
	}

	// 排序日期
	var dates []string
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	decOne := decimal.NewFromInt(1)

	// 为每个结果构建日期到每日记录的映射，并计算归一化基准值
	dailyMaps := make([]map[string]backtest.DailyRecord, len(results))
	baseValues := make([]decimal.Decimal, len(results))
	for i, r := range results {
		dailyMaps[i] = make(map[string]backtest.DailyRecord)
		for _, dr := range r.DailyRecords {
			dailyMaps[i][dr.Date] = dr
		}
		// 找到第一条记录作为基准值，用于归一化
		if len(r.DailyRecords) > 0 {
			baseValues[i] = r.DailyRecords[0].TotalValue
		} else {
			baseValues[i] = decOne
		}
	}

	// 等权重
	weight := decimal.NewFromInt(1).Div(decimal.NewFromInt(int64(len(results))))

	// 计算组合每日记录
	var portfolioRecords []PortfolioDailyRecord
	var cumulativeValue = decOne

	for _, date := range dates {
		// 当日各品种的日收益率
		var dailyReturnSum decimal.Decimal
		components := make(map[string]decimal.Decimal)
		validCount := 0

		for i, r := range results {
			if dr, ok := dailyMaps[i][date]; ok {
				dailyReturnSum = dailyReturnSum.Add(dr.DailyReturn)
				// 归一化净值：将净值归一化到共同日期范围的起点
				if !baseValues[i].IsZero() {
					components[r.Request.Symbol] = dr.TotalValue.Div(baseValues[i])
				} else {
					components[r.Request.Symbol] = dr.TotalValue
				}
				validCount++
			}
		}

		if validCount == 0 {
			continue
		}

		// 组合日收益率 = 各品种日收益率的等权平均
		avgDailyReturn := dailyReturnSum.Div(decimal.NewFromInt(int64(validCount)))

		// 累计净值
		cumulativeValue = cumulativeValue.Mul(decOne.Add(avgDailyReturn))

		record := PortfolioDailyRecord{
			Date:        date,
			TotalValue:  cumulativeValue,
			DailyReturn: avgDailyReturn,
			PnL:         cumulativeValue.Sub(decOne),
			Components:  components,
		}
		portfolioRecords = append(portfolioRecords, record)
	}

	// 收集所有持仓收益记录
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

	// 计算统计指标
	stats := calculatePortfolioStatistics(portfolioRecords, allReturns)

	// 提取ID和品种列表
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
	decOne := decimal.NewFromInt(1)
	stats.TotalReturn = stats.FinalValue.Sub(decOne)

	if len(records) > 1 {
		tradingDays := float64(len(records))
		years := tradingDays / 250.0
		if years > 0 {
			fv, _ := stats.FinalValue.Float64()
			stats.AnnualReturn = decimal.NewFromFloat(math.Pow(fv, 1/years) - 1)
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
		stats.WinRate = decimal.NewFromInt(int64(winCount)).Div(decimal.NewFromInt(int64(stats.TotalTrades)))
	}

	if winCount > 0 && lossCount > 0 {
		avgWin := totalWin.Div(decimal.NewFromInt(int64(winCount)))
		avgLoss := totalLoss.Div(decimal.NewFromInt(int64(lossCount)))
		if !avgLoss.IsZero() {
			stats.ProfitLossRatio = avgWin.Div(avgLoss)
		}
	}

	stats.SharpeRatio = calculatePortfolioSharpeRatio(records)

	if !stats.MaxDrawdownRatio.IsZero() {
		stats.CalmarRatio = stats.AnnualReturn.Div(stats.MaxDrawdownRatio)
	}

	return stats
}

func calculatePortfolioMaxDrawdown(records []PortfolioDailyRecord) (decimal.Decimal, decimal.Decimal) {
	if len(records) == 0 {
		return decimal.Zero, decimal.Zero
	}

	peak := decimal.NewFromInt(1)
	maxDD := decimal.Zero
	maxDDPercent := decimal.Zero

	for _, rec := range records {
		if rec.TotalValue.GreaterThan(peak) {
			peak = rec.TotalValue
		}
		drawdown := peak.Sub(rec.TotalValue)
		drawdownPercent := drawdown.Div(peak)

		if drawdown.GreaterThan(maxDD) {
			maxDD = drawdown
			maxDDPercent = drawdownPercent
		}
	}

	return maxDD, maxDDPercent
}

func calculatePortfolioPositionStats(returns []PortfolioPositionReturn) (winCount, lossCount int, totalWin, totalLoss decimal.Decimal) {
	totalWin = decimal.Zero
	totalLoss = decimal.Zero
	for _, pr := range returns {
		pnl := pr.Return.Mul(pr.Leverage).Mul(pr.Weight)
		if pnl.IsPositive() {
			winCount++
			totalWin = totalWin.Add(pnl)
		} else if pnl.IsNegative() {
			lossCount++
			totalLoss = totalLoss.Add(pnl.Abs())
		}
	}
	return
}

func calculatePortfolioSharpeRatio(records []PortfolioDailyRecord) decimal.Decimal {
	if len(records) < 2 {
		return decimal.Zero
	}

	var sum, sqSum float64
	for _, rec := range records {
		r, _ := rec.DailyReturn.Float64()
		sum += r
		sqSum += r * r
	}

	mean := sum / float64(len(records))
	variance := sqSum/float64(len(records)) - mean*mean

	if variance <= 0 {
		return decimal.Zero
	}

	stdDev := math.Sqrt(variance)
	if stdDev == 0 {
		return decimal.Zero
	}

	sharpe := mean / stdDev
	if math.IsInf(sharpe, 0) || math.IsNaN(sharpe) {
		return decimal.Zero
	}

	return decimal.NewFromFloat(sharpe * math.Sqrt(250))
}

// CheckTimeOverlap 检查多个回测结果的时间段是否有重叠
func CheckTimeOverlap(results []*ResultData) (bool, string) {
	if len(results) < 2 {
		return true, ""
	}

	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			r1, r2 := results[i], results[j]

			// 检查是否完全相同
			if r1.Request.StartDate == r2.Request.StartDate &&
				r1.Request.EndDate == r2.Request.EndDate {
				continue // 完全相同是允许的
			}

			// 检查是否有交集
			start1, end1 := r1.Request.StartDate, r1.Request.EndDate
			start2, end2 := r2.Request.StartDate, r2.Request.EndDate

			// 将日期字符串转换为可比较的格式
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
	// 简单字符串比较，假设格式为 YYYYMMDD
	// 重叠条件：start1 <= end2 且 start2 <= end1
	return start1 <= end2 && start2 <= end1
}

// GetCommonDateRange 获取多个回测结果的共同日期范围
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

// FilterResultsByDateRange 根据共同日期范围过滤结果
// startDate/endDate 格式为 YYYYMMDD，DailyRecord.Date 格式为 YYYY-MM-DD
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

// formatDateForComparison 将 YYYYMMDD 格式转换为 YYYY-MM-DD 格式
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
