package api

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

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
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return symbol
	}
	if !strings.HasSuffix(symbol, "USDT") {
		symbol += "USDT"
	}
	return symbol
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

func hasPositionSide(positions []map[string]interface{}, symbol, side string) bool {
	for _, pos := range positions {
		if pos["symbol"] == symbol && pos["side"] == side {
			return true
		}
	}
	return false
}

func formatTradeQuantity(t trader.Trader, symbol string, quantity float64) (float64, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("quantity 必须大于 0")
	}
	formatted, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return quantity, nil
	}
	parsed, err := strconv.ParseFloat(formatted, 64)
	if err != nil {
		return quantity, nil
	}
	return parsed, nil
}

func (s *Server) executeHermesTrade(t trader.Trader, req hermesTradeRequest) (map[string]interface{}, error) {
	symbol := normalizeHermesSymbol(req.Symbol)
	action := strings.ToLower(strings.TrimSpace(req.Action))

	switch action {
	case "open_long", "add_long":
		qty, err := formatTradeQuantity(t, symbol, req.Quantity)
		if err != nil {
			return nil, err
		}
		if action == "open_long" {
			positions, _ := t.GetPositions()
			if hasPositionSide(positions, symbol, "long") {
				return nil, fmt.Errorf("%s 已有多仓，请使用 add_long 加仓或先平仓", symbol)
			}
		}
		leverage := req.Leverage
		if leverage <= 0 {
			leverage = 5
		}
		return t.OpenLong(symbol, qty, leverage)

	case "open_short", "add_short":
		qty, err := formatTradeQuantity(t, symbol, req.Quantity)
		if err != nil {
			return nil, err
		}
		if action == "open_short" {
			positions, _ := t.GetPositions()
			if hasPositionSide(positions, symbol, "short") {
				return nil, fmt.Errorf("%s 已有空仓，请使用 add_short 加仓或先平仓", symbol)
			}
		}
		leverage := req.Leverage
		if leverage <= 0 {
			leverage = 5
		}
		return t.OpenShort(symbol, qty, leverage)

	case "close_long", "reduce_long":
		qty := req.Quantity
		if action == "close_long" {
			qty = 0
		} else if qty <= 0 {
			return nil, fmt.Errorf("reduce_long 需要 quantity > 0")
		} else {
			var err error
			qty, err = formatTradeQuantity(t, symbol, qty)
			if err != nil {
				return nil, err
			}
		}
		return t.CloseLong(symbol, qty)

	case "close_short", "reduce_short":
		qty := req.Quantity
		if action == "close_short" {
			qty = 0
		} else if qty <= 0 {
			return nil, fmt.Errorf("reduce_short 需要 quantity > 0")
		} else {
			var err error
			qty, err = formatTradeQuantity(t, symbol, qty)
			if err != nil {
				return nil, err
			}
		}
		return t.CloseShort(symbol, qty)

	default:
		return nil, fmt.Errorf("不支持的 action: %s", req.Action)
	}
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

	result, err := s.executeHermesTrade(t, req)
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
