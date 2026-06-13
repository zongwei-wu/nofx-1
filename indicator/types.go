package indicator

// Kline 统一 K 线输入结构（纯计算，零副作用）
type Kline struct {
	OpenTime int64
	Open     float64
	High     float64
	Low      float64
	Close    float64
	Volume   float64
}

// IndicatorResult 单值指标结果
type IndicatorResult struct {
	Current float64   `json:"current"`
	Series  []float64 `json:"series,omitempty"`
}

// MACDPoint MACD 序列点
type MACDPoint struct {
	MACD      float64 `json:"macd"`
	Signal    float64 `json:"signal"`
	Histogram float64 `json:"histogram"`
}

// MACDResult MACD 指标结果（含信号线/柱状图/交叉检测）
type MACDResult struct {
	MACD      float64     `json:"macd"`
	Signal    float64     `json:"signal"`
	Histogram float64     `json:"histogram"`
	Cross     string      `json:"cross,omitempty"` // "up" / "down" / "none"
	Series    []MACDPoint `json:"series,omitempty"`
}

// BollingerResult 布林带结果
type BollingerResult struct {
	Upper  float64 `json:"upper"`
	Middle float64 `json:"middle"`
	Lower  float64 `json:"lower"`
}

// StochasticResult 随机指标 KD 结果
type StochasticResult struct {
	K       float64   `json:"k"`
	D       float64   `json:"d"`
	KSeries []float64 `json:"k_series,omitempty"`
	DSeries []float64 `json:"d_series,omitempty"`
}

// ADXResult ADX 趋势强度结果
type ADXResult struct {
	ADX     float64 `json:"adx"`
	PlusDI  float64 `json:"plus_di"`
	MinusDI float64 `json:"minus_di"`
}

// IchimokuResult 一目均衡结果
type IchimokuResult struct {
	Tenkan  float64 `json:"tenkan"`
	Kijun   float64 `json:"kijun"`
	SenkouA float64 `json:"senkou_a"`
	SenkouB float64 `json:"senkou_b"`
	Chikou  float64 `json:"chikou"`
}

// Closes 提取收盘价序列
func Closes(klines []Kline) []float64 {
	out := make([]float64, len(klines))
	for i, k := range klines {
		out[i] = k.Close
	}
	return out
}
