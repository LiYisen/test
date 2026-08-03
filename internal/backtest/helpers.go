package backtest

// formatDateString 将 YYYYMMDD 格式转换为 YYYY-MM-DD
func formatDateString(date string) string {
	if len(date) == 8 {
		return date[:4] + "-" + date[4:6] + "-" + date[6:8]
	}
	return date
}

func FilterDominantKlines(allKlines []KLineWithContract, dominantMap map[string]string) []KLineWithContract {
	var result []KLineWithContract
	for _, kl := range allKlines {
		if dominant, ok := dominantMap[kl.Date]; ok && dominant == kl.Symbol {
			result = append(result, kl)
		}
	}
	return result
}

func FilterKlinesByDate(klines []KLineWithContract, startDate string) []KLineWithContract {
	var filtered []KLineWithContract
	formattedStart := formatDateString(startDate)
	for _, kl := range klines {
		if kl.Date >= formattedStart {
			filtered = append(filtered, kl)
		}
	}
	return filtered
}

func FilterDailyRecordsByDate(records []DailyRecord, startDate string) []DailyRecord {
	var filtered []DailyRecord
	formattedStart := formatDateString(startDate)
	for _, r := range records {
		if r.Date >= formattedStart {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
