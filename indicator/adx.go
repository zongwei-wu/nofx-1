package indicator

import "math"

// ADX 趋势强度指标
func ADX(klines []Kline, period int) ADXResult {
	if period <= 0 || len(klines) <= period*2 {
		return ADXResult{}
	}

	plusDM := make([]float64, len(klines))
	minusDM := make([]float64, len(klines))
	tr := make([]float64, len(klines))

	for i := 1; i < len(klines); i++ {
		upMove := klines[i].High - klines[i-1].High
		downMove := klines[i-1].Low - klines[i].Low
		if upMove > downMove && upMove > 0 {
			plusDM[i] = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDM[i] = downMove
		}
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close
		tr[i] = math.Max(high-low, math.Max(math.Abs(high-prevClose), math.Abs(low-prevClose)))
	}

	smoothPlus := wilderSmooth(plusDM[1:], period)
	smoothMinus := wilderSmooth(minusDM[1:], period)
	smoothTR := wilderSmooth(tr[1:], period)

	if len(smoothTR) == 0 {
		return ADXResult{}
	}

	plusDI := make([]float64, len(smoothTR))
	minusDI := make([]float64, len(smoothTR))
	dx := make([]float64, len(smoothTR))
	for i := range smoothTR {
		if smoothTR[i] > 0 {
			plusDI[i] = 100 * smoothPlus[i] / smoothTR[i]
			minusDI[i] = 100 * smoothMinus[i] / smoothTR[i]
		}
		diSum := plusDI[i] + minusDI[i]
		if diSum > 0 {
			dx[i] = 100 * math.Abs(plusDI[i]-minusDI[i]) / diSum
		}
	}

	adxSeries := wilderSmooth(dx, period)
	if len(adxSeries) == 0 {
		return ADXResult{}
	}

	last := len(adxSeries) - 1
	return ADXResult{
		ADX:     adxSeries[last],
		PlusDI:  plusDI[len(plusDI)-1],
		MinusDI: minusDI[len(minusDI)-1],
	}
}

func wilderSmooth(values []float64, period int) []float64 {
	if len(values) < period {
		return nil
	}
	out := make([]float64, 0, len(values)-period+1)
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	smoothed := sum / float64(period)
	out = append(out, smoothed)
	for i := period; i < len(values); i++ {
		smoothed = (smoothed*float64(period-1) + values[i]) / float64(period)
		out = append(out, smoothed)
	}
	return out
}
