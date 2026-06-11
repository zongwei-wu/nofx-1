package trader

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	gateMainnetBaseURL  = "https://api.gateio.ws/api/v4"
	gateTestnetBaseURL  = "https://fx-api-testnet.gateio.ws/api/v4"
	gateCacheDuration   = 15 * time.Second
)

// GateContract Gate USDT 永续合约规格
type GateContract struct {
	Name             string  `json:"name"`
	QuantoMultiplier float64 `json:"quanto_multiplier,string"`
	OrderSizeMin     float64 `json:"order_size_min,string"`
	OrderSizeMax     float64 `json:"order_size_max,string"`
	MarkPrice        float64 `json:"mark_price,string"`
	LastPrice        float64 `json:"last_price,string"`
	LeverageMin      float64 `json:"leverage_min,string"`
	LeverageMax      float64 `json:"leverage_max,string"`
	OrderPriceRound  int     `json:"order_price_round,string"`
}

// GateTrader Gate USDT 永续交易器
type GateTrader struct {
	apiKey    string
	secretKey string
	testnet   bool
	isCross   bool

	client  *http.Client
	baseURL string

	contracts   map[string]GateContract
	contractsMu sync.RWMutex

	cachedBalance    map[string]interface{}
	balanceCacheTime time.Time
	balanceCacheMu   sync.RWMutex

	cachedPositions    []map[string]interface{}
	positionsCacheTime time.Time
	positionsCacheMu   sync.RWMutex
}

// NewGateTrader 创建 Gate 交易器
func NewGateTrader(apiKey, secretKey string, testnet bool) (*GateTrader, error) {
	apiKey = strings.TrimSpace(apiKey)
	secretKey = strings.TrimSpace(secretKey)
	if apiKey == "" || secretKey == "" {
		return nil, fmt.Errorf("Gate API Key 和 Secret Key 不能为空")
	}

	t := &GateTrader{
		apiKey:    apiKey,
		secretKey: secretKey,
		testnet:   testnet,
		isCross:   true,
		baseURL:   gateMainnetBaseURL,
		contracts: make(map[string]GateContract),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if testnet {
		t.baseURL = gateTestnetBaseURL
		log.Printf("  🔬 使用 Gate 模拟盘 (Testnet)")
	} else {
		log.Printf("  🌐 使用 Gate 主网")
	}
	return t, nil
}

// SetBaseURL 供测试注入 mock server
func (t *GateTrader) SetBaseURL(url string) {
	t.baseURL = url
}

func convertSymbolToGate(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if strings.HasSuffix(symbol, "USDT") {
		base := strings.TrimSuffix(symbol, "USDT")
		return base + "_USDT"
	}
	return symbol
}

func convertGateToSymbol(name string) string {
	name = strings.ToUpper(strings.TrimSpace(name))
	parts := strings.Split(name, "_")
	if len(parts) >= 2 {
		return parts[0] + parts[1]
	}
	return name
}

func (t *GateTrader) sign(method, path, query, body, timestamp string) string {
	payload := strings.Join([]string{method, path + query, body, timestamp}, "\n")
	mac := hmac.New(sha512.New, []byte(t.secretKey))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (t *GateTrader) request(method, path string, body interface{}) (json.RawMessage, error) {
	var bodyStr string
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyStr = string(b)
		bodyReader = bytes.NewReader(b)
	}

	// Hash body for signature
	bodyHash := ""
	if bodyStr != "" {
		h := sha512.New()
		h.Write([]byte(bodyStr))
		bodyHash = hex.EncodeToString(h.Sum(nil))
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	queryStr := "" // no query params in signed requests here; set if needed
	sign := t.sign(method, path, queryStr, bodyHash, timestamp)

	req, err := http.NewRequest(method, t.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("KEY", t.apiKey)
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("SIGN", sign)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Gate 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Label   string `json:"label"`
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Message != "" {
			return nil, fmt.Errorf("Gate API 错误 [%s]: %s", errResp.Label, errResp.Message)
		}
		return nil, fmt.Errorf("Gate API HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (t *GateTrader) publicRequest(path string, query url.Values) (json.RawMessage, error) {
	fullURL := t.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Gate API HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

func (t *GateTrader) getContract(symbol string) (GateContract, error) {
	name := convertSymbolToGate(symbol)
	t.contractsMu.RLock()
	if c, ok := t.contracts[name]; ok {
		t.contractsMu.RUnlock()
		return c, nil
	}
	t.contractsMu.RUnlock()

	settle := "usdt"
	data, err := t.publicRequest("/futures/"+settle+"/contracts/"+name, nil)
	if err != nil {
		return GateContract{}, err
	}

	var c GateContract
	if err := json.Unmarshal(data, &c); err != nil {
		return GateContract{}, err
	}
	if c.Name == "" {
		return GateContract{}, fmt.Errorf("未找到 Gate 交易对 %s", name)
	}

	t.contractsMu.Lock()
	t.contracts[name] = c
	t.contractsMu.Unlock()
	return c, nil
}

func (t *GateTrader) baseToSize(symbol string, baseQty float64) (int64, error) {
	c, err := t.getContract(symbol)
	if err != nil {
		return 0, err
	}
	if c.QuantoMultiplier <= 0 {
		return 0, fmt.Errorf("%s quanto_multiplier 无效", c.Name)
	}
	// quanto_multiplier is USD value per contract
	// baseQty = number of contracts * quanto_multiplier / price
	// We approximate: contracts = baseQty / quanto_multiplier
	contracts := baseQty / c.QuantoMultiplier
	size := int64(contracts)
	if size <= 0 {
		size = 1 // minimum 1 contract
	}
	// Negative = short, positive = long in Gate
	// But we handle direction via side, so always positive size
	return size, nil
}

func (t *GateTrader) InvalidateAccountCache() {
	t.balanceCacheMu.Lock()
	t.cachedBalance = nil
	t.balanceCacheTime = time.Time{}
	t.balanceCacheMu.Unlock()

	t.positionsCacheMu.Lock()
	t.cachedPositions = nil
	t.positionsCacheTime = time.Time{}
	t.positionsCacheMu.Unlock()
}

// GetBalance 获取账户余额
func (t *GateTrader) GetBalance() (map[string]interface{}, error) {
	t.balanceCacheMu.RLock()
	if t.cachedBalance != nil && time.Since(t.balanceCacheTime) < gateCacheDuration {
		cached := t.cachedBalance
		t.balanceCacheMu.RUnlock()
		return cached, nil
	}
	t.balanceCacheMu.RUnlock()

	settle := "usdt"
	data, err := t.request(http.MethodGet, "/futures/"+settle+"/accounts", nil)
	if err != nil {
		return nil, fmt.Errorf("获取 Gate 余额失败: %w", err)
	}

	var account struct {
		Total           float64 `json:"total,string"`
		UnrealisedPnl   float64 `json:"unrealised_pnl,string"`
		Available       float64 `json:"available,string"`
		PositionMargin  float64 `json:"position_margin,string"`
	}
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, err
	}

	walletBal := account.Total - account.UnrealisedPnl
	result := map[string]interface{}{
		"totalWalletBalance":    walletBal,
		"availableBalance":      account.Available,
		"totalUnrealizedProfit": account.UnrealisedPnl,
	}

	t.balanceCacheMu.Lock()
	t.cachedBalance = result
	t.balanceCacheTime = time.Now()
	t.balanceCacheMu.Unlock()
	return result, nil
}

// GetPositions 获取所有持仓
func (t *GateTrader) GetPositions() ([]map[string]interface{}, error) {
	t.positionsCacheMu.RLock()
	if t.cachedPositions != nil && time.Since(t.positionsCacheTime) < gateCacheDuration {
		cached := t.cachedPositions
		t.positionsCacheMu.RUnlock()
		return cached, nil
	}
	t.positionsCacheMu.RUnlock()

	settle := "usdt"
	data, err := t.request(http.MethodGet, "/futures/"+settle+"/positions", nil)
	if err != nil {
		return nil, fmt.Errorf("获取 Gate 持仓失败: %w", err)
	}

	var items []struct {
		Contract        string  `json:"contract"`
		Size            int64   `json:"size"`
		Leverage        float64 `json:"leverage,string"`
		EntryPrice      float64 `json:"entry_price,string"`
		MarkPrice       float64 `json:"mark_price,string"`
		UnrealisedPnl   float64 `json:"unrealised_pnl,string"`
		LiquidationPrice float64 `json:"liq_price,string"`
		MarginMode      string  `json:"margin_mode"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, item := range items {
		if item.Size == 0 {
			continue
		}
		symbol := convertGateToSymbol(item.Contract)
		side := "long"
		if item.Size < 0 {
			side = "short"
		}

		absSize := math.Abs(float64(item.Size))
		baseAmt := absSize // Gate 用张数表示，baseAmt 用张数近似

		result = append(result, map[string]interface{}{
			"symbol":            symbol,
			"side":              side,
			"positionAmt":       baseAmt,
			"entryPrice":        item.EntryPrice,
			"markPrice":         item.MarkPrice,
			"unRealizedProfit":  item.UnrealisedPnl,
			"leverage":          item.Leverage,
			"liquidationPrice":  item.LiquidationPrice,
			"contracts":         absSize,
		})
	}

	t.positionsCacheMu.Lock()
	t.cachedPositions = result
	t.positionsCacheTime = time.Now()
	t.positionsCacheMu.Unlock()
	return result, nil
}

// SetMarginMode Gate 用 cross_/isolated_margin 字段在 position update 中设置
func (t *GateTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	t.isCross = isCrossMargin
	mode := "全仓"
	if !isCrossMargin {
		mode = "逐仓"
	}
	log.Printf("  ✓ Gate 仓位模式: %s", mode)
	return nil
}

// SetLeverage 设置杠杆
func (t *GateTrader) SetLeverage(symbol string, leverage int) error {
	name := convertSymbolToGate(symbol)
	settle := "usdt"
	_, err := t.request(http.MethodPost, "/futures/"+settle+"/positions/"+name+"/leverage",
		map[string]int{"leverage": leverage})
	if err != nil {
		if strings.Contains(err.Error(), "1034") {
			// leverage not changeable when position exists (expected)
			log.Printf("  ⚠ Gate %s 存在持仓，杠杆无法修改", symbol)
			return nil
		}
		return fmt.Errorf("设置 Gate 杠杆失败: %w", err)
	}
	log.Printf("  ✓ Gate %s 杠杆已设置为 %dx", symbol, leverage)
	return nil
}

// OpenLong 开多仓
func (t *GateTrader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	if err := t.CancelAllOrders(symbol); err != nil {
		log.Printf("  ⚠ 取消旧委托单失败: %v", err)
	}
	if err := t.SetLeverage(symbol, leverage); err != nil {
		return nil, err
	}
	return t.placeMarketOrder(symbol, quantity, "long")
}

// OpenShort 开空仓
func (t *GateTrader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	if err := t.CancelAllOrders(symbol); err != nil {
		log.Printf("  ⚠ 取消旧委托单失败: %v", err)
	}
	if err := t.SetLeverage(symbol, leverage); err != nil {
		return nil, err
	}
	return t.placeMarketOrder(symbol, quantity, "short")
}

func (t *GateTrader) placeMarketOrder(symbol string, baseQty float64, direction string) (map[string]interface{}, error) {
	size, err := t.baseToSize(symbol, baseQty)
	if err != nil {
		return nil, err
	}
	if size <= 0 {
		size = 1
	}

	name := convertSymbolToGate(symbol)
	contract, err := t.getContract(symbol)
	if err != nil {
		return nil, err
	}

	// For open: size positive, close=false
	// For close: reduce_only=true
	body := map[string]interface{}{
		"contract": name,
		"size":     size,
		"price":    "0",    // market order
		"tif":      "ioc",  // immediate or cancel for market
		"text":     "t-hermes",
	}

	if direction == "short" {
		// Gate uses negative size for short; but using side is safer when supported
		body["size"] = -size
	}

	settle := "usdt"
	data, err := t.request(http.MethodPost, "/futures/"+settle+"/orders", body)
	if err != nil {
		return nil, fmt.Errorf("Gate 下单失败: %w", err)
	}

	var order struct {
		ID         int64  `json:"id"`
		Contract   string `json:"contract"`
		Size       int64  `json:"size"`
		Status     string `json:"status"`
		FillPrice  float64 `json:"fill_price,string"`
	}
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, err
	}

	_ = contract
	log.Printf("✓ Gate %s %s 成功: %d 张 (%s)", direction, order.Contract, order.Size, name)
	t.InvalidateAccountCache()
	return map[string]interface{}{
		"orderId": fmt.Sprintf("%d", order.ID),
		"symbol":  symbol,
		"status":  order.Status,
	}, nil
}

// CloseLong 平多仓
func (t *GateTrader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	if quantity == 0 {
		positions, err := t.GetPositions()
		if err != nil {
			return nil, err
		}
		for _, pos := range positions {
			if pos["symbol"] == symbol && pos["side"] == "long" {
				quantity = pos["positionAmt"].(float64)
				break
			}
		}
		if quantity == 0 {
			return nil, fmt.Errorf("没有找到 %s 的多仓", symbol)
		}
	}
	result, err := t.placeCloseOrder(symbol, quantity, "long")
	if err != nil {
		return nil, err
	}
	_ = t.CancelAllOrders(symbol)
	return result, nil
}

// CloseShort 平空仓
func (t *GateTrader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	if quantity == 0 {
		positions, err := t.GetPositions()
		if err != nil {
			return nil, err
		}
		for _, pos := range positions {
			if pos["symbol"] == symbol && pos["side"] == "short" {
				quantity = pos["positionAmt"].(float64)
				break
			}
		}
		if quantity == 0 {
			return nil, fmt.Errorf("没有找到 %s 的空仓", symbol)
		}
	}
	result, err := t.placeCloseOrder(symbol, quantity, "short")
	if err != nil {
		return nil, err
	}
	_ = t.CancelAllOrders(symbol)
	return result, nil
}

func (t *GateTrader) placeCloseOrder(symbol string, baseQty float64, positionSide string) (map[string]interface{}, error) {
	size, err := t.baseToSize(symbol, baseQty)
	if err != nil {
		return nil, err
	}
	if size <= 0 {
		size = 1
	}

	name := convertSymbolToGate(symbol)
	// Close long: sell (negative size), Close short: buy (positive size)
	if positionSide == "long" {
		size = -size // sell to close long
	}

	body := map[string]interface{}{
		"contract":    name,
		"size":        size,
		"price":       "0",
		"tif":         "ioc",
		"reduce_only": true,
		"text":        "t-hermes",
	}

	settle := "usdt"
	data, err := t.request(http.MethodPost, "/futures/"+settle+"/orders", body)
	if err != nil {
		return nil, fmt.Errorf("Gate 平仓失败: %w", err)
	}

	var order struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, err
	}

	log.Printf("✓ Gate 平仓成功: %s %s %d 张", name, positionSide, size)
	t.InvalidateAccountCache()
	return map[string]interface{}{
		"orderId": fmt.Sprintf("%d", order.ID),
		"symbol":  symbol,
		"status":  order.Status,
	}, nil
}

// GetMarketPrice 获取市场价格
func (t *GateTrader) GetMarketPrice(symbol string) (float64, error) {
	name := convertSymbolToGate(symbol)
	settle := "usdt"
	q := url.Values{}
	q.Set("contract", name)
	data, err := t.publicRequest("/futures/"+settle+"/tickers", q)
	if err != nil {
		return 0, err
	}

	var tickers []struct {
		Contract string  `json:"contract"`
		Last     float64 `json:"last,string"`
	}
	if err := json.Unmarshal(data, &tickers); err != nil {
		return 0, err
	}
	for _, tk := range tickers {
		if tk.Contract == name {
			return tk.Last, nil
		}
	}
	return 0, fmt.Errorf("未找到 %s 价格", name)
}

// SetStopLoss 设置止损
func (t *GateTrader) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	return t.placePriceTriggerOrder(symbol, positionSide, quantity, stopPrice, 0, "stop_loss")
}

// SetTakeProfit 设置止盈
func (t *GateTrader) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	return t.placePriceTriggerOrder(symbol, positionSide, quantity, takeProfitPrice, 0, "take_profit")
}

func (t *GateTrader) placePriceTriggerOrder(symbol, positionSide string, baseQty, triggerPrice, _ float64, orderType string) error {
	size, err := t.baseToSize(symbol, baseQty)
	if err != nil {
		return err
	}
	if size <= 0 {
		size = 1
	}

	name := convertSymbolToGate(symbol)

	// Close trigger
	if positionSide == "long" {
		size = -size // sell to close long
	}

	settle := "usdt"
	body := map[string]interface{}{
		"initial": map[string]interface{}{
			"contract": name,
			"size":     size,
			"price":    "0",
			"tif":      "ioc",
			"reduce_only": true,
		},
		"trigger": map[string]interface{}{
			"strategy_type": 0, // 0 = price trigger
			"price":         fmt.Sprintf("%.8f", triggerPrice),
			"rule":          0,
		},
		"order_type": orderType,
	}

	// Use futures price orders endpoint
	_, err = t.request(http.MethodPost, "/futures/"+settle+"/price_orders", body)
	if err != nil {
		return fmt.Errorf("Gate 设置%s失败: %w", orderType, err)
	}
	return nil
}

// CancelStopLossOrders 仅取消止损
func (t *GateTrader) CancelStopLossOrders(symbol string) error {
	return t.cancelPriceOrders(symbol, "stop_loss")
}

// CancelTakeProfitOrders 仅取消止盈
func (t *GateTrader) CancelTakeProfitOrders(symbol string) error {
	return t.cancelPriceOrders(symbol, "take_profit")
}

func (t *GateTrader) cancelPriceOrders(symbol string, orderType string) error {
	name := convertSymbolToGate(symbol)
	settle := "usdt"

	q := url.Values{}
	q.Set("contract", name)
	q.Set("status", "open")
	data, err := t.request(http.MethodGet, "/futures/"+settle+"/price_orders?"+q.Encode(), nil)
	if err != nil {
		return err
	}

	type priceOrder struct {
		ID int `json:"id"`
	}
	var orders []priceOrder
	if err := json.Unmarshal(data, &orders); err != nil {
		return err
	}

	for _, o := range orders {
		_, _ = t.request(http.MethodDelete,
			fmt.Sprintf("/futures/%s/price_orders/%d", settle, o.ID), nil)
	}
	return nil
}

// CancelAllOrders 取消所有挂单
func (t *GateTrader) CancelAllOrders(symbol string) error {
	_ = t.cancelPriceOrders(symbol, "")
	name := convertSymbolToGate(symbol)
	settle := "usdt"

	// Cancel all open orders for this contract
	q := url.Values{}
	q.Set("contract", name)
	q.Set("status", "open")
	_, err := t.request(http.MethodDelete, "/futures/"+settle+"/orders?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	return nil
}

// CancelStopOrders 取消止盈/止损
func (t *GateTrader) CancelStopOrders(symbol string) error {
	return t.cancelPriceOrders(symbol, "")
}

// FormatQuantity 格式化数量
func (t *GateTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	c, err := t.getContract(symbol)
	if err != nil {
		return "", err
	}
	size := quantity / c.QuantoMultiplier
	return fmt.Sprintf("%d", int64(size)), nil
}

// GetLotStepSize 获取 1 张合约的标的币数量（quanto_multiplier 即 USD 面值）
func (t *GateTrader) GetLotStepSize(symbol string) (float64, error) {
	c, err := t.getContract(symbol)
	if err != nil {
		return 0, err
	}
	if c.QuantoMultiplier <= 0 {
		return 0, fmt.Errorf("%s quanto_multiplier 无效", symbol)
	}
	return c.QuantoMultiplier, nil
}
