package api

import "testing"

func TestLeadCloseOurPositionSide(t *testing.T) {
	tests := []struct {
		posSide, side, want string
	}{
		{"LONG", "SELL", "LONG"},
		{"SHORT", "BUY", "SHORT"},
		{"BOTH", "BUY", "SHORT"},
		{"BOTH", "SELL", "LONG"},
	}
	for _, tt := range tests {
		if got := leadCloseOurPositionSide(tt.posSide, tt.side); got != tt.want {
			t.Errorf("leadCloseOurPositionSide(%s,%s)=%s want %s", tt.posSide, tt.side, got, tt.want)
		}
	}
}
