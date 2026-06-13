package api

import (
	"fmt"
	"net/http"
	"strings"

	"nofx/indicator"
	"nofx/market"

	"github.com/gin-gonic/gin"
)

type indicatorMeta struct {
	Name        string  `json:"name"`
	Label       string  `json:"label"`
	Params      []int   `json:"params,omitempty"`
	DefaultMult float64 `json:"default_mult,omitempty"`
	Description string  `json:"description"`
}

var indicatorCatalog = []indicatorMeta{
	{Name: "SMA", Label: "简单移动平均", Params: []int{20}, Description: "SMA(period)"},
	{Name: "EMA", Label: "指数移动平均", Params: []int{20}, Description: "EMA(period)"},
	{Name: "MACD", Label: "MACD", Params: []int{12, 26, 9}, Description: "MACD(fast,slow,signal)"},
	{Name: "RSI", Label: "相对强弱", Params: []int{7, 14}, Description: "RSI(period)"},
	{Name: "BOLLINGER", Label: "布林带", Params: []int{20}, DefaultMult: 2.0, Description: "Bollinger(period,mult)"},
	{Name: "ATR", Label: "平均真实波幅", Params: []int{14}, Description: "ATR(period)"},
	{Name: "STOCHASTIC", Label: "随机指标", Params: []int{14, 3}, Description: "Stochastic(k,d)"},
	{Name: "OBV", Label: "能量潮", Description: "OBV()"},
	{Name: "VWAP", Label: "成交量加权均价", Description: "VWAP()"},
	{Name: "ADX", Label: "趋势强度", Params: []int{14}, Description: "ADX(period)"},
	{Name: "ICHIMOKU", Label: "一目均衡", Description: "Ichimoku()"},
	{Name: "MFI", Label: "资金流量", Params: []int{14}, Description: "MFI(period)"},
}

type computeIndicatorsRequest struct {
	ExchangeID string   `json:"exchange_id"`
	Symbol     string   `json:"symbol"`
	Interval   string   `json:"interval"`
	Limit      int      `json:"limit"`
	Indicators []string `json:"indicators"`
}

type compareIndicatorsRequest struct {
	Symbols    []string `json:"symbols"`
	Interval   string   `json:"interval"`
	Indicators []string `json:"indicators"`
}

func (s *Server) handleIndicatorsList(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"indicators": indicatorCatalog})
}

func (s *Server) handleIndicatorsCompute(c *gin.Context) {
	var req computeIndicatorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol 不能为空"})
		return
	}
	if req.Interval == "" {
		req.Interval = "1h"
	}
	if req.Limit <= 0 {
		req.Limit = 200
	}

	client := market.NewAPIClient()
	mk, err := client.GetKlines(market.Normalize(req.Symbol), req.Interval, req.Limit)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	ik := toIndicatorKlinesFromMarket(mk)
	results := computeIndicatorResults(req.Indicators, ik)
	c.JSON(http.StatusOK, gin.H{
		"symbol":     req.Symbol,
		"interval":   req.Interval,
		"indicators": results,
	})
}

func (s *Server) handleIndicatorsCompare(c *gin.Context) {
	var req compareIndicatorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Symbols) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbols 不能为空"})
		return
	}
	if req.Interval == "" {
		req.Interval = "1h"
	}

	client := market.NewAPIClient()
	comparison := make(map[string]interface{})
	for _, sym := range req.Symbols {
		mk, err := client.GetKlines(market.Normalize(sym), req.Interval, 200)
		if err != nil {
			comparison[sym] = gin.H{"error": err.Error()}
			continue
		}
		ik := toIndicatorKlinesFromMarket(mk)
		comparison[sym] = computeIndicatorResults(req.Indicators, ik)
	}
	c.JSON(http.StatusOK, gin.H{"comparison": comparison})
}

func toIndicatorKlinesFromMarket(klines []market.Kline) []indicator.Kline {
	out := make([]indicator.Kline, len(klines))
	for i, k := range klines {
		out[i] = indicator.Kline{OpenTime: k.OpenTime, Open: k.Open, High: k.High, Low: k.Low, Close: k.Close, Volume: k.Volume}
	}
	return out
}

func computeIndicatorResults(names []string, klines []indicator.Kline) map[string]interface{} {
	if len(names) == 0 {
		names = []string{"EMA20", "RSI14", "MACD", "ATR14"}
	}
	results := make(map[string]interface{})
	for _, name := range names {
		key := strings.ToUpper(strings.TrimSpace(name))
		switch {
		case strings.HasPrefix(key, "SMA"):
			period := parseIndicatorPeriod(key, "SMA", 20)
			results[key] = indicator.SMA(klines, period)
		case strings.HasPrefix(key, "EMA"):
			period := parseIndicatorPeriod(key, "EMA", 20)
			results[key] = indicator.EMA(klines, period)
		case strings.HasPrefix(key, "RSI"):
			period := parseIndicatorPeriod(key, "RSI", 14)
			results[key] = indicator.RSI(klines, period)
		case strings.HasPrefix(key, "ATR"):
			period := parseIndicatorPeriod(key, "ATR", 14)
			results[key] = indicator.ATR(klines, period)
		case strings.HasPrefix(key, "MFI"):
			period := parseIndicatorPeriod(key, "MFI", 14)
			results[key] = indicator.MFI(klines, period)
		case key == "MACD":
			results[key] = indicator.MACD(klines, 12, 26, 9)
		case key == "BOLLINGER" || key == "BOLL":
			results[key] = indicator.Bollinger(klines, 20, 2.0)
		case key == "STOCHASTIC" || key == "STOCH":
			results[key] = indicator.Stochastic(klines, 14, 3)
		case key == "OBV":
			results[key] = indicator.OBV(klines)
		case key == "VWAP":
			results[key] = indicator.VWAP(klines)
		case key == "ADX":
			results[key] = indicator.ADX(klines, 14)
		case key == "ICHIMOKU":
			results[key] = indicator.Ichimoku(klines)
		}
	}
	return results
}

func parseIndicatorPeriod(name, prefix string, defaultPeriod int) int {
	suffix := strings.TrimPrefix(strings.ToUpper(name), prefix)
	if suffix == "" {
		return defaultPeriod
	}
	var period int
	if _, err := fmt.Sscanf(suffix, "%d", &period); err != nil || period <= 0 {
		return defaultPeriod
	}
	return period
}

// Gateway handlers
func (s *Server) handleGatewayIndicatorsList(c *gin.Context)    { s.handleIndicatorsList(c) }
func (s *Server) handleGatewayIndicatorsCompute(c *gin.Context) { s.handleIndicatorsCompute(c) }
func (s *Server) handleGatewayIndicatorsCompare(c *gin.Context) { s.handleIndicatorsCompare(c) }
