package indicator

// VWAP 成交量加权均价
func VWAP(klines []Kline) []float64 {
	if len(klines) == 0 {
		return nil
	}
	out := make([]float64, len(klines))
	cumPV := 0.0
	cumVol := 0.0
	for i, k := range klines {
		typical := (k.High + k.Low + k.Close) / 3
		cumPV += typical * k.Volume
		cumVol += k.Volume
		if cumVol > 0 {
			out[i] = cumPV / cumVol
		}
	}
	return out
}
