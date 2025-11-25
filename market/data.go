package market

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FundingRateCache 资金费率缓存结构
// Binance Funding Rate 每 8 小时才更新一次，使用 1 小时缓存可显著减少 API 调用
type FundingRateCache struct {
	Rate      float64
	UpdatedAt time.Time
}

var (
	fundingRateMap sync.Map // map[string]*FundingRateCache
	frCacheTTL     = 1 * time.Hour
)

// Get 获取指定代币的市场数据
func Get(symbol string) (*Data, error) {
	var klines3m, klines15m, klines1h, klines4h []Kline
	var err error
	// 标准化symbol
	symbol = Normalize(symbol)
	// 获取3分钟K线数据 (最近10个)
	klines3m, err = WSMonitorCli.GetCurrentKlines(symbol, "3m") // 多获取一些用于计算
	if err != nil {
		return nil, fmt.Errorf("获取3分钟K线失败: %v", err)
	}

	// Data staleness detection: Prevent DOGEUSDT-style price freeze issues
	if isStaleData(klines3m, symbol) {
		log.Printf("⚠️  WARNING: %s detected stale data (consecutive price freeze), skipping symbol", symbol)
		return nil, fmt.Errorf("%s data is stale, possible cache failure", symbol)
	}

	// 获取15分钟K线数据
	klines15m, err = WSMonitorCli.GetCurrentKlines(symbol, "15m")
	if err != nil {
		return nil, fmt.Errorf("获取15分钟K线失败: %v", err)
	}

	// 获取1小时K线数据
	klines1h, err = WSMonitorCli.GetCurrentKlines(symbol, "1h")
	if err != nil {
		return nil, fmt.Errorf("获取1小时K线失败: %v", err)
	}

	// 获取4小时K线数据 (最近10个)
	klines4h, err = WSMonitorCli.GetCurrentKlines(symbol, "4h") // 多获取用于计算指标
	if err != nil {
		return nil, fmt.Errorf("获取4小时K线失败: %v", err)
	}

	// 检查数据是否为空
	if len(klines3m) == 0 {
		return nil, fmt.Errorf("3分钟K线数据为空")
	}
	if len(klines4h) == 0 {
		return nil, fmt.Errorf("4小时K线数据为空")
	}

	// 15m/1h 缺失时使用 3m 聚合回退
	if len(klines15m) == 0 {
		return nil, fmt.Errorf("15分钟K线数据为空")
	}
	if len(klines1h) == 0 {
		return nil, fmt.Errorf("1小时K线数据为空")
	}

	// 计算当前指标 (基于3分钟最新数据)
	currentPrice := klines3m[len(klines3m)-1].Close
	currentEMA20 := calculateEMA(klines3m, 20)
	currentMACD := calculateMACD(klines3m)
	currentRSI7 := calculateRSI(klines3m, 7)

	// 计算价格变化百分比（基于3m聚合参照）
	priceChange15m := 0.0
	if len(klines3m) >= 6 {
		price15mAgo := klines3m[len(klines3m)-6].Close
		if price15mAgo > 0 {
			priceChange15m = ((currentPrice - price15mAgo) / price15mAgo) * 100
		}
	}

	priceChange1h := 0.0
	if len(klines3m) >= 21 {
		price1hAgo := klines3m[len(klines3m)-21].Close
		if price1hAgo > 0 {
			priceChange1h = ((currentPrice - price1hAgo) / price1hAgo) * 100
		}
	}

	// 4小时价格变化 = 1个4小时K线前的价格
	priceChange4h := 0.0
	if len(klines4h) >= 2 {
		price4hAgo := klines4h[len(klines4h)-2].Close
		if price4hAgo > 0 {
			priceChange4h = ((currentPrice - price4hAgo) / price4hAgo) * 100
		}
	}

	// 获取OI数据
	oiData, err := getOpenInterestData(symbol)
	if err != nil {
		// OI失败不影响整体,使用默认值
		oiData = &OIData{Latest: 0, Average: 0}
	}

	// 获取Funding Rate
	fundingRate, _ := getFundingRate(symbol)

	// 计算日内系列数据
	intradayData := calculateIntradaySeries(klines3m)

	// 计算长期数据
	longerTermData := calculateLongerTermData(klines4h)

	// 计算15分钟K线上下文数据
	fifteenMinData := calculateKlineContextData(klines15m)

	// 计算1小时K线上下文数据
	oneHourData := calculateKlineContextData(klines1h)

	// 计算支撑压力位
	supportResistance := calculateSupportResistance(klines1h, klines4h, currentPrice)

	return &Data{
		Symbol:            symbol,
		CurrentPrice:      currentPrice,
		PriceChange15m:    priceChange15m,
		PriceChange1h:     priceChange1h,
		PriceChange4h:     priceChange4h,
		CurrentEMA20:      currentEMA20,
		CurrentMACD:       currentMACD,
		CurrentRSI7:       currentRSI7,
		OpenInterest:      oiData,
		FundingRate:       fundingRate,
		IntradaySeries:    intradayData,
		FifteenMinContext: fifteenMinData,
		OneHourContext:    oneHourData,
		LongerTermContext: longerTermData,
		SupportResistance: supportResistance,
	}, nil
}

func aggregateKlines(src []Kline, group int) []Kline {
	if group <= 1 || len(src) == 0 {
		return src
	}
	out := make([]Kline, 0, len(src)/group+1)
	for i := 0; i < len(src); i += group {
		end := i + group
		if end > len(src) {
			end = len(src)
		}
		g := src[i:end]
		if len(g) == 0 {
			continue
		}
		high := g[0].High
		low := g[0].Low
		vol := 0.0
		quoteVol := 0.0
		trades := 0
		takerBase := 0.0
		takerQuote := 0.0
		for _, k := range g {
			if k.High > high {
				high = k.High
			}
			if k.Low < low {
				low = k.Low
			}
			vol += k.Volume
			quoteVol += k.QuoteVolume
			trades += k.Trades
			takerBase += k.TakerBuyBaseVolume
			takerQuote += k.TakerBuyQuoteVolume
		}
		out = append(out, Kline{
			OpenTime:            g[0].OpenTime,
			CloseTime:           g[len(g)-1].CloseTime,
			Open:                g[0].Open,
			Close:               g[len(g)-1].Close,
			High:                high,
			Low:                 low,
			Volume:              vol,
			QuoteVolume:         quoteVol,
			Trades:              trades,
			TakerBuyBaseVolume:  takerBase,
			TakerBuyQuoteVolume: takerQuote,
		})
	}
	return out
}

// calculateEMA 计算EMA
func calculateEMA(klines []Kline, period int) float64 {
	if len(klines) < period {
		return 0
	}

	// 计算SMA作为初始EMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += klines[i].Close
	}
	ema := sum / float64(period)

	// 计算EMA
	multiplier := 2.0 / float64(period+1)
	for i := period; i < len(klines); i++ {
		ema = (klines[i].Close-ema)*multiplier + ema
	}

	return ema
}

// calculateMACD 计算MACD
func calculateMACD(klines []Kline) float64 {
	if len(klines) < 26 {
		return 0
	}

	// 计算12期和26期EMA
	ema12 := calculateEMA(klines, 12)
	ema26 := calculateEMA(klines, 26)

	// MACD = EMA12 - EMA26
	return ema12 - ema26
}

// calculateRSI 计算RSI
func calculateRSI(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	gains := 0.0
	losses := 0.0

	// 计算初始平均涨跌幅
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

	// 使用Wilder平滑方法计算后续RSI
	for i := period + 1; i < len(klines); i++ {
		change := klines[i].Close - klines[i-1].Close
		if change > 0 {
			avgGain = (avgGain*float64(period-1) + change) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + (-change)) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateATR 计算ATR
func calculateATR(klines []Kline, period int) float64 {
	if len(klines) <= period {
		return 0
	}

	trs := make([]float64, len(klines))
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		prevClose := klines[i-1].Close

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		trs[i] = math.Max(tr1, math.Max(tr2, tr3))
	}

	// 计算初始ATR
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += trs[i]
	}
	atr := sum / float64(period)

	// Wilder平滑
	for i := period + 1; i < len(klines); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}

	return atr
}

// calculateIntradaySeries 计算日内系列数据
func calculateIntradaySeries(klines []Kline) *IntradayData {
	data := &IntradayData{
		MidPrices:   make([]float64, 0, 10),
		EMA20Values: make([]float64, 0, 10),
		MACDValues:  make([]float64, 0, 10),
		RSI7Values:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
		Volume:      make([]float64, 0, 10),
	}

	// 获取最近10个数据点
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		data.MidPrices = append(data.MidPrices, klines[i].Close)
		data.Volume = append(data.Volume, klines[i].Volume)

		// 计算每个点的EMA20
		if i >= 19 {
			ema20 := calculateEMA(klines[:i+1], 20)
			data.EMA20Values = append(data.EMA20Values, ema20)
		}

		// 计算每个点的MACD
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}

		// 计算每个点的RSI
		if i >= 7 {
			rsi7 := calculateRSI(klines[:i+1], 7)
			data.RSI7Values = append(data.RSI7Values, rsi7)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	// 计算3m ATR14
	data.ATR14 = calculateATR(klines, 14)

	return data
}

// calculateLongerTermData 计算长期数据
func calculateLongerTermData(klines []Kline) *LongerTermData {
	data := &LongerTermData{
		MACDValues:  make([]float64, 0, 10),
		RSI14Values: make([]float64, 0, 10),
	}

	// 计算EMA
	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)

	// 计算ATR
	data.ATR3 = calculateATR(klines, 3)
	data.ATR14 = calculateATR(klines, 14)

	// 计算成交量
	if len(klines) > 0 {
		data.CurrentVolume = klines[len(klines)-1].Volume
		// 计算平均成交量
		sum := 0.0
		for _, k := range klines {
			sum += k.Volume
		}
		data.AverageVolume = sum / float64(len(klines))
	}

	// 计算MACD和RSI序列
	start := len(klines) - 10
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if i >= 25 {
			macd := calculateMACD(klines[:i+1])
			data.MACDValues = append(data.MACDValues, macd)
		}
		if i >= 14 {
			rsi14 := calculateRSI(klines[:i+1], 14)
			data.RSI14Values = append(data.RSI14Values, rsi14)
		}
	}

	return data
}

// getOpenInterestData 获取OI数据
func getOpenInterestData(symbol string) (*OIData, error) {
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/openInterest?symbol=%s", symbol)

	apiClient := NewAPIClient()
	resp, err := apiClient.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OpenInterest string `json:"openInterest"`
		Symbol       string `json:"symbol"`
		Time         int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	oi, _ := strconv.ParseFloat(result.OpenInterest, 64)

	return &OIData{
		Latest:  oi,
		Average: oi * 0.999, // 近似平均值
	}, nil
}

// getFundingRate 获取资金费率（优化：使用 1 小时缓存）
func getFundingRate(symbol string) (float64, error) {
	// 检查缓存（有效期 1 小时）
	// Funding Rate 每 8 小时才更新，1 小时缓存非常合理
	if cached, ok := fundingRateMap.Load(symbol); ok {
		cache := cached.(*FundingRateCache)
		if time.Since(cache.UpdatedAt) < frCacheTTL {
			// 缓存命中，直接返回
			return cache.Rate, nil
		}
	}

	// 缓存过期或不存在，调用 API
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)

	apiClient := NewAPIClient()
	resp, err := apiClient.client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Symbol          string `json:"symbol"`
		MarkPrice       string `json:"markPrice"`
		IndexPrice      string `json:"indexPrice"`
		LastFundingRate string `json:"lastFundingRate"`
		NextFundingTime int64  `json:"nextFundingTime"`
		InterestRate    string `json:"interestRate"`
		Time            int64  `json:"time"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	rate, _ := strconv.ParseFloat(result.LastFundingRate, 64)

	// 更新缓存
	fundingRateMap.Store(symbol, &FundingRateCache{
		Rate:      rate,
		UpdatedAt: time.Now(),
	})

	return rate, nil
}

// Format 格式化输出市场数据
func Format(data *Data) string {
	var sb strings.Builder

	// 使用动态精度格式化价格
	priceStr := formatPriceWithDynamicPrecision(data.CurrentPrice)
	sb.WriteString(fmt.Sprintf("current_price = %s, current_ema20 = %.3f, current_macd = %.3f, current_rsi (7 period) = %.3f\n\n",
		priceStr, data.CurrentEMA20, data.CurrentMACD, data.CurrentRSI7))

	sb.WriteString(fmt.Sprintf("price_change: 15m %+.2f%% | 1h %+.2f%% | 4h %+.2f%%\n\n",
		data.PriceChange15m, data.PriceChange1h, data.PriceChange4h))

	sb.WriteString(fmt.Sprintf("In addition, here is the latest %s open interest and funding rate for perps:\n\n",
		data.Symbol))

	if data.OpenInterest != nil {
		// 使用动态精度格式化 OI 数据
		oiLatestStr := formatPriceWithDynamicPrecision(data.OpenInterest.Latest)
		oiAverageStr := formatPriceWithDynamicPrecision(data.OpenInterest.Average)
		sb.WriteString(fmt.Sprintf("Open Interest: Latest: %s Average: %s\n\n",
			oiLatestStr, oiAverageStr))
	}

	sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))

	if data.IntradaySeries != nil {
		sb.WriteString("Intraday series (3‑minute intervals, oldest → latest):\n\n")

		if len(data.IntradaySeries.MidPrices) > 0 {
			sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
		}

		if len(data.IntradaySeries.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA indicators (20‑period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
		}

		if len(data.IntradaySeries.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
		}

		if len(data.IntradaySeries.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (7‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
		}

		if len(data.IntradaySeries.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
		}

		if len(data.IntradaySeries.Volume) > 0 {
			sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(data.IntradaySeries.Volume)))
		}

		sb.WriteString(fmt.Sprintf("3m ATR (14‑period): %.3f\n\n", data.IntradaySeries.ATR14))
	}

	if data.FifteenMinContext != nil {
		sb.WriteString("15‑minute context:\n\n")
		sb.WriteString(fmt.Sprintf("EMA20: %.3f vs. EMA50: %.3f\n\n", data.FifteenMinContext.EMA20, data.FifteenMinContext.EMA50))
		sb.WriteString(fmt.Sprintf("MACD: %.3f | RSI14: %.3f | ATR14: %.3f\n\n", data.FifteenMinContext.MACD, data.FifteenMinContext.RSI14, data.FifteenMinContext.ATR14))
		sb.WriteString(fmt.Sprintf("Volume: %.3f | Trend: %s | Pattern: %s\n\n", data.FifteenMinContext.Volume, data.FifteenMinContext.Trend, data.FifteenMinContext.Pattern))
	}

	if data.OneHourContext != nil {
		sb.WriteString("1‑hour context:\n\n")
		sb.WriteString(fmt.Sprintf("EMA20: %.3f vs. EMA50: %.3f\n\n", data.OneHourContext.EMA20, data.OneHourContext.EMA50))
		sb.WriteString(fmt.Sprintf("MACD: %.3f | RSI14: %.3f | ATR14: %.3f\n\n", data.OneHourContext.MACD, data.OneHourContext.RSI14, data.OneHourContext.ATR14))
		sb.WriteString(fmt.Sprintf("Volume: %.3f | Trend: %s | Pattern: %s\n\n", data.OneHourContext.Volume, data.OneHourContext.Trend, data.OneHourContext.Pattern))
	}

	if data.LongerTermContext != nil {
		sb.WriteString("Longer‑term context (4‑hour timeframe):\n\n")

		sb.WriteString(fmt.Sprintf("20‑Period EMA: %.3f vs. 50‑Period EMA: %.3f\n\n",
			data.LongerTermContext.EMA20, data.LongerTermContext.EMA50))

		sb.WriteString(fmt.Sprintf("3‑Period ATR: %.3f vs. 14‑Period ATR: %.3f\n\n",
			data.LongerTermContext.ATR3, data.LongerTermContext.ATR14))

		sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n\n",
			data.LongerTermContext.CurrentVolume, data.LongerTermContext.AverageVolume))

		if len(data.LongerTermContext.MACDValues) > 0 {
			sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
		}

		if len(data.LongerTermContext.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI indicators (14‑Period): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
		}
	}

	// 输出支撑压力位
	if data.SupportResistance != nil {
		sb.WriteString("Support & Resistance Levels:\n\n")

		// 最近的支撑和压力位（最重要）
		if data.SupportResistance.NearestSupport != nil {
			s := data.SupportResistance.NearestSupport
			sb.WriteString(fmt.Sprintf("📍 Nearest Support: %s (-%.2f%%) | Strength: %.0f/100 | Touches: %d\n\n",
				formatPriceWithDynamicPrecision(s.Price), s.Distance, s.Strength, s.TouchCount))
		}

		if data.SupportResistance.NearestResistance != nil {
			r := data.SupportResistance.NearestResistance
			sb.WriteString(fmt.Sprintf("📍 Nearest Resistance: %s (+%.2f%%) | Strength: %.0f/100 | Touches: %d\n\n",
				formatPriceWithDynamicPrecision(r.Price), r.Distance, r.Strength, r.TouchCount))
		}

		// 强支撑位
		if len(data.SupportResistance.StrongSupports) > 0 {
			sb.WriteString("Strong Supports:\n")
			for i, s := range data.SupportResistance.StrongSupports {
				sb.WriteString(fmt.Sprintf("  %d. %s (-%.2f%%) | Strength: %.0f | Touches: %d\n",
					i+1, formatPriceWithDynamicPrecision(s.Price), s.Distance, s.Strength, s.TouchCount))
			}
			sb.WriteString("\n")
		}

		// 强压力位
		if len(data.SupportResistance.StrongResistances) > 0 {
			sb.WriteString("Strong Resistances:\n")
			for i, r := range data.SupportResistance.StrongResistances {
				sb.WriteString(fmt.Sprintf("  %d. %s (+%.2f%%) | Strength: %.0f | Touches: %d\n",
					i+1, formatPriceWithDynamicPrecision(r.Price), r.Distance, r.Strength, r.TouchCount))
			}
			sb.WriteString("\n")
		}

		// 弱支撑位（可选）
		if len(data.SupportResistance.WeakSupports) > 0 {
			sb.WriteString("Weak Supports: ")
			for i, s := range data.SupportResistance.WeakSupports {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%s (-%.2f%%)",
					formatPriceWithDynamicPrecision(s.Price), s.Distance))
			}
			sb.WriteString("\n\n")
		}

		// 弱压力位（可选）
		if len(data.SupportResistance.WeakResistances) > 0 {
			sb.WriteString("Weak Resistances: ")
			for i, r := range data.SupportResistance.WeakResistances {
				if i > 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(fmt.Sprintf("%s (+%.2f%%)",
					formatPriceWithDynamicPrecision(r.Price), r.Distance))
			}
			sb.WriteString("\n\n")
		}

		// 斐波那契回撤位
		if data.SupportResistance.FibonacciLevels != nil && len(data.SupportResistance.FibonacciLevels.Levels) > 0 {
			fib := data.SupportResistance.FibonacciLevels
			sb.WriteString(fmt.Sprintf("Fibonacci Retracement (%s):\n", fib.Trend))
			sb.WriteString(fmt.Sprintf("  Swing High: %s | Swing Low: %s\n",
				formatPriceWithDynamicPrecision(fib.SwingHigh),
				formatPriceWithDynamicPrecision(fib.SwingLow)))

			// 按标准顺序输出斐波那契位
			fibKeys := []string{"0.236", "0.382", "0.5", "0.618", "0.786"}
			for _, key := range fibKeys {
				if price, ok := fib.Levels[key]; ok {
					distance := ((price - data.CurrentPrice) / data.CurrentPrice) * 100
					sb.WriteString(fmt.Sprintf("  %s: %s (%+.2f%%)\n", key,
						formatPriceWithDynamicPrecision(price), distance))
				}
			}
			sb.WriteString("\n")
		}

		// 布林带
		if data.SupportResistance.BollingerBands != nil {
			bb := data.SupportResistance.BollingerBands
			sb.WriteString("Bollinger Bands (20, 2):\n")
			sb.WriteString(fmt.Sprintf("  Upper: %s | Middle: %s | Lower: %s\n",
				formatPriceWithDynamicPrecision(bb.Upper),
				formatPriceWithDynamicPrecision(bb.Middle),
				formatPriceWithDynamicPrecision(bb.Lower)))
			sb.WriteString(fmt.Sprintf("  Width: %.2f%% | Position: %s", bb.Width, bb.Position))
			if bb.Squeeze {
				sb.WriteString(" | ⚠️ SQUEEZE (Low Volatility)")
			}
			sb.WriteString("\n\n")
		}

		// 成交量分布
		if data.SupportResistance.VolumeProfile != nil {
			vp := data.SupportResistance.VolumeProfile
			sb.WriteString("Volume Profile:\n")
			sb.WriteString(fmt.Sprintf("  POC (Point of Control): %s\n",
				formatPriceWithDynamicPrecision(vp.POC)))
			sb.WriteString(fmt.Sprintf("  Value Area: %s - %s (70%% volume)\n",
				formatPriceWithDynamicPrecision(vp.VAL),
				formatPriceWithDynamicPrecision(vp.VAH)))
			if len(vp.HighVolumeZones) > 0 {
				sb.WriteString(fmt.Sprintf("  High Volume Zones: %d areas detected\n", len(vp.HighVolumeZones)))
			}
			sb.WriteString("\n")
		}

		// 关键价格区域
		if len(data.SupportResistance.KeyPriceZones) > 0 {
			sb.WriteString("Key Price Zones:\n")
			for i, zone := range data.SupportResistance.KeyPriceZones {
				if i >= 3 {
					break // 只显示前3个最重要的区域
				}
				sb.WriteString(fmt.Sprintf("  %d. %s: %s - %s (Strength: %.0f)\n",
					i+1, zone.Type,
					formatPriceWithDynamicPrecision(zone.LowPrice),
					formatPriceWithDynamicPrecision(zone.HighPrice),
					zone.Strength))
			}
			sb.WriteString("\n")
		}

		// 突破信号
		if len(data.SupportResistance.BreakoutSignals) > 0 {
			sb.WriteString("⚡ Breakout Signals:\n")
			for _, signal := range data.SupportResistance.BreakoutSignals {
				confirmStatus := ""
				if signal.Confirmed {
					confirmStatus = " ✓ Confirmed"
				}
				sb.WriteString(fmt.Sprintf("  • %s at %s | %s | Strength: %.0f%s\n",
					signal.Type,
					formatPriceWithDynamicPrecision(signal.Price),
					signal.Direction,
					signal.Strength,
					confirmStatus))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func calculateKlineContextData(klines []Kline) *KlineContextData {
	data := &KlineContextData{}
	data.EMA20 = calculateEMA(klines, 20)
	data.EMA50 = calculateEMA(klines, 50)
	data.MACD = calculateMACD(klines)
	data.RSI14 = calculateRSI(klines, 14)
	data.ATR14 = calculateATR(klines, 14)
	if len(klines) > 0 {
		data.Volume = klines[len(klines)-1].Volume
	}
	data.Trend = determineTrend(klines)
	data.Pattern = determinePattern(klines)
	return data
}

func determineTrend(klines []Kline) string {
	if len(klines) < 20 {
		return "sideways"
	}
	currentPrice := klines[len(klines)-1].Close
	ema20 := calculateEMA(klines, 20)
	ema50 := calculateEMA(klines, 50)
	if currentPrice > ema20 && ema20 > ema50 {
		return "uptrend"
	} else if currentPrice < ema20 && ema20 < ema50 {
		return "downtrend"
	}
	return "sideways"
}

func determinePattern(klines []Kline) string {
	if len(klines) < 2 {
		return "none"
	}
	last := klines[len(klines)-1]
	prev := klines[len(klines)-2]
	if isHammer(last) {
		return "hammer"
	}
	if isEngulfing(prev, last) {
		return "engulfing"
	}
	return "none"
}

func isHammer(kline Kline) bool {
	body := math.Abs(kline.Close - kline.Open)
	lower := math.Min(kline.Open, kline.Close) - kline.Low
	upper := kline.High - math.Max(kline.Open, kline.Close)
	return body < (kline.High-kline.Low)*0.3 && lower > body*2 && upper < body
}

func isEngulfing(prevKline, currentKline Kline) bool {
	prevBody := math.Abs(prevKline.Close - prevKline.Open)
	currBody := math.Abs(currentKline.Close - currentKline.Open)
	return currBody > prevBody &&
		math.Min(currentKline.Open, currentKline.Close) < math.Min(prevKline.Open, prevKline.Close) &&
		math.Max(currentKline.Open, currentKline.Close) > math.Max(prevKline.Open, prevKline.Close)
}

// formatPriceWithDynamicPrecision 根据价格区间动态选择精度
// 这样可以完美支持从超低价 meme coin (< 0.0001) 到 BTC/ETH 的所有币种
func formatPriceWithDynamicPrecision(price float64) string {
	switch {
	case price < 0.0001:
		// 超低价 meme coin: 1000SATS, 1000WHY, DOGS
		// 0.00002070 → "0.00002070" (8位小数)
		return fmt.Sprintf("%.8f", price)
	case price < 0.001:
		// 低价 meme coin: NEIRO, HMSTR, HOT, NOT
		// 0.00015060 → "0.000151" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 0.01:
		// 中低价币: PEPE, SHIB, MEME
		// 0.00556800 → "0.005568" (6位小数)
		return fmt.Sprintf("%.6f", price)
	case price < 1.0:
		// 低价币: ASTER, DOGE, ADA, TRX
		// 0.9954 → "0.9954" (4位小数)
		return fmt.Sprintf("%.4f", price)
	case price < 100:
		// 中价币: SOL, AVAX, LINK, MATIC
		// 23.4567 → "23.4567" (4位小数)
		return fmt.Sprintf("%.4f", price)
	default:
		// 高价币: BTC, ETH (节省 Token)
		// 45678.9123 → "45678.91" (2位小数)
		return fmt.Sprintf("%.2f", price)
	}
}

// formatFloatSlice 格式化float64切片为字符串（使用动态精度）
func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = formatPriceWithDynamicPrecision(v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}

// Normalize 标准化symbol,确保是USDT交易对
func Normalize(symbol string) string {
	symbol = strings.ToUpper(symbol)
	if strings.HasSuffix(symbol, "USDT") {
		return symbol
	}
	return symbol + "USDT"
}

// parseFloat 解析float值
func parseFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case string:
		return strconv.ParseFloat(val, 64)
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

// isStaleData detects stale data (consecutive price freeze)
// Fix DOGEUSDT-style issue: consecutive N periods with completely unchanged prices indicate data source anomaly
func isStaleData(klines []Kline, symbol string) bool {
	if len(klines) < 5 {
		return false // Insufficient data to determine
	}

	// Detection threshold: 5 consecutive 3-minute periods with unchanged price (15 minutes without fluctuation)
	const stalePriceThreshold = 5
	const priceTolerancePct = 0.0001 // 0.01% fluctuation tolerance (avoid false positives)

	// Take the last stalePriceThreshold K-lines
	recentKlines := klines[len(klines)-stalePriceThreshold:]
	firstPrice := recentKlines[0].Close

	// Check if all prices are within tolerance
	for i := 1; i < len(recentKlines); i++ {
		priceDiff := math.Abs(recentKlines[i].Close-firstPrice) / firstPrice
		if priceDiff > priceTolerancePct {
			return false // Price fluctuation exists, data is normal
		}
	}

	// Additional check: MACD and volume
	// If price is unchanged but MACD/volume shows normal fluctuation, it might be a real market situation (extremely low volatility)
	// Check if volume is also 0 (data completely frozen)
	allVolumeZero := true
	for _, k := range recentKlines {
		if k.Volume > 0 {
			allVolumeZero = false
			break
		}
	}

	if allVolumeZero {
		log.Printf("⚠️  %s stale data confirmed: price freeze + zero volume", symbol)
		return true
	}

	// Price frozen but has volume: might be extremely low volatility market, allow but log warning
	log.Printf("⚠️  %s detected extreme price stability (no fluctuation for %d consecutive periods), but volume is normal", symbol, stalePriceThreshold)
	return false
}

// calculateSupportResistance 计算支撑位和压力位
// 综合使用多种方法：局部高低点、成交量分布、价格聚类
func calculateSupportResistance(klines1h []Kline, klines4h []Kline, currentPrice float64) *SupportResistanceLevels {
	if len(klines1h) < 20 || len(klines4h) < 20 {
		return &SupportResistanceLevels{}
	}

	levels := &SupportResistanceLevels{
		StrongSupports:    []PriceLevel{},
		WeakSupports:      []PriceLevel{},
		StrongResistances: []PriceLevel{},
		WeakResistances:   []PriceLevel{},
	}

	// 1. 从1小时和4小时K线提取局部高低点
	highPoints1h := findLocalHighs(klines1h, 3) // 3周期回看窗口
	lowPoints1h := findLocalLows(klines1h, 3)
	highPoints4h := findLocalHighs(klines4h, 2) // 4小时用2周期
	lowPoints4h := findLocalLows(klines4h, 2)

	// 2. 合并所有高低点
	allHighs := append(highPoints1h, highPoints4h...)
	allLows := append(lowPoints1h, lowPoints4h...)

	// 3. 价格聚类（将相近的价格合并）
	clusterThreshold := currentPrice * 0.005 // 0.5%的价格范围内视为同一价格位
	resistanceClusters := clusterPrices(allHighs, clusterThreshold)
	supportClusters := clusterPrices(allLows, clusterThreshold)

	// 4. 计算每个价格位的强度
	for _, cluster := range resistanceClusters {
		if cluster.Price <= currentPrice {
			continue // 压力位必须在当前价格上方
		}
		distance := (cluster.Price - currentPrice) / currentPrice * 100
		if distance > 15 { // 只关注15%范围内的压力位
			continue
		}

		level := PriceLevel{
			Price:      cluster.Price,
			Strength:   cluster.Strength,
			Distance:   distance,
			Source:     "high_cluster",
			TouchCount: cluster.TouchCount,
		}

		// 根据强度分类
		if cluster.Strength >= 60 {
			levels.StrongResistances = append(levels.StrongResistances, level)
		} else if cluster.Strength >= 30 {
			levels.WeakResistances = append(levels.WeakResistances, level)
		}
	}

	for _, cluster := range supportClusters {
		if cluster.Price >= currentPrice {
			continue // 支撑位必须在当前价格下方
		}
		distance := (currentPrice - cluster.Price) / currentPrice * 100
		if distance > 15 { // 只关注15%范围内的支撑位
			continue
		}

		level := PriceLevel{
			Price:      cluster.Price,
			Strength:   cluster.Strength,
			Distance:   distance,
			Source:     "low_cluster",
			TouchCount: cluster.TouchCount,
		}

		// 根据强度分类
		if cluster.Strength >= 60 {
			levels.StrongSupports = append(levels.StrongSupports, level)
		} else if cluster.Strength >= 30 {
			levels.WeakSupports = append(levels.WeakSupports, level)
		}
	}

	// 5. 按距离排序（最近的排在前面）
	sortPriceLevelsByDistance(levels.StrongSupports)
	sortPriceLevelsByDistance(levels.WeakSupports)
	sortPriceLevelsByDistance(levels.StrongResistances)
	sortPriceLevelsByDistance(levels.WeakResistances)

	// 6. 只保留最重要的3个
	if len(levels.StrongSupports) > 3 {
		levels.StrongSupports = levels.StrongSupports[:3]
	}
	if len(levels.StrongResistances) > 3 {
		levels.StrongResistances = levels.StrongResistances[:3]
	}
	if len(levels.WeakSupports) > 2 {
		levels.WeakSupports = levels.WeakSupports[:2]
	}
	if len(levels.WeakResistances) > 2 {
		levels.WeakResistances = levels.WeakResistances[:2]
	}

	// 7. 标记最近的支撑和压力位
	if len(levels.StrongSupports) > 0 {
		levels.NearestSupport = &levels.StrongSupports[0]
	} else if len(levels.WeakSupports) > 0 {
		levels.NearestSupport = &levels.WeakSupports[0]
	}

	if len(levels.StrongResistances) > 0 {
		levels.NearestResistance = &levels.StrongResistances[0]
	} else if len(levels.WeakResistances) > 0 {
		levels.NearestResistance = &levels.WeakResistances[0]
	}

	// 8. 计算斐波那契回撤位
	levels.FibonacciLevels = calculateFibonacciLevels(klines4h, currentPrice)

	// 9. 计算成交量分布
	levels.VolumeProfile = calculateVolumeProfile(klines1h, currentPrice)

	// 10. 计算布林带
	levels.BollingerBands = calculateBollingerBands(klines1h, 20, 2.0, currentPrice)

	// 11. 检测突破信号
	allSupports := append(levels.StrongSupports, levels.WeakSupports...)
	allResistances := append(levels.StrongResistances, levels.WeakResistances...)
	levels.BreakoutSignals = detectBreakouts(klines1h, allSupports, allResistances)

	// 12. 识别关键价格区域
	levels.KeyPriceZones = identifyKeyPriceZones(levels.StrongSupports, levels.StrongResistances, levels.VolumeProfile)

	return levels
}

// pricePoint 价格点（用于计算）
type pricePoint struct {
	Price  float64
	Volume float64
	Index  int
}

// findLocalHighs 找出局部高点
func findLocalHighs(klines []Kline, lookback int) []pricePoint {
	var highs []pricePoint
	for i := lookback; i < len(klines)-lookback; i++ {
		isLocalHigh := true
		current := klines[i].High

		// 检查左右两边的K线
		for j := 1; j <= lookback; j++ {
			if klines[i-j].High >= current || klines[i+j].High >= current {
				isLocalHigh = false
				break
			}
		}

		if isLocalHigh {
			highs = append(highs, pricePoint{
				Price:  current,
				Volume: klines[i].Volume,
				Index:  i,
			})
		}
	}
	return highs
}

// findLocalLows 找出局部低点
func findLocalLows(klines []Kline, lookback int) []pricePoint {
	var lows []pricePoint
	for i := lookback; i < len(klines)-lookback; i++ {
		isLocalLow := true
		current := klines[i].Low

		// 检查左右两边的K线
		for j := 1; j <= lookback; j++ {
			if klines[i-j].Low <= current || klines[i+j].Low <= current {
				isLocalLow = false
				break
			}
		}

		if isLocalLow {
			lows = append(lows, pricePoint{
				Price:  current,
				Volume: klines[i].Volume,
				Index:  i,
			})
		}
	}
	return lows
}

// priceCluster 价格聚类结果
type priceCluster struct {
	Price       float64
	Strength    float64
	TouchCount  int
	TotalVolume float64
}

// clusterPrices 将相近的价格聚类
func clusterPrices(points []pricePoint, threshold float64) []priceCluster {
	if len(points) == 0 {
		return []priceCluster{}
	}

	// 按价格排序
	sortedPoints := make([]pricePoint, len(points))
	copy(sortedPoints, points)
	for i := 0; i < len(sortedPoints)-1; i++ {
		for j := i + 1; j < len(sortedPoints); j++ {
			if sortedPoints[i].Price > sortedPoints[j].Price {
				sortedPoints[i], sortedPoints[j] = sortedPoints[j], sortedPoints[i]
			}
		}
	}

	var clusters []priceCluster
	currentCluster := priceCluster{
		Price:       sortedPoints[0].Price,
		TouchCount:  1,
		TotalVolume: sortedPoints[0].Volume,
	}

	for i := 1; i < len(sortedPoints); i++ {
		if math.Abs(sortedPoints[i].Price-currentCluster.Price) <= threshold {
			// 在同一聚类中，更新平均价格
			totalWeight := currentCluster.TotalVolume + sortedPoints[i].Volume
			currentCluster.Price = (currentCluster.Price*currentCluster.TotalVolume + sortedPoints[i].Price*sortedPoints[i].Volume) / totalWeight
			currentCluster.TotalVolume = totalWeight
			currentCluster.TouchCount++
		} else {
			// 计算当前聚类的强度并保存
			currentCluster.Strength = calculateClusterStrength(currentCluster)
			clusters = append(clusters, currentCluster)

			// 开始新的聚类
			currentCluster = priceCluster{
				Price:       sortedPoints[i].Price,
				TouchCount:  1,
				TotalVolume: sortedPoints[i].Volume,
			}
		}
	}

	// 保存最后一个聚类
	currentCluster.Strength = calculateClusterStrength(currentCluster)
	clusters = append(clusters, currentCluster)

	return clusters
}

// calculateClusterStrength 计算聚类强度（0-100）
func calculateClusterStrength(cluster priceCluster) float64 {
	// 强度 = 触及次数权重(60%) + 成交量权重(40%)
	touchScore := float64(cluster.TouchCount) * 15 // 每次触及+15分
	if touchScore > 60 {
		touchScore = 60 // 最高60分
	}

	volumeScore := 40.0 // 成交量基础分
	if cluster.TotalVolume > 0 {
		// 成交量越大，分数越高（使用对数刻度避免极端值）
		volumeScore = math.Min(40, math.Log10(cluster.TotalVolume+1)*8)
	}

	return touchScore + volumeScore
}

// sortPriceLevelsByDistance 按距离排序价格位
func sortPriceLevelsByDistance(levels []PriceLevel) {
	for i := 0; i < len(levels)-1; i++ {
		for j := i + 1; j < len(levels); j++ {
			if levels[i].Distance > levels[j].Distance {
				levels[i], levels[j] = levels[j], levels[i]
			}
		}
	}
}

// calculateFibonacciLevels 计算斐波那契回撤位
func calculateFibonacciLevels(klines []Kline, currentPrice float64) *FibonacciLevels {
	if len(klines) < 20 {
		return nil
	}

	// 找出最近的波段高点和低点（最近50个K线）
	lookback := 50
	if len(klines) < lookback {
		lookback = len(klines)
	}
	recentKlines := klines[len(klines)-lookback:]

	swingHigh := recentKlines[0].High
	swingLow := recentKlines[0].Low

	for _, k := range recentKlines {
		if k.High > swingHigh {
			swingHigh = k.High
		}
		if k.Low < swingLow {
			swingLow = k.Low
		}
	}

	// 判断趋势方向
	trend := "unknown"
	priceRange := swingHigh - swingLow
	if currentPrice > swingLow+(priceRange*0.6) {
		trend = "uptrend" // 价格在高位，计算回撤位
	} else if currentPrice < swingLow+(priceRange*0.4) {
		trend = "downtrend" // 价格在低位，计算反弹位
	}

	// 计算斐波那契回撤位
	levels := make(map[string]float64)
	if trend == "uptrend" {
		// 上升趋势：从高点往下回撤
		levels["0.236"] = swingHigh - priceRange*0.236
		levels["0.382"] = swingHigh - priceRange*0.382
		levels["0.5"] = swingHigh - priceRange*0.5
		levels["0.618"] = swingHigh - priceRange*0.618
		levels["0.786"] = swingHigh - priceRange*0.786
	} else if trend == "downtrend" {
		// 下降趋势：从低点往上反弹
		levels["0.236"] = swingLow + priceRange*0.236
		levels["0.382"] = swingLow + priceRange*0.382
		levels["0.5"] = swingLow + priceRange*0.5
		levels["0.618"] = swingLow + priceRange*0.618
		levels["0.786"] = swingLow + priceRange*0.786
	}

	return &FibonacciLevels{
		SwingHigh: swingHigh,
		SwingLow:  swingLow,
		Trend:     trend,
		Levels:    levels,
	}
}

// calculateVolumeProfile 计算成交量分布
func calculateVolumeProfile(klines []Kline, currentPrice float64) *VolumeProfile {
	if len(klines) < 20 {
		return nil
	}

	// 使用最近100个K线计算
	lookback := 100
	if len(klines) < lookback {
		lookback = len(klines)
	}
	recentKlines := klines[len(klines)-lookback:]

	// 找出价格范围
	minPrice := recentKlines[0].Low
	maxPrice := recentKlines[0].High
	for _, k := range recentKlines {
		if k.Low < minPrice {
			minPrice = k.Low
		}
		if k.High > maxPrice {
			maxPrice = k.High
		}
	}

	// 将价格范围分成30个区间
	numBins := 30
	binSize := (maxPrice - minPrice) / float64(numBins)
	volumeBins := make([]float64, numBins)
	binPrices := make([]float64, numBins)

	// 初始化每个区间的中心价格
	for i := 0; i < numBins; i++ {
		binPrices[i] = minPrice + float64(i)*binSize + binSize/2
	}

	// 分配成交量到各个价格区间
	for _, k := range recentKlines {
		// 简化：将K线的成交量平均分配到它覆盖的价格区间
		startBin := int((k.Low - minPrice) / binSize)
		endBin := int((k.High - minPrice) / binSize)

		if startBin < 0 {
			startBin = 0
		}
		if endBin >= numBins {
			endBin = numBins - 1
		}

		volumePerBin := k.Volume / float64(endBin-startBin+1)
		for i := startBin; i <= endBin; i++ {
			volumeBins[i] += volumePerBin
		}
	}

	// 找出POC（最大成交量价格）
	pocIndex := 0
	maxVolume := volumeBins[0]
	for i := 1; i < numBins; i++ {
		if volumeBins[i] > maxVolume {
			maxVolume = volumeBins[i]
			pocIndex = i
		}
	}
	poc := binPrices[pocIndex]

	// 计算Value Area（70%成交量区域）
	totalVolume := 0.0
	for _, v := range volumeBins {
		totalVolume += v
	}
	targetVolume := totalVolume * 0.70

	// 从POC向两侧扩展，直到包含70%的成交量
	accumulatedVolume := volumeBins[pocIndex]
	lowerIndex := pocIndex
	upperIndex := pocIndex

	for accumulatedVolume < targetVolume && (lowerIndex > 0 || upperIndex < numBins-1) {
		lowerVolume := 0.0
		upperVolume := 0.0

		if lowerIndex > 0 {
			lowerVolume = volumeBins[lowerIndex-1]
		}
		if upperIndex < numBins-1 {
			upperVolume = volumeBins[upperIndex+1]
		}

		if lowerVolume > upperVolume && lowerIndex > 0 {
			lowerIndex--
			accumulatedVolume += lowerVolume
		} else if upperIndex < numBins-1 {
			upperIndex++
			accumulatedVolume += upperVolume
		} else if lowerIndex > 0 {
			lowerIndex--
			accumulatedVolume += lowerVolume
		} else {
			break
		}
	}

	val := binPrices[lowerIndex]
	vah := binPrices[upperIndex]

	// 找出高成交量区域（成交量>平均值的150%）
	avgVolume := totalVolume / float64(numBins)
	highVolumeThreshold := avgVolume * 1.5
	var highVolumeZones []PriceZone

	i := 0
	for i < numBins {
		if volumeBins[i] > highVolumeThreshold {
			// 找到高成交量区域的起点
			startIdx := i
			zoneVolume := 0.0

			// 向后扫描连续的高成交量区间
			for i < numBins && volumeBins[i] > highVolumeThreshold {
				zoneVolume += volumeBins[i]
				i++
			}
			endIdx := i - 1

			zone := PriceZone{
				LowPrice:  binPrices[startIdx] - binSize/2,
				HighPrice: binPrices[endIdx] + binSize/2,
				Volume:    zoneVolume,
				Strength:  math.Min(100, (zoneVolume/totalVolume)*200), // 强度评分
				Type:      "high_volume_zone",
			}
			highVolumeZones = append(highVolumeZones, zone)
		} else {
			i++
		}
	}

	return &VolumeProfile{
		POC:             poc,
		VAH:             vah,
		VAL:             val,
		HighVolumeZones: highVolumeZones,
	}
}

// calculateBollingerBands 计算布林带
func calculateBollingerBands(klines []Kline, period int, stdDev float64, currentPrice float64) *BollingerBands {
	if len(klines) < period {
		return nil
	}

	// 计算中轨（SMA）
	sum := 0.0
	for i := len(klines) - period; i < len(klines); i++ {
		sum += klines[i].Close
	}
	middle := sum / float64(period)

	// 计算标准差
	variance := 0.0
	for i := len(klines) - period; i < len(klines); i++ {
		diff := klines[i].Close - middle
		variance += diff * diff
	}
	standardDeviation := math.Sqrt(variance / float64(period))

	// 计算上下轨
	upper := middle + stdDev*standardDeviation
	lower := middle - stdDev*standardDeviation

	// 计算带宽（波动率指标）
	width := ((upper - lower) / middle) * 100

	// 判断价格位置
	position := "in_band"
	if currentPrice > upper {
		position = "above_upper"
	} else if currentPrice < lower {
		position = "below_lower"
	}

	// 判断是否处于收缩状态（带宽 < 历史平均的70%）
	// 简化版：如果带宽 < 2%，认为是收缩状态
	squeeze := width < 2.0

	return &BollingerBands{
		Upper:    upper,
		Middle:   middle,
		Lower:    lower,
		Width:    width,
		Position: position,
		Squeeze:  squeeze,
	}
}

// detectBreakouts 检测突破信号
func detectBreakouts(klines []Kline, supportLevels []PriceLevel, resistanceLevels []PriceLevel) []BreakoutSignal {
	if len(klines) < 3 {
		return []BreakoutSignal{}
	}

	var signals []BreakoutSignal
	currentKline := klines[len(klines)-1]
	prevKline := klines[len(klines)-2]

	// 计算平均成交量
	avgVolume := 0.0
	lookback := 20
	if len(klines) < lookback {
		lookback = len(klines)
	}
	for i := len(klines) - lookback; i < len(klines); i++ {
		avgVolume += klines[i].Volume
	}
	avgVolume /= float64(lookback)

	// 检测压力位突破（向上突破）
	for _, resistance := range resistanceLevels {
		if prevKline.Close < resistance.Price && currentKline.Close > resistance.Price {
			// 计算突破强度（基于成交量）
			volumeRatio := currentKline.Volume / avgVolume
			strength := math.Min(100, volumeRatio*50)

			// 检查是否有回踩确认（简化版：检查后续K线）
			confirmed := false
			if len(klines) >= 4 {
				nextKline := klines[len(klines)-1]
				if nextKline.Low <= resistance.Price && nextKline.Close > resistance.Price {
					confirmed = true
				}
			}

			signals = append(signals, BreakoutSignal{
				Type:      "resistance_break",
				Price:     resistance.Price,
				Volume:    currentKline.Volume,
				Strength:  strength,
				Direction: "bullish",
				Confirmed: confirmed,
			})
		}
	}

	// 检测支撑位突破（向下突破）
	for _, support := range supportLevels {
		if prevKline.Close > support.Price && currentKline.Close < support.Price {
			volumeRatio := currentKline.Volume / avgVolume
			strength := math.Min(100, volumeRatio*50)

			confirmed := false
			if len(klines) >= 4 {
				nextKline := klines[len(klines)-1]
				if nextKline.High >= support.Price && nextKline.Close < support.Price {
					confirmed = true
				}
			}

			signals = append(signals, BreakoutSignal{
				Type:      "support_break",
				Price:     support.Price,
				Volume:    currentKline.Volume,
				Strength:  strength,
				Direction: "bearish",
				Confirmed: confirmed,
			})
		}
	}

	return signals
}

// identifyKeyPriceZones 识别关键价格区域
func identifyKeyPriceZones(supportLevels []PriceLevel, resistanceLevels []PriceLevel, volumeProfile *VolumeProfile) []PriceZone {
	var zones []PriceZone

	// 1. 从强支撑位创建支撑区域
	for _, support := range supportLevels {
		if support.Strength >= 60 {
			zone := PriceZone{
				LowPrice:  support.Price * 0.995, // 支撑位下0.5%
				HighPrice: support.Price * 1.005, // 支撑位上0.5%
				Volume:    0,                     // 从价格位计算，没有直接成交量数据
				Strength:  support.Strength,
				Type:      "support_zone",
			}
			zones = append(zones, zone)
		}
	}

	// 2. 从强压力位创建压力区域
	for _, resistance := range resistanceLevels {
		if resistance.Strength >= 60 {
			zone := PriceZone{
				LowPrice:  resistance.Price * 0.995,
				HighPrice: resistance.Price * 1.005,
				Volume:    0,
				Strength:  resistance.Strength,
				Type:      "resistance_zone",
			}
			zones = append(zones, zone)
		}
	}

	// 3. 添加成交量分布的高成交量区域
	if volumeProfile != nil {
		zones = append(zones, volumeProfile.HighVolumeZones...)
	}

	// 按强度排序，只保留最重要的5个区域
	for i := 0; i < len(zones)-1; i++ {
		for j := i + 1; j < len(zones); j++ {
			if zones[i].Strength < zones[j].Strength {
				zones[i], zones[j] = zones[j], zones[i]
			}
		}
	}

	if len(zones) > 5 {
		zones = zones[:5]
	}

	return zones
}
