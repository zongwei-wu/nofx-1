package indicator

// MFI 资金流量指标
func MFI(klines []Kline, period int) IndicatorResult {
	if period <= 0 || len(klines) <= period {
		return IndicatorResult{}
	}

	series := make([]float64, 0, len(klines)-period)
	for i := period; i < len(klines); i++ {
		posFlow := 0.0
		negFlow := 0.0
		for j := i - period + 1; j <= i; j++ {
			tp := (klines[j].High + klines[j].Low + klines[j].Close) / 3
			prevTP := (klines[j-1].High + klines[j-1].Low + klines[j-1].Close) / 3
			rawMF := tp * klines[j].Volume
			if tp > prevTP {
				posFlow += rawMF
			} else if tp < prevTP {
				negFlow += rawMF
			}
		}
		mfi := 50.0
		if negFlow > 0 {
			mr := posFlow / negFlow
			mfi = 100 - (100 / (1 + mr))
		} else if posFlow > 0 {
			mfi = 100
		}
		series = append(series, mfi)
	}

	current := 0.0
	if len(series) > 0 {
		current = series[len(series)-1]
	}
	return IndicatorResult{Current: current, Series: series}
}
