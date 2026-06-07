package api

import "strings"

// computeCopyTradeRealizedPnL 按开仓价与平仓价计算已实现盈亏（USDT）。
func computeCopyTradeRealizedPnL(positionSide string, qty, entryPrice, closePrice float64) float64 {
	if qty <= 0 || entryPrice <= 0 || closePrice <= 0 {
		return 0
	}
	switch strings.ToUpper(strings.TrimSpace(positionSide)) {
	case "SHORT":
		return (entryPrice - closePrice) * qty
	default:
		return (closePrice - entryPrice) * qty
	}
}

func copyTradePnLKind(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "OPEN":
		return "unrealized"
	case "CLOSED":
		return "realized"
	default:
		return ""
	}
}
