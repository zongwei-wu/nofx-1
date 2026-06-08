package hermes

import (
	"testing"
)

func TestExecuteTrade_UnknownAction(t *testing.T) {
	_, err := ExecuteTrade(nil, TradeRequest{
		Action:   "unknown_action",
		Symbol:   "BTCUSDT",
		Quantity: 1,
	})
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
}

func TestNormalizeSymbol(t *testing.T) {
	if got := NormalizeSymbol("btc"); got != "BTCUSDT" {
		t.Fatalf("got %s", got)
	}
	if got := NormalizeSymbol("ETHUSDT"); got != "ETHUSDT" {
		t.Fatalf("got %s", got)
	}
}
