package backtest

import "github.com/shopspring/decimal"

type Direction int

const (
	Buy Direction = iota
	Sell
	Close
	CloseShort
	CloseLong
)

func (d Direction) String() string {
	switch d {
	case Buy:
		return "Buy"
	case Sell:
		return "Sell"
	case Close:
		return "Close"
	case CloseShort:
		return "CloseShort"
	case CloseLong:
		return "CloseLong"
	default:
		return "Unknown"
	}
}

type TradeSignal struct {
	SignalDate string          `json:"signal_date"`
	Price      decimal.Decimal `json:"price"`
	Direction  Direction       `json:"direction"`
	Leverage   decimal.Decimal `json:"leverage"`
	Quantity   decimal.Decimal `json:"quantity"`
	SignalType string          `json:"signal_type"`
	Symbol     string          `json:"symbol"`
	OpenPrice  decimal.Decimal `json:"open_price"`
	OpenDate   string          `json:"open_date"`
}

type SignalPosition struct {
	Symbol    string          `json:"symbol"`
	Direction Direction       `json:"direction"`
	OpenPrice decimal.Decimal `json:"open_price"`
	OpenDate  string          `json:"open_date"`
	Leverage  decimal.Decimal `json:"leverage"`
}

type Position struct {
	Symbol       string          `json:"symbol"`
	Direction    Direction       `json:"direction"`
	OpenPrice    decimal.Decimal `json:"open_price"`
	OpenDate     string          `json:"open_date"`
	Quantity     decimal.Decimal `json:"quantity"`
	Leverage     decimal.Decimal `json:"leverage"`
	CurrentPrice decimal.Decimal `json:"current_price"`
}

type Account struct {
	Cash         decimal.Decimal        `json:"cash"`
	TotalValue   decimal.Decimal        `json:"total_value"`
	Positions    []Position             `json:"positions"`
	DailyRecords map[string]DailyRecord `json:"daily_records"`
}

type DailyRecord struct {
	Date        string          `json:"date"`
	Position    decimal.Decimal `json:"position"`
	Cash        decimal.Decimal `json:"cash"`
	TotalValue  decimal.Decimal `json:"total_value"`
	PnL         decimal.Decimal `json:"pnl"`
	DailyReturn decimal.Decimal `json:"daily_return"`
}

type YinYangElement struct {
	High    decimal.Decimal `json:"high"`
	Low     decimal.Decimal `json:"low"`
	IsValid bool            `json:"is_valid"`
}

type YinYangState struct {
	IsYang bool           `json:"is_yang"`
	Yang1  YinYangElement `json:"yang1"`
	Yin1   YinYangElement `json:"yin1"`
	Yang2  YinYangElement `json:"yang2"`
	Yin2   YinYangElement `json:"yin2"`
}

type StateRecord struct {
	Date       string          `json:"date"`
	Symbol     string          `json:"symbol"`
	Position   string          `json:"position"`
	ClosePrice float64         `json:"close_price"`
}

type TradeRecord struct {
	Date       string          `json:"date"`
	Symbol     string          `json:"symbol"`
	Direction  Direction       `json:"direction"`
	Price      decimal.Decimal `json:"price"`
	Quantity   decimal.Decimal `json:"quantity"`
	Leverage   decimal.Decimal `json:"leverage"`
	PnL        decimal.Decimal `json:"pnl"`
	TotalValue decimal.Decimal `json:"total_value"`
}

type KLineData struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume float64 `json:"volume"`
	Amount float64 `json:"amount"`
	Hold   float64 `json:"hold"`
	Settle float64 `json:"settle"`
}

type KLineWithContract struct {
	Symbol string `json:"symbol"`
	KLineData
}

type DominantContract struct {
	Date   string `json:"date"`
	Symbol string `json:"symbol"`
}
