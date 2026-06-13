package indicator

// Stochastic 随机指标 KD
func Stochastic(klines []Kline, kPeriod, dPeriod int) StochasticResult {
	if kPeriod <= 0 || dPeriod <= 0 || len(klines) < kPeriod {
		return StochasticResult{}
	}

	kSeries := make([]float64, 0, len(klines)-kPeriod+1)
	for i := kPeriod - 1; i < len(klines); i++ {
		window := klines[i-kPeriod+1 : i+1]
		highest := window[0].High
		lowest := window[0].Low
		for _, k := range window[1:] {
			if k.High > highest {
				highest = k.High
			}
			if k.Low < lowest {
				lowest = k.Low
			}
		}
		denom := highest - lowest
		kVal := 50.0
		if denom > 0 {
			kVal = ((klines[i].Close - lowest) / denom) * 100
		}
		kSeries = append(kSeries, kVal)
	}

	if len(kSeries) < dPeriod {
		return StochasticResult{K: kSeries[len(kSeries)-1], KSeries: kSeries}
	}

	dSeries := SMAFromValues(kSeries, dPeriod)
	k := kSeries[len(kSeries)-1]
	d := 0.0
	if len(dSeries) > 0 {
		d = dSeries[len(dSeries)-1]
	}
	return StochasticResult{K: k, D: d, KSeries: kSeries, DSeries: dSeries}
}

// SMAFromValues 对数值序列计算 SMA
func SMAFromValues(values []float64, period int) []float64 {
	if period <= 0 || len(values) < period {
		return nil
	}
	out := make([]float64, 0, len(values)-period+1)
	sum := 0.0
	for i := 0; i < len(values); i++ {
		sum += values[i]
		if i >= period {
			sum -= values[i-period]
		}
		if i >= period-1 {
			out = append(out, sum/float64(period))
		}
	}
	return out
}
