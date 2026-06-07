package api

import (
	"net/http"
	"strconv"
	"strings"

	"nofx/market"

	"github.com/gin-gonic/gin"
)

type klinePointResponse struct {
	Time  int64   `json:"time"`
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

// handleMarketKlines 代理币安合约 K 线（供前端价格事件图使用）
// GET /api/market/klines?symbol=BTCUSDT&interval=1h&limit=200
func (s *Server) handleMarketKlines(c *gin.Context) {
	symbol := strings.ToUpper(strings.TrimSpace(c.Query("symbol")))
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 symbol 参数"})
		return
	}
	if !strings.HasSuffix(symbol, "USDT") {
		symbol += "USDT"
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

	client := market.NewAPIClient()
	klines, err := client.GetKlines(symbol, interval, limit)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "获取K线失败", "detail": err.Error()})
		return
	}

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

	c.JSON(http.StatusOK, gin.H{
		"symbol":      symbol,
		"interval":    interval,
		"data_source": "rest",
		"klines":      out,
	})
}
