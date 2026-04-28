package fund

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func SaveFundResult(result *FundResult, baseDir string) error {
	fundDir := filepath.Join(baseDir, "funding", result.FundID, result.ID)

	if err := os.MkdirAll(fundDir, 0755); err != nil {
		return fmt.Errorf("创建基金结果目录失败: %w", err)
	}

	fundFile := filepath.Join(fundDir, "fund_result.json")
	fundData := map[string]interface{}{
		"id":            result.ID,
		"fund_id":       result.FundID,
		"fund_name":     result.FundName,
		"timestamp":     result.Timestamp,
		"start_date":    result.StartDate,
		"end_date":      result.EndDate,
		"statistics":    result.Statistics,
		"daily_records": result.DailyRecords,
	}

	fundJSON, err := json.MarshalIndent(fundData, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化基金结果失败: %w", err)
	}

	if err := os.WriteFile(fundFile, fundJSON, 0644); err != nil {
		return fmt.Errorf("写入基金结果文件失败: %w", err)
	}

	positionsDir := filepath.Join(fundDir, "positions")
	if err := os.MkdirAll(positionsDir, 0755); err != nil {
		return fmt.Errorf("创建品种结果目录失败: %w", err)
	}

	for symbol, posResult := range result.PositionResults {
		posFile := filepath.Join(positionsDir, fmt.Sprintf("%s_%s.json", symbol, posResult.Strategy))
		posData := map[string]interface{}{
			"symbol":        posResult.Symbol,
			"strategy":      posResult.Strategy,
			"weight":        posResult.Weight,
			"signals":       posResult.Signals,
			"daily_records": posResult.DailyRecords,
			"statistics":    posResult.Statistics,
		}

		posJSON, err := json.MarshalIndent(posData, "", "  ")
		if err != nil {
			return fmt.Errorf("序列化品种结果失败: %w", err)
		}

		if err := os.WriteFile(posFile, posJSON, 0644); err != nil {
			return fmt.Errorf("写入品种结果文件失败: %w", err)
		}
	}

	return nil
}

func LoadFundResult(baseDir, fundID, resultID string) (*FundResult, error) {
	fundingDir := filepath.Join(baseDir, "funding", fundID)

	targetDir := filepath.Join(fundingDir, resultID)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("未找到基金结果: %s/%s", fundID, resultID)
	}

	fundFile := filepath.Join(targetDir, "fund_result.json")
	fundJSON, err := os.ReadFile(fundFile)
	if err != nil {
		return nil, fmt.Errorf("读取基金结果文件失败: %w", err)
	}

	var result FundResult
	if err := json.Unmarshal(fundJSON, &result); err != nil {
		return nil, fmt.Errorf("解析基金结果失败: %w", err)
	}

	positionsDir := filepath.Join(targetDir, "positions")
	posEntries, err := os.ReadDir(positionsDir)
	if err == nil {
		result.PositionResults = make(map[string]*PositionResult)
		for _, entry := range posEntries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
				posFile := filepath.Join(positionsDir, entry.Name())
				posJSON, err := os.ReadFile(posFile)
				if err != nil {
					continue
				}

				var posResult PositionResult
				if err := json.Unmarshal(posJSON, &posResult); err != nil {
					continue
				}

				result.PositionResults[posResult.Symbol] = &posResult
			}
		}
	}

	return &result, nil
}

func ListFundResults(baseDir string) ([]map[string]interface{}, error) {
	fundingDir := filepath.Join(baseDir, "funding")

	if _, err := os.Stat(fundingDir); os.IsNotExist(err) {
		return []map[string]interface{}{}, nil
	}

	fundDirs, err := os.ReadDir(fundingDir)
	if err != nil {
		return nil, fmt.Errorf("读取基金目录失败: %w", err)
	}

	var results []map[string]interface{}

	for _, fundDir := range fundDirs {
		if !fundDir.IsDir() {
			continue
		}

		fundID := fundDir.Name()
		resultDirs, err := os.ReadDir(filepath.Join(fundingDir, fundID))
		if err != nil {
			continue
		}

		for _, resultDir := range resultDirs {
			if !resultDir.IsDir() {
				continue
			}

			fundFile := filepath.Join(fundingDir, fundID, resultDir.Name(), "fund_result.json")
			fundJSON, err := os.ReadFile(fundFile)
			if err != nil {
				continue
			}

			var fundData map[string]interface{}
			if err := json.Unmarshal(fundJSON, &fundData); err != nil {
				continue
			}

			results = append(results, map[string]interface{}{
				"id":         fundData["id"],
				"fund_id":    fundID,
				"fund_name":  fundData["fund_name"],
				"start_date": fundData["start_date"],
				"end_date":   fundData["end_date"],
				"timestamp":  fundData["timestamp"],
			})
		}
	}

	return results, nil
}

func DeleteFundResult(baseDir, fundID, resultID string) error {
	targetDir := filepath.Join(baseDir, "funding", fundID, resultID)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("未找到基金结果: %s/%s", fundID, resultID)
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("删除基金结果失败: %w", err)
	}

	return nil
}
