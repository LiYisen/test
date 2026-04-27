package fund

import (
	"futures-backtest/internal/backtest"

	"github.com/shopspring/decimal"
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
	Weight   decimal.Decimal        `json:"weight"`
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
	Weight       decimal.Decimal        `json:"weight"`
	Signals      []backtest.TradeSignal `json:"signals"`
	DailyRecords []backtest.DailyRecord `json:"daily_records"`
	Statistics   backtest.Statistics    `json:"statistics"`
}

type FundDailyRecord struct {
	Date        string                     `json:"date"`
	TotalValue  decimal.Decimal            `json:"total_value"`
	DailyReturn decimal.Decimal            `json:"daily_return"`
	PnL         decimal.Decimal            `json:"pnl"`
	Components  map[string]decimal.Decimal `json:"components"`
}

type FundStatistics struct {
	TotalReturn      decimal.Decimal `json:"total_return"`
	AnnualReturn     decimal.Decimal `json:"annual_return"`
	MaxDrawdown      decimal.Decimal `json:"max_drawdown"`
	MaxDrawdownRatio decimal.Decimal `json:"max_drawdown_ratio"`
	SharpeRatio      decimal.Decimal `json:"sharpe_ratio"`
	CalmarRatio      decimal.Decimal `json:"calmar_ratio"`
	WinRate          decimal.Decimal `json:"win_rate"`
	TradingDays      int             `json:"trading_days"`
	WinningTrades    int             `json:"winning_trades"`
	LosingTrades     int             `json:"losing_trades"`
	TotalTrades      int             `json:"total_trades"`
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
