package common

import (
	"fmt"
	"sort"
	"time"
)

// GenerateContractSymbols 根据品种代码和日期范围生成所有可能的合约符号
func GenerateContractSymbols(product, startDate, endDate string) []string {
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

// CalculateWarmupStartDateFromList 根据交易日列表计算预热起始日期
// 注意：不会修改输入的 tradingDays 切片
func CalculateWarmupStartDateFromList(tradingDays []string, startDate string, requiredDays int) string {
	if len(tradingDays) == 0 {
		startDateParsed, err := time.Parse("20060102", startDate)
		if err != nil {
			return startDate
		}
		warmupStart := startDateParsed.AddDate(0, 0, -requiredDays*2)
		return warmupStart.Format("20060102")
	}

	sorted := make([]string, len(tradingDays))
	copy(sorted, tradingDays)
	sort.Strings(sorted)

	if len(sorted) < requiredDays {
		startDateParsed, err := time.Parse("20060102", startDate)
		if err != nil {
			return startDate
		}
		warmupStart := startDateParsed.AddDate(0, 0, -requiredDays*2)
		return warmupStart.Format("20060102")
	}

	return sorted[len(sorted)-requiredDays]
}
