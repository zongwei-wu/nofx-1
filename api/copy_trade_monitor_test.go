package api

import (
	"testing"
)

func TestBuildLatestLeadActions_DedupesSameSymbolDirection(t *testing.T) {
	orders := []LeadOrder{
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 1000},
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "SELL", OrderTime: 3000},
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 2000},
	}

	out := buildLatestLeadActions(orders, 5)
	if len(out) != 1 {
		t.Fatalf("expected 1 action, got %d", len(out))
	}
	if out[0]["order_time"] != int64(3000) {
		t.Fatalf("order_time = %v, want 3000", out[0]["order_time"])
	}
	if out[0]["display"] != "平多" {
		t.Fatalf("display = %v, want 平多", out[0]["display"])
	}
}

func TestBuildLatestLeadActions_KeepsDifferentDirections(t *testing.T) {
	orders := []LeadOrder{
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 1000},
		{Symbol: "BTCUSDT", PositionSide: "SHORT", Side: "SELL", OrderTime: 2000},
	}

	out := buildLatestLeadActions(orders, 5)
	if len(out) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(out))
	}
}

func TestBuildLatestLeadActions_SortsByOrderTimeDesc(t *testing.T) {
	orders := []LeadOrder{
		{Symbol: "ETHUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 1000},
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 5000},
		{Symbol: "SOLUSDT", PositionSide: "SHORT", Side: "SELL", OrderTime: 3000},
	}

	out := buildLatestLeadActions(orders, 5)
	if len(out) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(out))
	}
	if out[0]["symbol"] != "BTCUSDT" {
		t.Fatalf("first symbol = %v, want BTCUSDT", out[0]["symbol"])
	}
	if out[1]["symbol"] != "SOLUSDT" {
		t.Fatalf("second symbol = %v, want SOLUSDT", out[1]["symbol"])
	}
	if out[2]["symbol"] != "ETHUSDT" {
		t.Fatalf("third symbol = %v, want ETHUSDT", out[2]["symbol"])
	}
}

func TestBuildLatestLeadActions_Limit(t *testing.T) {
	orders := []LeadOrder{
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 1000},
		{Symbol: "ETHUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 2000},
		{Symbol: "SOLUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 3000},
		{Symbol: "BNBUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 4000},
		{Symbol: "XRPUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 5000},
		{Symbol: "DOGEUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 6000},
	}

	out := buildLatestLeadActions(orders, 5)
	if len(out) != 5 {
		t.Fatalf("expected 5 actions, got %d", len(out))
	}
	if out[0]["symbol"] != "DOGEUSDT" {
		t.Fatalf("first symbol = %v, want DOGEUSDT", out[0]["symbol"])
	}
}

func TestBuildLatestLeadActions_SkipsEmptySymbol(t *testing.T) {
	orders := []LeadOrder{
		{Symbol: "", PositionSide: "LONG", Side: "BUY", OrderTime: 1000},
		{Symbol: "BTCUSDT", PositionSide: "LONG", Side: "BUY", OrderTime: 2000},
	}

	out := buildLatestLeadActions(orders, 5)
	if len(out) != 1 {
		t.Fatalf("expected 1 action, got %d", len(out))
	}
}
