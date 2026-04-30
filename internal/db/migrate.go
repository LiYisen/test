package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type JSONSymbolsFile struct {
	Symbols []Symbol `json:"symbols"`
}

type JSONFundsFile struct {
	Funds []Fund `json:"funds"`
}

func MigrateFromJSON(configDir string) error {
	if globalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	symbolsPath := filepath.Join(configDir, "symbols.json")
	if _, err := os.Stat(symbolsPath); err == nil {
		if err := migrateSymbols(symbolsPath); err != nil {
			return fmt.Errorf("迁移品种数据失败: %w", err)
		}
	}

	fundsPath := filepath.Join(configDir, "funds.json")
	if _, err := os.Stat(fundsPath); err == nil {
		if err := migrateFunds(fundsPath); err != nil {
			return fmt.Errorf("迁移基金数据失败: %w", err)
		}
	}

	return nil
}

func migrateSymbols(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	var file JSONSymbolsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}

	if len(file.Symbols) == 0 {
		return nil
	}

	return WithTx(func(tx *sql.Tx) error {
		return UpsertSymbolsTx(tx, file.Symbols)
	})
}

func migrateFunds(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	var file JSONFundsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}

	for _, f := range file.Funds {
		if err := UpsertFund(f); err != nil {
			return fmt.Errorf("迁移基金 %s 失败: %w", f.ID, err)
		}
	}

	return nil
}

func MigrateResultsFromDir(retDir string) error {
	if globalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	entries, err := os.ReadDir(retDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取结果目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(retDir, entry.Name())
		if err := migrateBacktestResult(filePath); err != nil {
			fmt.Printf("迁移回测结果 %s 失败: %v\n", entry.Name(), err)
		}
	}

	fundingDir := filepath.Join(retDir, "funding")
	if _, err := os.Stat(fundingDir); err == nil {
		if err := migrateFundResults(fundingDir); err != nil {
			fmt.Printf("迁移基金结果失败: %v\n", err)
		}
	}

	return nil
}

func migrateBacktestResult(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r := BacktestResult{}
	r.ID = getStr(raw, "id")

	req, ok := raw["request"].(map[string]interface{})
	if ok {
		r.Symbol = getStr(req, "symbol")
		r.Strategy = getStr(req, "strategy")
		r.StartDate = getStr(req, "start_date")
		r.EndDate = getStr(req, "end_date")
		r.Leverage = getFloat(req, "leverage")
	}

	stats, ok := raw["statistics"].(map[string]interface{})
	if ok {
		r.TotalReturn = getFloat(stats, "total_return")
		r.AnnualReturn = getFloat(stats, "annual_return")
		r.MaxDrawdown = getFloat(stats, "max_drawdown")
		r.MaxDrawdownRatio = getFloat(stats, "max_drawdown_ratio")
		r.WinRate = getFloat(stats, "win_rate")
		r.ProfitLossRatio = getFloat(stats, "profit_loss_ratio")
		r.WinningTrades = getInt(stats, "winning_trades")
		r.LosingTrades = getInt(stats, "losing_trades")
		r.TotalTrades = getInt(stats, "total_trades")
		r.TotalWin = getFloat(stats, "total_win")
		r.TotalLoss = getFloat(stats, "total_loss")
		r.SharpeRatio = getFloat(stats, "sharpe_ratio")
		r.CalmarRatio = getFloat(stats, "calmar_ratio")
		r.TradingDays = getInt(stats, "trading_days")
		r.FinalValue = getFloat(stats, "final_value")
	}

	if signals, ok := raw["signals"]; ok {
		b, err := json.Marshal(signals)
		if err != nil {
			return fmt.Errorf("序列化signals失败: %w", err)
		}
		r.Signals = string(b)
	}
	if dailyRecords, ok := raw["daily_records"]; ok {
		b, err := json.Marshal(dailyRecords)
		if err != nil {
			return fmt.Errorf("序列化daily_records失败: %w", err)
		}
		r.DailyRecords = string(b)
	}
	if positionReturns, ok := raw["position_returns"]; ok {
		b, err := json.Marshal(positionReturns)
		if err != nil {
			return fmt.Errorf("序列化position_returns失败: %w", err)
		}
		r.PositionReturns = string(b)
	}
	if stateHistory, ok := raw["state_history"]; ok {
		b, err := json.Marshal(stateHistory)
		if err != nil {
			return fmt.Errorf("序列化state_history失败: %w", err)
		}
		r.StateHistory = string(b)
	}
	if dominantMap, ok := raw["dominant_map"]; ok {
		b, err := json.Marshal(dominantMap)
		if err != nil {
			return fmt.Errorf("序列化dominant_map失败: %w", err)
		}
		r.DominantMap = string(b)
	}
	if klines, ok := raw["klines"]; ok {
		b, err := json.Marshal(klines)
		if err != nil {
			return fmt.Errorf("序列化klines失败: %w", err)
		}
		r.Klines = string(b)
	}

	return SaveBacktestResult(r)
}

func migrateFundResults(fundingDir string) error {
	fundDirs, err := os.ReadDir(fundingDir)
	if err != nil {
		return err
	}

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
			resultID := resultDir.Name()
			fundFile := filepath.Join(fundingDir, fundID, resultID, "fund_result.json")
			if _, err := os.Stat(fundFile); err != nil {
				continue
			}

			data, err := os.ReadFile(fundFile)
			if err != nil {
				continue
			}

			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				continue
			}

			r := FundResult{}
			r.ID = resultID
			r.FundID = getStr(raw, "fund_id")
			r.FundName = getStr(raw, "fund_name")
			r.StartDate = getStr(raw, "start_date")
			r.EndDate = getStr(raw, "end_date")
			r.Timestamp = getInt64(raw, "timestamp")

			stats, ok := raw["statistics"].(map[string]interface{})
			if ok {
				r.TotalReturn = getFloat(stats, "total_return")
				r.AnnualReturn = getFloat(stats, "annual_return")
				r.MaxDrawdown = getFloat(stats, "max_drawdown")
				r.MaxDrawdownRatio = getFloat(stats, "max_drawdown_ratio")
				r.SharpeRatio = getFloat(stats, "sharpe_ratio")
				r.CalmarRatio = getFloat(stats, "calmar_ratio")
				r.WinRate = getFloat(stats, "win_rate")
				r.TradingDays = getInt(stats, "trading_days")
				r.WinningTrades = getInt(stats, "winning_trades")
				r.LosingTrades = getInt(stats, "losing_trades")
				r.TotalTrades = getInt(stats, "total_trades")
			}

			if dailyRecords, ok := raw["daily_records"]; ok {
				b, err := json.Marshal(dailyRecords)
				if err != nil {
					fmt.Printf("序列化基金 %s/%s daily_records失败: %v\n", fundID, resultID, err)
					continue
				}
				r.DailyRecords = string(b)
			}

			positionsDir := filepath.Join(fundingDir, fundID, resultID, "positions")
			if posEntries, err := os.ReadDir(positionsDir); err == nil {
				posResults := make(map[string]interface{})
				for _, entry := range posEntries {
					if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
						continue
					}
					posData, err := os.ReadFile(filepath.Join(positionsDir, entry.Name()))
					if err != nil {
						continue
					}
					var posRaw map[string]interface{}
					if err := json.Unmarshal(posData, &posRaw); err != nil {
						continue
					}
					symbol := getStr(posRaw, "symbol")
					if symbol != "" {
						posResults[symbol] = posRaw
					}
				}
				b, err := json.Marshal(posResults)
				if err != nil {
					fmt.Printf("序列化基金 %s/%s position_results失败: %v\n", fundID, resultID, err)
					continue
				}
				r.PositionResults = string(b)
			}

			if err := SaveFundResult(r); err != nil {
				fmt.Printf("迁移基金结果 %s/%s 失败: %v\n", fundID, resultID, err)
			}
		}
	}

	return nil
}

func getStr(m map[string]interface{}, key string) string {
	v, ok := m[key].(string)
	if !ok {
		return ""
	}
	return v
}

func getFloat(m map[string]interface{}, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}

func getInt(m map[string]interface{}, key string) int {
	return int(getFloat(m, key))
}

func getInt64(m map[string]interface{}, key string) int64 {
	return int64(getFloat(m, key))
}
