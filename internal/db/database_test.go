package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	if err := ResetDB(dbPath); err != nil {
		t.Fatalf("初始化测试数据库失败: %v", err)
	}
}

func teardownTestDB(t *testing.T) {
	t.Helper()
	CloseDB()
}

func TestInitDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	if err := ResetDB(dbPath); err != nil {
		t.Fatalf("InitDB 失败: %v", err)
	}
	defer CloseDB()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("数据库文件未创建")
	}
}

func TestSymbolCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	symbol := Symbol{
		Code:     "RB",
		Name:     "螺纹钢",
		Exchange: "SHFE",
		Pinyin:   "lwg",
	}

	if err := UpsertSymbol(symbol); err != nil {
		t.Fatalf("UpsertSymbol 失败: %v", err)
	}

	got, err := GetSymbolByCode("RB")
	if err != nil {
		t.Fatalf("GetSymbolByCode 失败: %v", err)
	}
	if got == nil {
		t.Fatal("GetSymbolByCode 返回 nil")
	}
	if got.Name != "螺纹钢" {
		t.Errorf("品种名称不匹配: got=%s, want=螺纹钢", got.Name)
	}

	symbol.Name = "螺纹钢更新"
	if err := UpsertSymbol(symbol); err != nil {
		t.Fatalf("UpsertSymbol 更新失败: %v", err)
	}
	got, _ = GetSymbolByCode("RB")
	if got.Name != "螺纹钢更新" {
		t.Errorf("品种名称更新不匹配: got=%s, want=螺纹钢更新", got.Name)
	}

	symbols, err := GetAllSymbols()
	if err != nil {
		t.Fatalf("GetAllSymbols 失败: %v", err)
	}
	if len(symbols) != 1 {
		t.Errorf("GetAllSymbols 数量不匹配: got=%d, want=1", len(symbols))
	}

	if err := DeleteSymbol("RB"); err != nil {
		t.Fatalf("DeleteSymbol 失败: %v", err)
	}
	got, _ = GetSymbolByCode("RB")
	if got != nil {
		t.Error("DeleteSymbol 后品种仍存在")
	}
}

func TestUpsertSymbols(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	symbols := []Symbol{
		{Code: "RB", Name: "螺纹钢", Exchange: "SHFE", Pinyin: "lwg"},
		{Code: "HC", Name: "热轧卷板", Exchange: "SHFE", Pinyin: "rzjb"},
		{Code: "I", Name: "铁矿石", Exchange: "DCE", Pinyin: "tks"},
	}

	if err := UpsertSymbols(symbols); err != nil {
		t.Fatalf("UpsertSymbols 失败: %v", err)
	}

	all, err := GetAllSymbols()
	if err != nil {
		t.Fatalf("GetAllSymbols 失败: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("GetAllSymbols 数量不匹配: got=%d, want=3", len(all))
	}
}

func TestSearchSymbols(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	symbols := []Symbol{
		{Code: "RB", Name: "螺纹钢", Exchange: "SHFE", Pinyin: "lwg"},
		{Code: "HC", Name: "热轧卷板", Exchange: "SHFE", Pinyin: "rzjb"},
		{Code: "I", Name: "铁矿石", Exchange: "DCE", Pinyin: "tks"},
	}
	UpsertSymbols(symbols)

	results, err := SearchSymbols("铁")
	if err != nil {
		t.Fatalf("SearchSymbols 失败: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("SearchSymbols('铁') 数量不匹配: got=%d, want=1", len(results))
	}

	results, err = SearchSymbols("SHFE")
	if err != nil {
		t.Fatalf("SearchSymbols 失败: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("SearchSymbols('SHFE') 数量不匹配: got=%d, want=2", len(results))
	}
}

func TestStrategyCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	s := Strategy{
		Name:        "test_strategy",
		DisplayName: "测试策略",
		Description: "用于测试的策略",
		Enabled:     true,
		Params: []StrategyParam{
			{Name: "period", DisplayName: "周期", Type: "int", Default: 10, Min: 1, Max: 100, Description: "计算周期"},
			{Name: "leverage", DisplayName: "杠杆", Type: "float", Default: 3.0, Min: 0.1, Max: 10.0, Description: "杠杆系数"},
		},
	}

	if err := UpsertStrategy(s); err != nil {
		t.Fatalf("UpsertStrategy 失败: %v", err)
	}

	got, err := GetStrategyByName("test_strategy")
	if err != nil {
		t.Fatalf("GetStrategyByName 失败: %v", err)
	}
	if got == nil {
		t.Fatal("GetStrategyByName 返回 nil")
	}
	if got.DisplayName != "测试策略" {
		t.Errorf("策略显示名称不匹配: got=%s, want=测试策略", got.DisplayName)
	}
	if len(got.Params) != 2 {
		t.Errorf("策略参数数量不匹配: got=%d, want=2", len(got.Params))
	}
	if got.Params[0].Name != "period" {
		t.Errorf("第一个参数名称不匹配: got=%s, want=period", got.Params[0].Name)
	}

	all, err := GetAllStrategies()
	if err != nil {
		t.Fatalf("GetAllStrategies 失败: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("GetAllStrategies 数量不匹配: got=%d, want=1", len(all))
	}

	if err := DeleteStrategy("test_strategy"); err != nil {
		t.Fatalf("DeleteStrategy 失败: %v", err)
	}
	got, _ = GetStrategyByName("test_strategy")
	if got != nil {
		t.Error("DeleteStrategy 后策略仍存在")
	}
}

func TestFundCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	fund := Fund{
		ID:          "test_fund",
		Name:        "测试基金",
		Description: "用于测试的基金",
		StartDate:   "20240101",
		EndDate:     "20241231",
		Positions: []FundPosition{
			{Symbol: "RB", Strategy: "yinyang", Weight: 0.6, Params: map[string]interface{}{"leverage": float64(3)}},
			{Symbol: "TA", Strategy: "yinyang", Weight: 0.4, Params: map[string]interface{}{"leverage": float64(2)}},
		},
	}

	if err := UpsertFund(fund); err != nil {
		t.Fatalf("UpsertFund 失败: %v", err)
	}

	got, err := GetFundByID("test_fund")
	if err != nil {
		t.Fatalf("GetFundByID 失败: %v", err)
	}
	if got == nil {
		t.Fatal("GetFundByID 返回 nil")
	}
	if got.Name != "测试基金" {
		t.Errorf("基金名称不匹配: got=%s, want=测试基金", got.Name)
	}
	if len(got.Positions) != 2 {
		t.Errorf("持仓数量不匹配: got=%d, want=2", len(got.Positions))
	}
	if got.Positions[0].Weight != 0.6 {
		t.Errorf("持仓权重不匹配: got=%.1f, want=0.6", got.Positions[0].Weight)
	}

	all, err := GetAllFunds()
	if err != nil {
		t.Fatalf("GetAllFunds 失败: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("GetAllFunds 数量不匹配: got=%d, want=1", len(all))
	}

	if err := DeleteFund("test_fund"); err != nil {
		t.Fatalf("DeleteFund 失败: %v", err)
	}
	got, _ = GetFundByID("test_fund")
	if got != nil {
		t.Error("DeleteFund 后基金仍存在")
	}
}

func TestFundUpdate(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	fund := Fund{
		ID:          "test_fund",
		Name:        "测试基金",
		Description: "初始描述",
		StartDate:   "20240101",
		EndDate:     "20241231",
		Positions: []FundPosition{
			{Symbol: "RB", Strategy: "yinyang", Weight: 1.0},
		},
	}
	UpsertFund(fund)

	fund.Name = "更新基金"
	fund.Description = "更新描述"
	fund.Positions = []FundPosition{
		{Symbol: "RB", Strategy: "yinyang", Weight: 0.5},
		{Symbol: "HC", Strategy: "ma", Weight: 0.5},
	}
	if err := UpsertFund(fund); err != nil {
		t.Fatalf("UpsertFund 更新失败: %v", err)
	}

	got, _ := GetFundByID("test_fund")
	if got.Name != "更新基金" {
		t.Errorf("基金名称未更新: got=%s", got.Name)
	}
	if len(got.Positions) != 2 {
		t.Errorf("持仓数量未更新: got=%d, want=2", len(got.Positions))
	}
}

func TestBacktestResultCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	r := BacktestResult{
		ID:           "test_result_001",
		Symbol:       "RB",
		Strategy:     "yinyang",
		StartDate:    "20240101",
		EndDate:      "20241231",
		Leverage:     3.0,
		TotalReturn:  0.1523,
		AnnualReturn: 0.3012,
		WinRate:      0.55,
		TradingDays:  240,
		FinalValue:   1.1523,
		Signals:      `[{"date":"2024-01-02","direction":"long"}]`,
		DailyRecords: `[{"date":"2024-01-02","value":1.0}]`,
	}

	if err := SaveBacktestResult(r); err != nil {
		t.Fatalf("SaveBacktestResult 失败: %v", err)
	}

	got, err := GetBacktestResult("test_result_001")
	if err != nil {
		t.Fatalf("GetBacktestResult 失败: %v", err)
	}
	if got == nil {
		t.Fatal("GetBacktestResult 返回 nil")
	}
	if got.Symbol != "RB" {
		t.Errorf("品种不匹配: got=%s, want=RB", got.Symbol)
	}
	if got.TotalReturn != 0.1523 {
		t.Errorf("总收益率不匹配: got=%.4f, want=0.1523", got.TotalReturn)
	}

	results, err := ListBacktestResults()
	if err != nil {
		t.Fatalf("ListBacktestResults 失败: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("ListBacktestResults 数量不匹配: got=%d, want=1", len(results))
	}

	if err := DeleteBacktestResult("test_result_001"); err != nil {
		t.Fatalf("DeleteBacktestResult 失败: %v", err)
	}
	got, _ = GetBacktestResult("test_result_001")
	if got != nil {
		t.Error("DeleteBacktestResult 后结果仍存在")
	}
}

func TestFundResultCRUD(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	fund := Fund{
		ID:        "test_fund",
		Name:      "测试基金",
		StartDate: "20240101",
		EndDate:   "20241231",
		Positions: []FundPosition{
			{Symbol: "RB", Strategy: "yinyang", Weight: 1.0},
		},
	}
	UpsertFund(fund)

	r := FundResult{
		ID:           "result_001",
		FundID:       "test_fund",
		FundName:     "测试基金",
		StartDate:    "20240101",
		EndDate:      "20241231",
		Timestamp:    1700000000,
		TotalReturn:  0.25,
		WinRate:      0.6,
		TradingDays:  240,
		DailyRecords: `[{"date":"2024-01-02","value":1.0}]`,
	}

	if err := SaveFundResult(r); err != nil {
		t.Fatalf("SaveFundResult 失败: %v", err)
	}

	got, err := GetFundResult("test_fund", "result_001")
	if err != nil {
		t.Fatalf("GetFundResult 失败: %v", err)
	}
	if got == nil {
		t.Fatal("GetFundResult 返回 nil")
	}
	if got.FundName != "测试基金" {
		t.Errorf("基金名称不匹配: got=%s", got.FundName)
	}

	results, err := ListFundResultsByFundID("test_fund")
	if err != nil {
		t.Fatalf("ListFundResultsByFundID 失败: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("ListFundResultsByFundID 数量不匹配: got=%d, want=1", len(results))
	}

	if err := DeleteFundResult("test_fund", "result_001"); err != nil {
		t.Fatalf("DeleteFundResult 失败: %v", err)
	}
	got, _ = GetFundResult("test_fund", "result_001")
	if got != nil {
		t.Error("DeleteFundResult 后结果仍存在")
	}
}

func TestConfigMeta(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	if err := SetConfigMeta("default_strategy", "yinyang"); err != nil {
		t.Fatalf("SetConfigMeta 失败: %v", err)
	}

	got, err := GetConfigMeta("default_strategy")
	if err != nil {
		t.Fatalf("GetConfigMeta 失败: %v", err)
	}
	if got != "yinyang" {
		t.Errorf("配置值不匹配: got=%s, want=yinyang", got)
	}

	if err := SetConfigMeta("default_strategy", "ma"); err != nil {
		t.Fatalf("SetConfigMeta 更新失败: %v", err)
	}
	got, _ = GetConfigMeta("default_strategy")
	if got != "ma" {
		t.Errorf("配置值更新不匹配: got=%s, want=ma", got)
	}

	val, _ := GetConfigMeta("nonexistent")
	if val != "" {
		t.Errorf("不存在的配置应返回空字符串: got=%s", val)
	}
}

func TestExportTable(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	UpsertSymbol(Symbol{Code: "RB", Name: "螺纹钢", Exchange: "SHFE", Pinyin: "lwg"})

	jsonData, err := ExportTableToJSON("symbols")
	if err != nil {
		t.Fatalf("ExportTableToJSON 失败: %v", err)
	}
	if len(jsonData) == 0 {
		t.Error("ExportTableToJSON 返回空数据")
	}

	csvData, err := ExportTableToCSV("symbols")
	if err != nil {
		t.Fatalf("ExportTableToCSV 失败: %v", err)
	}
	if len(csvData) == 0 {
		t.Error("ExportTableToCSV 返回空数据")
	}
}

func TestExportTableSQLInjection(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	_, err := ExportTableToJSON("symbols; DROP TABLE symbols; --")
	if err == nil {
		t.Error("ExportTableToJSON 应拒绝非法表名")
	}

	_, err = ExportTableToCSV("symbols; DROP TABLE symbols; --")
	if err == nil {
		t.Error("ExportTableToCSV 应拒绝非法表名")
	}

	_, err = ExportTableToJSON("nonexistent_table")
	if err == nil {
		t.Error("ExportTableToJSON 应拒绝不存在的表名")
	}

	_, err = ExportTableToCSV("nonexistent_table")
	if err == nil {
		t.Error("ExportTableToCSV 应拒绝不存在的表名")
	}

	jsonData, err := ExportTableToJSON("backtest_results")
	if err != nil {
		t.Errorf("ExportTableToJSON 应允许合法表名 backtest_results: %v", err)
	}

	csvData, err := ExportTableToCSV("config_meta")
	if err != nil {
		t.Errorf("ExportTableToCSV 应允许合法表名 config_meta: %v", err)
	}
	_ = jsonData
	_ = csvData
}

func TestTransactionRollback(t *testing.T) {
	setupTestDB(t)
	defer teardownTestDB(t)

	err := WithTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO symbols (code, name, exchange, pinyin) VALUES (?, ?, ?, ?)`, "RB", "螺纹钢", "SHFE", "lwg")
		if err != nil {
			return err
		}
		return fmt.Errorf("模拟错误触发回滚")
	})
	if err == nil {
		t.Error("WithTx 应该返回错误")
	}

	got, _ := GetSymbolByCode("RB")
	if got != nil {
		t.Error("事务回滚后数据不应存在")
	}
}
