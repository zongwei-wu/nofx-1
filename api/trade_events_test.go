package api

import (
	"testing"
	"time"

	"nofx/logger"
)

func TestClassifyAIAction(t *testing.T) {
	if classifyAIAction("open_long", false) != "open" {
		t.Fatal("expected open")
	}
	if classifyAIAction("open_long", true) != "add" {
		t.Fatal("expected add")
	}
	if classifyAIAction("partial_close", true) != "reduce" {
		t.Fatal("expected reduce")
	}
	if classifyAIAction("close_long", true) != "close" {
		t.Fatal("expected close")
	}
}

func TestLeadOrderEventsAddReduce(t *testing.T) {
	orders := []LeadOrder{
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "BUY", ExecutedQty: 1, AvgPrice: 100, OrderTime: 1000},
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "BUY", ExecutedQty: 0.5, AvgPrice: 101, OrderTime: 2000},
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "SELL", ExecutedQty: 0.5, AvgPrice: 102, OrderTime: 3000},
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "SELL", ExecutedQty: 1, AvgPrice: 103, OrderTime: 4000},
	}
	evs := leadOrderEvents("p1", "trader1", orders)
	if len(evs) != 4 {
		t.Fatalf("want 4 events got %d", len(evs))
	}
	if evs[0].Type != "open" || evs[1].Type != "add" || evs[2].Type != "reduce" || evs[3].Type != "close" {
		t.Fatalf("types: %s %s %s %s", evs[0].Type, evs[1].Type, evs[2].Type, evs[3].Type)
	}
}

func TestIsAITradeAction(t *testing.T) {
	if !isAITradeAction("open_long") || !isAITradeAction("partial_close") {
		t.Fatal("expected trade actions")
	}
	if isAITradeAction("hold") || isAITradeAction("update_stop_loss") {
		t.Fatal("expected non-trade actions rejected")
	}
}

func TestCollectSymbolsFromAIRecords(t *testing.T) {
	records := []*logger.DecisionRecord{{
		Positions: []logger.PositionSnapshot{
			{Symbol: "ETHUSDT", PositionAmt: 1},
			{Symbol: "BTCUSDT", PositionAmt: 0},
		},
		Decisions: []logger.DecisionAction{
			{Action: "open_long", Symbol: "SOLUSDT", Success: true},
			{Action: "hold", Symbol: "XRPUSDT", Success: true},
			{Action: "open_long", Symbol: "ADAUSDT", Success: false},
		},
	}}
	syms := collectSymbolsFromAIRecords(records)
	if len(syms) != 2 {
		t.Fatalf("want 2 symbols got %v", syms)
	}
}

func TestAIEventsFromDecisionsSkipsNonTradeActions(t *testing.T) {
	records := []*logger.DecisionRecord{{
		Timestamp: time.Unix(0, 0),
		Decisions: []logger.DecisionAction{
			{Action: "hold", Symbol: "BTCUSDT", Success: true, Timestamp: time.Unix(1, 0)},
			{Action: "update_stop_loss", Symbol: "BTCUSDT", Success: true, Timestamp: time.Unix(2, 0)},
			{Action: "open_long", Symbol: "BTCUSDT", Quantity: 1, Price: 100, Success: true, Timestamp: time.Unix(3, 0)},
		},
	}}
	evs := aiEventsFromDecisions(records, "")
	if len(evs) != 1 || evs[0].Type != "open" {
		t.Fatalf("want 1 open event got %v", evs)
	}
}

func TestAICopyTradeEventsFromDB(t *testing.T) {
	db, userID, _ := setupCopyTradeSettingsDB(t)
	defer db.Close()

	s := &Server{database: db}
	traderID := "trader-copy-1"

	s.insertCopyTradeAIDecision(copyTradeAIDecisionParams{
		UserID:         userID,
		RunID:          42,
		PortfolioID:    "pf-1",
		Nickname:       "LeadA",
		Symbol:         "BTCUSDT",
		RecommendedQty: 0.1,
		ActionTaken:    "copied_open",
		Success:        true,
		AITraderID:     traderID,
		AITraderName:   "跟单分析员",
	})

	_, err := db.DB().Exec(`
		INSERT INTO copy_trade_records
		(user_id, portfolio_id, nickname, order_id, symbol, side, position_side,
		 executed_qty, avg_price, close_price, status, lead_order_time, copy_time, close_time)
		VALUES (?, 'pf-1', 'LeadA', 'ord-1', 'BTCUSDT', 'BUY', 'LONG',
		        0.1, 50000, 51000, 'CLOSED', 1000, datetime('now'), datetime('now'))`,
		userID)
	if err != nil {
		t.Fatal(err)
	}

	evs, err := s.aiCopyTradeEventsFromDB(userID, traderID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) < 2 {
		t.Fatalf("want at least 2 events (open+close), got %d", len(evs))
	}
	hasOpen, hasClose := false, false
	for _, e := range evs {
		if e.Source != "ai_copy_trade" {
			t.Fatalf("unexpected source %s", e.Source)
		}
		if e.Type == "open" {
			hasOpen = true
		}
		if e.Type == "close" {
			hasClose = true
		}
	}
	if !hasOpen || !hasClose {
		t.Fatalf("missing open/close: open=%v close=%v events=%+v", hasOpen, hasClose, evs)
	}
}

func TestAIEventsFromDecisions(t *testing.T) {
	records := []*logger.DecisionRecord{{
		Timestamp: time.Unix(0, 0),
		Decisions: []logger.DecisionAction{
			{Action: "open_long", Symbol: "BTCUSDT", Quantity: 1, Price: 100, Success: true, Timestamp: time.Unix(1, 0)},
			{Action: "partial_close", Symbol: "BTCUSDT", Quantity: 0.5, Price: 105, Success: true, Timestamp: time.Unix(2, 0)},
			{Action: "close_long", Symbol: "BTCUSDT", Quantity: 0, Price: 110, Success: true, Timestamp: time.Unix(3, 0)},
		},
	}}
	evs := aiEventsFromDecisions(records, "")
	if len(evs) != 3 {
		t.Fatalf("want 3 got %d", len(evs))
	}
}

