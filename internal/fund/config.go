package fund

import (
	"fmt"
	"math"
	"sync"

	"futures-backtest/internal/db"
)

var (
	configMu    sync.RWMutex
	fundConfigs map[string]*FundConfig
)

func LoadFundConfig(configPath string) error {
	funds, err := db.GetAllFunds()
	if err != nil {
		return fmt.Errorf("从数据库加载基金配置失败: %w", err)
	}

	configMu.Lock()
	fundConfigs = make(map[string]*FundConfig)
	for i := range funds {
		fund := &FundConfig{
			ID:          funds[i].ID,
			Name:        funds[i].Name,
			Description: funds[i].Description,
			StartDate:   funds[i].StartDate,
			EndDate:     funds[i].EndDate,
		}
		for _, pos := range funds[i].Positions {
			fund.Positions = append(fund.Positions, PositionConfig{
				Symbol:   pos.Symbol,
				Strategy: pos.Strategy,
				Weight:   pos.Weight,
				Params:   pos.Params,
			})
		}
		fundConfigs[fund.ID] = fund
	}
	configMu.Unlock()

	return nil
}

func GetFundConfig(fundID string) (*FundConfig, error) {
	configMu.RLock()
	if fundConfigs != nil {
		if fund, ok := fundConfigs[fundID]; ok {
			configMu.RUnlock()
			return fund, nil
		}
	}
	configMu.RUnlock()

	dbFund, err := db.GetFundByID(fundID)
	if err != nil {
		return nil, fmt.Errorf("查询基金配置失败: %w", err)
	}
	if dbFund == nil {
		return nil, fmt.Errorf("基金配置不存在: %s", fundID)
	}

	fund := &FundConfig{
		ID:          dbFund.ID,
		Name:        dbFund.Name,
		Description: dbFund.Description,
		StartDate:   dbFund.StartDate,
		EndDate:     dbFund.EndDate,
	}
	for _, pos := range dbFund.Positions {
		fund.Positions = append(fund.Positions, PositionConfig{
			Symbol:   pos.Symbol,
			Strategy: pos.Strategy,
			Weight:   pos.Weight,
			Params:   pos.Params,
		})
	}

	configMu.Lock()
	if fundConfigs == nil {
		fundConfigs = make(map[string]*FundConfig)
	}
	fundConfigs[fund.ID] = fund
	configMu.Unlock()

	return fund, nil
}

func GetAllFundConfigs() ([]*FundConfig, error) {
	dbFunds, err := db.GetAllFunds()
	if err != nil {
		return nil, fmt.Errorf("查询基金配置失败: %w", err)
	}

	var configs []*FundConfig
	for i := range dbFunds {
		fund := &FundConfig{
			ID:          dbFunds[i].ID,
			Name:        dbFunds[i].Name,
			Description: dbFunds[i].Description,
			StartDate:   dbFunds[i].StartDate,
			EndDate:     dbFunds[i].EndDate,
		}
		for _, pos := range dbFunds[i].Positions {
			fund.Positions = append(fund.Positions, PositionConfig{
				Symbol:   pos.Symbol,
				Strategy: pos.Strategy,
				Weight:   pos.Weight,
				Params:   pos.Params,
			})
		}
		configs = append(configs, fund)
	}

	configMu.Lock()
	fundConfigs = make(map[string]*FundConfig)
	for _, f := range configs {
		fundConfigs[f.ID] = f
	}
	configMu.Unlock()

	return configs, nil
}

func SaveFundConfig(configPath string, fund *FundConfig) error {
	if err := ValidateFundConfig(fund); err != nil {
		return err
	}

	dbFund := db.Fund{
		ID:          fund.ID,
		Name:        fund.Name,
		Description: fund.Description,
		StartDate:   fund.StartDate,
		EndDate:     fund.EndDate,
	}
	for _, pos := range fund.Positions {
		dbFund.Positions = append(dbFund.Positions, db.FundPosition{
			Symbol:   pos.Symbol,
			Strategy: pos.Strategy,
			Weight:   pos.Weight,
			Params:   pos.Params,
		})
	}

	if err := db.UpsertFund(dbFund); err != nil {
		return fmt.Errorf("保存基金配置到数据库失败: %w", err)
	}

	configMu.Lock()
	if fundConfigs == nil {
		fundConfigs = make(map[string]*FundConfig)
	}
	fundConfigs[fund.ID] = fund
	configMu.Unlock()

	return nil
}

func ValidateFundConfig(fund *FundConfig) error {
	if fund.ID == "" {
		return fmt.Errorf("基金ID不能为空")
	}
	if fund.Name == "" {
		return fmt.Errorf("基金名称不能为空")
	}
	if len(fund.Positions) == 0 {
		return fmt.Errorf("基金至少需要一个持仓品种")
	}

	totalWeight := 0.0
	for _, pos := range fund.Positions {
		if pos.Symbol == "" {
			return fmt.Errorf("品种代码不能为空")
		}
		if pos.Strategy == "" {
			return fmt.Errorf("策略名称不能为空")
		}
		if pos.Weight <= 0 {
			return fmt.Errorf("品种权重必须大于0: %s", pos.Symbol)
		}
		totalWeight += pos.Weight
	}

	if math.Abs(totalWeight-1.0) > 0.001 {
		return fmt.Errorf("权重总和必须为1，当前总和: %.6f", totalWeight)
	}

	return nil
}
