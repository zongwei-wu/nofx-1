package indicator

// MACD 指数平滑异同移动平均线（含信号线/柱状图/交叉检测）
func MACD(klines []Kline, fast, slow, signal int) MACDResult {
	if fast <= 0 || slow <= 0 || signal <= 0 || len(klines) < slow {
		return MACDResult{Cross: "none"}
	}

	macdSeries := make([]float64, 0, len(klines)-slow+1)
	for i := slow - 1; i < len(klines); i++ {
		slice := klines[:i+1]
		emaFast := EMA(slice, fast).Current
		emaSlow := EMA(slice, slow).Current
		macdSeries = append(macdSeries, emaFast-emaSlow)
	}

	if len(macdSeries) == 0 {
		return MACDResult{Cross: "none"}
	}

	signalSeries := emaFromSeries(macdSeries, signal)
	if len(signalSeries) == 0 {
		return MACDResult{
			MACD:  macdSeries[len(macdSeries)-1],
			Cross: "none",
		}
	}

	offset := len(macdSeries) - len(signalSeries)
	points := make([]MACDPoint, len(signalSeries))
	for i := range signalSeries {
		macdVal := macdSeries[offset+i]
		sigVal := signalSeries[i]
		points[i] = MACDPoint{
			MACD:      macdVal,
			Signal:    sigVal,
			Histogram: macdVal - sigVal,
		}
	}

	last := points[len(points)-1]
	cross := "none"
	if len(points) >= 2 {
		prev := points[len(points)-2]
		if prev.MACD <= prev.Signal && last.MACD > last.Signal {
			cross = "up"
		} else if prev.MACD >= prev.Signal && last.MACD < last.Signal {
			cross = "down"
		}
	}

	return MACDResult{
		MACD:      last.MACD,
		Signal:    last.Signal,
		Histogram: last.Histogram,
		Cross:     cross,
		Series:    points,
	}
}

// MACDLine 仅返回 MACD 线当前值（兼容 market 包旧用法）
func MACDLine(klines []Kline) float64 {
	return MACD(klines, 12, 26, 9).MACD
}
