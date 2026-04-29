package backtest

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
	SignalDate string    `json:"signal_date"`
	Price      float64   `json:"price"`
	Direction  Direction `json:"direction"`
	Leverage   float64   `json:"leverage"`
	Quantity   float64   `json:"quantity"`
	SignalType string    `json:"signal_type"`
	Symbol     string    `json:"symbol"`
	OpenPrice  float64   `json:"open_price"`
	OpenDate   string    `json:"open_date"`
}

type SignalPosition struct {
	Symbol    string    `json:"symbol"`
	Direction Direction `json:"direction"`
	OpenPrice float64   `json:"open_price"`
	OpenDate  string    `json:"open_date"`
	Leverage  float64   `json:"leverage"`
}

type Position struct {
	Symbol       string    `json:"symbol"`
	Direction    Direction `json:"direction"`
	OpenPrice    float64   `json:"open_price"`
	OpenDate     string    `json:"open_date"`
	Quantity     float64   `json:"quantity"`
	Leverage     float64   `json:"leverage"`
	CurrentPrice float64   `json:"current_price"`
}

type Account struct {
	Cash         float64               `json:"cash"`
	TotalValue   float64               `json:"total_value"`
	Positions    []Position            `json:"positions"`
	DailyRecords map[string]DailyRecord `json:"daily_records"`
}

type DailyRecord struct {
	Date        string  `json:"date"`
	Position    float64 `json:"position"`
	Cash        float64 `json:"cash"`
	TotalValue  float64 `json:"total_value"`
	PnL         float64 `json:"pnl"`
	DailyReturn float64 `json:"daily_return"`
}

type StateRecord struct {
	Date       string  `json:"date"`
	Symbol     string  `json:"symbol"`
	Position   string  `json:"position"`
	ClosePrice float64 `json:"close_price"`
}

type TradeRecord struct {
	Date       string    `json:"date"`
	Symbol     string    `json:"symbol"`
	Direction  Direction `json:"direction"`
	Price      float64   `json:"price"`
	Quantity   float64   `json:"quantity"`
	Leverage   float64   `json:"leverage"`
	PnL        float64   `json:"pnl"`
	TotalValue float64   `json:"total_value"`
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
