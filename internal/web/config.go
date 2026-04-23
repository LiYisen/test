package web

import (
	"bytes"
	"encoding/json"
	"futures-backtest/internal/strategy"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type SymbolConfig struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
	Pinyin   string `json:"pinyin"`
}

type Config struct {
	Symbols []SymbolConfig `json:"symbols"`
}

type symbolResponse struct {
	Symbols []SymbolConfig `json:"symbols"`
	Error   string         `json:"error"`
}

var (
	config     *Config
	configOnce sync.Once
	configMu   sync.RWMutex
)

func LoadConfig() *Config {
	configOnce.Do(func() {
		config = &Config{}
		loadConfigFromFile()
		if len(config.Symbols) > 0 {
			log.Printf("从配置文件加载了 %d 个品种", len(config.Symbols))
			go fetchSymbolsFromAkshare()
		} else {
			log.Println("配置文件为空，尝试从akshare获取品种列表...")
			fetchSymbolsFromAkshare()
			if len(config.Symbols) == 0 {
				log.Println("akshare获取失败，使用默认品种列表")
				setDefaultSymbols()
			}
		}
	})
	return config
}

func fetchSymbolsFromAkshare() {
	cmd := exec.Command("python", "scripts/get_symbols.py")
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("执行Python脚本失败: %v", err)
		return
	}

	output = bytes.TrimPrefix(output, []byte("\xef\xbb\xbf"))

	var resp symbolResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		log.Printf("解析Python输出失败: %v, 输出内容: %s", err, string(output[:min(200, len(output))]))
		return
	}

	if resp.Error != "" {
		log.Printf("Python脚本返回错误: %s", resp.Error)
		return
	}

	if len(resp.Symbols) == 0 {
		log.Println("Python脚本返回空品种列表")
		return
	}

	configMu.Lock()
	config.Symbols = resp.Symbols
	configMu.Unlock()

	log.Printf("从akshare获取了 %d 个品种", len(resp.Symbols))
	saveConfigToFile()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func saveConfigToFile() {
	configMu.RLock()
	defer configMu.RUnlock()

	os.MkdirAll("config", 0755)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("序列化配置失败: %v", err)
		return
	}

	if err := os.WriteFile("config/symbols.json", data, 0644); err != nil {
		log.Printf("写入配置文件失败: %v", err)
	}
}

func loadConfigFromFile() {
	file, err := os.Open("config/symbols.json")
	if err != nil {
		log.Printf("打开配置文件失败: %v", err)
		return
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		log.Printf("解析配置文件失败: %v", err)
		return
	}

	configMu.Lock()
	config.Symbols = cfg.Symbols
	configMu.Unlock()
}

func setDefaultSymbols() {
	configMu.Lock()
	defer configMu.Unlock()

	config.Symbols = []SymbolConfig{
		{Code: "RB", Name: "螺纹钢", Exchange: "SHFE", Pinyin: "lwg"},
		{Code: "HC", Name: "热轧卷板", Exchange: "SHFE", Pinyin: "rzjb"},
		{Code: "I", Name: "铁矿石", Exchange: "DCE", Pinyin: "tks"},
		{Code: "J", Name: "焦炭", Exchange: "DCE", Pinyin: "jt"},
		{Code: "JM", Name: "焦煤", Exchange: "DCE", Pinyin: "jm"},
		{Code: "AU", Name: "黄金", Exchange: "SHFE", Pinyin: "hj"},
		{Code: "AG", Name: "白银", Exchange: "SHFE", Pinyin: "by"},
		{Code: "CU", Name: "铜", Exchange: "SHFE", Pinyin: "t"},
		{Code: "AL", Name: "铝", Exchange: "SHFE", Pinyin: "l"},
		{Code: "ZN", Name: "锌", Exchange: "SHFE", Pinyin: "x"},
		{Code: "PB", Name: "铅", Exchange: "SHFE", Pinyin: "q"},
		{Code: "NI", Name: "镍", Exchange: "SHFE", Pinyin: "n"},
		{Code: "SN", Name: "锡", Exchange: "SHFE", Pinyin: "x"},
		{Code: "SS", Name: "不锈钢", Exchange: "SHFE", Pinyin: "bxg"},
		{Code: "WR", Name: "线材", Exchange: "SHFE", Pinyin: "xc"},
		{Code: "SP", Name: "纸浆", Exchange: "SHFE", Pinyin: "zj"},
		{Code: "FU", Name: "燃料油", Exchange: "SHFE", Pinyin: "rly"},
		{Code: "BU", Name: "沥青", Exchange: "SHFE", Pinyin: "lq"},
		{Code: "RU", Name: "橡胶", Exchange: "SHFE", Pinyin: "xj"},
		{Code: "NR", Name: "20号胶", Exchange: "INE", Pinyin: "ehj"},
		{Code: "SC", Name: "原油", Exchange: "INE", Pinyin: "yy"},
		{Code: "LU", Name: "低硫燃料油", Exchange: "INE", Pinyin: "drly"},
		{Code: "BC", Name: "国际铜", Exchange: "INE", Pinyin: "gjt"},
		{Code: "A", Name: "豆一", Exchange: "DCE", Pinyin: "dy"},
		{Code: "B", Name: "豆二", Exchange: "DCE", Pinyin: "de"},
		{Code: "M", Name: "豆粕", Exchange: "DCE", Pinyin: "dp"},
		{Code: "Y", Name: "豆油", Exchange: "DCE", Pinyin: "dy"},
		{Code: "P", Name: "棕榈油", Exchange: "DCE", Pinyin: "zly"},
		{Code: "C", Name: "玉米", Exchange: "DCE", Pinyin: "ym"},
		{Code: "CS", Name: "玉米淀粉", Exchange: "DCE", Pinyin: "ymdf"},
		{Code: "JD", Name: "鸡蛋", Exchange: "DCE", Pinyin: "jd"},
		{Code: "LH", Name: "生猪", Exchange: "DCE", Pinyin: "sz"},
		{Code: "PP", Name: "聚丙烯", Exchange: "DCE", Pinyin: "jbx"},
		{Code: "L", Name: "塑料", Exchange: "DCE", Pinyin: "sl"},
		{Code: "V", Name: "PVC", Exchange: "DCE", Pinyin: "pvc"},
		{Code: "EG", Name: "乙二醇", Exchange: "DCE", Pinyin: "yec"},
		{Code: "EB", Name: "苯乙烯", Exchange: "DCE", Pinyin: "bxy"},
		{Code: "PG", Name: "液化石油气", Exchange: "DCE", Pinyin: "yhsyq"},
		{Code: "CF", Name: "棉花", Exchange: "CZCE", Pinyin: "mh"},
		{Code: "SR", Name: "白糖", Exchange: "CZCE", Pinyin: "bt"},
		{Code: "TA", Name: "PTA", Exchange: "CZCE", Pinyin: "pta"},
		{Code: "MA", Name: "甲醇", Exchange: "CZCE", Pinyin: "jc"},
		{Code: "FG", Name: "玻璃", Exchange: "CZCE", Pinyin: "bl"},
		{Code: "SA", Name: "纯碱", Exchange: "CZCE", Pinyin: "cj"},
		{Code: "UR", Name: "尿素", Exchange: "CZCE", Pinyin: "ns"},
		{Code: "OI", Name: "菜油", Exchange: "CZCE", Pinyin: "cy"},
		{Code: "RM", Name: "菜粕", Exchange: "CZCE", Pinyin: "cp"},
		{Code: "ZC", Name: "动力煤", Exchange: "CZCE", Pinyin: "dlm"},
		{Code: "AP", Name: "苹果", Exchange: "CZCE", Pinyin: "pg"},
		{Code: "CJ", Name: "红枣", Exchange: "CZCE", Pinyin: "hz"},
		{Code: "PK", Name: "花生", Exchange: "CZCE", Pinyin: "hs"},
		{Code: "SI", Name: "工业硅", Exchange: "GFEX", Pinyin: "gyg"},
		{Code: "LC", Name: "碳酸锂", Exchange: "GFEX", Pinyin: "tsl"},
		{Code: "IF", Name: "沪深300", Exchange: "CFFEX", Pinyin: "hs300"},
		{Code: "IC", Name: "中证500", Exchange: "CFFEX", Pinyin: "zz500"},
		{Code: "IM", Name: "中证1000", Exchange: "CFFEX", Pinyin: "zz1000"},
		{Code: "IH", Name: "上证50", Exchange: "CFFEX", Pinyin: "sz50"},
		{Code: "T", Name: "十年国债", Exchange: "CFFEX", Pinyin: "sngz"},
		{Code: "TF", Name: "五年国债", Exchange: "CFFEX", Pinyin: "wngz"},
		{Code: "TS", Name: "二年国债", Exchange: "CFFEX", Pinyin: "engz"},
	}
	log.Printf("使用默认品种列表，共 %d 个品种", len(config.Symbols))
}

func GetSymbols() []SymbolConfig {
	configMu.RLock()
	defer configMu.RUnlock()
	if config == nil {
		configMu.RUnlock()
		LoadConfig()
		configMu.RLock()
	}
	return config.Symbols
}

func SearchSymbols(query string) []SymbolConfig {
	configMu.RLock()
	defer configMu.RUnlock()

	if config == nil {
		configMu.RUnlock()
		LoadConfig()
		configMu.RLock()
	}

	if query == "" {
		return config.Symbols
	}

	query = strings.ToLower(query)
	var results []SymbolConfig

	for _, s := range config.Symbols {
		if strings.Contains(strings.ToLower(s.Code), query) ||
			strings.Contains(s.Name, query) ||
			strings.Contains(strings.ToLower(s.Pinyin), query) {
			results = append(results, s)
		}
	}

	return results
}

func GetStrategies() []strategy.StrategyConfig {
	return strategy.DefaultRegistry.ListConfigs()
}
