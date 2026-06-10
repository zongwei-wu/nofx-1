package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client NOFX 交易所网关 HTTP 客户端（供外部 Hermes 服务使用）
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New 创建网关客户端
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) authHeader() string {
	return "ApiKey " + c.apiKey
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHeader())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &errBody)
		if errBody.Error != "" {
			return fmt.Errorf("%s: %s", res.Status, errBody.Error)
		}
		return fmt.Errorf("%s: %s", res.Status, string(data))
	}

	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	full := path
	if len(params) > 0 {
		full += "?" + params.Encode()
	}
	return c.do(ctx, http.MethodGet, full, nil, out)
}

// Capabilities 获取 Gateway 能力清单
func (c *Client) Capabilities(ctx context.Context) (*Capabilities, error) {
	var resp Capabilities
	if err := c.get(ctx, "/api/gateway/capabilities", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListExchanges 列出用户交易所配置
func (c *Client) ListExchanges(ctx context.Context) ([]ExchangeConfig, error) {
	var resp []ExchangeConfig
	if err := c.get(ctx, "/api/gateway/exchanges", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetBalance 获取账户余额
func (c *Client) GetBalance(ctx context.Context, exchangeID string) (*BalanceResponse, error) {
	params := url.Values{"exchange_id": {exchangeID}}
	var resp BalanceResponse
	if err := c.get(ctx, "/api/gateway/balance", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetPositions 获取持仓
func (c *Client) GetPositions(ctx context.Context, exchangeID string) (*PositionsResponse, error) {
	params := url.Values{"exchange_id": {exchangeID}}
	var resp PositionsResponse
	if err := c.get(ctx, "/api/gateway/positions", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetMarketPrice 获取市价
func (c *Client) GetMarketPrice(ctx context.Context, exchangeID, symbol string) (float64, error) {
	params := url.Values{
		"exchange_id": {exchangeID},
		"symbol":      {symbol},
	}
	var resp struct {
		Price float64 `json:"price"`
	}
	if err := c.get(ctx, "/api/gateway/market-price", params, &resp); err != nil {
		return 0, err
	}
	return resp.Price, nil
}

// GetKlines 获取 K 线
func (c *Client) GetKlines(ctx context.Context, exchangeID, symbol, interval string, limit int) (*KlineResponse, error) {
	params := url.Values{
		"exchange_id": {exchangeID},
		"symbol":      {symbol},
		"interval":    {interval},
		"limit":       {fmt.Sprintf("%d", limit)},
	}
	var resp KlineResponse
	if err := c.get(ctx, "/api/gateway/klines", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Trade 执行交易（API Key 路径无需 confirmed）
func (c *Client) Trade(ctx context.Context, req TradeRequest) (*TradeResponse, error) {
	var resp TradeResponse
	if err := c.do(ctx, http.MethodPost, "/api/gateway/trade", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SetLeverage 设置杠杆
func (c *Client) SetLeverage(ctx context.Context, req LeverageRequest) error {
	return c.do(ctx, http.MethodPost, "/api/gateway/leverage", req, nil)
}

// SetStopLoss 设置止损
func (c *Client) SetStopLoss(ctx context.Context, req StopOrderRequest) error {
	return c.do(ctx, http.MethodPost, "/api/gateway/stop-loss", req, nil)
}

// SetTakeProfit 设置止盈
func (c *Client) SetTakeProfit(ctx context.Context, req StopOrderRequest) error {
	return c.do(ctx, http.MethodPost, "/api/gateway/take-profit", req, nil)
}
