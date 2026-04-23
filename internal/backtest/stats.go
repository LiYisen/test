package backtest

import (
	"fmt"
	"math"

	"github.com/shopspring/decimal"
)

type Statistics struct {
	TotalReturn      decimal.Decimal
	AnnualReturn     decimal.Decimal
	MaxDrawdown      decimal.Decimal
	MaxDrawdownRatio decimal.Decimal
	WinRate          decimal.Decimal
	ProfitLossRatio  decimal.Decimal
	WinningTrades    int
	LosingTrades     int
	TotalTrades      int
	TotalWin         decimal.Decimal
	TotalLoss        decimal.Decimal
	SharpeRatio      decimal.Decimal
	CalmarRatio      decimal.Decimal
	TradingDays      int
	FinalValue       decimal.Decimal
}

func CalculateStatistics(dailyRecords []DailyRecord, positionReturns []PositionReturn) Statistics {
	stats := Statistics{
		TradingDays: len(dailyRecords),
	}

	if len(dailyRecords) == 0 {
		return stats
	}

	stats.FinalValue = dailyRecords[len(dailyRecords)-1].TotalValue
	decOne := decimal.NewFromInt(1)
	stats.TotalReturn = stats.FinalValue.Sub(decOne)

	if len(dailyRecords) > 1 {
		tradingDays := float64(len(dailyRecords))
		years := tradingDays / 250.0
		if years > 0 {
			fv, _ := stats.FinalValue.Float64()
			stats.AnnualReturn = decimal.NewFromFloat(math.Pow(fv, 1/years) - 1)
		}
	}

	stats.MaxDrawdown, stats.MaxDrawdownRatio = calculateMaxDrawdown(dailyRecords)

	winCount, lossCount, totalWin, totalLoss := calculatePositionStats(positionReturns)
	stats.WinningTrades = winCount
	stats.LosingTrades = lossCount
	stats.TotalTrades = len(positionReturns)
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

	stats.SharpeRatio = calculateSharpeRatio(dailyRecords)

	if !stats.MaxDrawdownRatio.IsZero() {
		stats.CalmarRatio = stats.AnnualReturn.Div(stats.MaxDrawdownRatio)
	}

	return stats
}

func calculateMaxDrawdown(records []DailyRecord) (decimal.Decimal, decimal.Decimal) {
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

func calculatePositionStats(returns []PositionReturn) (winCount, lossCount int, totalWin, totalLoss decimal.Decimal) {
	totalWin = decimal.Zero
	totalLoss = decimal.Zero
	for _, pr := range returns {
		pnl := pr.Return.Mul(pr.Leverage)
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

func calculateSharpeRatio(records []DailyRecord) decimal.Decimal {
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

func formatSignedPercent(d decimal.Decimal, places int32) string {
	s := d.Mul(decimal.NewFromInt(100)).StringFixed(places)
	if d.IsPositive() {
		return "+" + s
	}
	return s
}

func (s Statistics) Print() {
	fmt.Println("========== 策略统计 ==========")
	fmt.Printf("  %-20s: %d 天\n", "交易天数", s.TradingDays)
	fmt.Printf("  %-20s: %s%%\n", "总收益率", s.TotalReturn.Mul(decimal.NewFromInt(100)).StringFixed(2))
	fmt.Printf("  %-20s: %s%%\n", "年化收益率", s.AnnualReturn.Mul(decimal.NewFromInt(100)).StringFixed(2))
	fmt.Printf("  %-20s: %s%%\n", "最大回撤", s.MaxDrawdown.Mul(decimal.NewFromInt(100)).StringFixed(2))
	fmt.Printf("  %-20s: %s%%\n", "最大回撤比例", s.MaxDrawdownRatio.Mul(decimal.NewFromInt(100)).StringFixed(2))
	fmt.Printf("  %-20s: %s\n", "夏普比率", s.SharpeRatio.StringFixed(2))
	fmt.Printf("  %-20s: %s\n", "卡尔马比率", s.CalmarRatio.StringFixed(2))
	fmt.Printf("  %-20s: %d 次\n", "交易次数", s.TotalTrades)
	fmt.Printf("  %-20s: %d 次\n", "盈利次数", s.WinningTrades)
	fmt.Printf("  %-20s: %d 次\n", "亏损次数", s.LosingTrades)
	fmt.Printf("  %-20s: %s%%\n", "胜率", s.WinRate.Mul(decimal.NewFromInt(100)).StringFixed(2))
	fmt.Printf("  %-20s: %s\n", "盈亏比", s.ProfitLossRatio.StringFixed(2))
	fmt.Printf("  %-20s: %s\n", "最终净值", s.FinalValue.StringFixed(4))
}

type DailyDetail struct {
	Date         string
	Symbol       string
	Direction    Direction
	OpenPrice    decimal.Decimal
	Close        decimal.Decimal
	DailyReturn  decimal.Decimal
	CumReturn    decimal.Decimal
	SignalAction string
}

func GenerateDailyDetails(dailyRecords []DailyRecord, signals []TradeSignal, klines []KLineWithContract, dominantMap map[string]string) []DailyDetail {
	klineMap := make(map[string]KLineWithContract)
	for _, kl := range klines {
		klineMap[kl.Date+"|"+kl.Symbol] = kl
	}

	signalsByDate := make(map[string][]TradeSignal)
	for _, sig := range signals {
		signalsByDate[sig.SignalDate] = append(signalsByDate[sig.SignalDate], sig)
	}

	var details []DailyDetail
	var currentPos *TradeSignal
	decOne := decimal.NewFromInt(1)

	for _, rec := range dailyRecords {
		detail := DailyDetail{
			Date:        rec.Date,
			DailyReturn: rec.DailyReturn,
			CumReturn:   rec.TotalValue.Sub(decOne),
		}

		if dominant, ok := dominantMap[rec.Date]; ok {
			detail.Symbol = dominant
		}

		if kline, ok := klineMap[rec.Date+"|"+detail.Symbol]; ok {
			detail.Close = decimal.NewFromFloat(kline.Close)
		}

		if daySignals, ok := signalsByDate[rec.Date]; ok {
			for _, sig := range daySignals {
				if sig.Direction == Buy || sig.Direction == Sell {
					currentPos = &sig
				} else if sig.Direction == CloseLong || sig.Direction == CloseShort || sig.Direction == Close {
					currentPos = nil
				}
			}
		}

		if currentPos != nil {
			detail.Direction = currentPos.Direction
			detail.OpenPrice = currentPos.Price
			detail.SignalAction = formatDirection(currentPos.Direction)
		}

		if detail.SignalAction == "" {
			detail.SignalAction = "无持仓"
		}

		details = append(details, detail)
	}

	return details
}

func formatDirection(d Direction) string {
	switch d {
	case Buy:
		return "开多"
	case Sell:
		return "开空"
	case CloseLong:
		return "平多"
	case CloseShort:
		return "平空"
	case Close:
		return "平仓"
	default:
		return "无"
	}
}

func PrintDailyDetails(details []DailyDetail) {
	fmt.Printf("\n========== 每日持仓详情 ==========\n")
	fmt.Printf("%-12s %-8s %-10s %-10s %-12s %-12s %-12s\n",
		"日期", "合约", "方向", "开仓价", "收盘价", "日收益率", "累计收益率")
	fmt.Println("------------------------------------------------------------------------------------------")

	for _, d := range details {
		directionStr := d.SignalAction
		if d.OpenPrice.IsZero() {
			directionStr = "无持仓"
		}
		fmt.Printf("%-12s %-8s %-10s %-10s %-12s %s%% %s%%\n",
			d.Date, d.Symbol, directionStr, d.OpenPrice.StringFixed(2), d.Close.StringFixed(2), formatSignedPercent(d.DailyReturn, 4), formatSignedPercent(d.CumReturn, 4))
	}
}

func PrintPositionReturns(returns []PositionReturn) {
	fmt.Printf("\n========== 持仓收益记录 ==========\n")
	fmt.Printf("%-12s %-12s %-8s %-10s %-10s %-10s %-10s %-12s\n",
		"开仓日期", "平仓日期", "合约", "方向", "开仓价", "平仓价", "杠杆", "收益率")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, pr := range returns {
		directionStr := formatDirection(pr.Direction)
		fmt.Printf("%-12s %-12s %-8s %-10s %-10s %-10s %-10s %s%%\n",
			pr.OpenDate, pr.CloseDate, pr.Symbol, directionStr, pr.OpenPrice.StringFixed(2), pr.ClosePrice.StringFixed(2), pr.Leverage.StringFixed(2), formatSignedPercent(pr.Return, 4))
	}
}
