package logger

import (
	"math"
	"testing"
)

func TestComputeEquityMetrics(t *testing.T) {
	m := ComputeEquityMetrics(AccountSnapshot{
		TotalBalance:          1000,
		TotalUnrealizedProfit: 50,
		InitialBalance:        900,
	}, 800)

	if m.TotalEquity != 1050 {
		t.Fatalf("TotalEquity=%v, want 1050", m.TotalEquity)
	}
	if m.TotalPnL != 150 {
		t.Fatalf("TotalPnL=%v, want 150", m.TotalPnL)
	}
	wantPct := (150.0 / 900.0) * 100
	if math.Abs(m.TotalPnLPct-wantPct) > 1e-9 {
		t.Fatalf("TotalPnLPct=%v, want %v", m.TotalPnLPct, wantPct)
	}
	if m.InitialBase != 900 {
		t.Fatalf("InitialBase=%v, want 900", m.InitialBase)
	}
}

func TestComputeEquityMetricsFallbackInitial(t *testing.T) {
	m := ComputeEquityMetrics(AccountSnapshot{
		TotalBalance:          1200,
		TotalUnrealizedProfit: -100,
	}, 1000)

	if m.TotalEquity != 1100 {
		t.Fatalf("TotalEquity=%v, want 1100", m.TotalEquity)
	}
	if m.TotalPnL != 100 {
		t.Fatalf("TotalPnL=%v, want 100", m.TotalPnL)
	}
}
