package indicator

// RSI 相对强弱指标（Wilder 平滑）
func RSI(klines []Kline, period int) IndicatorResult {
	if period <= 0 || len(klines) <= period {
		return IndicatorResult{}
	}

	series := make([]float64, 0, len(klines)-period)

	gains := 0.0
	losses := 0.0
	for i := 1; i <= period; i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	series = append(series, calcRSI(avgGain, avgLoss))

	for i := period + 1; i < len(klines); i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + (-change)) / float64(period)
		}
		series = append(series, calcRSI(avgGain, avgLoss))
	}

	current := 0.0
	if len(series) > 0 {
		current = series[len(series)-1]
	}
	return IndicatorResult{Current: current, Series: series}
}

func calcRSI(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}
