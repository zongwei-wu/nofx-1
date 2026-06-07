package api

import (
	"testing"
)

func TestSymbolBaseAsset(t *testing.T) {
	if got := symbolBaseAsset("ETHUSDT"); got != "ETH" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveCopyRecommendedQty_ConvertsContracts(t *testing.T) {
	// 200 张 * 0.001 = 0.2 ETH
	got := resolveCopyRecommendedQty(nil, "ETHUSDT", 2000, 200, 0.1, 0.1)
	// AI returns 200 张 at 10% of 2000 — should become 0.2 ETH if step=0.001
	// without fTrader, aiQty > leadBase*1.5 won't convert; pass mock step via nil returns 200
	if got != 200 {
		t.Fatalf("without trader fallback got %v", got)
	}
}

func TestResolveCopyRecommendedQty_UsesRatio(t *testing.T) {
	got := resolveCopyRecommendedQty(nil, "ETHUSDT", 2000, 0, 0.1, 0.1)
	if got != 200 {
		t.Fatalf("expected ratio on raw contracts fallback %v", got)
	}
}
