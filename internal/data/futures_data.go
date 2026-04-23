// Package data 封装期货数据获取业务逻辑
package data

import (
	"fmt"
	"sync"
	"time"

	"futures-backtest/pkg/pyexec"
)

// KLine K线数据结构
// KLine K线数据结构
type KLine struct {
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
// TradeDate 交易日历结构
type TradeDate struct {
	Date          string `json:"date"`
	IsTradingDay  bool   `json:"is_trading_day"`
}

// FuturesInfo 期货信息结构
type FuturesInfo struct {
	Symbol  string `json:"symbol"`
	Name    string `json:"name"`
	Exchange string `json:"exchange"`
}

// cacheKey 缓存键
type cacheKey struct {
	dataType pyexec.DataType
	symbol   string
	start    string
	end      string
}

// DataCache 内存数据缓存
type DataCache struct {
	mu       sync.RWMutex
	kline    map[string][]KLine        // key: "symbol:start:end"
	calendar map[string][]TradeDate    // key: "start:end"
	info     map[string]FuturesInfo     // key: "symbol"
}

// newDataCache 创建新的缓存
func newDataCache() *DataCache {
	return &DataCache{
		kline:    make(map[string][]KLine),
		calendar: make(map[string][]TradeDate),
		info:     make(map[string]FuturesInfo),
	}
}

// FuturesDataManager 期货数据管理器
type FuturesDataManager struct {
	executor *pyexec.PyExecutor
	cache    *DataCache
}

// NewFuturesDataManager 创建新的期货数据管理器
func NewFuturesDataManager(executor *pyexec.PyExecutor) *FuturesDataManager {
	return &FuturesDataManager{
		executor: executor,
		cache:    newDataCache(),
	}
}

// klineCacheKey 生成K线缓存键
func klineCacheKey(symbol, start, end string) string {
	return fmt.Sprintf("%s:%s:%s", symbol, start, end)
}

// calendarCacheKey 生成日历缓存键
func calendarCacheKey(start, end string) string {
	return fmt.Sprintf("%s:%s", start, end)
}

// GetFuturesKLine 获取期货K线数据
func (m *FuturesDataManager) GetFuturesKLine(symbol, startDate, endDate string) ([]KLine, error) {
	key := klineCacheKey(symbol, startDate, endDate)

	// 检查缓存
	m.cache.mu.RLock()
	if data, ok := m.cache.kline[key]; ok {
		m.cache.mu.RUnlock()
		return data, nil
	}
	m.cache.mu.RUnlock()

	// 调用pyexec获取原始数据
	rawData, err := m.executor.GetFuturesKLine(symbol, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}

	// 转换为强类型结构
	klines := make([]KLine, 0, len(rawData))
	for _, item := range rawData {
		kl, err := parseKLine(item)
		if err != nil {
			return nil, fmt.Errorf("解析K线数据失败: %w", err)
		}
		klines = append(klines, kl)
	}

	// 写入缓存
	m.cache.mu.Lock()
	m.cache.kline[key] = klines
	m.cache.mu.Unlock()

	return klines, nil
}

// GetTradeCalendar 获取交易日历
func (m *FuturesDataManager) GetTradeCalendar(startDate, endDate string) ([]TradeDate, error) {
	key := calendarCacheKey(startDate, endDate)

	// 检查缓存
	m.cache.mu.RLock()
	if data, ok := m.cache.calendar[key]; ok {
		m.cache.mu.RUnlock()
		return data, nil
	}
	m.cache.mu.RUnlock()

	// 调用pyexec获取原始数据
	rawData, err := m.executor.GetTradeCalendar(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取交易日历失败: %w", err)
	}

	// 转换为强类型结构
	calendar := make([]TradeDate, 0, len(rawData))
	for _, item := range rawData {
		td, err := parseTradeDate(item)
		if err != nil {
			return nil, fmt.Errorf("解析交易日历数据失败: %w", err)
		}
		calendar = append(calendar, td)
	}

	// 写入缓存
	m.cache.mu.Lock()
	m.cache.calendar[key] = calendar
	m.cache.mu.Unlock()

	return calendar, nil
}

// GetFuturesInfo 获取期货信息
func (m *FuturesDataManager) GetFuturesInfo(symbol string) (FuturesInfo, error) {
	// 检查缓存
	m.cache.mu.RLock()
	if info, ok := m.cache.info[symbol]; ok {
		m.cache.mu.RUnlock()
		return info, nil
	}
	m.cache.mu.RUnlock()

	// 当前pyexec暂不支持期货信息查询，返回基于缓存或错误
	m.cache.mu.RLock()
	defer m.cache.mu.RUnlock()

	if info, ok := m.cache.info[symbol]; ok {
		return info, nil
	}

	return FuturesInfo{}, fmt.Errorf("期货信息未找到: %s（当前版本不支持自动获取，请通过SetFuturesInfo手动设置）", symbol)
}

// SetFuturesInfo 手动设置期货信息（用于补充pyexec暂不支持的数据）
func (m *FuturesDataManager) SetFuturesInfo(info FuturesInfo) {
	m.cache.mu.Lock()
	defer m.cache.mu.Unlock()
	m.cache.info[info.Symbol] = info
}

// ClearCache 清除所有缓存
func (m *FuturesDataManager) ClearCache() {
	m.cache.mu.Lock()
	defer m.cache.mu.Unlock()
	m.cache.kline = make(map[string][]KLine)
	m.cache.calendar = make(map[string][]TradeDate)
	m.cache.info = make(map[string]FuturesInfo)
}

// ClearExpiredCache 清除指定时间之前的缓存
func (m *FuturesDataManager) ClearExpiredCache(before time.Time) {
	m.cache.mu.Lock()
	defer m.cache.mu.Unlock()
	// 当前实现为全量清除，后续可按缓存时间精确清理
	m.cache.kline = make(map[string][]KLine)
	m.cache.calendar = make(map[string][]TradeDate)
}

// parseKLine 将map解析为KLine结构
// parseKLine 将map解析为KLine结构
func parseKLine(m map[string]interface{}) (KLine, error) {
	kl := KLine{}

	if v, ok := m["date"].(string); ok {
		kl.Date = v
	}
	kl.Open = getFloat64(m, "open")
	kl.High = getFloat64(m, "high")
	kl.Low = getFloat64(m, "low")
	kl.Close = getFloat64(m, "close")
	kl.Volume = getFloat64(m, "volume")
	kl.Amount = getFloat64(m, "amount")
	kl.Hold = getFloat64(m, "hold")
	kl.Settle = getFloat64(m, "settle")

	return kl, nil
}
// parseTradeDate 将map解析为TradeDate结构
func parseTradeDate(m map[string]interface{}) (TradeDate, error) {
	td := TradeDate{}

	if v, ok := m["date"].(string); ok {
		td.Date = v
	}
	if v, ok := m["is_trading_day"].(bool); ok {
		td.IsTradingDay = v
	}

	return td, nil
}

// getFloat64 从map中安全获取float64值
func getFloat64(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

// NewDefaultDataManager 创建默认的期货数据管理器
func NewDefaultDataManager() *FuturesDataManager {
	return NewFuturesDataManager(pyexec.NewDefaultExecutor())
}
