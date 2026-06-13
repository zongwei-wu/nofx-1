package indicator

import "math"

// Bollinger 布林带
func Bollinger(klines []Kline, period int, mult float64) BollingerResult {
	sma := SMA(klines, period)
	if sma.Current == 0 || len(sma.Series) == 0 {
		return BollingerResult{}
	}

	start := len(klines) - period
	if start < 0 {
		return BollingerResult{}
	}
	window := klines[start:]
	mean := sma.Current
	variance := 0.0
	for _, k := range window {
		diff := k.Close - mean
		variance += diff * diff
	}
	stddev := math.Sqrt(variance / float64(period))

	return BollingerResult{
		Upper:  mean + mult*stddev,
		Middle: mean,
		Lower:  mean - mult*stddev,
	}
}
