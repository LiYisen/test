package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type BacktestResult struct {
	ID               string  `json:"id"`
	Symbol           string  `json:"symbol"`
	Strategy         string  `json:"strategy"`
	StartDate        string  `json:"start_date"`
	EndDate          string  `json:"end_date"`
	Leverage         float64 `json:"leverage"`
	TotalReturn      float64 `json:"total_return"`
	AnnualReturn     float64 `json:"annual_return"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	MaxDrawdownRatio float64 `json:"max_drawdown_ratio"`
	WinRate          float64 `json:"win_rate"`
	ProfitLossRatio  float64 `json:"profit_loss_ratio"`
	WinningTrades    int     `json:"winning_trades"`
	LosingTrades     int     `json:"losing_trades"`
	TotalTrades      int     `json:"total_trades"`
	TotalWin         float64 `json:"total_win"`
	TotalLoss        float64 `json:"total_loss"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	CalmarRatio      float64 `json:"calmar_ratio"`
	TradingDays      int     `json:"trading_days"`
	FinalValue       float64 `json:"final_value"`
	Signals          string  `json:"signals"`
	DailyRecords     string  `json:"daily_records"`
	PositionReturns  string  `json:"position_returns"`
	StateHistory     string  `json:"state_history"`
	DominantMap      string  `json:"dominant_map"`
	Klines           string  `json:"klines"`
}

func SaveBacktestResult(r BacktestResult) error {
	if globalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := globalDB.Exec(`
		INSERT OR REPLACE INTO backtest_results (
			id, symbol, strategy, start_date, end_date, leverage,
			total_return, annual_return, max_drawdown, max_drawdown_ratio,
			win_rate, profit_loss_ratio, winning_trades, losing_trades,
			total_trades, total_win, total_loss, sharpe_ratio, calmar_ratio,
			trading_days, final_value, signals, daily_records,
			position_returns, state_history, dominant_map, klines
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.Symbol, r.Strategy, r.StartDate, r.EndDate, r.Leverage,
		r.TotalReturn, r.AnnualReturn, r.MaxDrawdown, r.MaxDrawdownRatio,
		r.WinRate, r.ProfitLossRatio, r.WinningTrades, r.LosingTrades,
		r.TotalTrades, r.TotalWin, r.TotalLoss, r.SharpeRatio, r.CalmarRatio,
		r.TradingDays, r.FinalValue, r.Signals, r.DailyRecords,
		r.PositionReturns, r.StateHistory, r.DominantMap, r.Klines)
	return err
}

func GetBacktestResult(id string) (*BacktestResult, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	var r BacktestResult
	err := globalDB.QueryRow(`
		SELECT id, symbol, strategy, start_date, end_date, leverage,
			total_return, annual_return, max_drawdown, max_drawdown_ratio,
			win_rate, profit_loss_ratio, winning_trades, losing_trades,
			total_trades, total_win, total_loss, sharpe_ratio, calmar_ratio,
			trading_days, final_value, signals, daily_records,
			position_returns, state_history, dominant_map, klines
		FROM backtest_results WHERE id = ?
	`, id).Scan(&r.ID, &r.Symbol, &r.Strategy, &r.StartDate, &r.EndDate, &r.Leverage,
		&r.TotalReturn, &r.AnnualReturn, &r.MaxDrawdown, &r.MaxDrawdownRatio,
		&r.WinRate, &r.ProfitLossRatio, &r.WinningTrades, &r.LosingTrades,
		&r.TotalTrades, &r.TotalWin, &r.TotalLoss, &r.SharpeRatio, &r.CalmarRatio,
		&r.TradingDays, &r.FinalValue, &r.Signals, &r.DailyRecords,
		&r.PositionReturns, &r.StateHistory, &r.DominantMap, &r.Klines)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func ListBacktestResults() ([]map[string]interface{}, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	rows, err := globalDB.Query(`
		SELECT id, symbol, strategy, start_date, end_date, leverage,
			total_return, annual_return, max_drawdown, max_drawdown_ratio,
			win_rate, trading_days, final_value
		FROM backtest_results ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, symbol, strategy, startDate, endDate string
		var leverage, totalReturn, annualReturn, maxDD, maxDDRatio, winRate, finalValue float64
		var tradingDays int
		if err := rows.Scan(&id, &symbol, &strategy, &startDate, &endDate, &leverage,
			&totalReturn, &annualReturn, &maxDD, &maxDDRatio, &winRate, &tradingDays, &finalValue); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"id":            id,
			"symbol":        symbol,
			"strategy":      strategy,
			"start_date":    startDate,
			"end_date":      endDate,
			"leverage":      leverage,
			"total_return":  totalReturn,
			"annual_return": annualReturn,
			"max_drawdown":  maxDD,
			"win_rate":      winRate,
			"trading_days":  tradingDays,
			"final_value":   finalValue,
		})
	}
	return results, rows.Err()
}

func DeleteBacktestResult(id string) error {
	if globalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := globalDB.Exec(`DELETE FROM backtest_results WHERE id = ?`, id)
	return err
}

type FundResult struct {
	ID               string  `json:"id"`
	FundID           string  `json:"fund_id"`
	FundName         string  `json:"fund_name"`
	StartDate        string  `json:"start_date"`
	EndDate          string  `json:"end_date"`
	Timestamp        int64   `json:"timestamp"`
	TotalReturn      float64 `json:"total_return"`
	AnnualReturn     float64 `json:"annual_return"`
	MaxDrawdown      float64 `json:"max_drawdown"`
	MaxDrawdownRatio float64 `json:"max_drawdown_ratio"`
	SharpeRatio      float64 `json:"sharpe_ratio"`
	CalmarRatio      float64 `json:"calmar_ratio"`
	WinRate          float64 `json:"win_rate"`
	TradingDays      int     `json:"trading_days"`
	WinningTrades    int     `json:"winning_trades"`
	LosingTrades     int     `json:"losing_trades"`
	TotalTrades      int     `json:"total_trades"`
	DailyRecords     string  `json:"daily_records"`
	PositionResults  string  `json:"position_results"`
}

func SaveFundResult(r FundResult) error {
	if globalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := globalDB.Exec(`
		INSERT OR REPLACE INTO fund_results (
			id, fund_id, fund_name, start_date, end_date, timestamp,
			total_return, annual_return, max_drawdown, max_drawdown_ratio,
			sharpe_ratio, calmar_ratio, win_rate, trading_days,
			winning_trades, losing_trades, total_trades,
			daily_records, position_results
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.FundID, r.FundName, r.StartDate, r.EndDate, r.Timestamp,
		r.TotalReturn, r.AnnualReturn, r.MaxDrawdown, r.MaxDrawdownRatio,
		r.SharpeRatio, r.CalmarRatio, r.WinRate, r.TradingDays,
		r.WinningTrades, r.LosingTrades, r.TotalTrades,
		r.DailyRecords, r.PositionResults)
	return err
}

func GetFundResult(fundID, resultID string) (*FundResult, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	var r FundResult
	err := globalDB.QueryRow(`
		SELECT id, fund_id, fund_name, start_date, end_date, timestamp,
			total_return, annual_return, max_drawdown, max_drawdown_ratio,
			sharpe_ratio, calmar_ratio, win_rate, trading_days,
			winning_trades, losing_trades, total_trades,
			daily_records, position_results
		FROM fund_results WHERE fund_id = ? AND id = ?
	`, fundID, resultID).Scan(&r.ID, &r.FundID, &r.FundName, &r.StartDate, &r.EndDate, &r.Timestamp,
		&r.TotalReturn, &r.AnnualReturn, &r.MaxDrawdown, &r.MaxDrawdownRatio,
		&r.SharpeRatio, &r.CalmarRatio, &r.WinRate, &r.TradingDays,
		&r.WinningTrades, &r.LosingTrades, &r.TotalTrades,
		&r.DailyRecords, &r.PositionResults)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func ListFundResults() ([]map[string]interface{}, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	rows, err := globalDB.Query(`
		SELECT id, fund_id, fund_name, start_date, end_date, timestamp,
			total_return, annual_return, max_drawdown, max_drawdown_ratio,
			sharpe_ratio, win_rate, trading_days
		FROM fund_results ORDER BY timestamp DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, fundID, fundName, startDate, endDate string
		var timestamp int64
		var totalReturn, annualReturn, maxDD, maxDDRatio, sharpe, winRate float64
		var tradingDays int
		if err := rows.Scan(&id, &fundID, &fundName, &startDate, &endDate, &timestamp,
			&totalReturn, &annualReturn, &maxDD, &maxDDRatio, &sharpe, &winRate, &tradingDays); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"id":            id,
			"fund_id":       fundID,
			"fund_name":     fundName,
			"start_date":    startDate,
			"end_date":      endDate,
			"timestamp":     timestamp,
			"total_return":  totalReturn,
			"annual_return": annualReturn,
			"max_drawdown":  maxDD,
			"sharpe_ratio":  sharpe,
			"win_rate":      winRate,
			"trading_days":  tradingDays,
		})
	}
	return results, rows.Err()
}

func ListFundResultsByFundID(fundID string) ([]map[string]interface{}, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	rows, err := globalDB.Query(`
		SELECT id, fund_id, fund_name, start_date, end_date, timestamp,
			total_return, annual_return, max_drawdown, max_drawdown_ratio,
			sharpe_ratio, win_rate, trading_days
		FROM fund_results WHERE fund_id = ? ORDER BY timestamp DESC
	`, fundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id, fID, fundName, startDate, endDate string
		var timestamp int64
		var totalReturn, annualReturn, maxDD, maxDDRatio, sharpe, winRate float64
		var tradingDays int
		if err := rows.Scan(&id, &fID, &fundName, &startDate, &endDate, &timestamp,
			&totalReturn, &annualReturn, &maxDD, &maxDDRatio, &sharpe, &winRate, &tradingDays); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"id":            id,
			"fund_id":       fID,
			"fund_name":     fundName,
			"start_date":    startDate,
			"end_date":      endDate,
			"timestamp":     timestamp,
			"total_return":  totalReturn,
			"annual_return": annualReturn,
			"max_drawdown":  maxDD,
			"sharpe_ratio":  sharpe,
			"win_rate":      winRate,
			"trading_days":  tradingDays,
		})
	}
	return results, rows.Err()
}

func DeleteFundResult(fundID, resultID string) error {
	if globalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := globalDB.Exec(`DELETE FROM fund_results WHERE fund_id = ? AND id = ?`, fundID, resultID)
	return err
}

func DeleteFundResultsByFundID(fundID string) error {
	if globalDB == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := globalDB.Exec(`DELETE FROM fund_results WHERE fund_id = ?`, fundID)
	return err
}

var allowedExportTables = map[string]bool{
	"symbols":          true,
	"strategies":       true,
	"strategy_params":  true,
	"funds":            true,
	"fund_positions":   true,
	"backtest_results": true,
	"fund_results":     true,
	"config_meta":      true,
}

func validateTableName(tableName string) error {
	if !allowedExportTables[tableName] {
		return fmt.Errorf("不允许导出表: %s", tableName)
	}
	return nil
}

func ExportTableToJSON(tableName string) ([]byte, error) {
	if globalDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	if err := validateTableName(tableName); err != nil {
		return nil, err
	}
	rows, err := globalDB.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	return json.MarshalIndent(results, "", "  ")
}

func ExportTableToCSV(tableName string) (string, error) {
	if globalDB == nil {
		return "", fmt.Errorf("数据库未初始化")
	}
	if err := validateTableName(tableName); err != nil {
		return "", err
	}
	rows, err := globalDB.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return "", err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}

	csv := ""
	for i, col := range columns {
		if i > 0 {
			csv += ","
		}
		csv += col
	}
	csv += "\n"

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return "", err
		}
		for i, val := range values {
			if i > 0 {
				csv += ","
			}
			b, ok := val.([]byte)
			if ok {
				csv += string(b)
			} else if val == nil {
				csv += ""
			} else {
				csv += fmt.Sprintf("%v", val)
			}
		}
		csv += "\n"
	}

	return csv, nil
}
