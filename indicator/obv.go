package indicator

// OBV 能量潮
func OBV(klines []Kline) []float64 {
	if len(klines) == 0 {
		return nil
	}
	out := make([]float64, len(klines))
	out[0] = klines[0].Volume
	for i := 1; i < len(klines); i++ {
		switch {
		case klines[i].Close > klines[i-1].Close:
			out[i] = out[i-1] + klines[i].Volume
		case klines[i].Close < klines[i-1].Close:
			out[i] = out[i-1] - klines[i].Volume
		default:
			out[i] = out[i-1]
		}
	}
	return out
}
