package api

import (
	"strings"
	"testing"
)

func TestBuildCopyTradeRiskPrompt_IncludesMarketAndAccount(t *testing.T) {
	account := copyTradeAIAccountContext{
		TotalEquity:      10000,
		AvailableBalance: 5000,
		OpenCount:        1,
		PositionsSummary: "  - BTCUSDT 数量=0.1\n",
	}
	market := "## BTCUSDT 最新市场数据\ncurrent_price = 65000\n"
	prompt := buildCopyTradeRiskPrompt(account, market, copyTradeAIPromptParams{
		Symbol: "BTCUSDT", Direction: "买入开多", LeadNickname: "LeadA",
		LeadPrice: 65000, LeadQty: 1,
	})

	for _, want := range []string{
		"最新账户资产",
		"10000.00 USDT",
		"最新行情",
		"current_price = 65000",
		"LeadA",
		"BTCUSDT",
	} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}

func TestFetchCopyTradeAIMarketSection_InvalidSymbol(t *testing.T) {
	out := fetchCopyTradeAIMarketSection("")
	if !strings.Contains(out, "无效交易对") {
		t.Fatalf("unexpected: %s", out)
	}
}
