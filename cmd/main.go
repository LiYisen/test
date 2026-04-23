package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"time"

	"futures-backtest/internal/backtest"
	"futures-backtest/internal/data"
	"futures-backtest/internal/strategy"
	"futures-backtest/internal/strategy/yinyang"

	"github.com/shopspring/decimal"
)

func init() {
	if runtime.GOOS == "windows" {
		_ = exec.Command("chcp", "65001").Run()
	}
}

func main() {
	symbol := flag.String("symbol", "IF", "品种代码（如 IF, ru, au）")
	startDate := flag.String("start", "20240101", "开始日期，格式: YYYYMMDD")
	endDate := flag.String("end", "20241231", "结束日期，格式: YYYYMMDD")
	leverage := flag.Float64("leverage", 3.0, "杠杆系数")
	strategyName := flag.String("strategy", "yinyang", "策略名称（如 yinyang）")
	flag.Parse()

	fmt.Println("========== 期货回测系统 ==========")
	fmt.Printf("品种: %s | 区间: %s ~ %s | 杠杆系数: %s | 策略: %s\n",
		*symbol, *startDate, *endDate, decimal.NewFromFloat(*leverage).StringFixed(2), *strategyName)
	fmt.Println()

	// 使用策略工厂创建策略
	factory, err := strategy.DefaultRegistry.Get(*strategyName)
	if err != nil {
		log.Fatalf("获取策略失败: %v", err)
	}

	strategyInstance := factory.Create(map[string]interface{}{
		"leverage": *leverage,
	})

	fmt.Printf("[0/5] 策略初始化完成: %s\n", factory.Description())

	dataManager := data.NewDefaultDataManager()
	fmt.Println("[1/5] 数据管理器初始化完成")

	calendar, err := dataManager.GetTradeCalendar(*startDate, *endDate)
	if err != nil {
		log.Fatalf("获取交易日历失败: %v", err)
	}
	fmt.Printf("[2/5] 获取交易日历完成，共 %d 天\n", len(calendar))

	contractSymbols := generateContractSymbols(*symbol, *startDate, *endDate)
	fmt.Printf("[3/5] 待查询合约列表: %v\n", contractSymbols)

	var allKlines []backtest.KLineWithContract
	for _, cs := range contractSymbols {
		klines, err := dataManager.GetFuturesKLine(cs, *startDate, *endDate)
		if err != nil {
			fmt.Printf("  合约 %s 获取失败: %v，跳过\n", cs, err)
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
		fmt.Printf("  合约 %s: %d 条K线\n", cs, len(klines))
	}
	fmt.Printf("  共获取 %d 条K线数据\n", len(allKlines))

	if len(allKlines) == 0 {
		log.Fatal("未获取到任何K线数据，无法继续回测")
	}

	identifier := data.NewDominantContractIdentifier(dataManager)
	dominantResult, err := identifier.Identify(*symbol, allKlines, *startDate, *endDate)
	if err != nil {
		log.Fatalf("识别主力合约失败: %v", err)
	}

	dominantMap := make(map[string]string, len(dominantResult))
	for t, sym := range dominantResult {
		dateStr := t.Format("2006-01-02")
		dominantMap[dateStr] = sym
	}
	fmt.Printf("[4/5] 识别主力合约完成，共 %d 天\n", len(dominantMap))

	// 创建换月处理器（阴阳线策略特有）
	var rollover backtest.RolloverHandler
	if adapter, ok := strategyInstance.(*yinyang.YinYangAdapter); ok {
		rollover = yinyang.NewRolloverHandler(adapter.GetStrategy())
	}

	signalEngine := backtest.NewSignalEngine(allKlines, dominantMap, strategyInstance, rollover)

	stateRecorder := yinyang.NewYinYangStateRecorder()
	signalEngine.SetStateRecorder(stateRecorder)

	signals, err := signalEngine.Calculate()
	if err != nil {
		log.Fatalf("计算交易信号失败: %v", err)
	}
	fmt.Printf("[5/5] 计算交易信号完成，共 %d 条信号\n", len(signals))

	dominantKlines := filterDominantKlines(allKlines, dominantMap)
	portfolioEngine := backtest.NewPortfolioEngine()
	dailyRecords, positionReturns, err := portfolioEngine.Calculate(signals, dominantKlines)
	if err != nil {
		log.Fatalf("计算资金收益失败: %v", err)
	}
	fmt.Printf("[6/6] 计算资金收益完成，共 %d 条持仓记录\n", len(positionReturns))

	stats := backtest.CalculateStatistics(dailyRecords, positionReturns)
	stats.Print()

	dailyDetails := backtest.GenerateDailyDetails(dailyRecords, signals, dominantKlines, dominantMap)
	backtest.PrintDailyDetails(dailyDetails)
	backtest.PrintPositionReturns(positionReturns)

	reporter := backtest.NewReporter(signals)
	reporter.SetStateHistory(stateRecorder.GetStateHistory())
	reporter.PrintSignals()
	reporter.PrintStateHistory()
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

func filterDominantKlines(allKlines []backtest.KLineWithContract, dominantMap map[string]string) []backtest.KLineWithContract {
	var dominantKlines []backtest.KLineWithContract
	for _, kl := range allKlines {
		if dominant, ok := dominantMap[kl.Date]; ok && dominant == kl.Symbol {
			dominantKlines = append(dominantKlines, kl)
		}
	}
	return dominantKlines
}
