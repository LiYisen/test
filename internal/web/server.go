package web

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"futures-backtest/internal/backtest"
	"futures-backtest/internal/data"
	"futures-backtest/internal/strategy"
	"futures-backtest/pkg/pyexec"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
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
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

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
		c.HTML(http.StatusOK, "portfolio.html", nil)
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

	result, err := s.runBacktest(req, resultID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) runBacktest(req BacktestRequest, resultID string) (*BacktestResponse, error) {
	_, err := s.dataManager.GetTradeCalendar(req.StartDate, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("获取交易日历失败: %w", err)
	}

	factory, err := strategy.DefaultRegistry.Get(req.Strategy)
	if err != nil {
		return nil, fmt.Errorf("获取策略失败: %w", err)
	}

	params := make(map[string]interface{})
	if req.Params != nil {
		for k, v := range req.Params {
			params[k] = v
		}
	}
	if _, ok := params["leverage"]; !ok {
		params["leverage"] = req.Leverage
	}

	warmupDays := factory.GetWarmupDays(params)

	var warmupStartDate string
	var backtestStartDateFormatted string
	if warmupDays > 0 {
		startDate, err := time.Parse("20060102", req.StartDate)
		if err != nil {
			return nil, fmt.Errorf("解析开始日期失败: %w", err)
		}

		warmupStart := startDate.AddDate(0, 0, -warmupDays*2)
		warmupStartDate = warmupStart.Format("20060102")
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
		return nil, fmt.Errorf("未获取到任何K线数据")
	}

	identifier := data.NewDominantContractIdentifier(s.dataManager)
	dominantResult, err := identifier.Identify(req.Symbol, allKlines, warmupStartDate, req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("识别主力合约失败: %w", err)
	}

	dominantMap := make(map[string]string, len(dominantResult))
	for t, sym := range dominantResult {
		dateStr := t.Format("2006-01-02")
		dominantMap[dateStr] = sym
	}

	sigStrategy := factory.Create(params)

	rollover := factory.CreateRolloverHandler(sigStrategy)
	stateRecorder := factory.CreateStateRecorder()

	signalEngine := backtest.NewSignalEngine(allKlines, dominantMap, sigStrategy, rollover)
	signalEngine.SetStateRecorder(stateRecorder)
	signalEngine.SetWarmupDays(warmupDays, backtestStartDateFormatted)

	signals, err := signalEngine.Calculate()
	if err != nil {
		return nil, fmt.Errorf("计算交易信号失败: %w", err)
	}

	dominantKlines := filterDominantKlines(allKlines, dominantMap)
	portfolioEngine := backtest.NewPortfolioEngine()
	dailyRecords, positionReturns, err := portfolioEngine.Calculate(signals, dominantKlines)
	if err != nil {
		return nil, fmt.Errorf("计算资金收益失败: %w", err)
	}

	stats := backtest.CalculateStatistics(dailyRecords, positionReturns)

	resultData := &ResultData{
		ID:              resultID,
		Request:         req,
		Signals:         signals,
		DailyRecords:    dailyRecords,
		PositionReturns: positionReturns,
		Statistics:      stats,
		StateHistory:    stateRecorder.GetStateHistory(),
		DominantMap:     dominantMap,
		Klines:          dominantKlines,
	}

	if err := s.saveResult(resultData); err != nil {
		return nil, fmt.Errorf("保存结果失败: %w", err)
	}

	return &BacktestResponse{
		ID:          resultID,
		Success:     true,
		Message:     "回测完成",
		Statistics:  convertStatistics(stats),
		SignalCount: len(signals),
		TradeCount:  len(positionReturns),
		TradingDays: len(dailyRecords),
	}, nil
}

func (s *Server) handleListResults(c *gin.Context) {
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
		"total_return":       stats.TotalReturn.StringFixed(4),
		"annual_return":      stats.AnnualReturn.StringFixed(4),
		"max_drawdown":       stats.MaxDrawdown.StringFixed(4),
		"max_drawdown_ratio": stats.MaxDrawdownRatio.StringFixed(4),
		"win_rate":           stats.WinRate.StringFixed(4),
		"profit_loss_ratio":  stats.ProfitLossRatio.StringFixed(4),
		"winning_trades":     stats.WinningTrades,
		"losing_trades":      stats.LosingTrades,
		"total_trades":       stats.TotalTrades,
		"total_win":          stats.TotalWin.StringFixed(4),
		"total_loss":         stats.TotalLoss.StringFixed(4),
		"sharpe_ratio":       stats.SharpeRatio.StringFixed(4),
		"calmar_ratio":       stats.CalmarRatio.StringFixed(4),
		"trading_days":       stats.TradingDays,
		"final_value":        stats.FinalValue.StringFixed(4),
	}
}

func convertDailyRecords(records []backtest.DailyRecord) []map[string]interface{} {
	if len(records) == 0 {
		return []map[string]interface{}{}
	}

	result := make([]map[string]interface{}, len(records))

	peak := records[0].TotalValue
	for i, r := range records {
		if r.TotalValue.GreaterThan(peak) {
			peak = r.TotalValue
		}

		drawdown := decimal.Zero
		if peak.GreaterThan(decimal.Zero) {
			drawdown = peak.Sub(r.TotalValue).Div(peak)
		}

		result[i] = map[string]interface{}{
			"date":         r.Date,
			"total_value":  r.TotalValue.StringFixed(4),
			"daily_return": r.DailyReturn.StringFixed(6),
			"pnl":          r.PnL.StringFixed(4),
			"drawdown":     drawdown.StringFixed(6),
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
			"price":     s.Price.StringFixed(2),
		})
	}

	result := make([]map[string]interface{}, len(records))

	peak := records[0].TotalValue
	for i, r := range records {
		if r.TotalValue.GreaterThan(peak) {
			peak = r.TotalValue
		}

		drawdown := decimal.Zero
		if peak.GreaterThan(decimal.Zero) {
			drawdown = peak.Sub(r.TotalValue).Div(peak)
		}

		record := map[string]interface{}{
			"date":         r.Date,
			"total_value":  r.TotalValue.StringFixed(4),
			"daily_return": r.DailyReturn.StringFixed(6),
			"pnl":          r.PnL.StringFixed(4),
			"drawdown":     drawdown.StringFixed(6),
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
			"price":     s.Price.StringFixed(2),
			"leverage":  s.Leverage.StringFixed(2),
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
			"price":     s.Price.StringFixed(2),
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
			"open_price":  r.OpenPrice.StringFixed(2),
			"close_price": r.ClosePrice.StringFixed(2),
			"leverage":    r.Leverage.StringFixed(2),
			"return":      r.Return.StringFixed(6),
		}
	}
	return result
}

func convertPortfolioDailyRecords(records []PortfolioDailyRecord) []map[string]interface{} {
	result := make([]map[string]interface{}, len(records))
	for i, r := range records {
		components := make(map[string]string)
		for k, v := range r.Components {
			components[k] = v.StringFixed(4)
		}
		result[i] = map[string]interface{}{
			"date":         r.Date,
			"total_value":  r.TotalValue.StringFixed(4),
			"daily_return": r.DailyReturn.StringFixed(6),
			"pnl":          r.PnL.StringFixed(4),
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
			"open_price":  r.OpenPrice.StringFixed(2),
			"close_price": r.ClosePrice.StringFixed(2),
			"leverage":    r.Leverage.StringFixed(2),
			"return":      r.Return.StringFixed(6),
			"weight":      r.Weight.StringFixed(4),
		}
	}
	return result
}

func convertPortfolioStatistics(stats PortfolioStatistics) map[string]interface{} {
	return map[string]interface{}{
		"total_return":       stats.TotalReturn.StringFixed(4),
		"annual_return":      stats.AnnualReturn.StringFixed(4),
		"max_drawdown":       stats.MaxDrawdown.StringFixed(4),
		"max_drawdown_ratio": stats.MaxDrawdownRatio.StringFixed(4),
		"win_rate":           stats.WinRate.StringFixed(4),
		"profit_loss_ratio":  stats.ProfitLossRatio.StringFixed(4),
		"winning_trades":     stats.WinningTrades,
		"losing_trades":      stats.LosingTrades,
		"total_trades":       stats.TotalTrades,
		"total_win":          stats.TotalWin.StringFixed(4),
		"total_loss":         stats.TotalLoss.StringFixed(4),
		"sharpe_ratio":       stats.SharpeRatio.StringFixed(4),
		"calmar_ratio":       stats.CalmarRatio.StringFixed(4),
		"trading_days":       stats.TradingDays,
		"final_value":        stats.FinalValue.StringFixed(4),
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
