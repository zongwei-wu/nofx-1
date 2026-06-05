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

func TestCollectSymbolsFromAIRecords(t *testing.T) {
	records := []*logger.DecisionRecord{{
		Positions: []logger.PositionSnapshot{
			{Symbol: "ETHUSDT", PositionAmt: 1},
			{Symbol: "BTCUSDT", PositionAmt: 0},
		},
		Decisions: []logger.DecisionAction{
			{Symbol: "SOLUSDT", Success: true},
			{Symbol: "XRPUSDT", Success: false},
		},
	}}
	syms := collectSymbolsFromAIRecords(records)
	if len(syms) != 2 {
		t.Fatalf("want 2 symbols got %v", syms)
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
