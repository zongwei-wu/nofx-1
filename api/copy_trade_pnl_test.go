package api

import "testing"

func TestComputeCopyTradeRealizedPnL(t *testing.T) {
	tests := []struct {
		side   string
		qty    float64
		entry  float64
		close  float64
		expect float64
	}{
		{"LONG", 2, 100, 110, 20},
		{"LONG", 2, 100, 90, -20},
		{"SHORT", 1, 200, 180, 20},
		{"SHORT", 1, 200, 220, -20},
		{"LONG", 0, 100, 110, 0},
	}
	for _, tt := range tests {
		got := computeCopyTradeRealizedPnL(tt.side, tt.qty, tt.entry, tt.close)
		if got != tt.expect {
			t.Fatalf("%s qty=%v entry=%v close=%v: got %v want %v",
				tt.side, tt.qty, tt.entry, tt.close, got, tt.expect)
		}
	}
}

func TestCopyTradePnLKind(t *testing.T) {
	if copyTradePnLKind("OPEN") != "unrealized" {
		t.Fatal("OPEN should be unrealized")
	}
	if copyTradePnLKind("CLOSED") != "realized" {
		t.Fatal("CLOSED should be realized")
	}
}
