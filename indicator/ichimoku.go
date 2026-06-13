package indicator

// Ichimoku 一目均衡表
func Ichimoku(klines []Kline) IchimokuResult {
	if len(klines) < 52 {
		return IchimokuResult{}
	}

	tenkan := midpoint(klines, 9)
	kijun := midpoint(klines, 26)
	senkouB := midpoint(klines, 52)
	senkouA := (tenkan + kijun) / 2
	chikou := klines[len(klines)-1].Close

	return IchimokuResult{
		Tenkan:  tenkan,
		Kijun:   kijun,
		SenkouA: senkouA,
		SenkouB: senkouB,
		Chikou:  chikou,
	}
}

func midpoint(klines []Kline, period int) float64 {
	if len(klines) < period {
		return 0
	}
	window := klines[len(klines)-period:]
	high := window[0].High
	low := window[0].Low
	for _, k := range window[1:] {
		if k.High > high {
			high = k.High
		}
		if k.Low < low {
			low = k.Low
		}
	}
	return (high + low) / 2
}
