package api

import (
	"fmt"
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
			if !d.Success || d.Symbol == "" {
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
			if d.Success && d.Symbol != "" {
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
// GET /api/trade-events?source=copy_trade|ai_trader&trader_id=&portfolio_id=&symbol=&from=&to=
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
		records, err := trader.GetDecisionLogger().GetLatestRecords(500)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		events = aiEventsFromDecisions(records, symbol)
		symbolCandidates = append(symbolCandidates, collectSymbolsFromAIRecords(records))

	case "copy_trade", "":
		userID := c.GetString("user_id")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
			return
		}
		portfolioID := strings.TrimSpace(c.Query("portfolio_id"))
		recEvents, err := s.copyTradeEventsFromRecords(userID, portfolioID, symbol)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "source 须为 copy_trade 或 ai_trader"})
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
