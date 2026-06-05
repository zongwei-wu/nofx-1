package api

import "testing"

func TestNormalizeSymbolUSDT(t *testing.T) {
	if normalizeSymbolUSDT("eth") != "ETHUSDT" {
		t.Fatal("expected ETHUSDT")
	}
	if normalizeSymbolUSDT("BTCUSDT") != "BTCUSDT" {
		t.Fatal("expected BTCUSDT")
	}
	if normalizeSymbolUSDT("  ") != "" {
		t.Fatal("expected empty")
	}
}

func TestMergeSymbolCatalog(t *testing.T) {
	out := mergeSymbolCatalog(
		[]string{"BTCUSDT", "ETHUSDT"},
		[]string{"ETHUSDT", "SOLUSDT"},
	)
	if len(out) != 3 {
		t.Fatalf("want 3 got %d: %v", len(out), out)
	}
	if out[0] != "ETHUSDT" || out[1] != "SOLUSDT" || out[2] != "BTCUSDT" {
		t.Fatalf("unexpected order: %v", out)
	}
}
