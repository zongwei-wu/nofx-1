package market

import "time"

func klineIntervalDuration(interval string) time.Duration {
	switch interval {
	case "1m":
		return time.Minute
	case "3m":
		return 3 * time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "2h":
		return 2 * time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return 3 * time.Minute
	}
}

// isKlineCacheStale 判断 WS 缓存是否过期（最后一根 K 线收盘时间过久未更新）。
func isKlineCacheStale(klines []Kline, interval string) bool {
	if len(klines) == 0 {
		return true
	}
	last := klines[len(klines)-1]
	closeMs := last.CloseTime
	if closeMs > 0 && closeMs < 1e12 {
		closeMs *= 1000
	}
	if closeMs <= 0 {
		return true
	}
	threshold := 2 * klineIntervalDuration(interval)
	return time.Since(time.UnixMilli(closeMs)) > threshold
}
