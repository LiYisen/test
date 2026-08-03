package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"time"

	"futures-backtest/internal/backtest"
	"futures-backtest/internal/common"
	"futures-backtest/internal/data"
	"futures-backtest/internal/strategy"
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
	fmt.Printf("品种: %s | 区间: %s ~ %s | 杠杆系数: %.2f | 策略: %s\n",
		*symbol, *startDate, *endDate, *leverage, *strategyName)
	fmt.Println()

	factory, err := strategy.DefaultRegistry.Get(*strategyName)
	if err != nil {
		log.Fatalf("获取策略失败: %v", err)
	}

	params := map[string]interface{}{
		"leverage": *leverage,
	}

	fmt.Printf("[0/5] 策略初始化完成: %s\n", factory.Description())

	dataManager := data.NewDefaultDataManager()
	fmt.Println("[1/5] 数据管理器初始化完成")

	calendar, err := dataManager.GetTradeCalendar(*startDate, *endDate)
	if err != nil {
		log.Fatalf("获取交易日历失败: %v", err)
	}
	fmt.Printf("[2/5] 获取交易日历完成，共 %d 天\n", len(calendar))

	contractSymbols := common.GenerateContractSymbols(*symbol, *startDate, *endDate)
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

	warmupDays := factory.GetWarmupDays(params)
	backtestStartDate := ""
	if warmupDays > 0 {
		start, _ := time.Parse("20060102", *startDate)
		backtestStartDate = start.Format("2006-01-02")
	}

	sigStrategy := factory.Create(params)
	rollover := factory.CreateRolloverHandler(sigStrategy)
	stateRecorder := factory.CreateStateRecorder()

	btStrategy := sigStrategy.(backtest.SignalStrategy)
	btRollover := rollover.(backtest.RolloverHandler)
	btStateRecorder := stateRecorder.(backtest.StateRecorder)

	service := backtest.NewBacktestService()
	result, err := service.Run(backtest.BacktestInput{
		Klines:            allKlines,
		DominantMap:       dominantMap,
		Symbol:            *symbol,
		StartDate:         *startDate,
		EndDate:           *endDate,
		Strategy:          btStrategy,
		Rollover:          btRollover,
		StateRecorder:     btStateRecorder,
		WarmupDays:        warmupDays,
		BacktestStartDate: backtestStartDate,
	})
	if err != nil {
		log.Fatalf("回测执行失败: %v", err)
	}

	fmt.Printf("[5/5] 计算交易信号完成，共 %d 条信号\n", len(result.Signals))
	fmt.Printf("[6/6] 计算资金收益完成，共 %d 条持仓记录\n", len(result.PositionReturns))

	result.Statistics.Print()

	dailyDetails := backtest.GenerateDailyDetails(result.DailyRecords, result.Signals, result.Klines, result.DominantMap)
	backtest.PrintDailyDetails(dailyDetails)
	backtest.PrintPositionReturns(result.PositionReturns)

	reporter := backtest.NewReporter(result.Signals)
	reporter.SetStateHistory(result.StateHistory)
	reporter.PrintSignals()
	reporter.PrintStateHistory()
}
