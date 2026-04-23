package backtest

import (
	"sort"

	"github.com/shopspring/decimal"
)

// PortfolioEngine 收益计算引擎，只依赖通用信号，不依赖策略类型
type PortfolioEngine struct{}

func NewPortfolioEngine() *PortfolioEngine {
	return &PortfolioEngine{}
}

// PositionReturn 持仓收益记录
type PositionReturn struct {
	OpenDate   string
	CloseDate  string
	Symbol     string
	Direction  Direction
	OpenPrice  decimal.Decimal
	ClosePrice decimal.Decimal
	Leverage   decimal.Decimal
	Return     decimal.Decimal
}

// Calculate 根据交易信号和K线数据计算收益
// 核心逻辑：信号驱动，与策略类型完全解耦
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

	closePriceMap := make(map[string]decimal.Decimal, len(klines))
	for _, kl := range klines {
		key := kl.Date + "|" + kl.Symbol
		closePriceMap[key] = decimal.NewFromFloat(kl.Close)
	}

	var records []DailyRecord
	var positionReturns []PositionReturn

	decOne := decimal.NewFromInt(1)
	totalValue := decOne
	var currentPos *struct {
		Symbol    string
		Direction Direction
		OpenPrice decimal.Decimal
		Leverage  decimal.Decimal
		OpenDate  string
	}

	for _, date := range tradingDates {
		if daySignals, ok := signalsByDate[date]; ok {
			for _, sig := range daySignals {
				if sig.Direction == Buy || sig.Direction == Sell {
					// 开新仓前，先平掉旧仓
					if currentPos != nil {
						pr := e.calcReturn(currentPos, sig.SignalDate, sig.Price)
						if pr != nil {
							positionReturns = append(positionReturns, *pr)
							totalValue = totalValue.Mul(decOne.Add(pr.Return.Mul(pr.Leverage)))
						}
					}
					currentPos = &struct {
						Symbol    string
						Direction Direction
						OpenPrice decimal.Decimal
						Leverage  decimal.Decimal
						OpenDate  string
					}{
						Symbol:    sig.Symbol,
						Direction: sig.Direction,
						OpenPrice: sig.Price,
						Leverage:  sig.Leverage,
						OpenDate:  sig.SignalDate,
					}
				} else if sig.Direction == CloseLong || sig.Direction == CloseShort || sig.Direction == Close {
					// 平仓信号
					if currentPos != nil {
						pr := e.calcReturn(currentPos, sig.SignalDate, sig.Price)
						if pr != nil {
							positionReturns = append(positionReturns, *pr)
							totalValue = totalValue.Mul(decOne.Add(pr.Return.Mul(pr.Leverage)))
						}
						currentPos = nil
					}
				}
			}
		}

		// 计算当日有效净值（包含未实现盈亏）
		effectiveValue := totalValue
		if currentPos != nil {
			key := date + "|" + currentPos.Symbol
			if closePrice, ok := closePriceMap[key]; ok && !closePrice.IsZero() {
				var unrealizedRet decimal.Decimal
				if currentPos.Direction == Buy {
					unrealizedRet = closePrice.Sub(currentPos.OpenPrice).Div(currentPos.OpenPrice)
				} else {
					unrealizedRet = currentPos.OpenPrice.Sub(closePrice).Div(currentPos.OpenPrice)
				}
				effectiveValue = totalValue.Mul(decOne.Add(unrealizedRet.Mul(currentPos.Leverage)))
			}
		}

		// 计算日收益率
		var dailyReturn decimal.Decimal
		if len(records) > 0 {
			dailyReturn = effectiveValue.Div(records[len(records)-1].TotalValue).Sub(decOne)
		}
		records = append(records, DailyRecord{
			Date:        date,
			Position:    decimal.Zero,
			Cash:        effectiveValue,
			TotalValue:  effectiveValue,
			PnL:         effectiveValue.Sub(decOne),
			DailyReturn: dailyReturn,
		})
	}

	return records, positionReturns, nil
}

// calcReturn 计算单次交易收益
func (e *PortfolioEngine) calcReturn(pos *struct {
	Symbol    string
	Direction Direction
	OpenPrice decimal.Decimal
	Leverage  decimal.Decimal
	OpenDate  string
}, closeDate string, closePrice decimal.Decimal) *PositionReturn {
	if pos == nil {
		return nil
	}

	var ret decimal.Decimal
	if pos.Direction == Buy {
		ret = closePrice.Sub(pos.OpenPrice).Div(pos.OpenPrice)
	} else {
		ret = pos.OpenPrice.Sub(closePrice).Div(pos.OpenPrice)
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
