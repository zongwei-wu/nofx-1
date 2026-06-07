package api

import (
	"fmt"
	"strings"

	"nofx/market"
	"nofx/trader"
)

type copyTradeAIAccountContext struct {
	TotalEquity           float64
	AvailableBalance      float64
	TotalUnrealizedProfit float64
	OpenCount             int
	PositionsSummary      string
}

type copyTradeAIPromptParams struct {
	Symbol            string
	BaseAsset         string
	Direction         string
	LeadNickname      string
	LeadPrice         float64
	LeadQtyContracts  float64
	LeadQtyBase       float64
}

func fetchCopyTradeAIAccountContext(t trader.Trader) copyTradeAIAccountContext {
	ctx := copyTradeAIAccountContext{
		PositionsSummary: "  无持仓\n",
	}
	if t == nil {
		return ctx
	}

	invalidateTraderAccountCache(t)

	balance, err := t.GetBalance()
	if err == nil && balance != nil {
		if wb, ok := balance["totalWalletBalance"].(float64); ok {
			ctx.TotalEquity = wb
		}
		if ab, ok := balance["availableBalance"].(float64); ok {
			ctx.AvailableBalance = ab
		}
		if upnl, ok := balance["totalUnrealizedProfit"].(float64); ok {
			ctx.TotalUnrealizedProfit = upnl
		}
	}

	positions, err := t.GetPositions()
	if err == nil && positions != nil {
		var lines []string
		for _, p := range positions {
			sym, _ := p["symbol"].(string)
			amt, _ := p["positionAmt"].(string)
			if amtF, ok := p["positionAmt"].(float64); ok {
				amt = fmt.Sprintf("%v", amtF)
			}
			upnl, _ := p["unRealizedProfit"].(string)
			if upnlF, ok := p["unRealizedProfit"].(float64); ok {
				upnl = fmt.Sprintf("%v", upnlF)
			}
			markPrice, _ := p["markPrice"].(float64)
			if sym != "" && amt != "" && amt != "0" {
				if markPrice > 0 {
					lines = append(lines, fmt.Sprintf("  - %s 数量=%s 标记价=%.4f 未实现盈亏=%s", sym, amt, markPrice, upnl))
				} else {
					lines = append(lines, fmt.Sprintf("  - %s 数量=%s 未实现盈亏=%s", sym, amt, upnl))
				}
				ctx.OpenCount++
			}
		}
		if len(lines) > 0 {
			ctx.PositionsSummary = strings.Join(lines, "\n") + "\n"
		}
	}

	return ctx
}

func fetchCopyTradeAIMarketSectionFromAPI(symbol string) string {
	client := market.NewAPIClient()
	klines3m, err3m := client.GetKlines(symbol, "3m", 30)
	klines4h, err4h := client.GetKlines(symbol, "4h", 12)
	if (err3m != nil || len(klines3m) == 0) && (err4h != nil || len(klines4h) == 0) {
		if err3m != nil {
			return fmt.Sprintf("市场数据暂不可用: %v\n", err3m)
		}
		return fmt.Sprintf("市场数据暂不可用: %v\n", err4h)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %s 最新K线 (API实时)\n", symbol))
	if len(klines3m) > 0 {
		sb.WriteString(fmt.Sprintf("3m 最近%d根 (O/H/L/C/V):\n", len(klines3m)))
		for _, k := range klines3m {
			sb.WriteString(fmt.Sprintf("  %.4f / %.4f / %.4f / %.4f / %.2f\n", k.Open, k.High, k.Low, k.Close, k.Volume))
		}
		sb.WriteString("\n")
	}
	if len(klines4h) > 0 {
		sb.WriteString(fmt.Sprintf("4h 最近%d根 (O/H/L/C/V):\n", len(klines4h)))
		for _, k := range klines4h {
			sb.WriteString(fmt.Sprintf("  %.4f / %.4f / %.4f / %.4f / %.2f\n", k.Open, k.High, k.Low, k.Close, k.Volume))
		}
	}
	return sb.String()
}

func fetchCopyTradeAIMarketSection(symbol string) string {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return "市场数据暂不可用: 无效交易对\n"
	}
	symbol = market.Normalize(symbol)

	if market.WSMonitorCli != nil {
		if md, err := market.Get(symbol); err == nil && md != nil {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("## %s 最新市场数据 (实时K线)\n", symbol))
			sb.WriteString(fmt.Sprintf("1h涨跌: %+.2f%% | 4h涨跌: %+.2f%%\n\n", md.PriceChange1h, md.PriceChange4h))
			sb.WriteString(market.Format(md))
			return sb.String()
		}
	}

	return fetchCopyTradeAIMarketSectionFromAPI(symbol)
}

func buildCopyTradeRiskPrompt(account copyTradeAIAccountContext, marketSection string, p copyTradeAIPromptParams, exchangeID string) string {
	exchangeLabel := exchangeOrderQtyLabel(exchangeID)
	return fmt.Sprintf(`你是一个专业的加密货币期货交易风控分析师。请分析以下跟单交易请求。

## 最新账户资产 (实时)
- 账户总权益: %.2f USDT
- 可用余额: %.2f USDT
- 未实现盈亏: %+.2f USDT
- 当前持仓数: %d
- 持仓详情:
%s
## 跟单标的最新行情
%s
## 跟单请求
- 带单员: %s
- 交易对: %s
- 操作: %s
- 带单员开仓价格: $%.2f
- 带单员开仓数量: %.4f 张（折合 %.6f %s）

## 分析要求
1. 结合最新K线与账户资金评估是否适合跟单
2. 考虑当前持仓集中度与行情波动风险
3. 建议合理跟单仓位（相对带单员仓位）
4. 风险过高时建议不跟单
5. recommended_qty 必须为%s下单数量（标的币数量，单位 %s），不要使用「张」

请以JSON输出: {"feasible":true/false,"reasoning":"理由","recommended_ratio":0.1,"recommended_qty":0.5,"suggestion":"建议"}`,
		account.TotalEquity,
		account.AvailableBalance,
		account.TotalUnrealizedProfit,
		account.OpenCount,
		account.PositionsSummary,
		marketSection,
		p.LeadNickname,
		p.Symbol,
		p.Direction,
		p.LeadPrice,
		p.LeadQtyContracts,
		p.LeadQtyBase,
		p.BaseAsset,
		exchangeLabel,
		p.BaseAsset,
	)
}

func buildCopyTradeRiskPromptDetailed(account copyTradeAIAccountContext, marketSection string, p copyTradeAIPromptParams, exchangeID string) string {
	exchangeLabel := exchangeOrderQtyLabel(exchangeID)
	return fmt.Sprintf(`你是一个专业的加密货币期货交易风控分析师。请分析以下跟单交易请求，给出仓位大小建议。

## 最新账户资产 (实时)
- 账户总权益: %.2f USDT
- 可用余额: %.2f USDT
- 未实现盈亏: %+.2f USDT
- 当前持仓数量: %d
- 当前持仓详情:
%s
## 跟单标的最新行情
%s
## 跟单请求
- 带单员: %s
- 交易对: %s
- 操作: %s
- 带单员开仓价格: $%.2f
- 带单员开仓数量: %.4f 张（折合 %.6f %s）

## 分析要求
1. 结合最新K线与账户可用资金评估是否足够开仓
2. 考虑当前持仓集中度风险
3. 建议一个合理的跟单仓位比例（相对于带单员仓位）
4. 如果风险过高，建议不跟单
5. recommended_qty 必须为%s下单数量（标的币数量，单位 %s），不要使用「张」

请以JSON格式输出，包含以下字段：
{
  "feasible": true/false,
  "reasoning": "详细分析理由",
  "recommended_ratio": 0.1,
  "recommended_qty": 0.5,
  "max_risk_usd": 100.0,
  "suggestion": "简短建议"
}`,
		account.TotalEquity,
		account.AvailableBalance,
		account.TotalUnrealizedProfit,
		account.OpenCount,
		account.PositionsSummary,
		marketSection,
		p.LeadNickname,
		p.Symbol,
		p.Direction,
		p.LeadPrice,
		p.LeadQtyContracts,
		p.LeadQtyBase,
		p.BaseAsset,
		exchangeLabel,
		p.BaseAsset,
	)
}
