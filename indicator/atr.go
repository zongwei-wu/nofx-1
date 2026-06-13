package indicator

import "math"

// ATR 平均真实波幅（Wilder 平滑）
func ATR(klines []Kline, period int) IndicatorResult {
	if period <= 0 || len(klines) <= period {
		return IndicatorResult{}
	}

	trs := make([]float64, len(klines))
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close
		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)
		trs[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	series := make([]float64, 0, len(klines)-period)
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)
	series = append(series, atr)

	for i := period + 1; i < len(klines); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
		series = append(series, atr)
	}

	current := 0.0
	if len(series) > 0 {
		current = series[len(series)-1]
	}
	return IndicatorResult{Current: current, Series: series}
}
