package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"futures-backtest/internal/backtest"
	"futures-backtest/internal/db"
)

type ResultData struct {
	ID              string                       `json:"id"`
	Request         BacktestRequest              `json:"request"`
	Signals         []backtest.TradeSignal       `json:"signals"`
	DailyRecords    []backtest.DailyRecord       `json:"daily_records"`
	PositionReturns []backtest.PositionReturn    `json:"position_returns"`
	Statistics      backtest.Statistics          `json:"statistics"`
	StateHistory    []backtest.StateRecord       `json:"state_history"`
	DominantMap     map[string]string            `json:"dominant_map"`
	Klines          []backtest.KLineWithContract `json:"klines"`
}

func (s *Server) saveResult(result *ResultData) error {
	if err := s.saveResultToFile(result); err != nil {
		return fmt.Errorf("保存结果文件失败: %w", err)
	}

	if err := s.saveResultToDB(result); err != nil {
		fmt.Printf("保存结果到数据库失败(不影响文件存储): %v\n", err)
	}

	return nil
}

func (s *Server) saveResultToFile(result *ResultData) error {
	filename := filepath.Join(s.retDir, result.ID+".json")
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建结果文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("编码结果数据失败: %w", err)
	}

	return nil
}

func (s *Server) saveResultToDB(result *ResultData) error {
	signalsJSON, err := json.Marshal(result.Signals)
	if err != nil {
		return fmt.Errorf("序列化signals失败: %w", err)
	}
	dailyRecordsJSON, err := json.Marshal(result.DailyRecords)
	if err != nil {
		return fmt.Errorf("序列化daily_records失败: %w", err)
	}
	positionReturnsJSON, err := json.Marshal(result.PositionReturns)
	if err != nil {
		return fmt.Errorf("序列化position_returns失败: %w", err)
	}
	stateHistoryJSON, err := json.Marshal(result.StateHistory)
	if err != nil {
		return fmt.Errorf("序列化state_history失败: %w", err)
	}
	dominantMapJSON, err := json.Marshal(result.DominantMap)
	if err != nil {
		return fmt.Errorf("序列化dominant_map失败: %w", err)
	}
	klinesJSON, err := json.Marshal(result.Klines)
	if err != nil {
		return fmt.Errorf("序列化klines失败: %w", err)
	}

	dbResult := db.BacktestResult{
		ID:               result.ID,
		Symbol:           result.Request.Symbol,
		Strategy:         result.Request.Strategy,
		StartDate:        result.Request.StartDate,
		EndDate:          result.Request.EndDate,
		Leverage:         result.Request.Leverage,
		TotalReturn:      result.Statistics.TotalReturn,
		AnnualReturn:     result.Statistics.AnnualReturn,
		MaxDrawdown:      result.Statistics.MaxDrawdown,
		MaxDrawdownRatio: result.Statistics.MaxDrawdownRatio,
		WinRate:          result.Statistics.WinRate,
		ProfitLossRatio:  result.Statistics.ProfitLossRatio,
		WinningTrades:    result.Statistics.WinningTrades,
		LosingTrades:     result.Statistics.LosingTrades,
		TotalTrades:      result.Statistics.TotalTrades,
		TotalWin:         result.Statistics.TotalWin,
		TotalLoss:        result.Statistics.TotalLoss,
		SharpeRatio:      result.Statistics.SharpeRatio,
		CalmarRatio:      result.Statistics.CalmarRatio,
		TradingDays:      result.Statistics.TradingDays,
		FinalValue:       result.Statistics.FinalValue,
		Signals:          string(signalsJSON),
		DailyRecords:     string(dailyRecordsJSON),
		PositionReturns:  string(positionReturnsJSON),
		StateHistory:     string(stateHistoryJSON),
		DominantMap:      string(dominantMapJSON),
		Klines:           string(klinesJSON),
	}

	return db.SaveBacktestResult(dbResult)
}

func (s *Server) loadResult(id string) (*ResultData, error) {
	dbResult, err := db.GetBacktestResult(id)
	if err == nil && dbResult != nil {
		return s.loadResultFromDB(dbResult)
	}

	return s.loadResultFromFile(id)
}

func (s *Server) loadResultFromDB(dbResult *db.BacktestResult) (*ResultData, error) {
	result := &ResultData{
		ID: dbResult.ID,
		Request: BacktestRequest{
			Symbol:    dbResult.Symbol,
			Strategy:  dbResult.Strategy,
			StartDate: dbResult.StartDate,
			EndDate:   dbResult.EndDate,
			Leverage:  dbResult.Leverage,
		},
		Statistics: backtest.Statistics{
			TotalReturn:      dbResult.TotalReturn,
			AnnualReturn:     dbResult.AnnualReturn,
			MaxDrawdown:      dbResult.MaxDrawdown,
			MaxDrawdownRatio: dbResult.MaxDrawdownRatio,
			WinRate:          dbResult.WinRate,
			ProfitLossRatio:  dbResult.ProfitLossRatio,
			WinningTrades:    dbResult.WinningTrades,
			LosingTrades:     dbResult.LosingTrades,
			TotalTrades:      dbResult.TotalTrades,
			TotalWin:         dbResult.TotalWin,
			TotalLoss:        dbResult.TotalLoss,
			SharpeRatio:      dbResult.SharpeRatio,
			CalmarRatio:      dbResult.CalmarRatio,
			TradingDays:      dbResult.TradingDays,
			FinalValue:       dbResult.FinalValue,
		},
	}

	if dbResult.Signals != "" && dbResult.Signals != "null" {
		json.Unmarshal([]byte(dbResult.Signals), &result.Signals)
	}
	if dbResult.DailyRecords != "" && dbResult.DailyRecords != "null" {
		json.Unmarshal([]byte(dbResult.DailyRecords), &result.DailyRecords)
	}
	if dbResult.PositionReturns != "" && dbResult.PositionReturns != "null" {
		json.Unmarshal([]byte(dbResult.PositionReturns), &result.PositionReturns)
	}
	if dbResult.StateHistory != "" && dbResult.StateHistory != "null" {
		json.Unmarshal([]byte(dbResult.StateHistory), &result.StateHistory)
	}
	if dbResult.DominantMap != "" && dbResult.DominantMap != "null" {
		json.Unmarshal([]byte(dbResult.DominantMap), &result.DominantMap)
	}
	if dbResult.Klines != "" && dbResult.Klines != "null" {
		json.Unmarshal([]byte(dbResult.Klines), &result.Klines)
	}

	return result, nil
}

func (s *Server) loadResultFromFile(id string) (*ResultData, error) {
	filename := filepath.Join(s.retDir, id+".json")
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("打开结果文件失败: %w", err)
	}
	defer file.Close()

	var result ResultData
	if err := json.NewDecoder(file).Decode(&result); err != nil {
		return nil, fmt.Errorf("解码结果数据失败: %w", err)
	}

	return &result, nil
}
