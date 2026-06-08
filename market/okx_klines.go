package market

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const okxPublicBaseURL = "https://www.okx.com"

// OKXKlineClient OKX 公共 K 线客户端
type OKXKlineClient struct {
	client *http.Client
}

func NewOKXKlineClient() *OKXKlineClient {
	return &OKXKlineClient{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func convertSymbolToOKXInstID(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if strings.Contains(symbol, "-") {
		return symbol
	}
	if strings.HasSuffix(symbol, "USDT") {
		base := strings.TrimSuffix(symbol, "USDT")
		return base + "-USDT-SWAP"
	}
	return symbol + "-USDT-SWAP"
}

func mapIntervalToOKXBar(interval string) (string, error) {
	switch strings.ToLower(interval) {
	case "1m":
		return "1m", nil
	case "3m":
		return "3m", nil
	case "5m":
		return "5m", nil
	case "15m":
		return "15m", nil
	case "30m":
		return "30m", nil
	case "1h":
		return "1H", nil
	case "2h":
		return "2H", nil
	case "4h":
		return "4H", nil
	case "1d":
		return "1D", nil
	default:
		return "", fmt.Errorf("不支持的 interval: %s", interval)
	}
}

type okxCandlesResponse struct {
	Code string     `json:"code"`
	Msg  string     `json:"msg"`
	Data [][]string `json:"data"`
}

// GetKlines 获取 OKX USDT 永续 K 线（公共接口）
func (c *OKXKlineClient) GetKlines(symbol, interval string, limit int) ([]Kline, error) {
	bar, err := mapIntervalToOKXBar(interval)
	if err != nil {
		return nil, err
	}
	instID := convertSymbolToOKXInstID(symbol)

	url := fmt.Sprintf("%s/api/v5/market/candles?instId=%s&bar=%s&limit=%d",
		okxPublicBaseURL, instID, bar, limit)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result okxCandlesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != "0" {
		return nil, fmt.Errorf("OKX API error: %s %s", result.Code, result.Msg)
	}

	klines := make([]Kline, 0, len(result.Data))
	for _, row := range result.Data {
		if len(row) < 6 {
			continue
		}
		openTime, _ := strconv.ParseInt(row[0], 10, 64)
		open, _ := strconv.ParseFloat(row[1], 64)
		high, _ := strconv.ParseFloat(row[2], 64)
		low, _ := strconv.ParseFloat(row[3], 64)
		closePx, _ := strconv.ParseFloat(row[4], 64)
		volume, _ := strconv.ParseFloat(row[5], 64)
		klines = append(klines, Kline{
			OpenTime: openTime,
			Open:     open,
			High:     high,
			Low:      low,
			Close:    closePx,
			Volume:   volume,
		})
	}

	// OKX 返回倒序，统一为正序
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	return klines, nil
}
