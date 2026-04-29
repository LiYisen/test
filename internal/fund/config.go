package fund

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
)

type FundConfigFile struct {
	Funds []FundConfig `json:"funds"`
}

var (
	configMu    sync.RWMutex
	fundConfigs map[string]*FundConfig
)

func LoadFundConfig(configPath string) error {
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			configMu.Lock()
			fundConfigs = make(map[string]*FundConfig)
			configMu.Unlock()
			return nil
		}
		return fmt.Errorf("打开基金配置文件失败: %w", err)
	}
	defer file.Close()

	var configFile FundConfigFile
	if err := json.NewDecoder(file).Decode(&configFile); err != nil {
		return fmt.Errorf("解析基金配置文件失败: %w", err)
	}

	configs := make(map[string]*FundConfig)
	for i := range configFile.Funds {
		fund := &configFile.Funds[i]
		configs[fund.ID] = fund
	}

	configMu.Lock()
	fundConfigs = configs
	configMu.Unlock()

	return nil
}

func GetFundConfig(fundID string) (*FundConfig, error) {
	configMu.RLock()
	defer configMu.RUnlock()

	if fundConfigs == nil {
		return nil, fmt.Errorf("基金配置未加载")
	}

	fund, ok := fundConfigs[fundID]
	if !ok {
		return nil, fmt.Errorf("基金配置不存在: %s", fundID)
	}

	return fund, nil
}

func GetAllFundConfigs() ([]*FundConfig, error) {
	configMu.RLock()
	defer configMu.RUnlock()

	if fundConfigs == nil {
		return nil, fmt.Errorf("基金配置未加载")
	}

	var configs []*FundConfig
	for _, fund := range fundConfigs {
		configs = append(configs, fund)
	}
	return configs, nil
}

func SaveFundConfig(configPath string, fund *FundConfig) error {
	configMu.Lock()
	defer configMu.Unlock()

	if fundConfigs == nil {
		fundConfigs = make(map[string]*FundConfig)

		file, err := os.Open(configPath)
		if err == nil {
			var configFile FundConfigFile
			if err := json.NewDecoder(file).Decode(&configFile); err == nil {
				for i := range configFile.Funds {
					existingFund := &configFile.Funds[i]
					fundConfigs[existingFund.ID] = existingFund
				}
			}
			file.Close()
		}
	}

	fundConfigs[fund.ID] = fund

	var configFile FundConfigFile
	for _, f := range fundConfigs {
		configFile.Funds = append(configFile.Funds, *f)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(configFile); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

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
