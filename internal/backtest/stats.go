package backtest

import (
	"fmt"
	"math"
)

type Statistics struct {
	TotalReturn      float64
	AnnualReturn     float64
	MaxDrawdown      float64
	MaxDrawdownRatio float64
	WinRate          float64
	ProfitLossRatio  float64
	WinningTrades    int
	LosingTrades     int
	TotalTrades      int
	TotalWin         float64
	TotalLoss        float64
	SharpeRatio      float64
	CalmarRatio      float64
	TradingDays      int
	FinalValue       float64
}

func CalculateStatistics(dailyRecords []DailyRecord, positionReturns []PositionReturn) Statistics {
	stats := Statistics{
		TradingDays: len(dailyRecords),
	}

	if len(dailyRecords) == 0 {
		return stats
	}

	stats.FinalValue = dailyRecords[len(dailyRecords)-1].TotalValue
	stats.TotalReturn = stats.FinalValue - 1

	if len(dailyRecords) > 1 {
		tradingDays := float64(len(dailyRecords))
		years := tradingDays / 250.0
		if years > 0 {
			stats.AnnualReturn = math.Pow(stats.FinalValue, 1/years) - 1
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
		stats.WinRate = float64(winCount) / float64(stats.TotalTrades)
	}

	if winCount > 0 && lossCount > 0 {
		avgWin := totalWin / float64(winCount)
		avgLoss := totalLoss / float64(lossCount)
		if avgLoss != 0 {
			stats.ProfitLossRatio = avgWin / avgLoss
		}
	}

	stats.SharpeRatio = calculateSharpeRatio(dailyRecords)

	if stats.MaxDrawdownRatio != 0 {
		stats.CalmarRatio = stats.AnnualReturn / stats.MaxDrawdownRatio
	}

	return stats
}

func calculateMaxDrawdown(records []DailyRecord) (float64, float64) {
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

func calculatePositionStats(returns []PositionReturn) (winCount, lossCount int, totalWin, totalLoss float64) {
	for _, pr := range returns {
		pnl := pr.Return * pr.Leverage
		if pnl > 0 {
			winCount++
			totalWin += pnl
		} else if pnl < 0 {
			lossCount++
			totalLoss += -pnl
		}
	}
	return
}

func calculateSharpeRatio(records []DailyRecord) float64 {
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

func formatSignedPercent(f float64, places int) string {
	pct := f * 100
	s := fmt.Sprintf("%.[1]*f%%", places, pct)
	if f > 0 {
		return "+" + s
	}
	return s
}

func (s Statistics) Print() {
	fmt.Println("========== 策略统计 ==========")
	fmt.Printf("  %-20s: %d 天\n", "交易天数", s.TradingDays)
	fmt.Printf("  %-20s: %.2f%%\n", "总收益率", s.TotalReturn*100)
	fmt.Printf("  %-20s: %.2f%%\n", "年化收益率", s.AnnualReturn*100)
	fmt.Printf("  %-20s: %.2f%%\n", "最大回撤", s.MaxDrawdown*100)
	fmt.Printf("  %-20s: %.2f%%\n", "最大回撤比例", s.MaxDrawdownRatio*100)
	fmt.Printf("  %-20s: %.2f\n", "夏普比率", s.SharpeRatio)
	fmt.Printf("  %-20s: %.2f\n", "卡尔马比率", s.CalmarRatio)
	fmt.Printf("  %-20s: %d 次\n", "交易次数", s.TotalTrades)
	fmt.Printf("  %-20s: %d 次\n", "盈利次数", s.WinningTrades)
	fmt.Printf("  %-20s: %d 次\n", "亏损次数", s.LosingTrades)
	fmt.Printf("  %-20s: %.2f%%\n", "胜率", s.WinRate*100)
	fmt.Printf("  %-20s: %.2f\n", "盈亏比", s.ProfitLossRatio)
	fmt.Printf("  %-20s: %.4f\n", "最终净值", s.FinalValue)
}

type DailyDetail struct {
	Date         string
	Symbol       string
	Direction    Direction
	OpenPrice    float64
	Close        float64
	DailyReturn  float64
	CumReturn    float64
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

	for _, rec := range dailyRecords {
		detail := DailyDetail{
			Date:        rec.Date,
			DailyReturn: rec.DailyReturn,
			CumReturn:   rec.TotalValue - 1,
		}

		if dominant, ok := dominantMap[rec.Date]; ok {
			detail.Symbol = dominant
		}

		if kline, ok := klineMap[rec.Date+"|"+detail.Symbol]; ok {
			detail.Close = kline.Close
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
		if d.OpenPrice == 0 {
			directionStr = "无持仓"
		}
		fmt.Printf("%-12s %-8s %-10s %-10.2f %-12.2f %s %s\n",
			d.Date, d.Symbol, directionStr, d.OpenPrice, d.Close, formatSignedPercent(d.DailyReturn, 4), formatSignedPercent(d.CumReturn, 4))
	}
}

func PrintPositionReturns(returns []PositionReturn) {
	fmt.Printf("\n========== 持仓收益记录 ==========\n")
	fmt.Printf("%-12s %-12s %-8s %-10s %-10s %-10s %-10s %-12s\n",
		"开仓日期", "平仓日期", "合约", "方向", "开仓价", "平仓价", "杠杆", "收益率")
	fmt.Println("--------------------------------------------------------------------------------")

	for _, pr := range returns {
		directionStr := formatDirection(pr.Direction)
		fmt.Printf("%-12s %-12s %-8s %-10s %-10.2f %-10.2f %-10.2f %s\n",
			pr.OpenDate, pr.CloseDate, pr.Symbol, directionStr, pr.OpenPrice, pr.ClosePrice, pr.Leverage, formatSignedPercent(pr.Return, 4))
	}
}
