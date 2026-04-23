package backtest

import (
	"fmt"
	"strings"
)

type Reporter struct {
	signals      []TradeSignal
	stateHistory []StateRecord
}

func NewReporter(signals []TradeSignal) *Reporter {
	return &Reporter{
		signals: signals,
	}
}

func (r *Reporter) SetStateHistory(history []StateRecord) {
	r.stateHistory = history
}

func (r *Reporter) PrintSignals() {
	if len(r.signals) == 0 {
		fmt.Println("\n========== 交易信号 ==========")
		fmt.Println("无交易信号")
		return
	}

	fmt.Println("\n========== 交易信号 ==========")
	fmt.Printf("%-12s %-12s %-10s %-12s %-8s %-12s\n",
		"日期", "合约", "方向", "价格", "杠杆", "类型")
	fmt.Println(strings.Repeat("-", 68))

	for _, sig := range r.signals {
		fmt.Printf("%-12s %-12s %-10s %-12s %-8s %-12s\n",
			sig.SignalDate, sig.Symbol, sig.Direction.String(),
			sig.Price.StringFixed(2), sig.Leverage.StringFixed(2), sig.SignalType)
	}

	fmt.Printf("\n共 %d 条交易信号\n", len(r.signals))
}

func (r *Reporter) PrintStateHistory() {
	if len(r.stateHistory) == 0 {
		fmt.Println("\n========== 策略状态历史 ==========")
		fmt.Println("无状态记录")
		return
	}

	fmt.Println("\n========== 策略状态历史 ==========")
	fmt.Printf("%-12s %-10s %-20s %s\n",
		"日期", "合约", "持仓方向", "持仓价格")
	fmt.Println(strings.Repeat("-", 60))

	for _, rec := range r.stateHistory {
		fmt.Printf("%-12s %-10s %-20s %.2f\n",
			rec.Date, rec.Symbol, rec.Position, rec.ClosePrice)
	}
}
