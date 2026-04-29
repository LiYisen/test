package fund

import (
	"encoding/json"
	"fmt"

	"futures-backtest/internal/db"
)

func SaveFundResult(result *FundResult, baseDir string) error {
	dailyRecordsJSON, err := json.Marshal(result.DailyRecords)
	if err != nil {
		return fmt.Errorf("序列化每日记录失败: %w", err)
	}

	positionResultsJSON, err := json.Marshal(result.PositionResults)
	if err != nil {
		return fmt.Errorf("序列化品种结果失败: %w", err)
	}

	dbResult := db.FundResult{
		ID:              result.ID,
		FundID:          result.FundID,
		FundName:        result.FundName,
		StartDate:       result.StartDate,
		EndDate:         result.EndDate,
		Timestamp:       result.Timestamp,
		TotalReturn:     result.Statistics.TotalReturn,
		AnnualReturn:    result.Statistics.AnnualReturn,
		MaxDrawdown:     result.Statistics.MaxDrawdown,
		MaxDrawdownRatio: result.Statistics.MaxDrawdownRatio,
		SharpeRatio:     result.Statistics.SharpeRatio,
		CalmarRatio:     result.Statistics.CalmarRatio,
		WinRate:         result.Statistics.WinRate,
		TradingDays:     result.Statistics.TradingDays,
		WinningTrades:   result.Statistics.WinningTrades,
		LosingTrades:    result.Statistics.LosingTrades,
		TotalTrades:     result.Statistics.TotalTrades,
		DailyRecords:    string(dailyRecordsJSON),
		PositionResults: string(positionResultsJSON),
	}

	if err := db.SaveFundResult(dbResult); err != nil {
		return fmt.Errorf("保存基金结果到数据库失败: %w", err)
	}

	return nil
}

func LoadFundResult(baseDir, fundID, resultID string) (*FundResult, error) {
	dbResult, err := db.GetFundResult(fundID, resultID)
	if err != nil {
		return nil, fmt.Errorf("从数据库加载基金结果失败: %w", err)
	}
	if dbResult == nil {
		return nil, fmt.Errorf("未找到基金结果: %s/%s", fundID, resultID)
	}

	result := &FundResult{
		ID:        dbResult.ID,
		FundID:    dbResult.FundID,
		FundName:  dbResult.FundName,
		StartDate: dbResult.StartDate,
		EndDate:   dbResult.EndDate,
		Timestamp: dbResult.Timestamp,
		Statistics: FundStatistics{
			TotalReturn:      dbResult.TotalReturn,
			AnnualReturn:     dbResult.AnnualReturn,
			MaxDrawdown:      dbResult.MaxDrawdown,
			MaxDrawdownRatio: dbResult.MaxDrawdownRatio,
			SharpeRatio:      dbResult.SharpeRatio,
			CalmarRatio:      dbResult.CalmarRatio,
			WinRate:          dbResult.WinRate,
			TradingDays:      dbResult.TradingDays,
			WinningTrades:    dbResult.WinningTrades,
			LosingTrades:     dbResult.LosingTrades,
			TotalTrades:      dbResult.TotalTrades,
		},
	}

	if dbResult.DailyRecords != "" && dbResult.DailyRecords != "[]" {
		if err := json.Unmarshal([]byte(dbResult.DailyRecords), &result.DailyRecords); err != nil {
			return nil, fmt.Errorf("解析每日记录失败: %w", err)
		}
	}

	if dbResult.PositionResults != "" && dbResult.PositionResults != "{}" {
		result.PositionResults = make(map[string]*PositionResult)
		if err := json.Unmarshal([]byte(dbResult.PositionResults), &result.PositionResults); err != nil {
			return nil, fmt.Errorf("解析品种结果失败: %w", err)
		}
	}

	return result, nil
}

func ListFundResults(baseDir string) ([]map[string]interface{}, error) {
	return db.ListFundResults()
}

func DeleteFundResult(baseDir, fundID, resultID string) error {
	return db.DeleteFundResult(fundID, resultID)
}
