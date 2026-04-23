package data

import (
	"fmt"
	"time"

	"futures-backtest/internal/backtest"
)

type DominantContractIdentifier struct {
	manager *FuturesDataManager
}

func NewDominantContractIdentifier(manager *FuturesDataManager) *DominantContractIdentifier {
	return &DominantContractIdentifier{
		manager: manager,
	}
}

func (d *DominantContractIdentifier) Identify(product string, allKlines []backtest.KLineWithContract, startDate, endDate string) (map[time.Time]string, error) {
	if len(allKlines) == 0 {
		return nil, fmt.Errorf("K线数据不能为空")
	}

	calendar, err := d.manager.GetTradeCalendar(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取交易日历失败: %w", err)
	}

	klinesByDate := make(map[string][]backtest.KLineWithContract)
	for _, kl := range allKlines {
		klinesByDate[kl.Date] = append(klinesByDate[kl.Date], kl)
	}

	result := make(map[time.Time]string)
	var currentDominant string
	var currentVolume float64
	var currentHold float64

	fmt.Println("\n========== 主力合约切换记录 ==========")

	for _, td := range calendar {
		if !td.IsTradingDay {
			continue
		}

		dayKlines, ok := klinesByDate[td.Date]
		if !ok || len(dayKlines) == 0 {
			continue
		}

		if currentDominant == "" {
			currentDominant = findInitialDominantContract(dayKlines)
			if currentDominant == "" {
				continue
			}
			for _, kl := range dayKlines {
				if kl.Symbol == currentDominant {
					currentVolume = kl.Volume
					currentHold = kl.Hold
					fmt.Printf("初始主力合约: %s | 日期: %s | 持仓量: %.0f | 成交量: %.0f\n",
						currentDominant, td.Date, currentHold, currentVolume)
					break
				}
			}
		} else {
			newDominant, newVolume, newHold, oldVol, oldHold := findSwitchDominantContract(dayKlines, currentDominant)
			if newDominant != "" && newDominant != currentDominant {
				fmt.Printf("主力切换: %s -> %s | 日期: %s\n", currentDominant, newDominant, td.Date)
				fmt.Printf("  旧主力 %s: 持仓量=%.0f, 成交量=%.0f\n", currentDominant, oldHold, oldVol)
				fmt.Printf("  新主力 %s: 持仓量=%.0f, 成交量=%.0f\n", newDominant, newHold, newVolume)
				fmt.Printf("  切换原因: 持仓量 %.0f > %.0f (%.1f%%), 成交量 %.0f > %.0f (%.1f%%)\n",
					newHold, oldHold, (newHold-oldHold)/oldHold*100,
					newVolume, oldVol, (newVolume-oldVol)/oldVol*100)
				currentDominant = newDominant
				currentVolume = newVolume
				currentHold = newHold
			} else {
				for _, kl := range dayKlines {
					if kl.Symbol == currentDominant {
						currentVolume = kl.Volume
						currentHold = kl.Hold
						break
					}
				}
			}
		}

		t, err := parseDate(td.Date)
		if err != nil {
			continue
		}
		result[t] = currentDominant
	}

	fmt.Println("========================================\n")

	return result, nil
}

func findInitialDominantContract(klines []backtest.KLineWithContract) string {
	if len(klines) == 0 {
		return ""
	}
	if len(klines) == 1 {
		return klines[0].Symbol
	}

	var maxHoldSymbol string
	var maxHold float64
	for _, kl := range klines {
		if kl.Hold > maxHold {
			maxHold = kl.Hold
			maxHoldSymbol = kl.Symbol
		}
	}

	return maxHoldSymbol
}

func findSwitchDominantContract(klines []backtest.KLineWithContract, currentSymbol string) (string, float64, float64, float64, float64) {
	var currentDayVol, currentDayHold float64
	for _, kl := range klines {
		if kl.Symbol == currentSymbol {
			currentDayVol = kl.Volume
			currentDayHold = kl.Hold
			break
		}
	}

	for _, kl := range klines {
		if kl.Symbol == currentSymbol {
			continue
		}
		if !isLaterContract(kl.Symbol, currentSymbol) {
			continue
		}
		if kl.Volume > currentDayVol && kl.Hold > currentDayHold {
			return kl.Symbol, kl.Volume, kl.Hold, currentDayVol, currentDayHold
		}
	}
	return "", 0, 0, currentDayVol, currentDayHold
}

func isLaterContract(newSymbol, currentSymbol string) bool {
	newYearMonth, err1 := extractYearMonth(newSymbol)
	currentYearMonth, err2 := extractYearMonth(currentSymbol)
	if err1 != nil || err2 != nil {
		return false
	}
	return newYearMonth > currentYearMonth
}

func extractYearMonth(symbol string) (int, error) {
	if len(symbol) < 4 {
		return 0, fmt.Errorf("invalid symbol format: %s", symbol)
	}
	yearMonthStr := symbol[len(symbol)-4:]
	var year, month int
	_, err := fmt.Sscanf(yearMonthStr, "%2d%2d", &year, &month)
	if err != nil {
		return 0, fmt.Errorf("failed to parse year month from symbol: %s", symbol)
	}
	fullYear := 2000 + year
	return fullYear*100 + month, nil
}

func parseDate(dateStr string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err == nil {
		return t, nil
	}
	return time.Parse("20060102", dateStr)
}
