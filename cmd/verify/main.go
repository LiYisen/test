package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"

	"futures-backtest/internal/backtest"
	"futures-backtest/internal/fund"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("  基金模式 vs 组合分析 对比验证")
	fmt.Println("  (从已有结果文件加载)")
	fmt.Println("========================================")

	rbFile := "ret/RB_yinyang_20240101_20240331_3_1777296806.json"
	taFile := "ret/TA_yinyang_20240101_20240331_2_1777296824.json"
	fundFile := "ret/funding/demo_fund/demo_fund_20240101_20240331_1777365805/fund_result.json"

	rbResult, err := loadResultData(rbFile)
	if err != nil {
		log.Fatalf("加载RB结果失败: %v", err)
	}
	taResult, err := loadResultData(taFile)
	if err != nil {
		log.Fatalf("加载TA结果失败: %v", err)
	}
	fundResult, err := loadFundResult(fundFile)
	if err != nil {
		log.Fatalf("加载基金结果失败: %v", err)
	}

	fmt.Printf("RB: 交易天数=%d, 总收益=%.2f%%, 杠杆=%.0f\n",
		rbResult.Statistics.TradingDays,
		rbResult.Statistics.TotalReturn*100,
		rbResult.Request.Leverage)
	fmt.Printf("TA: 交易天数=%d, 总收益=%.2f%%, 杠杆=%.0f\n",
		taResult.Statistics.TradingDays,
		taResult.Statistics.TotalReturn*100,
		taResult.Request.Leverage)
	fmt.Println()

	symbols := []symbolInput{
		{name: "RB", dailyRecords: rbResult.DailyRecords, statistics: rbResult.Statistics},
		{name: "TA", dailyRecords: taResult.DailyRecords, statistics: taResult.Statistics},
	}

	fmt.Println(">>> 第1步: 基金模式合并（加权，权重=0.5/0.5） <<<")
	fundRecords := mergeFundMode(symbols)
	fundStats := calcStats(fundRecords, symbols)
	printStats("基金模式(加权0.5/0.5)", fundStats)

	fmt.Println()
	fmt.Println(">>> 第2步: 组合分析模式合并（等权平均） <<<")
	portfolioRecords := mergePortfolioMode(symbols)
	portfolioStats := calcStats(portfolioRecords, symbols)
	printStats("组合分析(等权平均)", portfolioStats)

	fmt.Println()
	fmt.Println(">>> 第3步: FundEngine 实际结果 <<<")
	fundEngineRecords := make([]simpleRecord, len(fundResult.DailyRecords))
	for i, r := range fundResult.DailyRecords {
		fundEngineRecords[i] = simpleRecord{Date: r.Date, TotalValue: r.TotalValue, DailyReturn: r.DailyReturn}
	}
	fundEngineStats := simpleStats{
		TradingDays:      fundResult.Statistics.TradingDays,
		TotalReturn:      fundResult.Statistics.TotalReturn,
		AnnualReturn:     fundResult.Statistics.AnnualReturn,
		MaxDrawdownRatio: fundResult.Statistics.MaxDrawdownRatio,
		SharpeRatio:      fundResult.Statistics.SharpeRatio,
		WinRate:          fundResult.Statistics.WinRate,
		TotalTrades:      fundResult.Statistics.TotalTrades,
	}
	printStats("FundEngine实际", fundEngineStats)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  对比结果")
	fmt.Println("========================================")

	compare("基金模式(加权)", "组合分析(等权)", fundRecords, fundStats, portfolioRecords, portfolioStats)
	compare("基金模式(加权)", "FundEngine", fundRecords, fundStats, fundEngineRecords, fundEngineStats)
	compare("组合分析(等权)", "FundEngine", portfolioRecords, portfolioStats, fundEngineRecords, fundEngineStats)

	fmt.Println()
	fmt.Println(">>> 逐日对比（前30天） <<<")
	fmt.Printf("%-12s  %-14s  %-14s  %-14s  %-14s  %-14s\n",
		"日期", "基金净值", "组合净值", "FundEngine净值", "基金日收益", "组合日收益")
	fmt.Println("-----------------------------------------------------------------------------------------------")
	maxCompare := 30
	for i := 0; i < maxCompare && i < len(fundRecords) && i < len(portfolioRecords) && i < len(fundEngineRecords); i++ {
		fr := fundRecords[i]
		pr := portfolioRecords[i]
		er := fundEngineRecords[i]
		fmt.Printf("%-12s  %-14.6f  %-14.6f  %-14.6f  %-14.6f  %-14.6f\n",
			fr.Date,
			fr.TotalValue,
			pr.TotalValue,
			er.TotalValue,
			fr.DailyReturn,
			pr.DailyReturn)
	}

	fmt.Println()
	fmt.Println(">>> 关键差异分析 <<<")
	analyzeDifferences(fundRecords, portfolioRecords, fundEngineRecords, symbols)
}

type simpleRecord struct {
	Date        string
	TotalValue  float64
	DailyReturn float64
}

type simpleStats struct {
	TradingDays      int
	TotalReturn      float64
	AnnualReturn     float64
	MaxDrawdownRatio float64
	SharpeRatio      float64
	WinRate          float64
	TotalTrades      int
}

type resultData struct {
	ID              string                    `json:"id"`
	Request         backtestRequest           `json:"request"`
	DailyRecords    []backtest.DailyRecord    `json:"daily_records"`
	PositionReturns []backtest.PositionReturn `json:"position_returns"`
	Statistics      backtest.Statistics       `json:"statistics"`
}

type backtestRequest struct {
	Symbol    string                 `json:"symbol"`
	StartDate string                 `json:"start_date"`
	EndDate   string                 `json:"end_date"`
	Leverage  float64                `json:"leverage"`
	Strategy  string                 `json:"strategy"`
	Params    map[string]interface{} `json:"params"`
}

func loadResultData(filename string) (*resultData, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var result resultData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func loadFundResult(filename string) (*fund.FundResult, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var result fund.FundResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type symbolInput struct {
	name         string
	dailyRecords []backtest.DailyRecord
	statistics   backtest.Statistics
}

func mergeFundMode(symbols []symbolInput) []simpleRecord {
	dateSet := make(map[string]bool)
	for _, s := range symbols {
		for _, dr := range s.dailyRecords {
			dateSet[dr.Date] = true
		}
	}

	var dates []string
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	dailyMaps := make([]map[string]backtest.DailyRecord, len(symbols))
	for i, s := range symbols {
		dailyMaps[i] = make(map[string]backtest.DailyRecord)
		for _, dr := range s.dailyRecords {
			dailyMaps[i][dr.Date] = dr
		}
	}

	weight := 1.0 / float64(len(symbols))

	var records []simpleRecord
	var cumulativeValue = 1.0

	for _, date := range dates {
		var weightedReturn float64
		totalWeight := 0.0

		for i := range symbols {
			if dr, ok := dailyMaps[i][date]; ok {
				weightedReturn += dr.DailyReturn * weight
				totalWeight += weight
			}
		}

		if totalWeight == 0 {
			continue
		}

		cumulativeValue = cumulativeValue * (1 + weightedReturn)

		records = append(records, simpleRecord{
			Date:        date,
			TotalValue:  cumulativeValue,
			DailyReturn: weightedReturn,
		})
	}

	return records
}

func mergePortfolioMode(symbols []symbolInput) []simpleRecord {
	dateSet := make(map[string]bool)
	for _, s := range symbols {
		for _, dr := range s.dailyRecords {
			dateSet[dr.Date] = true
		}
	}

	var dates []string
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	dailyMaps := make([]map[string]backtest.DailyRecord, len(symbols))
	for i, s := range symbols {
		dailyMaps[i] = make(map[string]backtest.DailyRecord)
		for _, dr := range s.dailyRecords {
			dailyMaps[i][dr.Date] = dr
		}
	}

	var records []simpleRecord
	var cumulativeValue = 1.0

	for _, date := range dates {
		var dailyReturnSum float64
		validCount := 0

		for i := range symbols {
			if dr, ok := dailyMaps[i][date]; ok {
				dailyReturnSum += dr.DailyReturn
				validCount++
			}
		}

		if validCount == 0 {
			continue
		}

		avgDailyReturn := dailyReturnSum / float64(validCount)
		cumulativeValue = cumulativeValue * (1 + avgDailyReturn)

		records = append(records, simpleRecord{
			Date:        date,
			TotalValue:  cumulativeValue,
			DailyReturn: avgDailyReturn,
		})
	}

	return records
}

func calcStats(records []simpleRecord, symbols []symbolInput) simpleStats {
	stats := simpleStats{TradingDays: len(records)}
	if len(records) == 0 {
		return stats
	}

	stats.TotalReturn = records[len(records)-1].TotalValue - 1

	if len(records) > 1 {
		fv := records[len(records)-1].TotalValue
		years := float64(len(records)) / 250.0
		if years > 0 {
			stats.AnnualReturn = math.Pow(fv, 1/years) - 1
		}
	}

	peak := 1.0
	maxDDPercent := 0.0
	for _, rec := range records {
		if rec.TotalValue > peak {
			peak = rec.TotalValue
		}
		drawdownPercent := (peak - rec.TotalValue) / peak
		if drawdownPercent > maxDDPercent {
			maxDDPercent = drawdownPercent
		}
	}
	stats.MaxDrawdownRatio = maxDDPercent

	winCount, lossCount := 0, 0
	for _, s := range symbols {
		winCount += s.statistics.WinningTrades
		lossCount += s.statistics.LosingTrades
	}
	stats.TotalTrades = winCount + lossCount
	if stats.TotalTrades > 0 {
		stats.WinRate = float64(winCount) / float64(stats.TotalTrades)
	}

	var sum, sqSum float64
	for _, rec := range records {
		r := rec.DailyReturn
		sum += r
		sqSum += r * r
	}
	mean := sum / float64(len(records))
	variance := sqSum/float64(len(records)) - mean*mean
	if variance > 0 {
		stdDev := math.Sqrt(variance)
		if stdDev > 0 {
			stats.SharpeRatio = mean / stdDev * math.Sqrt(250)
		}
	}

	return stats
}

func printStats(name string, stats simpleStats) {
	fmt.Printf("%s:\n", name)
	fmt.Printf("  交易天数=%d, 总收益=%.2f%%, 年化=%.2f%%, 最大回撤=%.2f%%, 夏普=%.2f\n",
		stats.TradingDays,
		stats.TotalReturn*100,
		stats.AnnualReturn*100,
		stats.MaxDrawdownRatio*100,
		stats.SharpeRatio)
}

func compare(name1, name2 string, records1 []simpleRecord, stats1 simpleStats, records2 []simpleRecord, stats2 simpleStats) {
	fmt.Printf("\n--- %s vs %s ---\n", name1, name2)

	fmt.Printf("  总收益: %.6f vs %.6f", stats1.TotalReturn, stats2.TotalReturn)
	if stats1.TotalReturn == stats2.TotalReturn {
		fmt.Printf(" [一致]\n")
	} else {
		diff := stats1.TotalReturn - stats2.TotalReturn
		fmt.Printf(" [差异: %.8f]\n", diff)
	}

	fmt.Printf("  年化收益: %.6f vs %.6f", stats1.AnnualReturn, stats2.AnnualReturn)
	if stats1.AnnualReturn == stats2.AnnualReturn {
		fmt.Printf(" [一致]\n")
	} else {
		diff := stats1.AnnualReturn - stats2.AnnualReturn
		fmt.Printf(" [差异: %.8f]\n", diff)
	}

	fmt.Printf("  最大回撤: %.6f vs %.6f", stats1.MaxDrawdownRatio, stats2.MaxDrawdownRatio)
	if stats1.MaxDrawdownRatio == stats2.MaxDrawdownRatio {
		fmt.Printf(" [一致]\n")
	} else {
		diff := stats1.MaxDrawdownRatio - stats2.MaxDrawdownRatio
		fmt.Printf(" [差异: %.8f]\n", diff)
	}

	fmt.Printf("  交易天数: %d vs %d", stats1.TradingDays, stats2.TradingDays)
	if stats1.TradingDays == stats2.TradingDays {
		fmt.Printf(" [一致]\n")
	} else {
		fmt.Printf(" [差异: %d]\n", stats1.TradingDays-stats2.TradingDays)
	}

	minLen := len(records1)
	if len(records2) < minLen {
		minLen = len(records2)
	}
	diffCount := 0
	var maxDiff float64
	for i := 0; i < minLen; i++ {
		if records1[i].TotalValue != records2[i].TotalValue {
			diffCount++
			d := records1[i].TotalValue - records2[i].TotalValue
			if d < 0 {
				d = -d
			}
			if d > maxDiff {
				maxDiff = d
			}
		}
	}
	fmt.Printf("  逐日净值差异: %d/%d 天, 最大差异: %.8f\n", diffCount, minLen, maxDiff)
}

func analyzeDifferences(fundRecords, portfolioRecords, fundEngineRecords []simpleRecord, symbols []symbolInput) {
	dailyMaps := make([]map[string]backtest.DailyRecord, len(symbols))
	for i, s := range symbols {
		dailyMaps[i] = make(map[string]backtest.DailyRecord)
		for _, dr := range s.dailyRecords {
			dailyMaps[i][dr.Date] = dr
		}
	}

	fmt.Println("\n--- 逐日差异详情（前10个差异日） ---")
	fmt.Printf("%-12s  %-10s  %-12s  %-12s  %-12s  %-12s\n",
		"日期", "品种", "品种日收益", "基金加权收益", "组合等权收益", "差异原因")
	fmt.Println("-------------------------------------------------------------------------------------")

	diffShown := 0
	for i := 0; i < len(fundRecords) && diffShown < 10; i++ {
		fr := fundRecords[i]
		pr := portfolioRecords[i]

		if fr.DailyReturn != pr.DailyReturn {
			date := fr.Date
			details := ""
			for j, s := range symbols {
				if dr, ok := dailyMaps[j][date]; ok {
					details += fmt.Sprintf("%s:%.4f ", s.name, dr.DailyReturn)
				} else {
					details += fmt.Sprintf("%s:无数据 ", s.name)
				}
			}

			fmt.Printf("%-12s  %-10s  %-12s  %-12.6f  %-12.6f  %s\n",
				date, "", "", fr.DailyReturn, pr.DailyReturn, details)
			diffShown++
		}
	}

	if diffShown == 0 {
		fmt.Println("  无差异！基金模式和组合分析模式的日收益率完全一致。")
	}

	fmt.Println()
	fmt.Println("--- 差异原因分析 ---")

	allSame := true
	for i := 0; i < len(symbols); i++ {
		for j := i + 1; j < len(symbols); j++ {
			for date := range dailyMaps[i] {
				_, hasJ := dailyMaps[j][date]
				if !hasJ {
					fmt.Printf("  品种 %s 在 %s 有数据，但品种 %s 无数据 -> 导致权重差异\n",
						symbols[i].name, date, symbols[j].name)
					allSame = false
					break
				}
			}
		}
	}

	if allSame {
		fmt.Println("  所有品种在每个交易日都有数据，基金模式和组合分析模式应该产生一致结果。")
		fmt.Println("  但由于权重计算方式不同（加权 vs 等权平均），当权重不为均匀分配时会产生差异。")
	}

	fmt.Println()
	fmt.Println("--- 基金模式 vs FundEngine 差异分析 ---")
	minLen := len(fundRecords)
	if len(fundEngineRecords) < minLen {
		minLen = len(fundEngineRecords)
	}
	fmt.Printf("  基金模式天数: %d, FundEngine天数: %d\n", len(fundRecords), len(fundEngineRecords))

	fundDiffCount := 0
	for i := 0; i < minLen; i++ {
		if fundRecords[i].TotalValue != fundEngineRecords[i].TotalValue {
			fundDiffCount++
		}
	}
	fmt.Printf("  净值差异天数: %d/%d\n", fundDiffCount, minLen)

	if fundDiffCount > 0 {
		fmt.Println("  差异原因：FundEngine使用并发执行品种回测，可能产生不同的回测参数（如杠杆）。")
		fmt.Printf("  注意：RB杠杆=%.0f, TA杠杆=%.0f, 但基金配置中RB=3, TA=2\n",
			symbols[0].statistics.FinalValue, symbols[1].statistics.FinalValue)
	}
}
