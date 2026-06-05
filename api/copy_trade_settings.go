package api

import (
	"errors"
	"fmt"
	"net/http"

	"nofx/config"

	"github.com/gin-gonic/gin"
)

type copyTradeAIResult struct {
	AICfg        *config.AIModelConfig
	AITraderID   string
	AITraderName string
	FallbackUsed bool
}

func (s *Server) resolveCopyTradeAI(userID string) (*copyTradeAIResult, error) {
	settings, err := s.database.GetCopyTradeSettings(userID)
	if err != nil {
		return nil, err
	}

	if settings.AITraderID != "" {
		trader, aiModel, _, err := s.database.GetTraderConfig(userID, settings.AITraderID)
		if err == nil && aiModel != nil && aiModel.Enabled {
			return &copyTradeAIResult{
				AICfg:        aiModel,
				AITraderID:   trader.ID,
				AITraderName: trader.Name,
				FallbackUsed: false,
			}, nil
		}
	}

	aiModels, err := s.database.GetAIModels(userID)
	if err != nil {
		return nil, err
	}
	for _, m := range aiModels {
		if m.Enabled {
			return &copyTradeAIResult{
				AICfg:        m,
				AITraderID:   "",
				AITraderName: "",
				FallbackUsed: true,
			}, nil
		}
	}

	return nil, errors.New("无可用AI模型，请在跟单管理中选择AI交易员")
}

func (s *Server) handleGetCopyTradeSettings(c *gin.Context) {
	userID := c.GetString("user_id")

	settings, err := s.database.GetCopyTradeSettings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	resp := gin.H{
		"ai_trader_id":   settings.AITraderID,
		"ai_trader_name": "",
		"ai_model_name":  "",
		"fallback_used":  false,
	}

	if settings.AITraderID != "" {
		trader, aiModel, _, err := s.database.GetTraderConfig(userID, settings.AITraderID)
		if err == nil {
			resp["ai_trader_name"] = trader.Name
			if aiModel != nil {
				resp["ai_model_name"] = aiModel.Name
			}
		}
	} else {
		aiResult, err := s.resolveCopyTradeAI(userID)
		if err == nil && aiResult.FallbackUsed {
			resp["fallback_used"] = true
			if aiResult.AICfg != nil {
				resp["ai_model_name"] = aiResult.AICfg.Name
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) handleUpdateCopyTradeSettings(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		AITraderID string `json:"ai_trader_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	if req.AITraderID != "" {
		_, _, _, err := s.database.GetTraderConfig(userID, req.AITraderID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("AI交易员不存在: %s", req.AITraderID)})
			return
		}
	}

	if err := s.database.UpsertCopyTradeSettings(userID, req.AITraderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功", "ai_trader_id": req.AITraderID})
}
