package main

import (
	"fmt"
	"futures-backtest/internal/web"
	"log"
)

func main() {
	server := web.NewServer()
	fmt.Println("========================================")
	fmt.Println("  期货回测 Web 服务")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("访问地址: http://localhost:8080")
	fmt.Println()
	fmt.Println("API 端点:")
	fmt.Println("  POST /api/backtest       - 执行回测")
	fmt.Println("  GET  /api/results        - 列出所有结果")
	fmt.Println("  GET  /api/results/:id    - 获取结果概要")
	fmt.Println("  GET  /api/results/:id/data?type=... - 获取结果数据")
	fmt.Println()
	fmt.Println("数据类型 (type):")
	fmt.Println("  all     - 所有数据")
	fmt.Println("  daily   - 每日资金记录")
	fmt.Println("  signals - 交易信号")
	fmt.Println("  returns - 持仓收益")
	fmt.Println("  stats   - 统计指标")
	fmt.Println()
	fmt.Println("按 Ctrl+C 停止服务")
	fmt.Println("========================================")

	if err := server.Run(":8080"); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
