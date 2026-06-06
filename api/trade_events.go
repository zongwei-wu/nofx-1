package api

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"nofx/logger"

	"github.com/gin-gonic/gin"
)

// TradeEvent 统一交易事件（开/加/减/平）
type TradeEvent struct {
	Symbol string  `json:"symbol"`
	Time   int64   `json:"time"`
	Type   string  `json:"type"`
	Side   string  `json:"side"`
	Qty    float64 `json:"qty"`
	Price  float64 `json:"price"`
	Source string  `json:"source"`
	Detail string  `json:"detail"`
}

func normalizeTradeSymbol(sym string) string {
	sym = strings.ToUpper(strings.TrimSpace(sym))
	if sym == "" {
		return ""
	}
	if !strings.HasSuffix(sym, "USDT") {
		sym += "USDT"
	}
	return sym
}

func tradeEventTypeLabel(t string) string {
	switch t {
	case "open":
		return "开仓"
	case "add":
		return "加仓"
	case "reduce":
		return "减仓"
	case "close":
		return "平仓"
	default:
		return t
	}
}

func positionSideFromAction(action string) string {
	if strings.Contains(action, "short") {
		return "SHORT"
	}
	return "LONG"
}

func isAITradeAction(action string) bool {
	switch action {
	case "open_long", "open_short", "close_long", "close_short", "partial_close":
		return true
	default:
		return false
	}
}

func classifyAIAction(action string, hadOpen bool) string {
	switch action {
	case "open_long", "open_short":
		if hadOpen {
			return "add"
		}
		return "open"
	case "partial_close":
		return "reduce"
	case "close_long", "close_short":
		return "close"
	default:
		return ""
	}
}

func aiEventsFromDecisions(records []*logger.DecisionRecord, symbolFilter string) []TradeEvent {
	openByKey := make(map[string]bool)
	var events []TradeEvent

	for _, rec := range records {
		for _, d := range rec.Decisions {
			if !d.Success || d.Symbol == "" || !isAITradeAction(d.Action) {
				continue
			}
			sym := normalizeTradeSymbol(d.Symbol)
			if symbolFilter != "" && sym != symbolFilter {
				continue
			}
			evType := classifyAIAction(d.Action, openByKey[sym+"|"+positionSideFromAction(d.Action)])
			if evType == "" {
				continue
			}
			ts := d.Timestamp.UnixMilli()
			if ts == 0 {
				ts = rec.Timestamp.UnixMilli()
			}
			key := sym + "|" + positionSideFromAction(d.Action)
			if evType == "open" || evType == "add" {
				openByKey[key] = true
			}
			if evType == "close" {
				delete(openByKey, key)
			}
			if evType == "reduce" && d.Quantity <= 0 {
				continue
			}
			events = append(events, TradeEvent{
				Symbol: sym,
				Time:   ts,
				Type:   evType,
				Side:   positionSideFromAction(d.Action),
				Qty:    d.Quantity,
				Price:  d.Price,
				Source: "ai_trader",
				Detail: fmt.Sprintf("%s %s", tradeEventTypeLabel(evType), d.Action),
			})
		}
	}
	return events
}

func parseTimeToMs(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		if t > 1e12 {
			return t
		}
		return t * 1000
	case float64:
		n := int64(t)
		if n > 1e12 {
			return n
		}
		return n * 1000
	case string:
		if t == "" {
			return 0
		}
		if parsed, err := time.Parse("2006-01-02 15:04:05", t); err == nil {
			return parsed.UnixMilli()
		}
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed.UnixMilli()
		}
	case time.Time:
		if t.IsZero() {
			return 0
		}
		return t.UnixMilli()
	}
	return 0
}

func (s *Server) copyTradeEventsFromRecords(userID, portfolioID, symbolFilter string) ([]TradeEvent, error) {
	q := `
		SELECT portfolio_id, nickname, symbol, side, position_side, executed_qty, avg_price,
		       status, lead_order_time, copy_time, close_time, close_price
		FROM copy_trade_records WHERE user_id = ?`
	args := []interface{}{userID}
	if portfolioID != "" {
		q += " AND portfolio_id = ?"
		args = append(args, portfolioID)
	}
	if symbolFilter != "" {
		q += " AND symbol = ?"
		args = append(args, symbolFilter)
	}
	q += " ORDER BY COALESCE(copy_time, lead_order_time) ASC"

	rows, err := s.database.DB().Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TradeEvent
	for rows.Next() {
		var pid, nick, sym, side, posSide, status string
		var qty, avgPrice, closePrice float64
		var leadTime int64
		var copyTime, closeTime interface{}
		if err := rows.Scan(&pid, &nick, &sym, &side, &posSide, &qty, &avgPrice,
			&status, &leadTime, &copyTime, &closeTime, &closePrice); err != nil {
			continue
		}
		sym = normalizeTradeSymbol(sym)
		openMs := parseTimeToMs(copyTime)
		if openMs == 0 {
			openMs = leadTime
		}
		if status == "OPEN" || status == "FAILED" {
			if openMs > 0 && avgPrice > 0 {
				events = append(events, TradeEvent{
					Symbol: sym, Time: openMs, Type: "open", Side: posSide,
					Qty: qty, Price: avgPrice, Source: "copy_trade",
					Detail: fmt.Sprintf("跟单开仓 · %s", nick),
				})
			}
		}
		if status == "CLOSED" {
			closeMs := parseTimeToMs(closeTime)
			cp := closePrice
			if cp <= 0 {
				cp = avgPrice
			}
			if closeMs > 0 {
				events = append(events, TradeEvent{
					Symbol: sym, Time: closeMs, Type: "close", Side: posSide,
					Qty: qty, Price: cp, Source: "copy_trade",
					Detail: fmt.Sprintf("跟单平仓 · %s", nick),
				})
			}
		}
		_ = pid
		_ = side
	}
	return events, nil
}

func leadOrderPositionSide(o LeadOrder) string {
	if isLeadOpenOrder(o.PositionSide, o.Side) {
		ps := o.PositionSide
		if ps == "BOTH" {
			return "SHORT"
		}
		return ps
	}
	return leadCloseOurPositionSide(o.PositionSide, o.Side)
}

func leadOrderEvents(portfolioID, nickname string, orders []LeadOrder) []TradeEvent {
	posQty := make(map[string]float64)
	var events []TradeEvent

	sorted := make([]LeadOrder, len(orders))
	copy(sorted, orders)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].OrderTime < sorted[j].OrderTime
	})

	for _, o := range sorted {
		sym := normalizeTradeSymbol(o.Symbol)
		posSide := leadOrderPositionSide(o)
		key := sym + "|" + posSide
		prev := posQty[key]

		var evType string
		switch {
		case isLeadCloseOrder(o.PositionSide, o.Side):
			if prev <= o.ExecutedQty+1e-9 {
				evType = "close"
				posQty[key] = 0
			} else {
				evType = "reduce"
				posQty[key] = prev - o.ExecutedQty
			}
		case isLeadOpenOrder(o.PositionSide, o.Side):
			if prev < 0.001 {
				evType = "open"
			} else {
				evType = "add"
			}
			posQty[key] = prev + o.ExecutedQty
		default:
			continue
		}

		events = append(events, TradeEvent{
			Symbol: sym,
			Time:   o.OrderTime,
			Type:   evType,
			Side:   posSide,
			Qty:    o.ExecutedQty,
			Price:  o.AvgPrice,
			Source: "lead",
			Detail: fmt.Sprintf("带单%s · %s", tradeEventTypeLabel(evType), nickname),
		})
	}
	_ = portfolioID
	return events
}

func dedupeEvents(events []TradeEvent) []TradeEvent {
	seen := make(map[string]bool)
	var out []TradeEvent
	for _, e := range events {
		key := fmt.Sprintf("%s|%d|%s|%s|%.4f", e.Symbol, e.Time, e.Type, e.Source, e.Price)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Time < out[j].Time
	})
	return out
}

func collectSymbolsFromAIRecords(records []*logger.DecisionRecord) []string {
	seen := make(map[string]bool)
	for _, rec := range records {
		for _, pos := range rec.Positions {
			if sym := normalizeTradeSymbol(pos.Symbol); sym != "" && pos.PositionAmt != 0 {
				seen[sym] = true
			}
		}
		for _, d := range rec.Decisions {
			if d.Success && d.Symbol != "" && isAITradeAction(d.Action) {
				seen[normalizeTradeSymbol(d.Symbol)] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func (s *Server) distinctCopyTradeSymbols(userID, portfolioID string) ([]string, error) {
	q := `SELECT DISTINCT symbol FROM copy_trade_records WHERE user_id = ?`
	args := []interface{}{userID}
	if portfolioID != "" {
		q += ` AND portfolio_id = ?`
		args = append(args, portfolioID)
	}
	q += ` ORDER BY symbol`
	rows, err := s.database.DB().Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var sym string
		if rows.Scan(&sym) == nil {
			if norm := normalizeTradeSymbol(sym); norm != "" {
				out = append(out, norm)
			}
		}
	}
	return out, nil
}

func mergeSymbolLists(lists ...[]string) []string {
	seen := make(map[string]bool)
	for _, list := range lists {
		for _, sym := range list {
			if norm := normalizeTradeSymbol(sym); norm != "" {
				seen[norm] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for sym := range seen {
		out = append(out, sym)
	}
	sort.Strings(out)
	return out
}

func (s *Server) aiCopyTradePortfolioIDs(userID, traderID string) (map[string]bool, error) {
	rows, err := s.database.DB().Query(`
		SELECT DISTINCT portfolio_id
		FROM copy_trade_ai_decisions
		WHERE user_id = ? AND ai_trader_id = ? AND success = 1 AND action_taken = 'copied_open'`,
		userID, traderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var pid string
		if rows.Scan(&pid) == nil && pid != "" {
			out[pid] = true
		}
	}
	return out, nil
}

func (s *Server) lookupCopyRecordPrice(userID, portfolioID, symbol string, aroundMs int64) float64 {
	var avgPrice float64
	_ = s.database.DB().QueryRow(`
		SELECT avg_price FROM copy_trade_records
		WHERE user_id = ? AND portfolio_id = ? AND symbol = ? AND avg_price > 0
		ORDER BY ABS(COALESCE(copy_time, lead_order_time) - ?) ASC
		LIMIT 1`,
		userID, portfolioID, symbol, aroundMs).Scan(&avgPrice)
	return avgPrice
}

func (s *Server) aiCopyTradeEventsFromDB(userID, traderID, symbolFilter string) ([]TradeEvent, error) {
	portfolios, err := s.aiCopyTradePortfolioIDs(userID, traderID)
	if err != nil {
		return nil, err
	}

	var events []TradeEvent

	// 开仓：AI 跟单风控成功执行
	openQ := `
		SELECT portfolio_id, nickname, symbol, recommended_qty, created_at
		FROM copy_trade_ai_decisions
		WHERE user_id = ? AND ai_trader_id = ? AND success = 1 AND action_taken = 'copied_open'`
	openArgs := []interface{}{userID, traderID}
	if symbolFilter != "" {
		openQ += ` AND symbol = ?`
		openArgs = append(openArgs, symbolFilter)
	}
	openQ += ` ORDER BY created_at ASC`

	openRows, err := s.database.DB().Query(openQ, openArgs...)
	if err != nil {
		return nil, err
	}
	defer openRows.Close()

	for openRows.Next() {
		var pid, nick, sym string
		var qty float64
		var createdAt interface{}
		if err := openRows.Scan(&pid, &nick, &sym, &qty, &createdAt); err != nil {
			continue
		}
		sym = normalizeTradeSymbol(sym)
		ts := parseTimeToMs(createdAt)
		if ts == 0 {
			continue
		}
		price := s.lookupCopyRecordPrice(userID, pid, sym, ts/1000)
		events = append(events, TradeEvent{
			Symbol: sym,
			Time:   ts,
			Type:   "open",
			Side:   "LONG",
			Qty:    qty,
			Price:  price,
			Source: "ai_copy_trade",
			Detail: fmt.Sprintf("AI跟单开仓 · %s", nick),
		})
		if sym != "" {
			portfolios[pid] = true
		}
	}

	if len(portfolios) == 0 {
		return events, nil
	}

	// 平仓：该 AI 曾成功跟单开仓的组合下的已平仓记录
	closeQ := `
		SELECT portfolio_id, nickname, symbol, position_side, executed_qty, close_price, avg_price, close_time
		FROM copy_trade_records
		WHERE user_id = ? AND status = 'CLOSED'`
	closeArgs := []interface{}{userID}
	if symbolFilter != "" {
		closeQ += ` AND symbol = ?`
		closeArgs = append(closeArgs, symbolFilter)
	}
	closeQ += ` ORDER BY close_time ASC`

	closeRows, err := s.database.DB().Query(closeQ, closeArgs...)
	if err != nil {
		return nil, err
	}
	defer closeRows.Close()

	for closeRows.Next() {
		var pid, nick, sym, posSide string
		var qty, closePrice, avgPrice float64
		var closeTime interface{}
		if err := closeRows.Scan(&pid, &nick, &sym, &posSide, &qty, &closePrice, &avgPrice, &closeTime); err != nil {
			continue
		}
		if !portfolios[pid] {
			continue
		}
		sym = normalizeTradeSymbol(sym)
		closeMs := parseTimeToMs(closeTime)
		if closeMs == 0 {
			continue
		}
		cp := closePrice
		if cp <= 0 {
			cp = avgPrice
		}
		events = append(events, TradeEvent{
			Symbol: sym,
			Time:   closeMs,
			Type:   "close",
			Side:   posSide,
			Qty:    qty,
			Price:  cp,
			Source: "ai_copy_trade",
			Detail: fmt.Sprintf("AI跟单平仓 · %s", nick),
		})
	}

	// 补充：run_events 中的 copied_close（同一 AI 参与的 run）
	runRows, err := s.database.DB().Query(`
		SELECT e.portfolio_id, e.nickname, e.symbol, e.lead_position_side, e.created_at
		FROM copy_trade_run_events e
		INNER JOIN copy_trade_ai_decisions d ON d.run_id = e.run_id
		WHERE e.user_id = ? AND d.ai_trader_id = ? AND d.success = 1
		  AND e.action = 'copied_close'`,
		userID, traderID)
	if err == nil {
		defer runRows.Close()
		for runRows.Next() {
			var pid, nick, sym, posSide string
			var createdAt interface{}
			if runRows.Scan(&pid, &nick, &sym, &posSide, &createdAt) != nil {
				continue
			}
			if !portfolios[pid] {
				continue
			}
			sym = normalizeTradeSymbol(sym)
			if symbolFilter != "" && sym != symbolFilter {
				continue
			}
			ts := parseTimeToMs(createdAt)
			if ts == 0 {
				continue
			}
			events = append(events, TradeEvent{
				Symbol: sym,
				Time:   ts,
				Type:   "close",
				Side:   posSide,
				Qty:    0,
				Price:  0,
				Source: "ai_copy_trade",
				Detail: fmt.Sprintf("AI跟单平仓 · %s", nick),
			})
		}
	}

	return events, nil
}

func (s *Server) distinctAICopyTradeSymbols(userID, traderID string) ([]string, error) {
	rows, err := s.database.DB().Query(`
		SELECT DISTINCT symbol FROM copy_trade_ai_decisions
		WHERE user_id = ? AND ai_trader_id = ? AND success = 1 AND action_taken = 'copied_open'
		ORDER BY symbol`, userID, traderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var sym string
		if rows.Scan(&sym) == nil {
			if norm := normalizeTradeSymbol(sym); norm != "" {
				out = append(out, norm)
			}
		}
	}
	return out, nil
}

func filterEventsByTime(events []TradeEvent, fromMs, toMs int64) []TradeEvent {
	if fromMs == 0 && toMs == 0 {
		return events
	}
	var out []TradeEvent
	for _, e := range events {
		if fromMs > 0 && e.Time < fromMs {
			continue
		}
		if toMs > 0 && e.Time > toMs {
			continue
		}
		out = append(out, e)
	}
	return out
}

// handleTradeEvents 统一交易事件
// GET /api/trade-events?source=copy_trade|ai_trader|ai_copy_trade&trader_id=&portfolio_id=&symbol=&from=&to=
func (s *Server) handleTradeEvents(c *gin.Context) {
	source := strings.TrimSpace(c.Query("source"))
	symbol := strings.ToUpper(strings.TrimSpace(c.Query("symbol")))
	if symbol != "" && !strings.HasSuffix(symbol, "USDT") {
		symbol += "USDT"
	}
	fromMs, _ := strconv.ParseInt(c.Query("from"), 10, 64)
	toMs, _ := strconv.ParseInt(c.Query("to"), 10, 64)

	var events []TradeEvent

	var symbolCandidates [][]string

	switch source {
	case "ai_trader":
		_, traderID, err := s.getTraderFromQuery(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		trader, err := s.traderManager.GetTrader(traderID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		records, err := trader.GetDecisionLogger().GetLatestRecords(1000)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		events = aiEventsFromDecisions(records, symbol)
		symbolCandidates = append(symbolCandidates, collectSymbolsFromAIRecords(records))

	case "ai_copy_trade":
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}
		_, traderID, err := s.getTraderFromQuery(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[trade-events] ai_copy_trade userID=%s traderID=%s symbol=%s", userID, traderID, symbol)
		aiCopyEv, err := s.aiCopyTradeEventsFromDB(userID, traderID, symbol)
		if err != nil {
			log.Printf("[trade-events] ai_copy_trade error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[trade-events] ai_copy_trade returned %d events", len(aiCopyEv))
		events = append(events, aiCopyEv...)
		if syms, err := s.distinctAICopyTradeSymbols(userID, traderID); err == nil {
			symbolCandidates = append(symbolCandidates, syms)
		}

	case "copy_trade", "":
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}
		portfolioID := strings.TrimSpace(c.Query("portfolio_id"))
		log.Printf("[trade-events] copy_trade userID=%s portfolioID=%s symbol=%s", userID, portfolioID, symbol)
		recEvents, err := s.copyTradeEventsFromRecords(userID, portfolioID, symbol)
		if err != nil {
			log.Printf("[trade-events] copy_trade error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[trade-events] copy_trade returned %d events", len(recEvents))
		events = append(events, recEvents...)
		if recSyms, err := s.distinctCopyTradeSymbols(userID, portfolioID); err == nil {
			symbolCandidates = append(symbolCandidates, recSyms)
		}

		if portfolioID != "" {
			var nickname string
			_ = s.database.DB().QueryRow(
				"SELECT nickname FROM copy_trade_config WHERE user_id=? AND portfolio_id=? LIMIT 1",
				userID, portfolioID).Scan(&nickname)
			if history, err := getLeadOrdersForMonitor(portfolioID, false); err == nil && history.Data != nil {
				leadEv := leadOrderEvents(portfolioID, nickname, history.Data.List)
				if symbol != "" {
					for _, e := range leadEv {
						if e.Symbol == symbol {
							events = append(events, e)
						}
					}
				} else {
					events = append(events, leadEv...)
				}
			}
		} else {
			cfgRows, _ := s.database.DB().Query(
				"SELECT portfolio_id, nickname FROM copy_trade_config WHERE user_id=? AND enabled=1",
				userID)
			if cfgRows != nil {
				for cfgRows.Next() {
					var pid, nick string
					if cfgRows.Scan(&pid, &nick) != nil {
						continue
					}
					if history, err := getLeadOrdersForMonitor(pid, false); err == nil && history.Data != nil {
						events = append(events, leadOrderEvents(pid, nick, history.Data.List)...)
					}
				}
				cfgRows.Close()
			}
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "source 须为 copy_trade、ai_trader 或 ai_copy_trade"})
		return
	}

	events = dedupeEvents(events)
	events = filterEventsByTime(events, fromMs, toMs)

	eventSyms := make([]string, 0)
	for _, e := range events {
		if e.Symbol != "" {
			eventSyms = append(eventSyms, e.Symbol)
		}
	}
	symList := mergeSymbolLists(append(symbolCandidates, eventSyms)...)

	c.JSON(http.StatusOK, gin.H{
		"events":  events,
		"symbols": symList,
	})
}
