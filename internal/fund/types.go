package fund

import (
	"futures-backtest/internal/backtest"
)

type FundConfig struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	StartDate   string           `json:"start_date"`
	EndDate     string           `json:"end_date"`
	Positions   []PositionConfig `json:"positions"`
}

type PositionConfig struct {
	Symbol   string                 `json:"symbol"`
	Strategy string                 `json:"strategy"`
	Weight   float64                `json:"weight"`
	Params   map[string]interface{} `json:"params"`
}

type FundResult struct {
	ID              string                     `json:"id"`
	FundID          string                     `json:"fund_id"`
	FundName        string                     `json:"fund_name"`
	Timestamp       int64                      `json:"timestamp"`
	StartDate       string                     `json:"start_date"`
	EndDate         string                     `json:"end_date"`
	Statistics      FundStatistics             `json:"statistics"`
	DailyRecords    []FundDailyRecord          `json:"daily_records"`
	PositionResults map[string]*PositionResult `json:"position_results"`
}

type PositionResult struct {
	Symbol       string                 `json:"symbol"`
	Strategy     string                 `json:"strategy"`
	Weight       float64                `json:"weight"`
	Signals      []backtest.TradeSignal `json:"signals"`
	DailyRecords []backtest.DailyRecord `json:"daily_records"`
	Statistics   backtest.Statistics    `json:"statistics"`
}

type FundDailyRecord struct {
	Date        string             `json:"date"`
	TotalValue  float64            `json:"total_value"`
	DailyReturn float64            `json:"daily_return"`
	PnL         float64            `json:"pnl"`
	Components  map[string]float64 `json:"components"`
}

type FundStatistics struct {
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
}

type FundBacktestRequest struct {
	FundID    string `json:"fund_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type FundBacktestResponse struct {
	ID          string `json:"id"`
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	TradingDays int    `json:"trading_days"`
}
