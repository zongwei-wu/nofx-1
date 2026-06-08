package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"nofx/hermes"
	"nofx/market"
	"nofx/trader"

	"github.com/gin-gonic/gin"
)

type hermesTradeRequest struct {
	ExchangeID string  `json:"exchange_id" binding:"required"`
	Action     string  `json:"action" binding:"required"`
	Symbol     string  `json:"symbol" binding:"required"`
	Quantity   float64 `json:"quantity"`
	Leverage   int     `json:"leverage"`
	Confirmed  bool    `json:"confirmed"`
}

type hermesLeverageRequest struct {
	ExchangeID string `json:"exchange_id" binding:"required"`
	Symbol     string `json:"symbol" binding:"required"`
	Leverage   int    `json:"leverage" binding:"required"`
}

type hermesStopOrderRequest struct {
	ExchangeID   string  `json:"exchange_id" binding:"required"`
	Symbol       string  `json:"symbol" binding:"required"`
	PositionSide string  `json:"position_side" binding:"required"`
	Quantity     float64 `json:"quantity" binding:"required"`
	StopPrice    float64 `json:"stop_price" binding:"required"`
}

func normalizeHermesSymbol(symbol string) string {
	return hermes.NormalizeSymbol(symbol)
}

func klinesToResponse(symbol, interval, dataSource string, klines []market.Kline) []klinePointResponse {
	out := make([]klinePointResponse, 0, len(klines))
	for _, k := range klines {
		ts := k.OpenTime / 1000
		if k.OpenTime < 1e12 {
			ts = k.OpenTime
		}
		out = append(out, klinePointResponse{
			Time:  ts,
			Open:  k.Open,
			High:  k.High,
			Low:   k.Low,
			Close: k.Close,
		})
	}
	return out
}

func (s *Server) handleHermesKlines(c *gin.Context) {
	exchangeID := strings.TrimSpace(c.Query("exchange_id"))
	if exchangeID == "" {
		exchangeID = "binance"
	}

	symbol := normalizeHermesSymbol(c.Query("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 symbol 参数"})
		return
	}

	interval := c.DefaultQuery("interval", "1h")
	allowed := map[string]bool{
		"1m": true, "3m": true, "5m": true, "15m": true, "30m": true,
		"1h": true, "2h": true, "4h": true, "1d": true,
	}
	if !allowed[interval] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的 interval"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}

	var klines []market.Kline
	var err error
	var dataSource string

	switch exchangeID {
	case "binance":
		client := market.NewAPIClient()
		klines, err = client.GetKlines(symbol, interval, limit)
		dataSource = "binance"
	case "okx":
		client := market.NewOKXKlineClient()
		klines, err = client.GetKlines(symbol, interval, limit)
		dataSource = "okx"
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("交易所 %s 暂不支持 K 线查询（仅 binance/okx）", exchangeID),
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "获取K线失败", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exchange_id": exchangeID,
		"symbol":      symbol,
		"interval":    interval,
		"data_source": dataSource,
		"klines":      klinesToResponse(symbol, interval, dataSource, klines),
	})
}

func (s *Server) handleHermesBalance(c *gin.Context) {
	userID := c.GetString("user_id")
	exchangeID := strings.TrimSpace(c.Query("exchange_id"))
	if exchangeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 exchange_id 参数"})
		return
	}

	t, ex, err := s.getHermesTrader(userID, exchangeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	balance, err := t.GetBalance()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "获取余额失败", "detail": err.Error()})
		return
	}

	snapshot := extractBalanceSnapshot(balance)
	c.JSON(http.StatusOK, gin.H{
		"exchange_id":       ex.ID,
		"testnet":           ex.Testnet,
		"balance":           balance,
		"total_equity":      snapshot.TotalEquity,
		"available_balance": snapshot.AvailableBalance,
	})
}

func (s *Server) handleHermesPositions(c *gin.Context) {
	userID := c.GetString("user_id")
	exchangeID := strings.TrimSpace(c.Query("exchange_id"))
	if exchangeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 exchange_id 参数"})
		return
	}

	t, ex, err := s.getHermesTrader(userID, exchangeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	positions, err := t.GetPositions()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "获取持仓失败", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exchange_id": ex.ID,
		"positions":   positions,
	})
}

func (s *Server) handleHermesMarketPrice(c *gin.Context) {
	userID := c.GetString("user_id")
	exchangeID := strings.TrimSpace(c.Query("exchange_id"))
	if exchangeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 exchange_id 参数"})
		return
	}

	symbol := normalizeHermesSymbol(c.Query("symbol"))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 symbol 参数"})
		return
	}

	t, ex, err := s.getHermesTrader(userID, exchangeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	price, err := t.GetMarketPrice(symbol)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "获取市价失败", "detail": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"exchange_id": ex.ID,
		"symbol":      symbol,
		"price":       price,
	})
}

func (s *Server) executeHermesTradeHTTP(t trader.Trader, req hermesTradeRequest) (map[string]interface{}, error) {
	return hermes.ExecuteTrade(t, hermes.TradeRequest{
		Action:   req.Action,
		Symbol:   req.Symbol,
		Quantity: req.Quantity,
		Leverage: req.Leverage,
	})
}

func (s *Server) handleHermesTrade(c *gin.Context) {
	userID := c.GetString("user_id")

	var req hermesTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !req.Confirmed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "交易需用户确认，请设置 confirmed: true"})
		return
	}

	t, ex, err := s.getHermesTrader(userID, req.ExchangeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.executeHermesTradeHTTP(t, req)
	if err != nil {
		log.Printf("[HERMES] user=%s exchange=%s action=%s symbol=%s error=%v",
			userID, req.ExchangeID, req.Action, req.Symbol, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[HERMES] user=%s exchange=%s action=%s symbol=%s qty=%.8f success",
		userID, req.ExchangeID, req.Action, req.Symbol, req.Quantity)

	c.JSON(http.StatusOK, gin.H{
		"exchange_id": ex.ID,
		"action":      req.Action,
		"symbol":      normalizeHermesSymbol(req.Symbol),
		"result":      result,
	})
}

func (s *Server) handleHermesLeverage(c *gin.Context) {
	userID := c.GetString("user_id")

	var req hermesLeverageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Leverage <= 0 || req.Leverage > 125 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "leverage 必须在 1-125 之间"})
		return
	}

	t, ex, err := s.getHermesTrader(userID, req.ExchangeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	symbol := normalizeHermesSymbol(req.Symbol)
	if err := t.SetLeverage(symbol, req.Leverage); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[HERMES] user=%s exchange=%s set_leverage symbol=%s leverage=%d",
		userID, req.ExchangeID, symbol, req.Leverage)

	c.JSON(http.StatusOK, gin.H{
		"exchange_id": ex.ID,
		"symbol":      symbol,
		"leverage":    req.Leverage,
		"status":      "ok",
	})
}

func (s *Server) handleHermesStopLoss(c *gin.Context) {
	userID := c.GetString("user_id")

	var req hermesStopOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, ex, err := s.getHermesTrader(userID, req.ExchangeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	symbol := normalizeHermesSymbol(req.Symbol)
	positionSide := strings.ToUpper(strings.TrimSpace(req.PositionSide))
	if err := t.SetStopLoss(symbol, positionSide, req.Quantity, req.StopPrice); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[HERMES] user=%s exchange=%s set_stop_loss symbol=%s side=%s",
		userID, req.ExchangeID, symbol, positionSide)

	c.JSON(http.StatusOK, gin.H{
		"exchange_id":   ex.ID,
		"symbol":        symbol,
		"position_side": positionSide,
		"status":        "ok",
	})
}

func (s *Server) handleHermesTakeProfit(c *gin.Context) {
	userID := c.GetString("user_id")

	var req hermesStopOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, ex, err := s.getHermesTrader(userID, req.ExchangeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	symbol := normalizeHermesSymbol(req.Symbol)
	positionSide := strings.ToUpper(strings.TrimSpace(req.PositionSide))
	if err := t.SetTakeProfit(symbol, positionSide, req.Quantity, req.StopPrice); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[HERMES] user=%s exchange=%s set_take_profit symbol=%s side=%s",
		userID, req.ExchangeID, symbol, positionSide)

	c.JSON(http.StatusOK, gin.H{
		"exchange_id":   ex.ID,
		"symbol":        symbol,
		"position_side": positionSide,
		"status":        "ok",
	})
}
