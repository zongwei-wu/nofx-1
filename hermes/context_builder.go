package hermes

import (
	"fmt"
	"strings"
	"time"

	"nofx/config"
	"nofx/decision"
	"nofx/logger"
	"nofx/market"
	"nofx/trader"
)

// BuildContext 构建 Hermes AI 决策上下文
func BuildContext(
	t trader.Trader,
	settings *config.HermesSettings,
	callCount int,
	startTime time.Time,
	decisionLogger logger.IDecisionLogger,
) (*decision.Context, error) {
	balance, err := t.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("获取账户余额失败: %w", err)
	}

	totalWalletBalance := 0.0
	totalUnrealizedProfit := 0.0
	availableBalance := 0.0
	if wallet, ok := balance["totalWalletBalance"].(float64); ok {
		totalWalletBalance = wallet
	} else if wallet, ok := balance["wallet_balance"].(float64); ok {
		totalWalletBalance = wallet
	}
	if unrealized, ok := balance["totalUnrealizedProfit"].(float64); ok {
		totalUnrealizedProfit = unrealized
	} else if unrealized, ok := balance["unrealized_profit"].(float64); ok {
		totalUnrealizedProfit = unrealized
	}
	if avail, ok := balance["availableBalance"].(float64); ok {
		availableBalance = avail
	} else if avail, ok := balance["available_balance"].(float64); ok {
		availableBalance = avail
	}

	totalEquity := totalWalletBalance + totalUnrealizedProfit

	positions, err := t.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("获取持仓失败: %w", err)
	}

	var positionInfos []decision.PositionInfo
	totalMarginUsed := 0.0
	for _, pos := range positions {
		symbol, _ := pos["symbol"].(string)
		side, _ := pos["side"].(string)
		entryPrice, _ := pos["entryPrice"].(float64)
		markPrice, _ := pos["markPrice"].(float64)
		quantity, _ := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity
		}
		if quantity == 0 {
			continue
		}
		unrealizedPnl, _ := pos["unRealizedProfit"].(float64)
		liquidationPrice, _ := pos["liquidationPrice"].(float64)
		leverage := 10
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}
		marginUsed := (quantity * markPrice) / float64(leverage)
		totalMarginUsed += marginUsed
		pnlPct := 0.0
		if marginUsed > 0 {
			pnlPct = (unrealizedPnl / marginUsed) * 100
		}
		positionInfos = append(positionInfos, decision.PositionInfo{
			Symbol:           symbol,
			Side:             side,
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			Quantity:         quantity,
			Leverage:         leverage,
			UnrealizedPnL:    unrealizedPnl,
			UnrealizedPnLPct: pnlPct,
			LiquidationPrice: liquidationPrice,
			MarginUsed:       marginUsed,
			UpdateTime:       time.Now().UnixMilli(),
		})
	}

	candidateCoins := buildCandidateCoins(settings)
	initialBalance := settings.InitialBalance
	if initialBalance <= 0 {
		initialBalance = totalEquity
	}
	totalPnL := totalEquity - initialBalance
	totalPnLPct := 0.0
	if initialBalance > 0 {
		totalPnLPct = (totalPnL / initialBalance) * 100
	}
	marginUsedPct := 0.0
	if totalEquity > 0 {
		marginUsedPct = (totalMarginUsed / totalEquity) * 100
	}

	var performance interface{}
	if decisionLogger != nil {
		if p, err := decisionLogger.AnalyzePerformance(100); err == nil {
			performance = p
		}
	}

	return &decision.Context{
		CurrentTime:     time.Now().Format("2006-01-02 15:04:05"),
		RuntimeMinutes:  int(time.Since(startTime).Minutes()),
		CallCount:       callCount,
		BTCETHLeverage:  settings.BTCETHLeverage,
		AltcoinLeverage: settings.AltcoinLeverage,
		Account: decision.AccountInfo{
			TotalEquity:      totalEquity,
			AvailableBalance: availableBalance,
			UnrealizedPnL:    totalUnrealizedProfit,
			TotalPnL:         totalPnL,
			TotalPnLPct:      totalPnLPct,
			MarginUsed:       totalMarginUsed,
			MarginUsedPct:    marginUsedPct,
			PositionCount:    len(positionInfos),
		},
		Positions:      positionInfos,
		CandidateCoins: candidateCoins,
		Performance:    performance,
	}, nil
}

func buildCandidateCoins(settings *config.HermesSettings) []decision.CandidateCoin {
	if len(settings.TradingCoins) > 0 {
		out := make([]decision.CandidateCoin, 0, len(settings.TradingCoins))
		for _, s := range settings.TradingCoins {
			sym := market.Normalize(s)
			if sym == "" {
				continue
			}
			out = append(out, decision.CandidateCoin{Symbol: sym, Sources: []string{"hermes"}})
		}
		return out
	}
	defaults := []string{"BTCUSDT", "ETHUSDT"}
	out := make([]decision.CandidateCoin, 0, len(defaults))
	for _, sym := range defaults {
		out = append(out, decision.CandidateCoin{Symbol: sym, Sources: []string{"hermes_default"}})
	}
	return out
}

// ResolveOpenAction 若已有同向持仓则将 open 映射为 add
func ResolveOpenAction(t trader.Trader, d *decision.Decision) (string, float64, error) {
	action := strings.ToLower(d.Action)
	symbol := NormalizeSymbol(d.Symbol)
	qty := d.PositionSizeUSD
	if qty <= 0 {
		md, err := market.Get(symbol)
		if err != nil {
			return "", 0, err
		}
		qty = 100 / md.CurrentPrice
		if qty <= 0 {
			return "", 0, fmt.Errorf("无法计算开仓数量")
		}
	} else {
		md, err := market.Get(symbol)
		if err != nil {
			return "", 0, err
		}
		qty = qty / md.CurrentPrice
	}

	switch action {
	case "open_long":
		positions, _ := t.GetPositions()
		if hasPositionSide(positions, symbol, "long") {
			action = "add_long"
		}
	case "open_short":
		positions, _ := t.GetPositions()
		if hasPositionSide(positions, symbol, "short") {
			action = "add_short"
		}
	}
	return action, qty, nil
}
