package gateway

import (
	"fmt"
	"strconv"
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

func formatTradeQuantity(t trader.Trader, symbol string, quantity float64) (float64, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("quantity 必须大于 0")
	}
	formatted, err := t.FormatQuantity(symbol, quantity)
	if err != nil {
		return quantity, nil
	}
	parsed, err := strconv.ParseFloat(formatted, 64)
	if err != nil {
		return quantity, nil
	}
	return parsed, nil
}

// ExecuteTrade 执行网关交易（独立于 AI 交易员路径）
func ExecuteTrade(t trader.Trader, req TradeRequest) (map[string]interface{}, error) {
	symbol := NormalizeSymbol(req.Symbol)
	action := strings.ToLower(strings.TrimSpace(req.Action))

	switch action {
	case "open_long", "add_long":
		qty, err := formatTradeQuantity(t, symbol, req.Quantity)
		if err != nil {
			return nil, err
		}
		if action == "open_long" {
			positions, _ := t.GetPositions()
			if hasPositionSide(positions, symbol, "long") {
				return nil, fmt.Errorf("%s 已有多仓，请使用 add_long", symbol)
			}
		}
		leverage := req.Leverage
		if leverage <= 0 {
			leverage = 5
		}
		return t.OpenLong(symbol, qty, leverage)

	case "open_short", "add_short":
		qty, err := formatTradeQuantity(t, symbol, req.Quantity)
		if err != nil {
			return nil, err
		}
		if action == "open_short" {
			positions, _ := t.GetPositions()
			if hasPositionSide(positions, symbol, "short") {
				return nil, fmt.Errorf("%s 已有空仓，请使用 add_short", symbol)
			}
		}
		leverage := req.Leverage
		if leverage <= 0 {
			leverage = 5
		}
		return t.OpenShort(symbol, qty, leverage)

	case "close_long", "reduce_long":
		qty := req.Quantity
		if action == "close_long" {
			qty = 0
		} else if qty <= 0 {
			return nil, fmt.Errorf("reduce_long 需要 quantity > 0")
		} else {
			var err error
			qty, err = formatTradeQuantity(t, symbol, qty)
			if err != nil {
				return nil, err
			}
		}
		return t.CloseLong(symbol, qty)

	case "close_short", "reduce_short":
		qty := req.Quantity
		if action == "close_short" {
			qty = 0
		} else if qty <= 0 {
			return nil, fmt.Errorf("reduce_short 需要 quantity > 0")
		} else {
			var err error
			qty, err = formatTradeQuantity(t, symbol, qty)
			if err != nil {
				return nil, err
			}
		}
		return t.CloseShort(symbol, qty)

	default:
		return nil, fmt.Errorf("不支持的 action: %s", req.Action)
	}
}
