package web

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"futures-backtest/internal/backtest"
	"futures-backtest/internal/data"
	"futures-backtest/internal/db"
	"futures-backtest/internal/fund"
	"futures-backtest/internal/strategy"
	"futures-backtest/pkg/pyexec"

	"github.com/gin-gonic/gin"
)

// Server Web服务器
type Server struct {
	router      *gin.Engine
	dataManager *data.FuturesDataManager
	retDir      string
}

// NewServer 创建新的Web服务器
func NewServer() *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, err interface{}) {
		log.Printf("[PANIC] 捕获到panic: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("服务器内部错误: %v", err)})
	}))
	r.Use(corsMiddleware())

	if err := db.InitDB(db.GetDefaultDBPath()); err != nil {
		log.Printf("初始化数据库失败: %v", err)
	}

	executor := pyexec.NewDefaultExecutor()
	dataManager := data.NewFuturesDataManager(executor)

	retDir := filepath.Join(".", "ret")
	_ = os.MkdirAll(retDir, 0755)

	s := &Server{
		router:      r,
		dataManager: dataManager,
		retDir:      retDir,
	}

	LoadConfig()

	s.setupRoutes()

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			fund.GetTaskManager().CleanupOldTasks(30 * time.Minute)
		}
	}()

	return s
}

// Run 启动服务器
func (s *Server) Run(addr string) error {
	fmt.Printf("Web服务启动: http://localhost%s\n", addr)
	return s.router.Run(addr)
}

func (s *Server) setupRoutes() {
	s.router.Static("/static", "./web/static")
	s.router.LoadHTMLGlob("web/templates/*")

	s.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	s.router.GET("/portfolio", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	s.router.GET("/fund", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	api := s.router.Group("/api")
	{
		api.GET("/symbols", s.handleGetSymbols)
		api.GET("/strategies", s.handleGetStrategies)
		api.POST("/backtest", s.handleBacktest)
		api.GET("/results", s.handleListResults)
		api.GET("/results/:id", s.handleGetResult)
		api.DELETE("/results/:id", s.handleDeleteResult)
		api.GET("/results/:id/data", s.handleGetResultData)
		api.POST("/portfolio", s.handlePortfolioAnalysis)

		api.GET("/funds", s.handleGetFunds)
		api.GET("/funds/:id", s.handleGetFund)
		api.POST("/funds", s.handleCreateFund)
		api.POST("/funds/backtest", s.handleFundBacktest)
		api.GET("/funds/backtest/progress/:task_id", s.handleFundBacktestProgress)
		api.POST("/funds/test", func(c *gin.Context) {
			log.Printf("[测试] 收到POST请求")
			c.JSON(http.StatusOK, gin.H{"message": "POST test OK"})
		})
		api.GET("/funds/results", s.handleListFundResults)
		api.GET("/funds/results/:fund_id/:result_id", s.handleGetFundResult)
		api.DELETE("/funds/results/:fund_id/:result_id", s.handleDeleteFundResult)
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// BacktestRequest 回测请求
type BacktestRequest struct {
	Symbol    string                 `json:"symbol" binding:"required"`
	StartDate string                 `json:"start_date" binding:"required"`
	EndDate   string                 `json:"end_date" binding:"required"`
	Leverage  float64                `json:"leverage"`
	Strategy  string                 `json:"strategy"`
	Params    map[string]interface{} `json:"params"`
}

// BacktestResponse 回测响应
type BacktestResponse struct {
	ID          string                 `json:"id"`
	Success     bool                   `json:"success"`
	Message     string                 `json:"message"`
	Statistics  map[string]interface{} `json:"statistics"`
	SignalCount int                    `json:"signal_count"`
	TradeCount  int                    `json:"trade_count"`
	TradingDays int                    `json:"trading_days"`
}

func (s *Server) handleGetSymbols(c *gin.Context) {
	query := c.Query("q")
	var symbols []SymbolConfig
	if query != "" {
		symbols = SearchSymbols(query)
	} else {
		symbols = GetSymbols()
	}
	c.JSON(http.StatusOK, gin.H{"symbols": symbols})
}

func (s *Server) handleGetStrategies(c *gin.Context) {
	strategies := GetStrategies()
	c.JSON(http.StatusOK, gin.H{"strategies": strategies})
}

func (s *Server) handleBacktest(c *gin.Context) {
	var req BacktestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Leverage <= 0 {
		req.Leverage = 3.0
	}

	if req.Strategy == "" {
		req.Strategy = "yinyang"
	}

	resultID := fmt.Sprintf("%s_%s_%s_%s_%.0f_%d",
		req.Symbol, req.Strategy, req.StartDate, req.EndDate, req.Leverage, time.Now().Unix())

	result, actualResultID, err := s.runBacktest(req, resultID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if actualResultID != "" {
		result.ID = actualResultID
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) runBacktest(req BacktestRequest, resultID string) (*BacktestResponse, string, error) {
	_, err := s.dataManager.GetTradeCalendar(req.StartDate, req.EndDate)
	if err != nil {
		return nil, "", fmt.Errorf("获取交易日历失败: %w", err)
	}

	factory, err := strategy.DefaultRegistry.Get(req.Strategy)
	if err != nil {
		return nil, "", fmt.Errorf("获取策略失败: %w", err)
	}

	params := make(map[string]interface{})
	if req.Params != nil {
		for k, v := range req.Params {
			params[k] = v
		}
	}

	warmupDays := factory.GetWarmupDays(params)

	var warmupStartDate string
	var backtestStartDateFormatted string
	if warmupDays > 0 {
		startDate, err := time.Parse("20060102", req.StartDate)
		if err != nil {
			return nil, "", fmt.Errorf("解析开始日期失败: %w", err)
		}

		requiredTradingDays := warmupDays + 5

		calendar, err := s.dataManager.GetTradeCalendar("20000101", req.StartDate)
		if err == nil && len(calendar) > 0 {
			warmupStartDate = calculateWarmupStartDate(calendar, req.StartDate, requiredTradingDays)
		} else {
			warmupStart := startDate.AddDate(0, 0, -requiredTradingDays*2)
			warmupStartDate = warmupStart.Format("20060102")
		}

		backtestStartDateFormatted = startDate.Format("2006-01-02")
	} else {
		warmupStartDate = req.StartDate
		backtestStartDateFormatted = ""
	}

	contractSymbols := generateContractSymbols(req.Symbol, warmupStartDate, req.EndDate)

	var allKlines []backtest.KLineWithContract
	for _, cs := range contractSymbols {
		klines, err := s.dataManager.GetFuturesKLine(cs, warmupStartDate, req.EndDate)
		if err != nil {
			continue
		}
		for _, kl := range klines {
			allKlines = append(allKlines, backtest.KLineWithContract{
				Symbol: cs,
				KLineData: backtest.KLineData{
					Date:   kl.Date,
					Open:   kl.Open,
					High:   kl.High,
					Low:    kl.Low,
					Close:  kl.Close,
					Volume: kl.Volume,
					Amount: kl.Amount,
					Hold:   kl.Hold,
					Settle: kl.Settle,
				},
			})
		}
	}

	if len(allKlines) == 0 {
		return nil, "", fmt.Errorf("未获取到任何K线数据")
	}

	identifier := data.NewDominantContractIdentifier(s.dataManager)
	dominantResult, err := identifier.Identify(req.Symbol, allKlines, warmupStartDate, req.EndDate)
	if err != nil {
		return nil, "", fmt.Errorf("识别主力合约失败: %w", err)
	}

	dominantMap := make(map[string]string, len(dominantResult))
	for t, sym := range dominantResult {
		dateStr := t.Format("2006-01-02")
		dominantMap[dateStr] = sym
	}

	sigStrategy := factory.Create(params)

	var actualLeverage float64 = 1.0
	if v, ok := params["leverage"]; ok {
		if f, ok := v.(float64); ok {
			actualLeverage = f
		}
	}
	resultID = fmt.Sprintf("%s_%s_%s_%s_%.0f_%d",
		req.Symbol, req.Strategy, req.StartDate, req.EndDate, actualLeverage, time.Now().Unix())

	rollover := factory.CreateRolloverHandler(sigStrategy)
	stateRecorder := factory.CreateStateRecorder()

	signalEngine := backtest.NewSignalEngine(allKlines, dominantMap, sigStrategy, rollover)
	signalEngine.SetStateRecorder(stateRecorder)
	signalEngine.SetWarmupDays(warmupDays, backtestStartDateFormatted)

	signals, err := signalEngine.Calculate()
	if err != nil {
		return nil, "", fmt.Errorf("计算交易信号失败: %w", err)
	}

	dominantKlines := filterDominantKlines(allKlines, dominantMap)
	portfolioEngine := backtest.NewPortfolioEngine()
	dailyRecords, positionReturns, err := portfolioEngine.Calculate(signals, dominantKlines)
	if err != nil {
		return nil, "", fmt.Errorf("计算资金收益失败: %w", err)
	}

	stats := backtest.CalculateStatistics(dailyRecords, positionReturns)

	filteredKlines := dominantKlines
	filteredDailyRecords := dailyRecords
	if warmupDays > 0 {
		filteredKlines = filterKlinesByDate(dominantKlines, req.StartDate)
		filteredDailyRecords = filterDailyRecordsByDate(dailyRecords, req.StartDate)
	}

	resultData := &ResultData{
		ID:              resultID,
		Request:         req,
		Signals:         signals,
		DailyRecords:    filteredDailyRecords,
		PositionReturns: positionReturns,
		Statistics:      stats,
		StateHistory:    stateRecorder.GetStateHistory(),
		DominantMap:     dominantMap,
		Klines:          filteredKlines,
	}

	if err := s.saveResult(resultData); err != nil {
		return nil, "", fmt.Errorf("保存结果失败: %w", err)
	}

	return &BacktestResponse{
		ID:          resultID,
		Success:     true,
		Message:     "回测完成",
		Statistics:  convertStatistics(stats),
		SignalCount: len(signals),
		TradeCount:  len(positionReturns),
		TradingDays: len(dailyRecords),
	}, resultID, nil
}

func (s *Server) handleListResults(c *gin.Context) {
	dbResults, err := db.ListBacktestResults()
	if err == nil && len(dbResults) > 0 {
		var results []map[string]string
		for _, r := range dbResults {
			id, _ := r["id"].(string)
			results = append(results, map[string]string{
				"id":   id,
				"name": id + ".json",
			})
		}
		c.JSON(http.StatusOK, gin.H{"results": results})
		return
	}

	entries, err := os.ReadDir(s.retDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var results []map[string]string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".json" {
			results = append(results, map[string]string{
				"id":   name[:len(name)-5],
				"name": name,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (s *Server) handleGetResult(c *gin.Context) {
	id := c.Param("id")
	result, err := s.loadResult(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "结果不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           result.ID,
		"request":      result.Request,
		"statistics":   convertStatistics(result.Statistics),
		"signal_count": len(result.Signals),
		"trade_count":  len(result.PositionReturns),
		"trading_days": len(result.DailyRecords),
	})
}

func (s *Server) handleDeleteResult(c *gin.Context) {
	id := c.Param("id")

	db.DeleteBacktestResult(id)

	filename := filepath.Join(s.retDir, id+".json")

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "结果不存在"})
		return
	}

	if err := os.Remove(filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

func (s *Server) handlePortfolioAnalysis(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.IDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要选择2个回测结果"})
		return
	}

	// 加载所有选中的结果
	var results []*ResultData
	for _, id := range req.IDs {
		result, err := s.loadResult(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "结果不存在: " + id})
			return
		}
		results = append(results, result)
	}

	// 检查时间段重叠
	if ok, msg := CheckTimeOverlap(results); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	// 获取共同日期范围
	commonStart, commonEnd := GetCommonDateRange(results)
	if commonStart == "" || commonEnd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法确定共同日期范围"})
		return
	}

	// 过滤到共同日期范围
	filteredResults := FilterResultsByDateRange(results, commonStart, commonEnd)

	// 计算组合
	portfolio, err := CalculatePortfolio(filteredResults)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"symbols":          portfolio.Symbols,
		"common_start":     commonStart,
		"common_end":       commonEnd,
		"daily_records":    convertPortfolioDailyRecords(portfolio.DailyRecords),
		"position_returns": convertPortfolioPositionReturns(portfolio.PositionReturns),
		"statistics":       convertPortfolioStatistics(portfolio.Statistics),
	})
}

func (s *Server) handleGetResultData(c *gin.Context) {
	id := c.Param("id")
	dataType := c.Query("type")
	if dataType == "" {
		dataType = "all"
	}

	result, err := s.loadResult(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "结果不存在"})
		return
	}

	switch dataType {
	case "daily":
		c.JSON(http.StatusOK, gin.H{"daily_records": convertDailyRecordsWithSignals(result.DailyRecords, result.Signals)})
	case "signals":
		c.JSON(http.StatusOK, gin.H{"signals": convertSignals(result.Signals)})
	case "returns":
		c.JSON(http.StatusOK, gin.H{"position_returns": convertPositionReturns(result.PositionReturns)})
	case "stats":
		c.JSON(http.StatusOK, gin.H{"statistics": convertStatistics(result.Statistics)})
	case "klines":
		c.JSON(http.StatusOK, gin.H{"klines": convertKlines(result.Klines, result.Signals)})
	default:
		c.JSON(http.StatusOK, gin.H{
			"daily_records":    convertDailyRecordsWithSignals(result.DailyRecords, result.Signals),
			"signals":          convertSignals(result.Signals),
			"position_returns": convertPositionReturns(result.PositionReturns),
			"statistics":       convertStatistics(result.Statistics),
			"klines":           convertKlines(result.Klines, result.Signals),
		})
	}
}

func convertStatistics(stats backtest.Statistics) map[string]interface{} {
	return map[string]interface{}{
		"total_return":       fmt.Sprintf("%.4f", stats.TotalReturn),
		"annual_return":      fmt.Sprintf("%.4f", stats.AnnualReturn),
		"max_drawdown":       fmt.Sprintf("%.4f", stats.MaxDrawdown),
		"max_drawdown_ratio": fmt.Sprintf("%.4f", stats.MaxDrawdownRatio),
		"win_rate":           fmt.Sprintf("%.4f", stats.WinRate),
		"profit_loss_ratio":  fmt.Sprintf("%.4f", stats.ProfitLossRatio),
		"winning_trades":     stats.WinningTrades,
		"losing_trades":      stats.LosingTrades,
		"total_trades":       stats.TotalTrades,
		"total_win":          fmt.Sprintf("%.4f", stats.TotalWin),
		"total_loss":         fmt.Sprintf("%.4f", stats.TotalLoss),
		"sharpe_ratio":       fmt.Sprintf("%.4f", stats.SharpeRatio),
		"calmar_ratio":       fmt.Sprintf("%.4f", stats.CalmarRatio),
		"trading_days":       stats.TradingDays,
		"final_value":        fmt.Sprintf("%.4f", stats.FinalValue),
	}
}

func convertDailyRecords(records []backtest.DailyRecord) []map[string]interface{} {
	if len(records) == 0 {
		return []map[string]interface{}{}
	}

	result := make([]map[string]interface{}, len(records))

	peak := records[0].TotalValue
	for i, r := range records {
		if r.TotalValue > peak {
			peak = r.TotalValue
		}

		drawdown := 0.0
		if peak > 0 {
			drawdown = (peak - r.TotalValue) / peak
		}

		result[i] = map[string]interface{}{
			"date":         r.Date,
			"total_value":  fmt.Sprintf("%.4f", r.TotalValue),
			"daily_return": fmt.Sprintf("%.6f", r.DailyReturn),
			"pnl":          fmt.Sprintf("%.4f", r.PnL),
			"drawdown":     fmt.Sprintf("%.6f", drawdown),
		}
	}
	return result
}

func convertDailyRecordsWithSignals(records []backtest.DailyRecord, signals []backtest.TradeSignal) []map[string]interface{} {
	if len(records) == 0 {
		return []map[string]interface{}{}
	}

	signalMap := make(map[string][]map[string]interface{})
	for _, s := range signals {
		signalMap[s.SignalDate] = append(signalMap[s.SignalDate], map[string]interface{}{
			"direction": s.Direction.String(),
			"type":      s.SignalType,
			"price":     fmt.Sprintf("%.2f", s.Price),
		})
	}

	result := make([]map[string]interface{}, len(records))

	peak := records[0].TotalValue
	for i, r := range records {
		if r.TotalValue > peak {
			peak = r.TotalValue
		}

		drawdown := 0.0
		if peak > 0 {
			drawdown = (peak - r.TotalValue) / peak
		}

		record := map[string]interface{}{
			"date":         r.Date,
			"total_value":  fmt.Sprintf("%.4f", r.TotalValue),
			"daily_return": fmt.Sprintf("%.6f", r.DailyReturn),
			"pnl":          fmt.Sprintf("%.4f", r.PnL),
			"drawdown":     fmt.Sprintf("%.6f", drawdown),
		}

		if signalList, exists := signalMap[r.Date]; exists {
			record["signals"] = signalList
		}

		result[i] = record
	}
	return result
}

func convertSignals(signals []backtest.TradeSignal) []map[string]interface{} {
	result := make([]map[string]interface{}, len(signals))
	for i, s := range signals {
		result[i] = map[string]interface{}{
			"date":      s.SignalDate,
			"symbol":    s.Symbol,
			"direction": s.Direction.String(),
			"price":     fmt.Sprintf("%.2f", s.Price),
			"leverage":  fmt.Sprintf("%.2f", s.Leverage),
			"type":      s.SignalType,
		}
	}
	return result
}

func convertKlines(klines []backtest.KLineWithContract, signals []backtest.TradeSignal) []map[string]interface{} {
	if len(klines) == 0 {
		return []map[string]interface{}{}
	}

	signalMap := make(map[string][]map[string]interface{})
	for _, s := range signals {
		signalMap[s.SignalDate] = append(signalMap[s.SignalDate], map[string]interface{}{
			"direction": s.Direction.String(),
			"type":      s.SignalType,
			"price":     fmt.Sprintf("%.2f", s.Price),
		})
	}

	result := make([]map[string]interface{}, len(klines))
	for i, k := range klines {
		kline := map[string]interface{}{
			"date":   k.Date,
			"symbol": k.Symbol,
			"open":   k.Open,
			"high":   k.High,
			"low":    k.Low,
			"close":  k.Close,
			"volume": k.Volume,
		}

		if signalList, exists := signalMap[k.Date]; exists {
			kline["signals"] = signalList
		}

		result[i] = kline
	}
	return result
}

func convertPositionReturns(returns []backtest.PositionReturn) []map[string]interface{} {
	result := make([]map[string]interface{}, len(returns))
	for i, r := range returns {
		result[i] = map[string]interface{}{
			"open_date":   r.OpenDate,
			"close_date":  r.CloseDate,
			"symbol":      r.Symbol,
			"direction":   r.Direction.String(),
			"open_price":  fmt.Sprintf("%.2f", r.OpenPrice),
			"close_price": fmt.Sprintf("%.2f", r.ClosePrice),
			"leverage":    fmt.Sprintf("%.2f", r.Leverage),
			"return":      fmt.Sprintf("%.6f", r.Return),
		}
	}
	return result
}

func convertPortfolioDailyRecords(records []PortfolioDailyRecord) []map[string]interface{} {
	result := make([]map[string]interface{}, len(records))
	for i, r := range records {
		components := make(map[string]string)
		for k, v := range r.Components {
			components[k] = fmt.Sprintf("%.4f", v)
		}
		result[i] = map[string]interface{}{
			"date":         r.Date,
			"total_value":  fmt.Sprintf("%.4f", r.TotalValue),
			"daily_return": fmt.Sprintf("%.6f", r.DailyReturn),
			"pnl":          fmt.Sprintf("%.4f", r.PnL),
			"components":   components,
		}
	}
	return result
}

func convertPortfolioPositionReturns(returns []PortfolioPositionReturn) []map[string]interface{} {
	result := make([]map[string]interface{}, len(returns))
	for i, r := range returns {
		result[i] = map[string]interface{}{
			"open_date":   r.OpenDate,
			"close_date":  r.CloseDate,
			"symbol":      r.Symbol,
			"direction":   r.Direction,
			"open_price":  fmt.Sprintf("%.2f", r.OpenPrice),
			"close_price": fmt.Sprintf("%.2f", r.ClosePrice),
			"leverage":    fmt.Sprintf("%.2f", r.Leverage),
			"return":      fmt.Sprintf("%.6f", r.Return),
			"weight":      fmt.Sprintf("%.4f", r.Weight),
		}
	}
	return result
}

func convertPortfolioStatistics(stats PortfolioStatistics) map[string]interface{} {
	return map[string]interface{}{
		"total_return":       fmt.Sprintf("%.4f", stats.TotalReturn),
		"annual_return":      fmt.Sprintf("%.4f", stats.AnnualReturn),
		"max_drawdown":       fmt.Sprintf("%.4f", stats.MaxDrawdown),
		"max_drawdown_ratio": fmt.Sprintf("%.4f", stats.MaxDrawdownRatio),
		"win_rate":           fmt.Sprintf("%.4f", stats.WinRate),
		"profit_loss_ratio":  fmt.Sprintf("%.4f", stats.ProfitLossRatio),
		"winning_trades":     stats.WinningTrades,
		"losing_trades":      stats.LosingTrades,
		"total_trades":       stats.TotalTrades,
		"total_win":          fmt.Sprintf("%.4f", stats.TotalWin),
		"total_loss":         fmt.Sprintf("%.4f", stats.TotalLoss),
		"sharpe_ratio":       fmt.Sprintf("%.4f", stats.SharpeRatio),
		"calmar_ratio":       fmt.Sprintf("%.4f", stats.CalmarRatio),
		"trading_days":       stats.TradingDays,
		"final_value":        fmt.Sprintf("%.4f", stats.FinalValue),
	}
}

func filterDominantKlines(allKlines []backtest.KLineWithContract, dominantMap map[string]string) []backtest.KLineWithContract {
	var result []backtest.KLineWithContract
	for _, kl := range allKlines {
		if dominant, ok := dominantMap[kl.Date]; ok && dominant == kl.Symbol {
			result = append(result, kl)
		}
	}
	return result
}

func calculateWarmupStartDate(calendar []data.TradeDate, startDate string, requiredDays int) string {
	var tradingDays []string
	for _, td := range calendar {
		if td.IsTradingDay {
			tradingDays = append(tradingDays, td.Date)
		}
	}

	startIdx := -1
	for i, date := range tradingDays {
		if date == startDate {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		startDateParsed, _ := time.Parse("20060102", startDate)
		warmupStart := startDateParsed.AddDate(0, 0, -requiredDays*2)
		return warmupStart.Format("20060102")
	}

	warmupIdx := startIdx - requiredDays
	if warmupIdx < 0 {
		warmupIdx = 0
	}

	return tradingDays[warmupIdx]
}

func filterKlinesByDate(klines []backtest.KLineWithContract, startDate string) []backtest.KLineWithContract {
	var filtered []backtest.KLineWithContract
	startDateFormatted := startDate[:4] + "-" + startDate[4:6] + "-" + startDate[6:8]
	for _, kl := range klines {
		if kl.Date >= startDateFormatted {
			filtered = append(filtered, kl)
		}
	}
	return filtered
}

func filterDailyRecordsByDate(records []backtest.DailyRecord, startDate string) []backtest.DailyRecord {
	var filtered []backtest.DailyRecord
	startDateFormatted := startDate[:4] + "-" + startDate[4:6] + "-" + startDate[6:8]
	for _, r := range records {
		if r.Date >= startDateFormatted {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func generateContractSymbols(product, startDate, endDate string) []string {
	start, err := time.Parse("20060102", startDate)
	if err != nil {
		return []string{product}
	}
	end, err := time.Parse("20060102", endDate)
	if err != nil {
		return []string{product}
	}

	limitDate := end.AddDate(1, 0, 0)
	limitYear := limitDate.Year()
	limitMonth := int(limitDate.Month())

	startYear := start.Year()
	startMonth := int(start.Month()) + 1
	if startMonth > 12 {
		startMonth = 1
		startYear++
	}

	seen := make(map[string]bool)
	var symbols []string

	year := startYear
	month := startMonth
	for {
		if year > limitYear || (year == limitYear && month > limitMonth) {
			return symbols
		}
		sym := fmt.Sprintf("%s%02d%02d", product, year%100, month)
		if !seen[sym] {
			seen[sym] = true
			symbols = append(symbols, sym)
		}
		month++
		if month > 12 {
			month = 1
			year++
		}
	}
}

func (s *Server) handleGetFunds(c *gin.Context) {
	if err := fund.LoadFundConfig(""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载基金配置失败: " + err.Error()})
		return
	}

	funds, err := fund.GetAllFundConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"funds": funds})
}

func (s *Server) handleGetFund(c *gin.Context) {
	fundID := c.Param("id")

	if err := fund.LoadFundConfig(""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载基金配置失败: " + err.Error()})
		return
	}

	fundConfig, err := fund.GetFundConfig(fundID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"fund": fundConfig})
}

func (s *Server) handleCreateFund(c *gin.Context) {
	var fundConfig fund.FundConfig
	if err := c.ShouldBindJSON(&fundConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := fund.ValidateFundConfig(&fundConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := fund.SaveFundConfig("", &fundConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存基金配置失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "fund": fundConfig})
}

func (s *Server) handleFundBacktest(c *gin.Context) {
	fmt.Println("=== handleFundBacktest called ===")
	log.Printf("[基金回测] 收到请求: Method=%s, Path=%s", c.Request.Method, c.Request.URL.Path)

	var req fund.FundBacktestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[基金回测] 请求解析失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[基金回测] 开始: fund_id=%s, start=%s, end=%s", req.FundID, req.StartDate, req.EndDate)

	if err := fund.LoadFundConfig(""); err != nil {
		log.Printf("[基金回测] 加载配置失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载基金配置失败: " + err.Error()})
		return
	}
	log.Printf("[基金回测] 配置加载成功")

	fundConfig, err := fund.GetFundConfig(req.FundID)
	if err != nil {
		log.Printf("[基金回测] 获取配置失败: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[基金回测] 配置OK: %s, 品种数=%d", fundConfig.Name, len(fundConfig.Positions))

	taskManager := fund.GetTaskManager()
	task := taskManager.CreateTask(req.FundID, fundConfig.Name, req.StartDate, req.EndDate)
	task.TotalSteps = len(fundConfig.Positions)

	go s.executeFundBacktest(task.ID, *fundConfig, req.StartDate, req.EndDate)

	log.Printf("[基金回测] 创建异步任务: taskID=%s", task.ID)

	c.JSON(http.StatusOK, gin.H{
		"async":   true,
		"task_id": task.ID,
		"message": "回测任务已创建",
	})
}

func (s *Server) executeFundBacktest(taskID string, config fund.FundConfig, startDate, endDate string) {
	taskManager := fund.GetTaskManager()

	engine := fund.NewFundEngine(s.dataManager)
	engine.SetProgressCallback(func(progress int, step string) {
		taskManager.UpdateProgress(taskID, progress, step)
	})

	log.Printf("[基金回测] 异步任务开始执行: taskID=%s", taskID)

	result, err := engine.RunBacktest(config, startDate, endDate)
	if err != nil {
		log.Printf("[基金回测] 异步任务执行失败: taskID=%s, err=%v", taskID, err)
		taskManager.FailTask(taskID, err.Error())
		return
	}

	log.Printf("[基金回测] 异步任务执行成功: taskID=%s, 交易天数=%d, resultID=%s", taskID, result.Statistics.TradingDays, result.ID)

	if err := fund.SaveFundResult(result, s.retDir); err != nil {
		log.Printf("[基金回测] 异步任务保存失败: taskID=%s, err=%v", taskID, err)
		taskManager.FailTask(taskID, "保存结果失败: "+err.Error())
		return
	}

	taskManager.CompleteTask(taskID, result.ID)
	log.Printf("[基金回测] 异步任务完成: taskID=%s", taskID)
}

func (s *Server) handleFundBacktestProgress(c *gin.Context) {
	taskID := c.Param("task_id")

	taskManager := fund.GetTaskManager()
	task, ok := taskManager.GetTask(taskID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}

	resp := gin.H{
		"task_id":      task.ID,
		"fund_id":      task.FundID,
		"fund_name":    task.FundName,
		"status":       string(task.Status),
		"progress":     task.Progress,
		"current_step": task.CurrentStep,
		"message":      task.Message,
		"total_steps":  task.TotalSteps,
	}

	if task.Status == fund.TaskStatusCompleted {
		resp["result_id"] = task.ResultID
	}

	if task.Status == fund.TaskStatusFailed {
		resp["error"] = task.Error
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleListFundResults(c *gin.Context) {
	results, err := fund.ListFundResults(s.retDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (s *Server) handleGetFundResult(c *gin.Context) {
	fundID := c.Param("fund_id")
	resultID := c.Param("result_id")

	result, err := fund.LoadFundResult(s.retDir, fundID, resultID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	positionSummaries := make(map[string]interface{})
	for symbol, pos := range result.PositionResults {
		positionSummaries[symbol] = map[string]interface{}{
			"strategy":     pos.Strategy,
			"weight":       fmt.Sprintf("%.4f", pos.Weight),
			"total_return": fmt.Sprintf("%.4f", pos.Statistics.TotalReturn),
			"win_rate":     fmt.Sprintf("%.4f", pos.Statistics.WinRate),
			"trading_days": pos.Statistics.TradingDays,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":               result.ID,
		"fund_id":          result.FundID,
		"fund_name":        result.FundName,
		"start_date":       result.StartDate,
		"end_date":         result.EndDate,
		"statistics":       convertFundStatistics(result.Statistics),
		"daily_records":    convertFundDailyRecords(result.DailyRecords),
		"position_results": positionSummaries,
	})
}

func (s *Server) handleDeleteFundResult(c *gin.Context) {
	fundID := c.Param("fund_id")
	resultID := c.Param("result_id")

	if err := fund.DeleteFundResult(s.retDir, fundID, resultID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

func convertFundStatistics(stats fund.FundStatistics) map[string]interface{} {
	return map[string]interface{}{
		"total_return":       fmt.Sprintf("%.4f", stats.TotalReturn),
		"annual_return":      fmt.Sprintf("%.4f", stats.AnnualReturn),
		"max_drawdown":       fmt.Sprintf("%.4f", stats.MaxDrawdown),
		"max_drawdown_ratio": fmt.Sprintf("%.4f", stats.MaxDrawdownRatio),
		"sharpe_ratio":       fmt.Sprintf("%.4f", stats.SharpeRatio),
		"calmar_ratio":       fmt.Sprintf("%.4f", stats.CalmarRatio),
		"win_rate":           fmt.Sprintf("%.4f", stats.WinRate),
		"trading_days":       stats.TradingDays,
		"winning_trades":     stats.WinningTrades,
		"losing_trades":      stats.LosingTrades,
		"total_trades":       stats.TotalTrades,
	}
}

func convertFundDailyRecords(records []fund.FundDailyRecord) []map[string]interface{} {
	result := make([]map[string]interface{}, len(records))
	for i, r := range records {
		components := make(map[string]string)
		for k, v := range r.Components {
			components[k] = fmt.Sprintf("%.4f", v)
		}
		result[i] = map[string]interface{}{
			"date":         r.Date,
			"total_value":  fmt.Sprintf("%.4f", r.TotalValue),
			"daily_return": fmt.Sprintf("%.6f", r.DailyReturn),
			"pnl":          fmt.Sprintf("%.4f", r.PnL),
			"components":   components,
		}
	}
	return result
}
