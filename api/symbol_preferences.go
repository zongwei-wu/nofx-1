package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"nofx/market"

	"github.com/gin-gonic/gin"
)

func normalizeSymbolUSDT(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return ""
	}
	return market.Normalize(symbol)
}

func (s *Server) getDefaultCoinsCatalog() []string {
	defaultCoinsStr, _ := s.database.GetSystemConfig("default_coins")
	var defaultCoins []string
	if defaultCoinsStr != "" {
		_ = json.Unmarshal([]byte(defaultCoinsStr), &defaultCoins)
	}
	if len(defaultCoins) == 0 {
		defaultCoins = []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT", "XRPUSDT", "DOGEUSDT", "ADAUSDT", "HYPEUSDT"}
	}
	seen := make(map[string]bool)
	var out []string
	for _, sym := range defaultCoins {
		n := normalizeSymbolUSDT(sym)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

func mergeSymbolCatalog(defaultCoins, userSymbols []string) []string {
	seen := make(map[string]bool)
	var out []string
	appendUnique := func(list []string) {
		for _, sym := range list {
			n := normalizeSymbolUSDT(sym)
			if n == "" || seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
	}
	appendUnique(userSymbols)
	appendUnique(defaultCoins)
	return out
}

func (s *Server) aggregateUserSymbolNotional(userID string) map[string]float64 {
	result := make(map[string]float64)
	traders, err := s.database.GetTraders(userID)
	if err != nil {
		return result
	}
	for _, tr := range traders {
		at, err := s.traderManager.GetTrader(tr.ID)
		if err != nil {
			continue
		}
		positions, err := at.GetPositions()
		if err != nil {
			continue
		}
		for _, pos := range positions {
			sym, _ := pos["symbol"].(string)
			sym = normalizeSymbolUSDT(sym)
			if sym == "" {
				continue
			}
			markPrice, _ := pos["mark_price"].(float64)
			if markPrice == 0 {
				markPrice, _ = pos["markPrice"].(float64)
			}
			qty, _ := pos["quantity"].(float64)
			if qty == 0 {
				qty, _ = pos["positionAmt"].(float64)
			}
			if qty < 0 {
				qty = -qty
			}
			if markPrice > 0 && qty > 0 {
				result[sym] += markPrice * qty
			}
		}
	}
	return result
}

// handleGetSymbolPreferences 获取用户币种排序偏好
// GET /api/symbol-preferences
func (s *Server) handleGetSymbolPreferences(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	prefs, err := s.database.GetUserSymbolPreferences(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	normalized := make([]string, 0, len(prefs.Symbols))
	seen := make(map[string]bool)
	for _, sym := range prefs.Symbols {
		n := normalizeSymbolUSDT(sym)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		normalized = append(normalized, n)
	}
	c.JSON(http.StatusOK, gin.H{
		"symbols":          normalized,
		"use_custom_order": prefs.UseCustomOrder,
		"catalog":          mergeSymbolCatalog(s.getDefaultCoinsCatalog(), normalized),
	})
}

type symbolPreferencesRequest struct {
	Symbols        []string `json:"symbols"`
	UseCustomOrder bool     `json:"use_custom_order"`
}

// handlePutSymbolPreferences 保存用户币种排序偏好
// PUT /api/symbol-preferences
func (s *Server) handlePutSymbolPreferences(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	var req symbolPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求体"})
		return
	}
	normalized := make([]string, 0, len(req.Symbols))
	seen := make(map[string]bool)
	for _, sym := range req.Symbols {
		n := normalizeSymbolUSDT(sym)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		normalized = append(normalized, n)
	}
	if err := s.database.UpsertUserSymbolPreferences(userID, normalized, req.UseCustomOrder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"symbols":          normalized,
		"use_custom_order": req.UseCustomOrder,
		"catalog":          mergeSymbolCatalog(s.getDefaultCoinsCatalog(), normalized),
	})
}

// handleResetSymbolPreferences 恢复默认排序（按持仓价值）
// POST /api/symbol-preferences/reset
func (s *Server) handleResetSymbolPreferences(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	prefs, err := s.database.GetUserSymbolPreferences(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := s.database.UpsertUserSymbolPreferences(userID, prefs.Symbols, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"symbols":          prefs.Symbols,
		"use_custom_order": false,
		"catalog":          mergeSymbolCatalog(s.getDefaultCoinsCatalog(), prefs.Symbols),
	})
}

// handleGetSymbolValues 聚合用户各币种持仓名义价值
// GET /api/symbol-values
func (s *Server) handleGetSymbolValues(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"values": s.aggregateUserSymbolNotional(userID),
	})
}
