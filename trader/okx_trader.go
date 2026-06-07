package trader

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	okxMainnetBaseURL = "https://www.okx.com"
	okxCacheDuration  = 15 * time.Second
)

// OKXInstrument OKX 合约规格
type OKXInstrument struct {
	InstID string
	CtVal  float64 // 每张合约面值（标的币）
	LotSz  float64 // 下单张数步进
	MinSz  float64 // 最小张数
	TickSz float64 // 价格步进
}

// OKXTrader OKX USDT 永续交易器
type OKXTrader struct {
	apiKey     string
	secretKey  string
	passphrase string
	testnet    bool
	isCross    bool

	client *http.Client
	baseURL string

	instruments   map[string]OKXInstrument
	instrumentsMu sync.RWMutex
	posModeOnce   sync.Once
	posModeErr    error

	cachedBalance     map[string]interface{}
	balanceCacheTime  time.Time
	balanceCacheMutex sync.RWMutex

	cachedPositions     []map[string]interface{}
	positionsCacheTime  time.Time
	positionsCacheMutex sync.RWMutex
}

// NewOKXTrader 创建 OKX 交易器
func NewOKXTrader(apiKey, secretKey, passphrase string, testnet bool) (*OKXTrader, error) {
	apiKey = strings.TrimSpace(apiKey)
	secretKey = strings.TrimSpace(secretKey)
	passphrase = strings.TrimSpace(passphrase)
	if apiKey == "" || secretKey == "" || passphrase == "" {
		return nil, fmt.Errorf("OKX API Key、Secret Key 和 Passphrase 不能为空")
	}

	t := &OKXTrader{
		apiKey:      apiKey,
		secretKey:   secretKey,
		passphrase:  passphrase,
		testnet:     testnet,
		isCross:     true,
		baseURL:     okxMainnetBaseURL,
		instruments: make(map[string]OKXInstrument),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if testnet {
		log.Printf("  🔬 使用 OKX 模拟盘 (Demo Trading)")
	} else {
		log.Printf("  🌐 使用 OKX 主网")
	}

	return t, nil
}

func (t *OKXTrader) ensureLongShortModeOnce() error {
	t.posModeOnce.Do(func() {
		t.posModeErr = t.ensureLongShortMode()
		if t.posModeErr != nil {
			log.Printf("⚠️ 设置 OKX 双向持仓模式失败: %v (若已是 long_short_mode 可忽略)", t.posModeErr)
			t.posModeErr = nil // 已是目标模式时不阻断交易
		}
	})
	return t.posModeErr
}

// SetBaseURL 供测试注入 mock server
func (t *OKXTrader) SetBaseURL(url string) {
	t.baseURL = url
}

func convertSymbolToOKX(symbol string) string {
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

func convertOKXToSymbol(instID string) string {
	instID = strings.ToUpper(strings.TrimSpace(instID))
	if !strings.Contains(instID, "-") {
		return instID
	}
	parts := strings.Split(instID, "-")
	if len(parts) >= 2 {
		return parts[0] + parts[1]
	}
	return instID
}

type okxResponse struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func (t *OKXTrader) sign(timestamp, method, path, body string) string {
	prehash := timestamp + method + path + body
	mac := hmac.New(sha256.New, []byte(t.secretKey))
	mac.Write([]byte(prehash))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (t *OKXTrader) request(method, path string, body interface{}) (json.RawMessage, error) {
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

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	req, err := http.NewRequest(method, t.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("OK-ACCESS-KEY", t.apiKey)
	req.Header.Set("OK-ACCESS-SIGN", t.sign(timestamp, method, path, bodyStr))
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", t.passphrase)
	req.Header.Set("Content-Type", "application/json")
	if t.testnet {
		req.Header.Set("x-simulated-trading", "1")
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OKX 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result okxResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 OKX 响应失败: %w, body=%s", err, string(respBody))
	}
	if result.Code != "0" {
		return nil, fmt.Errorf("OKX API 错误 [%s]: %s", result.Code, result.Msg)
	}
	return result.Data, nil
}

func (t *OKXTrader) publicRequest(path string) (json.RawMessage, error) {
	req, err := http.NewRequest(http.MethodGet, t.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if t.testnet {
		req.Header.Set("x-simulated-trading", "1")
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result okxResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 OKX 响应失败: %w", err)
	}
	if result.Code != "0" {
		return nil, fmt.Errorf("OKX API 错误 [%s]: %s", result.Code, result.Msg)
	}
	return result.Data, nil
}

func (t *OKXTrader) ensureLongShortMode() error {
	_, err := t.request(http.MethodPost, "/api/v5/account/set-position-mode", map[string]string{
		"posMode": "long_short_mode",
	})
	return err
}

func (t *OKXTrader) tdMode() string {
	if t.isCross {
		return "cross"
	}
	return "isolated"
}

func (t *OKXTrader) getInstrument(symbol string) (OKXInstrument, error) {
	instID := convertSymbolToOKX(symbol)
	t.instrumentsMu.RLock()
	if inst, ok := t.instruments[instID]; ok {
		t.instrumentsMu.RUnlock()
		return inst, nil
	}
	t.instrumentsMu.RUnlock()

	data, err := t.publicRequest("/api/v5/public/instruments?instType=SWAP&instId=" + instID)
	if err != nil {
		return OKXInstrument{}, err
	}

	var items []struct {
		InstID string `json:"instId"`
		CtVal  string `json:"ctVal"`
		LotSz  string `json:"lotSz"`
		MinSz  string `json:"minSz"`
		TickSz string `json:"tickSz"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return OKXInstrument{}, err
	}
	if len(items) == 0 {
		return OKXInstrument{}, fmt.Errorf("未找到交易对 %s", instID)
	}

	ctVal, _ := strconv.ParseFloat(items[0].CtVal, 64)
	lotSz, _ := strconv.ParseFloat(items[0].LotSz, 64)
	minSz, _ := strconv.ParseFloat(items[0].MinSz, 64)
	tickSz, _ := strconv.ParseFloat(items[0].TickSz, 64)
	inst := OKXInstrument{
		InstID: instID,
		CtVal:  ctVal,
		LotSz:  lotSz,
		MinSz:  minSz,
		TickSz: tickSz,
	}

	t.instrumentsMu.Lock()
	t.instruments[instID] = inst
	t.instrumentsMu.Unlock()
	return inst, nil
}

func roundDownToStep(value, step float64) float64 {
	if step <= 0 {
		return value
	}
	return math.Floor(value/step+1e-12) * step
}

func (t *OKXTrader) baseToContracts(symbol string, baseQty float64) (float64, error) {
	inst, err := t.getInstrument(symbol)
	if err != nil {
		return 0, err
	}
	if inst.CtVal <= 0 {
		return 0, fmt.Errorf("%s ctVal 无效", inst.InstID)
	}
	contracts := roundDownToStep(baseQty/inst.CtVal, inst.LotSz)
	if contracts < inst.MinSz {
		return 0, fmt.Errorf("数量过小: %.8f 标的币 → %.4f 张 (最小 %.4f 张)", baseQty, contracts, inst.MinSz)
	}
	return contracts, nil
}

func (t *OKXTrader) contractsToBase(symbol string, contracts float64) (float64, error) {
	inst, err := t.getInstrument(symbol)
	if err != nil {
		return 0, err
	}
	return contracts * inst.CtVal, nil
}

func formatFloatTrim(v float64, decimals int) string {
	s := strconv.FormatFloat(v, 'f', decimals, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

func (t *OKXTrader) formatContracts(symbol string, contracts float64) (string, error) {
	inst, err := t.getInstrument(symbol)
	if err != nil {
		return "", err
	}
	rounded := roundDownToStep(contracts, inst.LotSz)
	decimals := 0
	if inst.LotSz < 1 {
		decimals = int(math.Ceil(-math.Log10(inst.LotSz)))
	}
	return formatFloatTrim(rounded, decimals), nil
}

// InvalidateAccountCache 清除账户与持仓缓存
func (t *OKXTrader) InvalidateAccountCache() {
	t.balanceCacheMutex.Lock()
	t.cachedBalance = nil
	t.balanceCacheTime = time.Time{}
	t.balanceCacheMutex.Unlock()

	t.positionsCacheMutex.Lock()
	t.cachedPositions = nil
	t.positionsCacheTime = time.Time{}
	t.positionsCacheMutex.Unlock()
}

// GetBalance 获取账户余额
func (t *OKXTrader) GetBalance() (map[string]interface{}, error) {
	t.balanceCacheMutex.RLock()
	if t.cachedBalance != nil && time.Since(t.balanceCacheTime) < okxCacheDuration {
		cached := t.cachedBalance
		t.balanceCacheMutex.RUnlock()
		return cached, nil
	}
	t.balanceCacheMutex.RUnlock()

	data, err := t.request(http.MethodGet, "/api/v5/account/balance?ccy=USDT", nil)
	if err != nil {
		return nil, fmt.Errorf("获取 OKX 余额失败: %w", err)
	}

	var items []struct {
		Details []struct {
			Ccy      string `json:"ccy"`
			Eq       string `json:"eq"`
			AvailEq  string `json:"availEq"`
			Upl      string `json:"upl"`
			CashBal  string `json:"cashBal"`
			AvailBal string `json:"availBal"`
		} `json:"details"`
		TotalEq string `json:"totalEq"`
		Upl     string `json:"upl"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("OKX 余额数据为空")
	}

	totalEq, _ := strconv.ParseFloat(items[0].TotalEq, 64)
	totalUpl, _ := strconv.ParseFloat(items[0].Upl, 64)
	availEq := 0.0
	walletBal := 0.0
	for _, d := range items[0].Details {
		if d.Ccy == "USDT" {
			availEq, _ = strconv.ParseFloat(d.AvailEq, 64)
			if availEq == 0 {
				availEq, _ = strconv.ParseFloat(d.AvailBal, 64)
			}
			walletBal, _ = strconv.ParseFloat(d.CashBal, 64)
			if d.Upl != "" {
				upl, _ := strconv.ParseFloat(d.Upl, 64)
				totalUpl = upl
			}
			break
		}
	}
	if walletBal == 0 {
		walletBal = totalEq - totalUpl
	}

	result := map[string]interface{}{
		"totalWalletBalance":    walletBal,
		"availableBalance":      availEq,
		"totalUnrealizedProfit": totalUpl,
	}

	t.balanceCacheMutex.Lock()
	t.cachedBalance = result
	t.balanceCacheTime = time.Now()
	t.balanceCacheMutex.Unlock()
	return result, nil
}

// GetPositions 获取所有持仓
func (t *OKXTrader) GetPositions() ([]map[string]interface{}, error) {
	t.positionsCacheMutex.RLock()
	if t.cachedPositions != nil && time.Since(t.positionsCacheTime) < okxCacheDuration {
		cached := t.cachedPositions
		t.positionsCacheMutex.RUnlock()
		return cached, nil
	}
	t.positionsCacheMutex.RUnlock()

	data, err := t.request(http.MethodGet, "/api/v5/account/positions?instType=SWAP", nil)
	if err != nil {
		return nil, fmt.Errorf("获取 OKX 持仓失败: %w", err)
	}

	var items []struct {
		InstID  string `json:"instId"`
		Pos     string `json:"pos"`
		PosSide string `json:"posSide"`
		AvgPx   string `json:"avgPx"`
		MarkPx  string `json:"markPx"`
		Upl     string `json:"upl"`
		Lever   string `json:"lever"`
		LiqPx   string `json:"liqPx"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, item := range items {
		contracts, _ := strconv.ParseFloat(item.Pos, 64)
		if contracts == 0 {
			continue
		}
		symbol := convertOKXToSymbol(item.InstID)
		baseAmt, _ := t.contractsToBase(symbol, math.Abs(contracts))
		entryPx, _ := strconv.ParseFloat(item.AvgPx, 64)
		markPx, _ := strconv.ParseFloat(item.MarkPx, 64)
		upl, _ := strconv.ParseFloat(item.Upl, 64)
		lever, _ := strconv.ParseFloat(item.Lever, 64)
		liqPx, _ := strconv.ParseFloat(item.LiqPx, 64)

		side := strings.ToLower(item.PosSide)
		if side == "" {
			if contracts > 0 {
				side = "long"
			} else {
				side = "short"
			}
		}

		result = append(result, map[string]interface{}{
			"symbol":            symbol,
			"side":              side,
			"positionAmt":       baseAmt,
			"entryPrice":        entryPx,
			"markPrice":         markPx,
			"unRealizedProfit":  upl,
			"leverage":          lever,
			"liquidationPrice":  liqPx,
			"contracts":         math.Abs(contracts),
		})
	}

	t.positionsCacheMutex.Lock()
	t.cachedPositions = result
	t.positionsCacheTime = time.Now()
	t.positionsCacheMutex.Unlock()
	return result, nil
}

// SetMarginMode 设置仓位模式
func (t *OKXTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	t.isCross = isCrossMargin
	mode := "全仓"
	if !isCrossMargin {
		mode = "逐仓"
		instID := convertSymbolToOKX(symbol)
		_, err := t.request(http.MethodPost, "/api/v5/account/set-isolated-mode", map[string]string{
			"isoMode": "automatic",
			"type":    "MARGIN",
		})
		if err != nil && !strings.Contains(err.Error(), "already") {
			log.Printf("  ⚠️ OKX 设置逐仓模式: %v", err)
		}
		log.Printf("  ✓ OKX %s 仓位模式: %s", instID, mode)
	} else {
		log.Printf("  ✓ OKX 使用全仓模式 (%s)", mode)
	}
	return nil
}

// SetLeverage 设置杠杆
func (t *OKXTrader) SetLeverage(symbol string, leverage int) error {
	instID := convertSymbolToOKX(symbol)
	_, err := t.request(http.MethodPost, "/api/v5/account/set-leverage", map[string]string{
		"instId":  instID,
		"lever":   strconv.Itoa(leverage),
		"mgnMode": t.tdMode(),
	})
	if err != nil {
		if strings.Contains(err.Error(), "same") || strings.Contains(err.Error(), "59000") {
			log.Printf("  ✓ %s 杠杆已是 %dx", symbol, leverage)
			return nil
		}
		return fmt.Errorf("设置杠杆失败: %w", err)
	}
	log.Printf("  ✓ %s 杠杆已设置为 %dx", symbol, leverage)
	return nil
}

// OpenLong 开多仓
func (t *OKXTrader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	_ = t.ensureLongShortModeOnce()
	if err := t.CancelAllOrders(symbol); err != nil {
		log.Printf("  ⚠ 取消旧委托单失败: %v", err)
	}
	if err := t.SetLeverage(symbol, leverage); err != nil {
		return nil, err
	}
	return t.placeMarketOrder(symbol, "buy", "long", quantity)
}

// OpenShort 开空仓
func (t *OKXTrader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	if err := t.CancelAllOrders(symbol); err != nil {
		log.Printf("  ⚠ 取消旧委托单失败: %v", err)
	}
	if err := t.SetLeverage(symbol, leverage); err != nil {
		return nil, err
	}
	return t.placeMarketOrder(symbol, "sell", "short", quantity)
}

func (t *OKXTrader) placeMarketOrder(symbol, side, posSide string, baseQty float64) (map[string]interface{}, error) {
	contracts, err := t.baseToContracts(symbol, baseQty)
	if err != nil {
		return nil, err
	}
	sz, err := t.formatContracts(symbol, contracts)
	if err != nil {
		return nil, err
	}

	instID := convertSymbolToOKX(symbol)
	data, err := t.request(http.MethodPost, "/api/v5/trade/order", map[string]string{
		"instId":  instID,
		"tdMode":  t.tdMode(),
		"side":    side,
		"posSide": posSide,
		"ordType": "market",
		"sz":      sz,
	})
	if err != nil {
		return nil, fmt.Errorf("下单失败: %w", err)
	}

	var orders []struct {
		OrdID   string `json:"ordId"`
		InstID  string `json:"instId"`
		SCode   string `json:"sCode"`
		SMsg    string `json:"sMsg"`
	}
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, fmt.Errorf("OKX 下单无返回")
	}
	if orders[0].SCode != "" && orders[0].SCode != "0" {
		return nil, fmt.Errorf("OKX 下单失败 [%s]: %s", orders[0].SCode, orders[0].SMsg)
	}

	log.Printf("✓ OKX %s %s 成功: %s 张 (%s)", side, posSide, sz, instID)
	t.InvalidateAccountCache()
	return map[string]interface{}{
		"orderId": orders[0].OrdID,
		"symbol":  symbol,
		"status":  "filled",
	}, nil
}

// CloseLong 平多仓
func (t *OKXTrader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
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
	result, err := t.placeCloseOrder(symbol, "sell", "long", quantity)
	if err != nil {
		return nil, err
	}
	_ = t.CancelAllOrders(symbol)
	return result, nil
}

// CloseShort 平空仓
func (t *OKXTrader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
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
	result, err := t.placeCloseOrder(symbol, "buy", "short", quantity)
	if err != nil {
		return nil, err
	}
	_ = t.CancelAllOrders(symbol)
	return result, nil
}

func (t *OKXTrader) placeCloseOrder(symbol, side, posSide string, baseQty float64) (map[string]interface{}, error) {
	contracts, err := t.baseToContracts(symbol, baseQty)
	if err != nil {
		return nil, err
	}
	sz, err := t.formatContracts(symbol, contracts)
	if err != nil {
		return nil, err
	}
	instID := convertSymbolToOKX(symbol)
	data, err := t.request(http.MethodPost, "/api/v5/trade/order", map[string]string{
		"instId":     instID,
		"tdMode":     t.tdMode(),
		"side":       side,
		"posSide":    posSide,
		"ordType":    "market",
		"sz":         sz,
		"reduceOnly": "true",
	})
	if err != nil {
		return nil, fmt.Errorf("平仓失败: %w", err)
	}
	var orders []struct {
		OrdID string `json:"ordId"`
		SCode string `json:"sCode"`
		SMsg  string `json:"sMsg"`
	}
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}
	if len(orders) > 0 && orders[0].SCode != "" && orders[0].SCode != "0" {
		return nil, fmt.Errorf("OKX 平仓失败 [%s]: %s", orders[0].SCode, orders[0].SMsg)
	}
	log.Printf("✓ OKX 平仓成功: %s %s %s 张", instID, posSide, sz)
	t.InvalidateAccountCache()
	orderID := ""
	if len(orders) > 0 {
		orderID = orders[0].OrdID
	}
	return map[string]interface{}{
		"orderId": orderID,
		"symbol":  symbol,
		"status":  "filled",
	}, nil
}

// GetMarketPrice 获取市场价格
func (t *OKXTrader) GetMarketPrice(symbol string) (float64, error) {
	instID := convertSymbolToOKX(symbol)
	data, err := t.publicRequest("/api/v5/market/ticker?instId=" + instID)
	if err != nil {
		return 0, err
	}
	var items []struct {
		Last string `json:"last"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, fmt.Errorf("未找到 %s 价格", instID)
	}
	return strconv.ParseFloat(items[0].Last, 64)
}

// SetStopLoss 设置止损单
func (t *OKXTrader) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	return t.placeAlgoStop(symbol, positionSide, quantity, stopPrice, 0)
}

// SetTakeProfit 设置止盈单
func (t *OKXTrader) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	return t.placeAlgoStop(symbol, positionSide, quantity, 0, takeProfitPrice)
}

func (t *OKXTrader) placeAlgoStop(symbol, positionSide string, baseQty, slPx, tpPx float64) error {
	contracts, err := t.baseToContracts(symbol, baseQty)
	if err != nil {
		return err
	}
	sz, err := t.formatContracts(symbol, contracts)
	if err != nil {
		return err
	}

	posSide := strings.ToLower(positionSide)
	side := "sell"
	if posSide == "short" {
		side = "buy"
	}

	instID := convertSymbolToOKX(symbol)
	body := map[string]string{
		"instId":  instID,
		"tdMode":  t.tdMode(),
		"side":    side,
		"posSide": posSide,
		"ordType": "conditional",
		"sz":      sz,
	}
	if slPx > 0 {
		body["slTriggerPx"] = formatFloatTrim(slPx, 8)
		body["slOrdPx"] = "-1"
	}
	if tpPx > 0 {
		body["tpTriggerPx"] = formatFloatTrim(tpPx, 8)
		body["tpOrdPx"] = "-1"
	}

	_, err = t.request(http.MethodPost, "/api/v5/trade/order-algo", body)
	if err != nil {
		return fmt.Errorf("设置止盈止损失败: %w", err)
	}
	return nil
}

func (t *OKXTrader) listPendingAlgos(symbol string) ([]map[string]interface{}, error) {
	instID := convertSymbolToOKX(symbol)
	data, err := t.request(http.MethodGet, "/api/v5/trade/orders-algo-pending?instType=SWAP&instId="+instID+"&ordType=conditional", nil)
	if err != nil {
		return nil, err
	}
	var items []map[string]interface{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (t *OKXTrader) cancelAlgos(symbol string, filter func(map[string]interface{}) bool) error {
	algos, err := t.listPendingAlgos(symbol)
	if err != nil {
		return err
	}
	var toCancel []map[string]string
	for _, algo := range algos {
		if filter != nil && !filter(algo) {
			continue
		}
		algoID, _ := algo["algoId"].(string)
		instID, _ := algo["instId"].(string)
		if algoID != "" && instID != "" {
			toCancel = append(toCancel, map[string]string{
				"algoId": algoID,
				"instId": instID,
			})
		}
	}
	if len(toCancel) == 0 {
		return nil
	}
	_, err = t.request(http.MethodPost, "/api/v5/trade/cancel-algos", toCancel)
	return err
}

// CancelStopLossOrders 仅取消止损单
func (t *OKXTrader) CancelStopLossOrders(symbol string) error {
	return t.cancelAlgos(symbol, func(algo map[string]interface{}) bool {
		sl, _ := algo["slTriggerPx"].(string)
		return sl != "" && sl != "0"
	})
}

// CancelTakeProfitOrders 仅取消止盈单
func (t *OKXTrader) CancelTakeProfitOrders(symbol string) error {
	return t.cancelAlgos(symbol, func(algo map[string]interface{}) bool {
		tp, _ := algo["tpTriggerPx"].(string)
		return tp != "" && tp != "0"
	})
}

// CancelAllOrders 取消该币种的所有挂单
func (t *OKXTrader) CancelAllOrders(symbol string) error {
	_ = t.cancelAlgos(symbol, nil)
	instID := convertSymbolToOKX(symbol)
	data, err := t.request(http.MethodGet, "/api/v5/trade/orders-pending?instType=SWAP&instId="+instID, nil)
	if err != nil {
		return err
	}
	var orders []struct {
		InstID string `json:"instId"`
		OrdID  string `json:"ordId"`
	}
	if err := json.Unmarshal(data, &orders); err != nil {
		return err
	}
	if len(orders) == 0 {
		return nil
	}
	var batch []map[string]string
	for _, o := range orders {
		batch = append(batch, map[string]string{"instId": o.InstID, "ordId": o.OrdID})
	}
	_, err = t.request(http.MethodPost, "/api/v5/trade/cancel-batch-orders", batch)
	return err
}

// CancelStopOrders 取消止盈/止损单
func (t *OKXTrader) CancelStopOrders(symbol string) error {
	return t.cancelAlgos(symbol, nil)
}

// FormatQuantity 格式化数量（标的币）
func (t *OKXTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	inst, err := t.getInstrument(symbol)
	if err != nil {
		return "", err
	}
	if inst.CtVal <= 0 {
		return "", fmt.Errorf("ctVal 无效")
	}
	contracts := roundDownToStep(quantity/inst.CtVal, inst.LotSz)
	baseQty := contracts * inst.CtVal
	decimals := 8
	if inst.CtVal >= 1 {
		decimals = 4
	}
	return formatFloatTrim(baseQty, decimals), nil
}

// GetLotStepSize 获取 1 张合约对应的标的币数量（ctVal）
func (t *OKXTrader) GetLotStepSize(symbol string) (float64, error) {
	inst, err := t.getInstrument(symbol)
	if err != nil {
		return 0, err
	}
	if inst.CtVal <= 0 {
		return 0, fmt.Errorf("%s ctVal 无效", symbol)
	}
	return inst.CtVal, nil
}

// GetContractSpec 返回 OKX 合约规格（供跟单换算）
func (t *OKXTrader) GetContractSpec(symbol string) (OKXInstrument, error) {
	return t.getInstrument(symbol)
}
