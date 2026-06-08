package api

import (
	"net/http"
	"strings"

	"nofx/hermes"
	"nofx/logger"

	"github.com/gin-gonic/gin"
)

type hermesSettingsUpdateRequest struct {
	AutonomousEnabled    *bool    `json:"autonomous_enabled"`
	ExchangeID           string   `json:"exchange_id"`
	AIModelID            string   `json:"ai_model_id"`
	ScanIntervalMinutes  *int     `json:"scan_interval_minutes"`
	SystemPromptTemplate string   `json:"system_prompt_template"`
	BTCETHLeverage       *int     `json:"btc_eth_leverage"`
	AltcoinLeverage      *int     `json:"altcoin_leverage"`
	TradingCoins         []string `json:"trading_coins"`
	InitialBalance       *float64 `json:"initial_balance"`
}

func (s *Server) handleGetHermesSettings(c *gin.Context) {
	userID := c.GetString("user_id")
	settings, err := s.database.GetHermesSettings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	exchanges, _ := s.database.GetExchanges(userID)
	models, _ := s.database.GetAIModels(userID)
	readyErr := hermes.ValidateReadyToStart(settings, exchanges, models)

	status := map[string]interface{}{}
	if runner, ok := s.hermesManager.GetRunner(userID); ok {
		status = runner.Status()
	} else {
		status = map[string]interface{}{
			"user_id":    userID,
			"is_running": settings.RunnerEnabled,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"settings":       settings,
		"ready_to_start": readyErr == nil,
		"ready_error":    errString(readyErr),
		"runner_status":  status,
	})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *Server) handlePutHermesSettings(c *gin.Context) {
	userID := c.GetString("user_id")
	var req hermesSettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings, err := s.database.GetHermesSettings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.AutonomousEnabled != nil {
		settings.AutonomousEnabled = *req.AutonomousEnabled
	}
	if strings.TrimSpace(req.ExchangeID) != "" {
		settings.ExchangeID = strings.TrimSpace(req.ExchangeID)
	}
	if strings.TrimSpace(req.AIModelID) != "" {
		settings.AIModelID = strings.TrimSpace(req.AIModelID)
	}
	if req.ScanIntervalMinutes != nil && *req.ScanIntervalMinutes > 0 {
		settings.ScanIntervalMinutes = *req.ScanIntervalMinutes
	}
	if strings.TrimSpace(req.SystemPromptTemplate) != "" {
		settings.SystemPromptTemplate = strings.TrimSpace(req.SystemPromptTemplate)
	}
	if req.BTCETHLeverage != nil && *req.BTCETHLeverage > 0 {
		settings.BTCETHLeverage = *req.BTCETHLeverage
	}
	if req.AltcoinLeverage != nil && *req.AltcoinLeverage > 0 {
		settings.AltcoinLeverage = *req.AltcoinLeverage
	}
	if req.TradingCoins != nil {
		settings.TradingCoins = req.TradingCoins
	}
	if req.InitialBalance != nil && *req.InitialBalance > 0 {
		settings.InitialBalance = *req.InitialBalance
	}

	if err := s.database.UpsertHermesSettings(settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"settings": settings, "message": "Hermes 设置已保存"})
}

func (s *Server) handleHermesStart(c *gin.Context) {
	userID := c.GetString("user_id")
	settings, err := s.database.GetHermesSettings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	exchanges, err := s.database.GetExchanges(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	models, err := s.database.GetAIModels(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := hermes.ValidateReadyToStart(settings, exchanges, models); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.hermesManager.Start(s.database, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	runner, _ := s.hermesManager.GetRunner(userID)
	c.JSON(http.StatusOK, gin.H{"message": "Hermes Runner 已启动", "status": runner.Status()})
}

func (s *Server) handleHermesStop(c *gin.Context) {
	userID := c.GetString("user_id")
	if err := s.hermesManager.Stop(s.database, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Hermes Runner 已停止"})
}

func (s *Server) handleHermesStatus(c *gin.Context) {
	userID := c.GetString("user_id")
	settings, _ := s.database.GetHermesSettings(userID)
	if runner, ok := s.hermesManager.GetRunner(userID); ok {
		c.JSON(http.StatusOK, gin.H{"settings": settings, "status": runner.Status()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"settings": settings,
		"status": map[string]interface{}{
			"user_id":    userID,
			"is_running": false,
		},
	})
}

func (s *Server) handleHermesDecisions(c *gin.Context) {
	userID := c.GetString("user_id")
	runner, ok := s.hermesManager.GetRunner(userID)
	if !ok {
		logDir := "decision_logs/hermes/" + userID
		dl := logger.NewDecisionLogger(logDir)
		records, err := dl.GetLatestRecords(30)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"records": records})
		return
	}
	records, err := runner.GetRecentDecisions(30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"records": records})
}

func (s *Server) handleHermesRequireSetup(c *gin.Context) {
	userID := c.GetString("user_id")
	exchanges, _ := s.database.GetExchanges(userID)
	settings, _ := s.database.GetHermesSettings(userID)
	models, _ := s.database.GetAIModels(userID)

	missing := []string{}
	if settings == nil || strings.TrimSpace(settings.ExchangeID) == "" {
		missing = append(missing, "exchange_id")
	} else if err := hermes.ValidateReadyToStart(settings, exchanges, models); err != nil {
		missing = append(missing, err.Error())
	}
	if settings != nil && !settings.AutonomousEnabled {
		missing = append(missing, "autonomous_enabled（需用户授权半自动交易）")
	}

	c.JSON(http.StatusOK, gin.H{
		"configured": len(missing) == 0,
		"missing":    missing,
		"settings":   settings,
	})
}
