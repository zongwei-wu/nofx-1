package gateway

import (
	"fmt"
	"strings"

	"nofx/trader"
)

// TradeRequest 网关交易请求
type TradeRequest struct {
	Action   string
	Symbol   string
	Quantity float64
	Leverage int
}

func NormalizeSymbol(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return symbol
	}
	if !strings.HasSuffix(symbol, "USDT") {
		symbol += "USDT"
	}
	return symbol
}

func hasPositionSide(positions []map[string]interface{}, symbol, side string) bool {
	for _, pos := range positions {
		if pos["symbol"] == symbol && pos["side"] == side {
			return true
		}
	}
	return false
}

// ExecuteTrade 执行网关交易（独立于 AI 交易员路径）
// 数量由各 trader 方法内部的 baseToSize/FormatQuantity 自行处理，不在此处转换
func ExecuteTrade(t trader.Trader, req TradeRequest) (map[string]interface{}, error) {
	symbol := NormalizeSymbol(req.Symbol)
	action := strings.ToLower(strings.TrimSpace(req.Action))

	switch action {
	case "open_long", "add_long":
		if action == "open_long" {
			positions, _ := t.GetPositions()
			if hasPositionSide(positions, symbol, "long") {
				return nil, fmt.Errorf("%s 已有多仓，请使用 add_long", symbol)
			}
		}
		if action == "add_long" {
			return t.OpenLong(symbol, req.Quantity, 0)
		}
		leverage := req.Leverage
		if leverage <= 0 {
			leverage = 5
		}
		return t.OpenLong(symbol, req.Quantity, leverage)

	case "open_short", "add_short":
		if action == "open_short" {
			positions, _ := t.GetPositions()
			if hasPositionSide(positions, symbol, "short") {
				return nil, fmt.Errorf("%s 已有空仓，请使用 add_short", symbol)
			}
		}
		if action == "add_short" {
			return t.OpenShort(symbol, req.Quantity, 0)
		}
		leverage := req.Leverage
		if leverage <= 0 {
			leverage = 5
		}
		return t.OpenShort(symbol, req.Quantity, leverage)

	case "close_long", "reduce_long":
		qty := req.Quantity
		if action == "close_long" {
			qty = 0
		}
		return t.CloseLong(symbol, qty)

	case "close_short", "reduce_short":
		qty := req.Quantity
		if action == "close_short" {
			qty = 0
		}
		return t.CloseShort(symbol, qty)

	default:
		return nil, fmt.Errorf("不支持的 action: %s", req.Action)
	}
}
