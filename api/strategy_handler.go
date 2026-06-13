package api

import (
	"encoding/json"
	"net/http"
	"time"

	"nofx/config"
	"nofx/indicator"
	"nofx/market"
	"nofx/strategy"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) handleListStrategies(c *gin.Context) {
	userID := c.GetString("user_id")
	records, err := s.database.GetStrategies(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"strategies": records})
}

func (s *Server) handleCreateStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	var req strategy.StrategyConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 不能为空"})
		return
	}
	if req.Symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol 不能为空"})
		return
	}
	if req.ExchangeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exchange_id 不能为空"})
		return
	}
	if req.Timeframe == "" {
		req.Timeframe = "1h"
	}
	if req.Direction == "" {
		req.Direction = "long_only"
	}

	id := uuid.New().String()
	req.ID = id
	req.UserID = userID
	configJSON, _ := json.Marshal(req)

	record := &config.StrategyRecord{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Config: string(configJSON),
		Status: strategy.StatusDraft,
	}
	if err := s.database.CreateStrategy(record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"strategy": record})
}

func (s *Server) handleGetStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	record, err := s.database.GetStrategy(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "策略不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"strategy": record})
}

func (s *Server) handleUpdateStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	existing, err := s.database.GetStrategy(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "策略不存在"})
		return
	}

	var req strategy.StrategyConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = id
	req.UserID = userID
	if req.Name == "" {
		req.Name = existing.Name
	}
	configJSON, _ := json.Marshal(req)

	record := &config.StrategyRecord{
		ID:     id,
		UserID: userID,
		Name:   req.Name,
		Config: string(configJSON),
		Status: existing.Status,
	}
	if err := s.database.UpdateStrategy(record); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"strategy": record})
}

func (s *Server) handleDeleteStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	s.strategyManager.PauseStrategy(id)
	if err := s.database.DeleteStrategy(userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (s *Server) handleValidateStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	record, err := s.database.GetStrategy(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "策略不存在"})
		return
	}

	cfg, err := s.parseStrategyRecord(record)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	signal, err := s.evaluateStrategyNow(cfg)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"signal": signal})
}

func (s *Server) handleActivateStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	record, err := s.database.GetStrategy(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "策略不存在"})
		return
	}

	cfg, err := s.parseStrategyRecord(record)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.database.UpdateStrategyStatus(userID, id, strategy.StatusActive); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := s.strategyManager.StartStrategy(userID, cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "active", "strategy_id": id})
}

func (s *Server) handlePauseStrategy(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	s.strategyManager.PauseStrategy(id)
	if err := s.database.UpdateStrategyStatus(userID, id, strategy.StatusPaused); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "paused", "strategy_id": id})
}

func (s *Server) handleStrategySignals(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	if _, err := s.database.GetStrategy(userID, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "策略不存在"})
		return
	}
	signals, err := s.database.GetStrategySignals(id, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"signals": signals})
}

func (s *Server) parseStrategyRecord(record *config.StrategyRecord) (strategy.StrategyConfig, error) {
	var cfg strategy.StrategyConfig
	if err := json.Unmarshal([]byte(record.Config), &cfg); err != nil {
		return cfg, err
	}
	cfg.ID = record.ID
	cfg.UserID = record.UserID
	if cfg.Name == "" {
		cfg.Name = record.Name
	}
	return cfg, nil
}

func (s *Server) evaluateStrategyNow(cfg strategy.StrategyConfig) (strategy.Signal, error) {
	symbol := market.Normalize(cfg.Symbol)
	client := market.NewAPIClient()
	mkKlines, err := client.GetKlines(symbol, cfg.Timeframe, 200)
	if err != nil {
		return strategy.Signal{}, err
	}
	ik := make([]indicator.Kline, len(mkKlines))
	for i, k := range mkKlines {
		ik[i] = indicator.Kline{OpenTime: k.OpenTime, Open: k.Open, High: k.High, Low: k.Low, Close: k.Close, Volume: k.Volume}
	}
	engine := strategy.NewEngine()
	signal := engine.Evaluate(cfg, ik)
	signal.Timestamp = time.Now()
	return signal, nil
}

// Gateway handlers（API Key 认证，复用同一逻辑）

func (s *Server) handleGatewayListStrategies(c *gin.Context) {
	s.handleListStrategies(c)
}

func (s *Server) handleGatewayCreateStrategy(c *gin.Context) {
	s.handleCreateStrategy(c)
}

func (s *Server) handleGatewayGetStrategy(c *gin.Context) {
	s.handleGetStrategy(c)
}

func (s *Server) handleGatewayUpdateStrategy(c *gin.Context) {
	s.handleUpdateStrategy(c)
}

func (s *Server) handleGatewayDeleteStrategy(c *gin.Context) {
	s.handleDeleteStrategy(c)
}

func (s *Server) handleGatewayValidateStrategy(c *gin.Context) {
	s.handleValidateStrategy(c)
}

func (s *Server) handleGatewayActivateStrategy(c *gin.Context) {
	s.handleActivateStrategy(c)
}

func (s *Server) handleGatewayPauseStrategy(c *gin.Context) {
	s.handlePauseStrategy(c)
}

func (s *Server) handleGatewayStrategySignals(c *gin.Context) {
	s.handleStrategySignals(c)
}
