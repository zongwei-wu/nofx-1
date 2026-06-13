package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"nofx/backtest"
	"nofx/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type backtestRequest struct {
	Symbol         string  `json:"symbol"`
	Timeframe      string  `json:"timeframe"`
	StartTime      string  `json:"start_time"`
	EndTime        string  `json:"end_time"`
	InitialCapital float64 `json:"initial_capital"`
}

func (s *Server) handleStartBacktest(c *gin.Context) {
	userID := c.GetString("user_id")
	strategyID := c.Param("id")

	record, err := s.database.GetStrategy(userID, strategyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "策略不存在"})
		return
	}

	var req backtestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空 body，使用策略默认参数
		req = backtestRequest{}
	}

	cfg, err := s.parseStrategyRecord(record)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	symbol := req.Symbol
	if symbol == "" {
		symbol = cfg.Symbol
	}
	timeframe := req.Timeframe
	if timeframe == "" {
		timeframe = cfg.Timeframe
	}
	if timeframe == "" {
		timeframe = "1h"
	}

	btID := uuid.New().String()
	btRecord := &config.BacktestRecord{
		ID:             btID,
		UserID:         userID,
		StrategyID:     strategyID,
		Symbol:         symbol,
		Timeframe:      timeframe,
		StartTime:      time.Now().Add(-30 * 24 * time.Hour),
		EndTime:        time.Now(),
		InitialCapital: req.InitialCapital,
		Status:         "pending",
	}
	if req.InitialCapital <= 0 {
		btRecord.InitialCapital = 1000
	}
	if req.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			btRecord.StartTime = t
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			btRecord.EndTime = t
		}
	}

	if err := s.database.CreateBacktest(btRecord); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	go s.runBacktestAsync(btID, btRecord)

	c.JSON(http.StatusAccepted, gin.H{"backtest_id": btID, "status": "pending"})
}

func (s *Server) runBacktestAsync(btID string, btRecord *config.BacktestRecord) {
	_ = s.database.UpdateBacktestResult(btID, "running", "", "")

	record, err := s.database.GetStrategy(btRecord.UserID, btRecord.StrategyID)
	if err != nil {
		s.database.UpdateBacktestResult(btID, "failed", "", err.Error())
		return
	}
	parsed, err := s.parseStrategyRecord(record)
	if err != nil {
		s.database.UpdateBacktestResult(btID, "failed", "", err.Error())
		return
	}

	loader := backtest.NewDataLoader()
	klines, err := loader.LoadKlines(btRecord.Symbol, btRecord.Timeframe, 500)
	if err != nil {
		s.database.UpdateBacktestResult(btID, "failed", "", err.Error())
		return
	}

	btCfg := backtest.BacktestConfig{
		Symbol:         btRecord.Symbol,
		Timeframe:      btRecord.Timeframe,
		StartTime:      btRecord.StartTime,
		EndTime:        btRecord.EndTime,
		InitialCapital: btRecord.InitialCapital,
	}
	engine := backtest.NewEngine(btCfg, parsed)
	result := engine.Run(klines)
	resultJSON, err := result.ToJSON()
	if err != nil {
		s.database.UpdateBacktestResult(btID, "failed", "", err.Error())
		return
	}
	s.database.UpdateBacktestResult(btID, "done", resultJSON, "")
	log.Printf("[BACKTEST] %s 完成: 收益 %.2f%%", btID, result.Metrics.TotalReturnPct)
}

func (s *Server) handleListBacktests(c *gin.Context) {
	userID := c.GetString("user_id")
	records, err := s.database.GetBacktests(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"backtests": records})
}

func (s *Server) handleGetBacktest(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	record, err := s.database.GetBacktest(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "回测不存在"})
		return
	}

	resp := gin.H{
		"id":              record.ID,
		"strategy_id":     record.StrategyID,
		"symbol":          record.Symbol,
		"timeframe":       record.Timeframe,
		"status":          record.Status,
		"error":           record.Error,
		"initial_capital": record.InitialCapital,
		"created_at":      record.CreatedAt,
		"completed_at":    record.CompletedAt,
	}
	if record.Result != "" {
		var result interface{}
		if err := json.Unmarshal([]byte(record.Result), &result); err == nil {
			resp["result"] = result
		}
	}
	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleGatewayStartBacktest(c *gin.Context) {
	s.handleStartBacktest(c)
}

func (s *Server) handleGatewayListBacktests(c *gin.Context) {
	s.handleListBacktests(c)
}

func (s *Server) handleGatewayGetBacktest(c *gin.Context) {
	s.handleGetBacktest(c)
}
