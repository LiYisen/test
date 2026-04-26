package strategy

import (
	"fmt"
	"sync"

	"futures-backtest/internal/backtest"
	"futures-backtest/internal/strategy/ma"
	"futures-backtest/internal/strategy/yinyang"
)

type StrategyConfig struct {
	Name        string                `json:"name"`
	DisplayName string                `json:"display_name"`
	Description string                `json:"description"`
	Params      []StrategyParamConfig `json:"params"`
	Enabled     bool                  `json:"enabled"`
}

type StrategyParamConfig struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Type        string  `json:"type"`
	Default     float64 `json:"default"`
	Min         float64 `json:"min"`
	Max         float64 `json:"max"`
	Description string  `json:"description"`
}

type FactoryRegistry struct {
	factories map[string]StrategyFactory
	mu        sync.RWMutex
}

func NewFactoryRegistry() *FactoryRegistry {
	return &FactoryRegistry{
		factories: make(map[string]StrategyFactory),
	}
}

func (r *FactoryRegistry) Register(factory StrategyFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[factory.Name()] = factory
}

func (r *FactoryRegistry) Get(name string) (StrategyFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	factory, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("策略 %s 未注册", name)
	}
	return factory, nil
}

func (r *FactoryRegistry) List() []StrategyFactory {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]StrategyFactory, 0, len(r.factories))
	for _, f := range r.factories {
		list = append(list, f)
	}
	return list
}

func (r *FactoryRegistry) ListConfigs() []StrategyConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]StrategyConfig, 0, len(r.factories))
	for _, f := range r.factories {
		config := StrategyConfig{
			Name:        f.Name(),
			DisplayName: f.DisplayName(),
			Description: f.Description(),
			Params:      f.GetParams(),
			Enabled:     true,
		}
		list = append(list, config)
	}
	return list
}

type YinYangFactory struct{}

func NewYinYangFactory() *YinYangFactory {
	return &YinYangFactory{}
}

func (f *YinYangFactory) Name() string {
	return "yinyang"
}

func (f *YinYangFactory) DisplayName() string {
	return "阴阳线突破策略"
}

func (f *YinYangFactory) Description() string {
	return "基于阴阳线形态的趋势跟踪策略，通过识别连续阳线和连续阴线形成的阴阳集合，在价格突破集合边界时产生交易信号"
}

func (f *YinYangFactory) GetParams() []StrategyParamConfig {
	return []StrategyParamConfig{
		{
			Name:        "leverage",
			DisplayName: "杠杆系数",
			Type:        "float",
			Default:     3.0,
			Min:         0.1,
			Max:         10.0,
			Description: "账户承担的百分比风险系数",
		},
	}
}

func (f *YinYangFactory) Create(params map[string]interface{}) SignalStrategy {
	leverage := 3.0
	if v, ok := params["leverage"]; ok {
		if f, ok := v.(float64); ok {
			leverage = f
		}
	}
	strategy := yinyang.NewYinYangStrategy(leverage)
	return yinyang.NewYinYangAdapter(strategy)
}

func (f *YinYangFactory) CreateRolloverHandler(strategy SignalStrategy) backtest.RolloverHandler {
	adapter, ok := strategy.(*yinyang.YinYangAdapter)
	if !ok {
		return nil
	}
	return yinyang.NewRolloverHandler(adapter.GetStrategy())
}

func (f *YinYangFactory) CreateStateRecorder() backtest.StateRecorder {
	return yinyang.NewYinYangStateRecorder()
}

func (f *YinYangFactory) GetWarmupDays(params map[string]interface{}) int {
	return 20
}

type MAFactory struct{}

func NewMAFactory() *MAFactory {
	return &MAFactory{}
}

func (f *MAFactory) Name() string {
	return "ma"
}

func (f *MAFactory) DisplayName() string {
	return "双均线交叉策略"
}

func (f *MAFactory) Description() string {
	return "基于双均线交叉的趋势跟踪策略，短期均线上穿长期均线产生做多信号（金叉），短期均线下穿长期均线产生做空信号（死叉），信号在下一根K线开盘价执行"
}

func (f *MAFactory) GetParams() []StrategyParamConfig {
	return []StrategyParamConfig{
		{
			Name:        "short_period",
			DisplayName: "短期均线周期",
			Type:        "int",
			Default:     5,
			Min:         2,
			Max:         50,
			Description: "短期移动平均线的计算周期",
		},
		{
			Name:        "long_period",
			DisplayName: "长期均线周期",
			Type:        "int",
			Default:     20,
			Min:         5,
			Max:         200,
			Description: "长期移动平均线的计算周期",
		},
		{
			Name:        "leverage",
			DisplayName: "杠杆系数",
			Type:        "float",
			Default:     1.0,
			Min:         0.1,
			Max:         10.0,
			Description: "账户承担的百分比风险系数",
		},
	}
}

func (f *MAFactory) Create(params map[string]interface{}) SignalStrategy {
	shortPeriod := 5
	longPeriod := 20
	leverage := 1.0

	if v, ok := params["short_period"]; ok {
		if f, ok := v.(float64); ok {
			shortPeriod = int(f)
		}
	}

	if v, ok := params["long_period"]; ok {
		if f, ok := v.(float64); ok {
			longPeriod = int(f)
		}
	}

	if v, ok := params["leverage"]; ok {
		if f, ok := v.(float64); ok {
			leverage = f
		}
	}

	strategy := ma.NewMAStrategy(shortPeriod, longPeriod, leverage)
	return ma.NewMAAdapter(strategy)
}

func (f *MAFactory) CreateRolloverHandler(strategy SignalStrategy) backtest.RolloverHandler {
	adapter, ok := strategy.(*ma.MAAdapter)
	if !ok {
		return nil
	}
	return ma.NewRolloverHandler(adapter.GetStrategy())
}

func (f *MAFactory) CreateStateRecorder() backtest.StateRecorder {
	return backtest.NewDefaultStateRecorder()
}

func (f *MAFactory) GetWarmupDays(params map[string]interface{}) int {
	longPeriod := 20
	if v, ok := params["long_period"]; ok {
		if f, ok := v.(float64); ok {
			longPeriod = int(f)
		}
	}
	return longPeriod + 1
}

var DefaultRegistry = NewFactoryRegistry()

func init() {
	DefaultRegistry.Register(NewYinYangFactory())
	DefaultRegistry.Register(NewMAFactory())
}
