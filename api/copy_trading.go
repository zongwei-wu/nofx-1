package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const copyTradeTimerIntervalSec = 300

// LeadOrder 币安跟单带单员成交记录
type LeadOrder struct {
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`
	PositionSide string  `json:"positionSide"`
	ExecutedQty  float64 `json:"executedQty"`
	AvgPrice     float64 `json:"avgPrice"`
	TotalPnl     float64 `json:"totalPnl"`
	OrderTime    int64   `json:"orderTime"`
}

// LeadOrderHistory 与本地缓存格式一致
type LeadOrderHistory struct {
	Data *struct {
		List []LeadOrder `json:"list"`
	} `json:"data"`
}

var (
	leadOrdersMemCache   = make(map[string]*leadOrdersCacheEntry)
	leadOrdersMemCacheMu sync.RWMutex
)

type leadOrdersCacheEntry struct {
	history   *LeadOrderHistory
	fetchedAt time.Time
}

func copyTradingCacheDir() string {
	if dir := strings.TrimSpace(os.Getenv("COPY_TRADING_CACHE_DIR")); dir != "" {
		return dir
	}
	return filepath.Join("data", "copy-trading")
}

func ensureCopyTradingCacheDir() error {
	return os.MkdirAll(copyTradingCacheDir(), 0o755)
}

func orderCachePath(portfolioID string) string {
	return filepath.Join(copyTradingCacheDir(), fmt.Sprintf("orders_%s.json", portfolioID))
}

func leaderboardCachePath() string {
	return filepath.Join(copyTradingCacheDir(), "leaderboard.json")
}

func binanceBapiBaseURL() string {
	if base := strings.TrimSpace(os.Getenv("BINANCE_BAPI_BASE_URL")); base != "" {
		return strings.TrimRight(base, "/")
	}
	return "https://www.binance.com"
}

type leaderboardCacheFile struct {
	Ts   int64           `json:"ts"`
	Data json.RawMessage `json:"data"`
}

func writeLeaderboardCache(data json.RawMessage) error {
	if len(data) == 0 {
		return fmt.Errorf("排行榜数据为空")
	}
	if err := ensureCopyTradingCacheDir(); err != nil {
		return err
	}
	payload, err := json.Marshal(leaderboardCacheFile{Ts: time.Now().Unix(), Data: data})
	if err != nil {
		return err
	}
	return os.WriteFile(leaderboardCachePath(), payload, 0o644)
}

func readLeaderboardCache() (json.RawMessage, int64, error) {
	raw, err := os.ReadFile(leaderboardCachePath())
	if err != nil {
		return nil, 0, err
	}
	var cached leaderboardCacheFile
	if err := json.Unmarshal(raw, &cached); err != nil {
		return nil, 0, err
	}
	if len(cached.Data) == 0 {
		return nil, 0, fmt.Errorf("缓存中无 data 字段")
	}
	return cached.Data, cached.Ts, nil
}

func emptyLeaderboardData() json.RawMessage {
	return json.RawMessage(`{"highestPnlLeads":[],"highestRoiLeads":[]}`)
}

// binanceLeadListPayloadPNL 高盈亏 Tab（30 日 PNL 排序）
func binanceLeadListPayloadPNL() map[string]interface{} {
	return map[string]interface{}{
		"pageNumber":       1,
		"pageSize":         20,
		"timeRange":        "30D",
		"dataType":         "PNL",
		"favoriteOnly":     false,
		"hideFull":         false,
		"nickname":         "",
		"order":            "DESC",
		"userAsset":        0,
		"portfolioType":    "ALL",
		"useAiRecommended": true,
		"PAGE_SIZE":        20,
	}
}

// binanceLeadListPayloadROI 高收益 Tab（30 日 ROI 排序）
func binanceLeadListPayloadROI() map[string]interface{} {
	return map[string]interface{}{
		"pageNumber":       1,
		"pageSize":         20,
		"timeRange":        "30D",
		"dataType":         "ROI",
		"favoriteOnly":     false,
		"hideFull":         false,
		"nickname":         "",
		"order":            "DESC",
		"userAsset":        0,
		"portfolioType":    "ALL",
		"useAiRecommended": true,
	}
}

func postBinanceCopyTradeBapi(path string, payload interface{}) (json.RawMessage, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	binanceURL := binanceBapiBaseURL() + path

	req, err := http.NewRequest("POST", binanceURL, strings.NewReader(string(jsonPayload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Origin", binanceBapiBaseURL())
	req.Header.Set("Referer", binanceBapiBaseURL()+"/zh-CN/copy-trading")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		snippet := string(body)
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet)
	}

	var binanceResp struct {
		Code    string          `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &binanceResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if binanceResp.Code != "000000" {
		return nil, fmt.Errorf("币安错误 %s: %s", binanceResp.Code, binanceResp.Message)
	}
	if len(binanceResp.Data) == 0 {
		return nil, fmt.Errorf("币安返回空 data")
	}
	return binanceResp.Data, nil
}

const binanceLeadPortfolioListPath = "/bapi/futures/v1/friendly/future/copy-trade/lead-portfolio/list"

func fetchBinanceLeadPortfolioListPNL() (json.RawMessage, error) {
	return postBinanceCopyTradeBapi(binanceLeadPortfolioListPath, binanceLeadListPayloadPNL())
}

func fetchBinanceLeadPortfolioListROI() (json.RawMessage, error) {
	return postBinanceCopyTradeBapi(binanceLeadPortfolioListPath, binanceLeadListPayloadROI())
}

func fetchBinanceRecommendLeadList() (json.RawMessage, error) {
	return postBinanceCopyTradeBapi(
		"/bapi/futures/v1/friendly/future/copy-trade/home-page/recommend-lead-list",
		map[string]interface{}{},
	)
}

func leadsFromListData(data json.RawMessage) ([]interface{}, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("空数据")
	}
	var withList struct {
		List []interface{} `json:"list"`
	}
	if err := json.Unmarshal(data, &withList); err == nil && len(withList.List) > 0 {
		return withList.List, nil
	}
	var withPnl struct {
		HighestPnlLeads []interface{} `json:"highestPnlLeads"`
	}
	if err := json.Unmarshal(data, &withPnl); err == nil && len(withPnl.HighestPnlLeads) > 0 {
		return withPnl.HighestPnlLeads, nil
	}
	var withRoi struct {
		HighestRoiLeads []interface{} `json:"highestRoiLeads"`
	}
	if err := json.Unmarshal(data, &withRoi); err == nil && len(withRoi.HighestRoiLeads) > 0 {
		return withRoi.HighestRoiLeads, nil
	}
	var arr []interface{}
	if err := json.Unmarshal(data, &arr); err == nil && len(arr) > 0 {
		return arr, nil
	}
	return nil, fmt.Errorf("无法解析 lead list")
}

func mergeLeaderboardFromRecommend(rec json.RawMessage) (pnl, roi []interface{}, ok bool) {
	var data struct {
		HighestPnlLeads []interface{} `json:"highestPnlLeads"`
		HighestRoiLeads []interface{} `json:"highestRoiLeads"`
	}
	if err := json.Unmarshal(rec, &data); err != nil {
		return nil, nil, false
	}
	return data.HighestPnlLeads, data.HighestRoiLeads, true
}

func buildLeaderboardJSON(pnlLeads, roiLeads []interface{}) (json.RawMessage, error) {
	if pnlLeads == nil {
		pnlLeads = []interface{}{}
	}
	if roiLeads == nil {
		roiLeads = []interface{}{}
	}
	out, err := json.Marshal(map[string]interface{}{
		"highestPnlLeads": pnlLeads,
		"highestRoiLeads": roiLeads,
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func fetchBinanceLeaderboard() (json.RawMessage, error) {
	pnlRaw, pnlErr := fetchBinanceLeadPortfolioListPNL()
	roiRaw, roiErr := fetchBinanceLeadPortfolioListROI()

	var pnlLeads, roiLeads []interface{}
	if pnlErr == nil {
		pnlLeads, pnlErr = leadsFromListData(pnlRaw)
	}
	if roiErr == nil {
		roiLeads, roiErr = leadsFromListData(roiRaw)
	}

	if pnlErr == nil && roiErr == nil {
		return buildLeaderboardJSON(pnlLeads, roiLeads)
	}

	rec, recErr := fetchBinanceRecommendLeadList()
	if recErr != nil {
		if pnlErr != nil && roiErr != nil {
			return nil, fmt.Errorf("排行榜请求失败: PNL=%v, ROI=%v", pnlErr, roiErr)
		}
		return buildLeaderboardJSON(pnlLeads, roiLeads)
	}
	recPnl, recRoi, _ := mergeLeaderboardFromRecommend(rec)
	if pnlErr != nil {
		pnlLeads = recPnl
	}
	if roiErr != nil {
		roiLeads = recRoi
	}
	return buildLeaderboardJSON(pnlLeads, roiLeads)
}

func fetchBinanceLeadOrders(portfolioID string, pageSize int) (*LeadOrderHistory, error) {
	if portfolioID == "" {
		return nil, fmt.Errorf("portfolio_id 为空")
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	now := time.Now()
	endTime := now.UnixMilli()
	startTime := now.AddDate(0, -1, 0).UnixMilli()

	payload := map[string]interface{}{
		"portfolioId": portfolioID,
		"startTime":   startTime,
		"endTime":     endTime,
		"pageSize":    pageSize,
	}
	binanceData, err := postBinanceCopyTradeBapi(
		"/bapi/futures/v1/friendly/future/copy-trade/lead-portfolio/order-history",
		payload,
	)
	if err != nil {
		return nil, err
	}

	var data struct {
		List []LeadOrder `json:"list"`
	}
	if err := json.Unmarshal(binanceData, &data); err != nil {
		return nil, fmt.Errorf("解析订单数据失败: %w", err)
	}
	return &LeadOrderHistory{Data: &data}, nil
}

func writeOrderCache(portfolioID string, history *LeadOrderHistory) error {
	if history == nil {
		return fmt.Errorf("订单数据为空")
	}
	if err := ensureCopyTradingCacheDir(); err != nil {
		return err
	}
	payload := map[string]interface{}{
		"ts":   time.Now().Unix(),
		"data": history.Data,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if err := os.WriteFile(orderCachePath(portfolioID), data, 0o644); err != nil {
		return err
	}
	setLeadOrdersMemCache(portfolioID, history)
	return nil
}

func readOrderCache(portfolioID string) (*LeadOrderHistory, error) {
	data, err := os.ReadFile(orderCachePath(portfolioID))
	if err != nil {
		return nil, err
	}
	var history LeadOrderHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}
	if history.Data == nil {
		return nil, fmt.Errorf("缓存中无 data 字段")
	}
	return &history, nil
}

func setLeadOrdersMemCache(portfolioID string, history *LeadOrderHistory) {
	leadOrdersMemCacheMu.Lock()
	defer leadOrdersMemCacheMu.Unlock()
	leadOrdersMemCache[portfolioID] = &leadOrdersCacheEntry{history: history, fetchedAt: time.Now()}
}

func getLeadOrdersCached(portfolioID string, maxAge time.Duration) (*LeadOrderHistory, bool) {
	leadOrdersMemCacheMu.RLock()
	entry, ok := leadOrdersMemCache[portfolioID]
	leadOrdersMemCacheMu.RUnlock()
	if ok && time.Since(entry.fetchedAt) < maxAge {
		return entry.history, true
	}
	if history, err := readOrderCache(portfolioID); err == nil {
		setLeadOrdersMemCache(portfolioID, history)
		return history, true
	}
	return nil, false
}

func refreshOrderCache(portfolioID string, pageSize int) (*LeadOrderHistory, error) {
	history, err := fetchBinanceLeadOrders(portfolioID, pageSize)
	if err != nil {
		return nil, err
	}
	_ = writeOrderCache(portfolioID, history)
	return history, nil
}

func getLeadOrdersForMonitor(portfolioID string, forceRefresh bool) (*LeadOrderHistory, error) {
	if !forceRefresh {
		if h, ok := getLeadOrdersCached(portfolioID, 60*time.Second); ok {
			return h, nil
		}
	}
	return refreshOrderCache(portfolioID, 20)
}

func isLeadOpenOrder(positionSide, side string) bool {
	return (positionSide == "LONG" && side == "BUY") ||
		(positionSide == "SHORT" && side == "SELL") ||
		(positionSide == "BOTH" && side == "SELL")
}

func isLeadCloseOrder(positionSide, side string) bool {
	return (positionSide == "LONG" && side == "SELL") ||
		(positionSide == "SHORT" && side == "BUY") ||
		(positionSide == "BOTH" && side == "BUY")
}

func leadActionLabel(positionSide, side string) string {
	if isLeadOpenOrder(positionSide, side) {
		if positionSide == "LONG" || (positionSide == "BOTH" && side == "BUY") {
			return "lead_open"
		}
		return "lead_open"
	}
	if isLeadCloseOrder(positionSide, side) {
		return "lead_close"
	}
	return "lead_other"
}

func leadActionDisplay(positionSide, side string) string {
	if isLeadOpenOrder(positionSide, side) {
		if positionSide == "LONG" || side == "BUY" {
			return "开多"
		}
		return "开空"
	}
	if isLeadCloseOrder(positionSide, side) {
		if positionSide == "LONG" {
			return "平多"
		}
		return "平空"
	}
	return side + " " + positionSide
}
