package strategy

import (
	"fmt"
	"sync"

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
			Description: f.Description(),
			Enabled:     true,
		}
		if yf, ok := f.(*YinYangFactory); ok {
			config.DisplayName = yf.DisplayName()
			config.Params = yf.GetParams()
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

var DefaultRegistry = NewFactoryRegistry()

func init() {
	DefaultRegistry.Register(NewYinYangFactory())
}
