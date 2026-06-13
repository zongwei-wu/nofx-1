package backtest

import (
	"encoding/json"
	"fmt"
)

// ToJSON 序列化回测结果为 JSON
func (r *BacktestResult) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SummaryMarkdown 生成简要 Markdown 报告
func (r *BacktestResult) SummaryMarkdown() string {
	m := r.Metrics
	return fmt.Sprintf(`# 回测报告: %s

- 交易对: %s (%s)
- 初始资金: %.2f USDT
- 最终权益: %.2f USDT
- 总收益率: %.2f%%
- 年化收益: %.2f%%
- 最大回撤: %.2f%%
- 夏普比率: %.2f
- 胜率: %.2f%%
- 盈亏比: %.2f
- 总交易次数: %d
`,
		r.StrategyName, r.Symbol, r.Timeframe,
		r.InitialCapital, r.FinalEquity,
		m.TotalReturnPct, m.AnnualizedReturnPct, m.MaxDrawdownPct,
		m.SharpeRatio, m.WinRatePct, m.ProfitFactor, m.TotalTrades,
	)
}
