package api

import (
	"strings"
)

func symbolBaseAsset(symbol string) string {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if strings.HasSuffix(symbol, "USDT") {
		return strings.TrimSuffix(symbol, "USDT")
	}
	return symbol
}

// leadContractsToBaseQty 带单「张」→ 执行交易所下单标的币数量（如 ETH）
func leadContractsToBaseQty(conv contractQtyConverter, symbol string, contracts float64) (float64, error) {
	return leadContractsToBaseQtyGeneric(conv, symbol, contracts)
}

// resolveCopyRecommendedQty 将 AI/默认比例结果规范为交易所下单数量（标的币）
func resolveCopyRecommendedQty(
	conv contractQtyConverter,
	symbol string,
	leadContracts float64,
	aiQty float64,
	aiRatio float64,
	defaultRatio float64,
) float64 {
	leadBase, _ := leadContractsToBaseQty(conv, symbol, leadContracts)
	if leadBase <= 0 {
		leadBase = leadContracts
	}

	recommended := leadBase * defaultRatio
	if aiRatio > 0 && aiRatio <= 1 {
		recommended = leadBase * aiRatio
	}

	if aiQty <= 0 {
		return recommended
	}

	// AI 常按「张」返回数量；若明显大于标的币口径则换算
	if aiQty > leadBase*1.5 {
		if converted, err := leadContractsToBaseQty(conv, symbol, aiQty); err == nil && converted > 0 {
			return converted
		}
	}
	return aiQty
}

func copyTradePromptParams(
	symbol, direction, nickname string,
	leadPrice, leadContracts float64,
	conv contractQtyConverter,
) copyTradeAIPromptParams {
	base := symbolBaseAsset(symbol)
	leadBase, _ := leadContractsToBaseQty(conv, symbol, leadContracts)
	return copyTradeAIPromptParams{
		Symbol:           symbol,
		BaseAsset:        base,
		Direction:        direction,
		LeadNickname:     nickname,
		LeadPrice:        leadPrice,
		LeadQtyContracts: leadContracts,
		LeadQtyBase:      leadBase,
	}
}
