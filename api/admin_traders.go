package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"nofx/trader"

	"github.com/gin-gonic/gin"
)

func (s *Server) stopTraderForUser(ownerUserID, traderID string) error {
	if _, _, _, err := s.database.GetTraderConfig(ownerUserID, traderID); err != nil {
		return fmt.Errorf("交易员不存在或无访问权限")
	}

	if s.traderManager == nil {
		return fmt.Errorf("交易员不存在")
	}
	at, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		return fmt.Errorf("交易员不存在")
	}

	status := at.GetStatus()
	if isRunning, ok := status["is_running"].(bool); ok && !isRunning {
		return fmt.Errorf("交易员已停止")
	}

	at.Stop()
	if err := s.database.UpdateTraderStatus(ownerUserID, traderID, false); err != nil {
		log.Printf("⚠️  更新交易员状态失败: %v", err)
	}
	log.Printf("⏹  交易员 %s 已停止 (owner=%s)", at.GetName(), ownerUserID)
	return nil
}

func (s *Server) mergeTraderRunningStatus(traderID string, dbRunning bool) bool {
	if s.traderManager == nil {
		return dbRunning
	}
	if at, err := s.traderManager.GetTrader(traderID); err == nil {
		status := at.GetStatus()
		if running, ok := status["is_running"].(bool); ok {
			return running
		}
	}
	return dbRunning
}

func (s *Server) getAdminTraderRuntime(traderID string) (*trader.AutoTrader, error) {
	record, err := s.database.GetTraderByID(traderID)
	if err != nil {
		return nil, err
	}
	if s.traderManager == nil {
		return nil, fmt.Errorf("交易员不存在")
	}
	if err := s.traderManager.LoadUserTraders(s.database, record.UserID); err != nil {
		log.Printf("⚠️ 加载用户 %s 交易员失败: %v", record.UserID, err)
	}
	return s.traderManager.GetTrader(traderID)
}

// handleAdminListTraders 跨用户分页查询 AI 交易员
func (s *Server) handleAdminListTraders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current", c.DefaultQuery("page", "1")))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	userID := strings.TrimSpace(c.Query("user_id"))
	email := strings.TrimSpace(c.Query("email"))
	name := strings.TrimSpace(c.Query("name"))

	var isRunning *bool
	if v := strings.TrimSpace(c.Query("is_running")); v != "" {
		running := v == "true" || v == "1"
		isRunning = &running
	}

	items, total, err := s.database.ListTradersAdmin(page, pageSize, userID, email, isRunning, name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "查询失败"})
		return
	}

	data := make([]gin.H, 0, len(items))
	for _, item := range items {
		running := s.mergeTraderRunningStatus(item.ID, item.IsRunning)
		data = append(data, gin.H{
			"id":                     item.ID,
			"user_id":                item.UserID,
			"user_email":             item.UserEmail,
			"name":                   item.Name,
			"ai_model_id":            item.AIModelID,
			"ai_model_name":          item.AIModelName,
			"exchange_id":            item.ExchangeID,
			"exchange_name":          item.ExchangeName,
			"exchange_type":          item.ExchangeType,
			"initial_balance":        item.InitialBalance,
			"scan_interval_minutes":  item.ScanIntervalMinutes,
			"is_running":             running,
			"btc_eth_leverage":       item.BTCETHLeverage,
			"altcoin_leverage":       item.AltcoinLeverage,
			"trading_symbols":        item.TradingSymbols,
			"system_prompt_template": item.SystemPromptTemplate,
			"is_cross_margin":        item.IsCrossMargin,
			"use_coin_pool":          item.UseCoinPool,
			"use_oi_top":             item.UseOITop,
			"created_at":             item.CreatedAt,
			"updated_at":             item.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": data, "total": total, "success": true})
}

// handleAdminGetTrader 交易员详情（脱敏）
func (s *Server) handleAdminGetTrader(c *gin.Context) {
	traderID := c.Param("id")
	record, err := s.database.GetTraderByID(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "交易员不存在"})
		return
	}

	owner, err := s.database.GetUserByID(record.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "所属用户不存在"})
		return
	}

	traderCfg, aiModel, exchange, err := s.database.GetTraderConfig(record.UserID, traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "获取交易员配置失败"})
		return
	}

	isRunning := s.mergeTraderRunningStatus(traderID, traderCfg.IsRunning)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":                     traderCfg.ID,
			"user_id":                traderCfg.UserID,
			"user_email":             owner.Email,
			"name":                   traderCfg.Name,
			"ai_model_id":            traderCfg.AIModelID,
			"ai_model_name":          aiModel.Name,
			"ai_model_provider":      aiModel.Provider,
			"exchange_id":            traderCfg.ExchangeID,
			"exchange_name":          exchange.Name,
			"exchange_type":          exchange.Type,
			"initial_balance":        traderCfg.InitialBalance,
			"scan_interval_minutes":  traderCfg.ScanIntervalMinutes,
			"is_running":             isRunning,
			"btc_eth_leverage":       traderCfg.BTCETHLeverage,
			"altcoin_leverage":       traderCfg.AltcoinLeverage,
			"trading_symbols":        traderCfg.TradingSymbols,
			"custom_prompt":          traderCfg.CustomPrompt,
			"override_base_prompt":   traderCfg.OverrideBasePrompt,
			"system_prompt_template": traderCfg.SystemPromptTemplate,
			"is_cross_margin":        traderCfg.IsCrossMargin,
			"use_coin_pool":          traderCfg.UseCoinPool,
			"use_oi_top":             traderCfg.UseOITop,
			"created_at":             traderCfg.CreatedAt,
			"updated_at":             traderCfg.UpdatedAt,
		},
	})
}

// handleAdminGetTraderAccount 只读账户信息
func (s *Server) handleAdminGetTraderAccount(c *gin.Context) {
	traderID := c.Param("id")
	at, err := s.getAdminTraderRuntime(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "交易员未加载或不存在"})
		return
	}

	account, err := at.GetAccountInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": fmt.Sprintf("获取账户信息失败: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": account})
}

// handleAdminGetTraderPositions 只读持仓
func (s *Server) handleAdminGetTraderPositions(c *gin.Context) {
	traderID := c.Param("id")
	at, err := s.getAdminTraderRuntime(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "交易员未加载或不存在"})
		return
	}

	positions, err := at.GetPositions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": fmt.Sprintf("获取持仓失败: %v", err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": positions})
}

// handleAdminStopTrader 强制停止交易员
func (s *Server) handleAdminStopTrader(c *gin.Context) {
	traderID := c.Param("id")
	record, err := s.database.GetTraderByID(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "交易员不存在"})
		return
	}

	if s.traderManager != nil {
		if err := s.traderManager.LoadUserTraders(s.database, record.UserID); err != nil {
			log.Printf("⚠️ 加载用户交易员失败: %v", err)
		}
	}

	if err := s.stopTraderForUser(record.UserID, traderID); err != nil {
		if err.Error() == "交易员已停止" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "交易员已停止"})
}
