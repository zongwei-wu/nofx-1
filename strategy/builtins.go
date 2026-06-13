package strategy

// BuiltinTemplates 内置策略模板
var BuiltinTemplates = map[string]StrategyConfig{
	"macd_golden_cross": {
		Name:      "MACD 金叉做多",
		Symbol:    "ETHUSDT",
		Timeframe: "1h",
		Direction: "long_only",
		EntryConditions: []ConditionGroup{
			{
				{Indicator: "MACD", Operator: "cross_above", Value: 0},
				{Indicator: "RSI14", Operator: "<", Value: 70},
			},
		},
		ExitConditions: []ConditionGroup{
			{
				{Indicator: "MACD", Operator: "cross_below", Value: 0},
			},
		},
		ExitRules: ExitRules{TakeProfitPct: 3, StopLossPct: 1.5},
		Risk:      RiskConfig{MaxPositions: 3, Leverage: 5, RiskPerTrade: 0.02},
	},
	"rsi_oversold": {
		Name:      "RSI 超卖反弹",
		Symbol:    "BTCUSDT",
		Timeframe: "4h",
		Direction: "long_only",
		EntryConditions: []ConditionGroup{
			{
				{Indicator: "RSI14", Operator: "<", Value: 30},
				{Indicator: "CLOSE", Operator: ">", Value: 0},
			},
		},
		ExitConditions: []ConditionGroup{
			{
				{Indicator: "RSI14", Operator: ">", Value: 70},
			},
		},
		ExitRules: ExitRules{TakeProfitPct: 5, StopLossPct: 2},
		Risk:      RiskConfig{MaxPositions: 2, Leverage: 3, RiskPerTrade: 0.01},
	},
}

// GetBuiltinTemplate 获取内置模板（返回副本）
func GetBuiltinTemplate(name string) (StrategyConfig, bool) {
	tpl, ok := BuiltinTemplates[name]
	if !ok {
		return StrategyConfig{}, false
	}
	return tpl, true
}
