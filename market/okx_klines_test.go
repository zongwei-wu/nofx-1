package market

import "testing"

func TestMapIntervalToOKXBar(t *testing.T) {
	bar, err := mapIntervalToOKXBar("1h")
	if err != nil || bar != "1H" {
		t.Fatalf("got %s %v", bar, err)
	}
	_, err = mapIntervalToOKXBar("invalid")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConvertSymbolToOKXInstID(t *testing.T) {
	if got := convertSymbolToOKXInstID("BTCUSDT"); got != "BTC-USDT-SWAP" {
		t.Fatalf("got %s", got)
	}
}
