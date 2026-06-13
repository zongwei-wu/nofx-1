package indicator

// SMA 简单移动平均
func SMA(klines []Kline, period int) IndicatorResult {
	if period <= 0 || len(klines) < period {
		return IndicatorResult{}
	}

	series := make([]float64, 0, len(klines)-period+1)
	sum := 0.0
	for i := 0; i < len(klines); i++ {
		sum += klines[i].Close
		if i >= period {
			sum -= klines[i-period].Close
		}
		if i >= period-1 {
			series = append(series, sum/float64(period))
		}
	}

	current := 0.0
	if len(series) > 0 {
		current = series[len(series)-1]
	}
	return IndicatorResult{Current: current, Series: series}
}
