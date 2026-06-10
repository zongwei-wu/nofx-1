package client

// TradeRequest 网关交易请求
type TradeRequest struct {
	ExchangeID string  `json:"exchange_id"`
	Action     string  `json:"action"`
	Symbol     string  `json:"symbol"`
	Quantity   float64 `json:"quantity,omitempty"`
	Leverage   int     `json:"leverage,omitempty"`
}

// LeverageRequest 设置杠杆
type LeverageRequest struct {
	ExchangeID string `json:"exchange_id"`
	Symbol     string `json:"symbol"`
	Leverage   int    `json:"leverage"`
}

// StopOrderRequest 止损/止盈
type StopOrderRequest struct {
	ExchangeID   string  `json:"exchange_id"`
	Symbol       string  `json:"symbol"`
	PositionSide string  `json:"position_side"`
	Quantity     float64 `json:"quantity"`
	StopPrice    float64 `json:"stop_price"`
}

// Capabilities Gateway 能力清单
type Capabilities struct {
	Version         string   `json:"version"`
	Exchanges       []string `json:"exchanges"`
	Features        []string `json:"features"`
	KlinesExchanges []string `json:"klines_exchanges"`
}

// ExchangeConfig 交易所配置（脱敏）
type ExchangeConfig struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Testnet bool   `json:"testnet"`
	APIKey  string `json:"apiKey,omitempty"`
}

// KlineResponse K 线响应
type KlineResponse struct {
	ExchangeID string       `json:"exchange_id"`
	Symbol     string       `json:"symbol"`
	Interval   string       `json:"interval"`
	DataSource string       `json:"data_source"`
	Klines     []KlinePoint `json:"klines"`
}

// KlinePoint K 线点
type KlinePoint struct {
	Time  int64   `json:"time"`
	Open  float64 `json:"open"`
	High  float64 `json:"high"`
	Low   float64 `json:"low"`
	Close float64 `json:"close"`
}

// BalanceResponse 余额响应
type BalanceResponse struct {
	ExchangeID       string                 `json:"exchange_id"`
	Testnet          bool                   `json:"testnet"`
	Balance          map[string]interface{} `json:"balance"`
	TotalEquity      float64                `json:"total_equity"`
	AvailableBalance float64                `json:"available_balance"`
}

// PositionsResponse 持仓响应
type PositionsResponse struct {
	ExchangeID string                   `json:"exchange_id"`
	Positions  []map[string]interface{} `json:"positions"`
}

// TradeResponse 交易响应
type TradeResponse struct {
	ExchangeID string                 `json:"exchange_id"`
	Action     string                 `json:"action"`
	Symbol     string                 `json:"symbol"`
	Result     map[string]interface{} `json:"result"`
}
