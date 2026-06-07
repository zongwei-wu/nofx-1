package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func mapFuturesPositionsForAPI(raw []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(raw))
	for _, pos := range raw {
		symbol, _ := pos["symbol"].(string)
		if symbol == "" {
			continue
		}
		side, _ := pos["side"].(string)
		entryPrice, _ := pos["entryPrice"].(float64)
		markPrice, _ := pos["markPrice"].(float64)
		quantity, _ := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity
		}
		if quantity == 0 {
			continue
		}
		unrealizedPnl, _ := pos["unRealizedProfit"].(float64)
		liquidationPrice, _ := pos["liquidationPrice"].(float64)
		leverage := 10
		if lev, ok := pos["leverage"].(float64); ok && lev > 0 {
			leverage = int(lev)
		}
		marginUsed := (quantity * markPrice) / float64(leverage)
		pnlPct := 0.0
		if marginUsed > 0 {
			pnlPct = (unrealizedPnl / marginUsed) * 100
		}
		out = append(out, map[string]interface{}{
			"symbol":             symbol,
			"side":               strings.ToLower(side),
			"entry_price":        entryPrice,
			"mark_price":         markPrice,
			"quantity":           quantity,
			"leverage":           leverage,
			"unrealized_pnl":     unrealizedPnl,
			"unrealized_pnl_pct": pnlPct,
			"liquidation_price":  liquidationPrice,
			"margin_used":        marginUsed,
		})
	}
	return out
}

// handleGetCopyTradeExchangePositions 跟单账户在交易所的实时持仓（与 AI 看板 positions API 口径一致）
func (s *Server) handleGetCopyTradeExchangePositions(c *gin.Context) {
	userID := c.GetString("user_id")
	exchangeCfg, err := s.getExecutionExchangeForUser(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fTrader, err := createTraderForExchange(userID, exchangeCfg)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	raw, err := fTrader.GetPositions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, mapFuturesPositionsForAPI(raw))
}

func (s *Server) copyTradeExchangePositionOverlays(userID, symbolFilter string) ([]map[string]interface{}, error) {
	exchangeCfg, err := s.getExecutionExchangeForUser(userID)
	if err != nil {
		return nil, err
	}
	fTrader, err := createTraderForExchange(userID, exchangeCfg)
	if err != nil {
		return nil, err
	}
	raw, err := fTrader.GetPositions()
	if err != nil {
		return nil, err
	}

	out := make([]map[string]interface{}, 0)
	for _, p := range mapFuturesPositionsForAPI(raw) {
		sym, _ := p["symbol"].(string)
		side, _ := p["side"].(string)
		entryPrice, _ := p["entry_price"].(float64)
		unrealizedPnl, _ := p["unrealized_pnl"].(float64)
		qty, _ := p["quantity"].(float64)
		sym = normalizeTradeSymbol(sym)
		if sym == "" || entryPrice <= 0 {
			continue
		}
		if symbolFilter != "" && sym != normalizeTradeSymbol(symbolFilter) {
			continue
		}
		out = append(out, map[string]interface{}{
			"symbol":         sym,
			"position_side":  side,
			"entry_price":    entryPrice,
			"unrealized_pnl": unrealizedPnl,
			"qty":            qty,
		})
	}
	return out, nil
}
