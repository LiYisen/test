package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"futures-backtest/internal/db"
	"futures-backtest/internal/strategy"
)

func main() {
	flag.Parse()

	dbPath := db.GetDefaultDBPath()
	if err := db.InitDB(dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "初始化数据库失败: %v\n", err)
		os.Exit(1)
	}
	defer db.CloseDB()

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		return
	}

	switch args[0] {
	case "tables":
		listTables()
	case "show":
		if len(args) < 2 {
			fmt.Println("用法: dbcli show <表名>")
			return
		}
		showTable(args[1])
	case "symbols":
		listSymbols()
	case "search":
		if len(args) < 2 {
			fmt.Println("用法: dbcli search <关键词>")
			return
		}
		searchSymbols(strings.Join(args[1:], " "))
	case "strategies":
		listStrategies()
	case "funds":
		listFunds()
	case "fund":
		if len(args) < 2 {
			fmt.Println("用法: dbcli fund <基金ID>")
			return
		}
		showFund(args[1])
	case "results":
		listResults()
	case "fund-results":
		listFundResults()
	case "export":
		if len(args) < 3 {
			fmt.Println("用法: dbcli export <表名> <json|csv>")
			return
		}
		exportTable(args[1], args[2])
	case "migrate":
		migrateData()
	case "count":
		if len(args) < 2 {
			fmt.Println("用法: dbcli count <表名>")
			return
		}
		countTable(args[1])
	case "delete":
		if len(args) < 3 {
			fmt.Println("用法: dbcli delete <表名> <ID>")
			return
		}
		deleteRecord(args[1], args[2])
	default:
		fmt.Printf("未知命令: %s\n", args[0])
		printUsage()
	}
}

func printUsage() {
	fmt.Println("期货回测数据库查询工具")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  tables          列出所有表")
	fmt.Println("  show <表名>     显示表内容")
	fmt.Println("  count <表名>    显示表记录数")
	fmt.Println("  symbols         列出所有品种")
	fmt.Println("  search <关键词> 搜索品种")
	fmt.Println("  strategies      列出所有策略")
	fmt.Println("  funds           列出所有基金")
	fmt.Println("  fund <ID>       显示基金详情")
	fmt.Println("  results         列出回测结果")
	fmt.Println("  fund-results    列出基金回测结果")
	fmt.Println("  export <表> <格式> 导出表数据 (json/csv)")
	fmt.Println("  delete <表> <ID> 删除记录")
	fmt.Println("  migrate         从JSON文件迁移数据")
}

func listTables() {
	tables := []string{"symbols", "funds", "fund_positions",
		"backtest_results", "fund_results"}
	fmt.Println("数据库表:")
	for _, t := range tables {
		fmt.Printf("  %s\n", t)
	}
}

func showTable(name string) {
	switch name {
	case "symbols":
		listSymbols()
	case "strategies":
		listStrategies()
	case "funds":
		listFunds()
	case "backtest_results":
		listResults()
	case "fund_results":
		listFundResults()
	default:
		data, err := db.ExportTableToJSON(name)
		if err != nil {
			fmt.Printf("查询表 %s 失败: %v\n", name, err)
			return
		}
		fmt.Println(string(data))
	}
}

func listSymbols() {
	symbols, err := db.GetAllSymbols()
	if err != nil {
		fmt.Printf("查询品种失败: %v\n", err)
		return
	}
	fmt.Printf("共 %d 个品种:\n", len(symbols))
	fmt.Printf("%-8s %-20s %-10s %-10s\n", "代码", "名称", "交易所", "拼音")
	fmt.Println(strings.Repeat("-", 50))
	for _, s := range symbols {
		fmt.Printf("%-8s %-20s %-10s %-10s\n", s.Code, s.Name, s.Exchange, s.Pinyin)
	}
}

func searchSymbols(query string) {
	symbols, err := db.SearchSymbols(query)
	if err != nil {
		fmt.Printf("搜索品种失败: %v\n", err)
		return
	}
	fmt.Printf("搜索 '%s' 找到 %d 个品种:\n", query, len(symbols))
	for _, s := range symbols {
		fmt.Printf("  %-8s %-20s %-10s\n", s.Code, s.Name, s.Exchange)
	}
}

func listStrategies() {
	strategies := strategy.DefaultRegistry.ListConfigs()
	fmt.Printf("共 %d 个策略:\n", len(strategies))
	for _, s := range strategies {
		enabled := "启用"
		if !s.Enabled {
			enabled = "禁用"
		}
		fmt.Printf("  %s (%s) [%s] - %s\n", s.Name, s.DisplayName, enabled, s.Description)
		if len(s.Params) > 0 {
			for _, p := range s.Params {
				fmt.Printf("    参数: %s (%s) 类型=%s 默认=%.1f 范围=[%.1f, %.1f]\n",
					p.Name, p.DisplayName, p.Type, p.Default, p.Min, p.Max)
			}
		}
	}
}

func listFunds() {
	funds, err := db.GetAllFunds()
	if err != nil {
		fmt.Printf("查询基金失败: %v\n", err)
		return
	}
	fmt.Printf("共 %d 个基金:\n", len(funds))
	for _, f := range funds {
		fmt.Printf("  %s: %s (%s ~ %s)\n", f.ID, f.Name, f.StartDate, f.EndDate)
		for _, p := range f.Positions {
			paramsJSON, _ := json.Marshal(p.Params)
			fmt.Printf("    品种: %s 策略: %s 权重: %.2f 参数: %s\n", p.Symbol, p.Strategy, p.Weight, string(paramsJSON))
		}
	}
}

func showFund(fundID string) {
	fund, err := db.GetFundByID(fundID)
	if err != nil {
		fmt.Printf("查询基金失败: %v\n", err)
		return
	}
	if fund == nil {
		fmt.Printf("基金 %s 不存在\n", fundID)
		return
	}
	fmt.Printf("基金ID: %s\n", fund.ID)
	fmt.Printf("名称: %s\n", fund.Name)
	fmt.Printf("描述: %s\n", fund.Description)
	fmt.Printf("日期范围: %s ~ %s\n", fund.StartDate, fund.EndDate)
	fmt.Printf("持仓品种:\n")
	for _, p := range fund.Positions {
		paramsJSON, _ := json.Marshal(p.Params)
		fmt.Printf("  %s/%s 权重=%.2f 参数=%s\n", p.Symbol, p.Strategy, p.Weight, string(paramsJSON))
	}
}

func listResults() {
	results, err := db.ListBacktestResults()
	if err != nil {
		fmt.Printf("查询回测结果失败: %v\n", err)
		return
	}
	fmt.Printf("共 %d 条回测结果:\n", len(results))
	fmt.Printf("%-40s %-6s %-10s %-12s %-12s %-8s %-10s\n",
		"ID", "品种", "策略", "开始日期", "结束日期", "杠杆", "总收益率")
	fmt.Println(strings.Repeat("-", 100))
	for _, r := range results {
		id, _ := r["id"].(string)
		symbol, _ := r["symbol"].(string)
		strategy, _ := r["strategy"].(string)
		startDate, _ := r["start_date"].(string)
		endDate, _ := r["end_date"].(string)
		leverage, _ := r["leverage"].(float64)
		totalReturn, _ := r["total_return"].(float64)
		fmt.Printf("%-40s %-6s %-10s %-12s %-12s %-8.1f %-10.4f\n",
			truncate(id, 38), symbol, strategy, startDate, endDate, leverage, totalReturn)
	}
}

func listFundResults() {
	results, err := db.ListFundResults()
	if err != nil {
		fmt.Printf("查询基金结果失败: %v\n", err)
		return
	}
	fmt.Printf("共 %d 条基金回测结果:\n", len(results))
	fmt.Printf("%-40s %-15s %-12s %-12s %-10s\n",
		"ID", "基金名称", "开始日期", "结束日期", "总收益率")
	fmt.Println(strings.Repeat("-", 90))
	for _, r := range results {
		id, _ := r["id"].(string)
		fundName, _ := r["fund_name"].(string)
		startDate, _ := r["start_date"].(string)
		endDate, _ := r["end_date"].(string)
		totalReturn, _ := r["total_return"].(float64)
		fmt.Printf("%-40s %-15s %-12s %-12s %-10.4f\n",
			truncate(id, 38), fundName, startDate, endDate, totalReturn)
	}
}

func exportTable(tableName, format string) {
	switch strings.ToLower(format) {
	case "json":
		data, err := db.ExportTableToJSON(tableName)
		if err != nil {
			fmt.Printf("导出失败: %v\n", err)
			return
		}
		fmt.Println(string(data))
	case "csv":
		data, err := db.ExportTableToCSV(tableName)
		if err != nil {
			fmt.Printf("导出失败: %v\n", err)
			return
		}
		fmt.Println(data)
	default:
		fmt.Printf("不支持的格式: %s (支持: json, csv)\n", format)
	}
}

func countTable(tableName string) {
	d := db.GetDB()
	if d == nil {
		fmt.Println("数据库未初始化")
		return
	}
	var count int
	err := d.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	if err != nil {
		fmt.Printf("查询失败: %v\n", err)
		return
	}
	fmt.Printf("%s: %d 条记录\n", tableName, count)
}

func deleteRecord(tableName, id string) {
	switch tableName {
	case "backtest_results":
		if err := db.DeleteBacktestResult(id); err != nil {
			fmt.Printf("删除失败: %v\n", err)
			return
		}
		fmt.Printf("已删除回测结果: %s\n", id)
	case "fund_results":
		if err := db.DeleteFundResult("", id); err != nil {
			fmt.Printf("删除失败: %v\n", err)
			return
		}
		fmt.Printf("已删除基金结果: %s\n", id)
	case "funds":
		if err := db.DeleteFund(id); err != nil {
			fmt.Printf("删除失败: %v\n", err)
			return
		}
		fmt.Printf("已删除基金: %s\n", id)
	case "symbols":
		if err := db.DeleteSymbol(id); err != nil {
			fmt.Printf("删除失败: %v\n", err)
			return
		}
		fmt.Printf("已删除品种: %s\n", id)
	default:
		fmt.Printf("不支持的表: %s\n", tableName)
	}
}

func migrateData() {
	fmt.Println("开始从JSON文件迁移数据...")
	if err := db.MigrateFromJSON("config"); err != nil {
		fmt.Printf("迁移配置数据失败: %v\n", err)
	} else {
		fmt.Println("配置数据迁移完成")
	}

	if err := db.MigrateResultsFromDir("ret"); err != nil {
		fmt.Printf("迁移结果数据失败: %v\n", err)
	} else {
		fmt.Println("结果数据迁移完成")
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}
