package indicator

// EMA 指数移动平均
func EMA(klines []Kline, period int) IndicatorResult {
	if period <= 0 || len(klines) < period {
		return IndicatorResult{}
	}

	multiplier := 2.0 / float64(period+1)
	series := make([]float64, 0, len(klines)-period+1)

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += klines[i].Close
	}
	ema := sum / float64(period)
	series = append(series, ema)

	for i := period; i < len(klines); i++ {
		ema = (klines[i].Close-ema)*multiplier + ema
		series = append(series, ema)
	}

	return IndicatorResult{Current: ema, Series: series}
}

// emaFromSeries 对任意序列计算 EMA（内部复用）
func emaFromSeries(values []float64, period int) []float64 {
	if period <= 0 || len(values) < period {
		return nil
	}
	multiplier := 2.0 / float64(period+1)
	out := make([]float64, 0, len(values)-period+1)

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	ema := sum / float64(period)
	out = append(out, ema)

	for i := period; i < len(values); i++ {
		ema = (values[i]-ema)*multiplier + ema
		out = append(out, ema)
	}
	return out
}
