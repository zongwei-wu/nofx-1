package hermes

import (
	"fmt"
	"strings"

	"nofx/config"
	"nofx/decision"
	"nofx/trader"
)

// ExecuteDecision 将 AI 决策映射为 Hermes 交易执行
func ExecuteDecision(t trader.Trader, d *decision.Decision) (map[string]interface{}, error) {
	if d == nil {
		return nil, fmt.Errorf("决策为空")
	}
	action := strings.ToLower(strings.TrimSpace(d.Action))
	symbol := NormalizeSymbol(d.Symbol)

	switch action {
	case "open_long", "open_short":
		resolved, qty, err := ResolveOpenAction(t, d)
		if err != nil {
			return nil, err
		}
		return ExecuteTrade(t, TradeRequest{
			Action:   resolved,
			Symbol:   symbol,
			Quantity: qty,
			Leverage: d.Leverage,
		})

	case "close_long", "close_short":
		return ExecuteTrade(t, TradeRequest{Action: action, Symbol: symbol})

	case "partial_close":
		positions, err := t.GetPositions()
		if err != nil {
			return nil, err
		}
		var side string
		var qty float64
		for _, pos := range positions {
			if pos["symbol"] != symbol {
				continue
			}
			side, _ = pos["side"].(string)
			amt, _ := pos["positionAmt"].(float64)
			if amt < 0 {
				amt = -amt
			}
			pct := d.ClosePercentage
			if pct <= 0 || pct > 100 {
				pct = 50
			}
			qty = amt * (pct / 100.0)
			break
		}
		if side == "" || qty <= 0 {
			return nil, fmt.Errorf("未找到可部分平仓的持仓: %s", symbol)
		}
		reduceAction := "reduce_long"
		if side == "short" {
			reduceAction = "reduce_short"
		}
		return ExecuteTrade(t, TradeRequest{Action: reduceAction, Symbol: symbol, Quantity: qty})

	case "update_stop_loss":
		positions, _ := t.GetPositions()
		qty, side := positionQtySide(positions, symbol)
		if qty <= 0 {
			return nil, fmt.Errorf("无持仓可设止损: %s", symbol)
		}
		price := d.NewStopLoss
		if price <= 0 {
			price = d.StopLoss
		}
		if err := t.SetStopLoss(symbol, strings.ToUpper(side), qty, price); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "ok", "action": action}, nil

	case "update_take_profit":
		positions, _ := t.GetPositions()
		qty, side := positionQtySide(positions, symbol)
		if qty <= 0 {
			return nil, fmt.Errorf("无持仓可设止盈: %s", symbol)
		}
		price := d.NewTakeProfit
		if price <= 0 {
			price = d.TakeProfit
		}
		if err := t.SetTakeProfit(symbol, strings.ToUpper(side), qty, price); err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "ok", "action": action}, nil

	case "hold", "wait":
		return map[string]interface{}{"status": "skipped", "action": action}, nil

	default:
		return nil, fmt.Errorf("Hermes 不支持的 action: %s", d.Action)
	}
}

func positionQtySide(positions []map[string]interface{}, symbol string) (float64, string) {
	for _, pos := range positions {
		if pos["symbol"] != symbol {
			continue
		}
		side, _ := pos["side"].(string)
		amt, _ := pos["positionAmt"].(float64)
		if amt < 0 {
			amt = -amt
		}
		return amt, side
	}
	return 0, ""
}

// SetInitialBalanceIfZero 首次运行时用净值填充 initial_balance
func SetInitialBalanceIfZero(settings *config.HermesSettings, equity float64) bool {
	if settings.InitialBalance > 0 {
		return false
	}
	settings.InitialBalance = equity
	return true
}
