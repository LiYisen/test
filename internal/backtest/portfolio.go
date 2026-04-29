package backtest

import (
	"sort"
)

type PortfolioEngine struct{}

func NewPortfolioEngine() *PortfolioEngine {
	return &PortfolioEngine{}
}

type PositionReturn struct {
	OpenDate   string
	CloseDate  string
	Symbol     string
	Direction  Direction
	OpenPrice  float64
	ClosePrice float64
	Leverage   float64
	Return     float64
}

func (e *PortfolioEngine) Calculate(signals []TradeSignal, klines []KLineWithContract) ([]DailyRecord, []PositionReturn, error) {
	sort.Slice(klines, func(i, j int) bool {
		return klines[i].Date < klines[j].Date
	})

	klineMap := make(map[string]KLineWithContract, len(klines))
	tradingDatesSet := make(map[string]bool)
	tradingDates := make([]string, 0)

	for _, kl := range klines {
		key := kl.Date + "|" + kl.Symbol
		klineMap[key] = kl
		if !tradingDatesSet[kl.Date] {
			tradingDatesSet[kl.Date] = true
			tradingDates = append(tradingDates, kl.Date)
		}
	}
	sort.Strings(tradingDates)

	signalsByDate := make(map[string][]TradeSignal)
	for _, sig := range signals {
		signalsByDate[sig.SignalDate] = append(signalsByDate[sig.SignalDate], sig)
	}

	closePriceMap := make(map[string]float64, len(klines))
	for _, kl := range klines {
		key := kl.Date + "|" + kl.Symbol
		closePriceMap[key] = kl.Close
	}

	var records []DailyRecord
	var positionReturns []PositionReturn

	totalValue := 1.0
	var currentPos *struct {
		Symbol    string
		Direction Direction
		OpenPrice float64
		Leverage  float64
		OpenDate  string
	}

	for _, date := range tradingDates {
		if daySignals, ok := signalsByDate[date]; ok {
			for _, sig := range daySignals {
				if sig.Direction == Buy || sig.Direction == Sell {
					if currentPos != nil {
						pr := e.calcReturn(currentPos, sig.SignalDate, sig.Price)
						if pr != nil {
							positionReturns = append(positionReturns, *pr)
							totalValue = totalValue * (1 + pr.Return*pr.Leverage)
						}
					}
					currentPos = &struct {
						Symbol    string
						Direction Direction
						OpenPrice float64
						Leverage  float64
						OpenDate  string
					}{
						Symbol:    sig.Symbol,
						Direction: sig.Direction,
						OpenPrice: sig.Price,
						Leverage:  sig.Leverage,
						OpenDate:  sig.SignalDate,
					}
				} else if sig.Direction == CloseLong || sig.Direction == CloseShort || sig.Direction == Close {
					if currentPos != nil {
						pr := e.calcReturn(currentPos, sig.SignalDate, sig.Price)
						if pr != nil {
							positionReturns = append(positionReturns, *pr)
							totalValue = totalValue * (1 + pr.Return*pr.Leverage)
						}
						currentPos = nil
					}
				}
			}
		}

		effectiveValue := totalValue
		if currentPos != nil {
			key := date + "|" + currentPos.Symbol
			if closePrice, ok := closePriceMap[key]; ok && closePrice != 0 {
				var unrealizedRet float64
				if currentPos.Direction == Buy {
					unrealizedRet = (closePrice - currentPos.OpenPrice) / currentPos.OpenPrice
				} else {
					unrealizedRet = (currentPos.OpenPrice - closePrice) / currentPos.OpenPrice
				}
				effectiveValue = totalValue * (1 + unrealizedRet*currentPos.Leverage)
			}
		}

		var dailyReturn float64
		if len(records) > 0 {
			dailyReturn = effectiveValue/records[len(records)-1].TotalValue - 1
		}
		records = append(records, DailyRecord{
			Date:        date,
			Position:    0,
			Cash:        effectiveValue,
			TotalValue:  effectiveValue,
			PnL:         effectiveValue - 1,
			DailyReturn: dailyReturn,
		})
	}

	return records, positionReturns, nil
}

func (e *PortfolioEngine) calcReturn(pos *struct {
	Symbol    string
	Direction Direction
	OpenPrice float64
	Leverage  float64
	OpenDate  string
}, closeDate string, closePrice float64) *PositionReturn {
	if pos == nil {
		return nil
	}

	var ret float64
	if pos.Direction == Buy {
		ret = (closePrice - pos.OpenPrice) / pos.OpenPrice
	} else {
		ret = (pos.OpenPrice - closePrice) / pos.OpenPrice
	}

	return &PositionReturn{
		OpenDate:   pos.OpenDate,
		CloseDate:  closeDate,
		Symbol:     pos.Symbol,
		Direction:  pos.Direction,
		OpenPrice:  pos.OpenPrice,
		ClosePrice: closePrice,
		Leverage:   pos.Leverage,
		Return:     ret,
	}
}
