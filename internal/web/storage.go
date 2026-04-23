package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"futures-backtest/internal/backtest"
)

// ResultData 回测结果数据
type ResultData struct {
	ID              string                        `json:"id"`
	Request         BacktestRequest               `json:"request"`
	Signals         []backtest.TradeSignal        `json:"signals"`
	DailyRecords    []backtest.DailyRecord        `json:"daily_records"`
	PositionReturns []backtest.PositionReturn     `json:"position_returns"`
	Statistics      backtest.Statistics           `json:"statistics"`
	StateHistory    []backtest.StateRecord        `json:"state_history"`
	DominantMap     map[string]string             `json:"dominant_map"`
	Klines          []backtest.KLineWithContract  `json:"klines"`
}

// saveResult 保存回测结果到ret目录
func (s *Server) saveResult(result *ResultData) error {
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

// loadResult 从ret目录加载回测结果
func (s *Server) loadResult(id string) (*ResultData, error) {
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
