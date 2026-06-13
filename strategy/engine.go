package strategy

import (
	"fmt"
	"nofx/indicator"
	"strings"
)

// IndicatorValues 指标计算结果快照
type IndicatorValues struct {
	Current map[string]float64
	Prev    map[string]float64
	Price   float64
}

// Engine 策略条件评估引擎
type Engine struct{}

// NewEngine 创建策略引擎
func NewEngine() *Engine {
	return &Engine{}
}

// Evaluate 评估策略信号
func (e *Engine) Evaluate(cfg StrategyConfig, klines []indicator.Kline) Signal {
	if len(klines) == 0 {
		return Signal{Type: SignalHold, Symbol: cfg.Symbol}
	}

	values := e.computeAllIndicators(cfg, klines)
	prevKlines := klines
	if len(klines) > 1 {
		prevKlines = klines[:len(klines)-1]
	}
	prevValues := e.computeAllIndicators(cfg, prevKlines)

	price := klines[len(klines)-1].Close
	signal := Signal{
		Type:       SignalHold,
		Symbol:     cfg.Symbol,
		Price:      price,
		Indicators: values.Current,
		Reason:     "no condition matched",
	}

	// 离场条件优先
	if len(cfg.ExitConditions) > 0 && e.evaluateGroups(cfg.ExitConditions, values, prevValues) {
		signal.Type = SignalSell
		signal.Reason = "exit conditions met"
		return signal
	}

	// 入场条件
	if e.evaluateGroups(cfg.EntryConditions, values, prevValues) {
		if cfg.Direction == "short_only" {
			signal.Type = SignalSell
			signal.Reason = "entry conditions met (short)"
		} else {
			signal.Type = SignalBuy
			signal.Reason = "entry conditions met (long)"
		}
	}

	return signal
}

func (e *Engine) evaluateGroups(groups []ConditionGroup, values, prev IndicatorValues) bool {
	if len(groups) == 0 {
		return false
	}
	for _, group := range groups {
		if e.evaluateGroup(group, values, prev) {
			return true
		}
	}
	return false
}

func (e *Engine) evaluateGroup(group ConditionGroup, values, prev IndicatorValues) bool {
	if len(group) == 0 {
		return false
	}
	for _, cond := range group {
		if !e.evaluateCondition(cond, values, prev) {
			return false
		}
	}
	return true
}

func (e *Engine) evaluateCondition(cond Condition, values, prev IndicatorValues) bool {
	cur := resolveIndicatorValue(cond.Indicator, values)
	prevVal := resolveIndicatorValue(cond.Indicator, prev)

	switch cond.Operator {
	case ">":
		return cur > cond.Value
	case "<":
		return cur < cond.Value
	case ">=":
		return cur >= cond.Value
	case "<=":
		return cur <= cond.Value
	case "cross_above":
		return prevVal <= cond.Value && cur > cond.Value
	case "cross_below":
		return prevVal >= cond.Value && cur < cond.Value
	default:
		return false
	}
}

func resolveIndicatorValue(name string, values IndicatorValues) float64 {
	key := strings.ToUpper(strings.TrimSpace(name))
	if key == "PRICE" || key == "CLOSE" {
		return values.Price
	}
	if v, ok := values.Current[key]; ok {
		return v
	}
	return 0
}

func (e *Engine) computeAllIndicators(cfg StrategyConfig, klines []indicator.Kline) IndicatorValues {
	names := extractIndicators(cfg)
	current := make(map[string]float64)
	prev := make(map[string]float64)

	price := 0.0
	if len(klines) > 0 {
		price = klines[len(klines)-1].Close
	}

	for _, name := range names {
		cur, prv := computeIndicatorPair(name, klines)
		current[strings.ToUpper(name)] = cur
		if prv != 0 || cur != 0 {
			prev[strings.ToUpper(name)] = prv
		}
	}

	return IndicatorValues{Current: current, Prev: prev, Price: price}
}

func extractIndicators(cfg StrategyConfig) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(groups []ConditionGroup) {
		for _, g := range groups {
			for _, c := range g {
				key := strings.ToUpper(strings.TrimSpace(c.Indicator))
				if key != "" && !seen[key] {
					seen[key] = true
					out = append(out, key)
				}
			}
		}
	}
	add(cfg.EntryConditions)
	add(cfg.AddPositionConditions)
	add(cfg.ExitConditions)
	return out
}

func computeIndicatorPair(name string, klines []indicator.Kline) (current, prev float64) {
	key := strings.ToUpper(strings.TrimSpace(name))

	if key == "PRICE" || key == "CLOSE" {
		if len(klines) == 0 {
			return 0, 0
		}
		cur := klines[len(klines)-1].Close
		prv := cur
		if len(klines) > 1 {
			prv = klines[len(klines)-2].Close
		}
		return cur, prv
	}

	current = computeSingleIndicator(key, klines)
	if len(klines) > 1 {
		prev = computeSingleIndicator(key, klines[:len(klines)-1])
	}
	return
}

func computeSingleIndicator(name string, klines []indicator.Kline) float64 {
	switch {
	case strings.HasPrefix(name, "RSI"):
		period := parsePeriod(name, "RSI", 14)
		return indicator.RSI(klines, period).Current
	case strings.HasPrefix(name, "EMA"):
		period := parsePeriod(name, "EMA", 20)
		return indicator.EMA(klines, period).Current
	case strings.HasPrefix(name, "SMA"):
		period := parsePeriod(name, "SMA", 20)
		return indicator.SMA(klines, period).Current
	case strings.HasPrefix(name, "ATR"):
		period := parsePeriod(name, "ATR", 14)
		return indicator.ATR(klines, period).Current
	case strings.HasPrefix(name, "MFI"):
		period := parsePeriod(name, "MFI", 14)
		return indicator.MFI(klines, period).Current
	case name == "MACD":
		return indicator.MACD(klines, 12, 26, 9).MACD
	case name == "MACD_SIGNAL":
		return indicator.MACD(klines, 12, 26, 9).Signal
	case name == "MACD_HIST":
		return indicator.MACD(klines, 12, 26, 9).Histogram
	case name == "BOLL_UPPER":
		return indicator.Bollinger(klines, 20, 2.0).Upper
	case name == "BOLL_MIDDLE":
		return indicator.Bollinger(klines, 20, 2.0).Middle
	case name == "BOLL_LOWER":
		return indicator.Bollinger(klines, 20, 2.0).Lower
	case name == "STOCH_K":
		return indicator.Stochastic(klines, 14, 3).K
	case name == "STOCH_D":
		return indicator.Stochastic(klines, 14, 3).D
	case name == "ADX":
		return indicator.ADX(klines, 14).ADX
	case name == "PLUS_DI":
		return indicator.ADX(klines, 14).PlusDI
	case name == "MINUS_DI":
		return indicator.ADX(klines, 14).MinusDI
	case name == "ICHIMOKU_TENKAN":
		return indicator.Ichimoku(klines).Tenkan
	case name == "ICHIMOKU_KIJUN":
		return indicator.Ichimoku(klines).Kijun
	case name == "OBV":
		series := indicator.OBV(klines)
		if len(series) > 0 {
			return series[len(series)-1]
		}
	case name == "VWAP":
		series := indicator.VWAP(klines)
		if len(series) > 0 {
			return series[len(series)-1]
		}
	}
	return 0
}

func parsePeriod(name, prefix string, defaultPeriod int) int {
	suffix := strings.TrimPrefix(strings.ToUpper(name), prefix)
	if suffix == "" {
		return defaultPeriod
	}
	var period int
	if _, err := fmt.Sscanf(suffix, "%d", &period); err != nil || period <= 0 {
		return defaultPeriod
	}
	return period
}
