package strategy

import (
	"testing"

	"nofx/indicator"
)

func makeKlines(closes []float64) []indicator.Kline {
	klines := make([]indicator.Kline, len(closes))
	for i, c := range closes {
		klines[i] = indicator.Kline{
			Open:   c - 0.5,
			High:   c + 1,
			Low:    c - 1,
			Close:  c,
			Volume: 1000,
		}
	}
	return klines
}

func TestEngine_EntryConditionRSI(t *testing.T) {
	// 构造 RSI 较低的价格序列（持续下跌）
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = 100 - float64(i)*2
	}
	klines := makeKlines(closes)

	cfg := StrategyConfig{
		Symbol:    "BTCUSDT",
		Direction: "long_only",
		EntryConditions: []ConditionGroup{
			{{Indicator: "RSI14", Operator: "<", Value: 35}},
		},
	}

	engine := NewEngine()
	signal := engine.Evaluate(cfg, klines)
	if signal.Type != SignalBuy && signal.Type != SignalHold {
		t.Errorf("unexpected signal type: %s", signal.Type)
	}
}

func TestEngine_ExitCondition(t *testing.T) {
	closes := make([]float64, 30)
	for i := range closes {
		closes[i] = 100 + float64(i)
	}
	klines := makeKlines(closes)

	cfg := StrategyConfig{
		Symbol:    "ETHUSDT",
		Direction: "long_only",
		ExitConditions: []ConditionGroup{
			{{Indicator: "RSI14", Operator: ">", Value: 50}},
		},
	}

	engine := NewEngine()
	signal := engine.Evaluate(cfg, klines)
	if signal.Type == SignalBuy {
		t.Error("exit condition should not produce BUY when only exit rules defined")
	}
}

func TestEvaluateGroups_OR(t *testing.T) {
	engine := NewEngine()
	values := IndicatorValues{Current: map[string]float64{"RSI14": 25}, Price: 100}
	prev := IndicatorValues{Current: map[string]float64{"RSI14": 28}, Price: 99}

	groups := []ConditionGroup{
		{{Indicator: "RSI14", Operator: "<", Value: 30}},
		{{Indicator: "RSI14", Operator: ">", Value: 80}},
	}
	if !engine.evaluateGroups(groups, values, prev) {
		t.Error("expected first OR group to match")
	}
}

func TestEvaluateGroups_AND(t *testing.T) {
	engine := NewEngine()
	values := IndicatorValues{Current: map[string]float64{"RSI14": 25, "EMA20": 100}, Price: 100}
	prev := values

	group := ConditionGroup{
		{Indicator: "RSI14", Operator: "<", Value: 30},
		{Indicator: "CLOSE", Operator: ">", Value: 50},
	}
	if !engine.evaluateGroup(group, values, prev) {
		t.Error("expected AND group to match")
	}
}

func TestGetBuiltinTemplate(t *testing.T) {
	tpl, ok := GetBuiltinTemplate("macd_golden_cross")
	if !ok {
		t.Fatal("builtin template not found")
	}
	if tpl.Name == "" {
		t.Error("template name empty")
	}
}
