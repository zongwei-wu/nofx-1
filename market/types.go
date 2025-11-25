package market

import "time"

// Data 市场数据结构
type Data struct {
	Symbol            string
	CurrentPrice      float64
	PriceChange15m    float64 // 15分钟价格变化百分比
	PriceChange1h     float64 // 1小时价格变化百分比
	PriceChange4h     float64 // 4小时价格变化百分比
	CurrentEMA20      float64
	CurrentMACD       float64
	CurrentRSI7       float64
	OpenInterest      *OIData
	FundingRate       float64
	IntradaySeries    *IntradayData
	FifteenMinContext *KlineContextData // 15分钟K线上下文数据
	OneHourContext    *KlineContextData // 1小时K线上下文数据
	LongerTermContext *LongerTermData
	SupportResistance *SupportResistanceLevels // 支撑压力位
}

// OIData Open Interest数据
type OIData struct {
	Latest  float64
	Average float64
}

// IntradayData 日内数据(3分钟间隔)
type IntradayData struct {
	MidPrices   []float64
	EMA20Values []float64
	MACDValues  []float64
	RSI7Values  []float64
	RSI14Values []float64
	Volume      []float64
	ATR14       float64
}

// KlineContextData K线上下文数据
type KlineContextData struct {
	EMA20   float64
	EMA50   float64
	MACD    float64
	RSI14   float64
	ATR14   float64
	Volume  float64
	Trend   string // 趋势: uptrend, downtrend, sideways
	Pattern string // 形态模式
}

// LongerTermData 长期数据(4小时时间框架)
type LongerTermData struct {
	EMA20         float64
	EMA50         float64
	ATR3          float64
	ATR14         float64
	CurrentVolume float64
	AverageVolume float64
	MACDValues    []float64
	RSI14Values   []float64
}

// SupportResistanceLevels 支撑压力位数据
type SupportResistanceLevels struct {
	StrongSupports    []PriceLevel     `json:"strong_supports"`    // 强支撑位（从近到远）
	WeakSupports      []PriceLevel     `json:"weak_supports"`      // 弱支撑位
	StrongResistances []PriceLevel     `json:"strong_resistances"` // 强压力位（从近到远）
	WeakResistances   []PriceLevel     `json:"weak_resistances"`   // 弱压力位
	NearestSupport    *PriceLevel      `json:"nearest_support"`    // 最近支撑位
	NearestResistance *PriceLevel      `json:"nearest_resistance"` // 最近压力位
	FibonacciLevels   *FibonacciLevels `json:"fibonacci_levels"`   // 斐波那契回撤位
	VolumeProfile     *VolumeProfile   `json:"volume_profile"`     // 成交量分布
	BollingerBands    *BollingerBands  `json:"bollinger_bands"`    // 布林带
	KeyPriceZones     []PriceZone      `json:"key_price_zones"`    // 关键价格区域
	BreakoutSignals   []BreakoutSignal `json:"breakout_signals"`   // 突破信号
}

// PriceLevel 价格位信息
type PriceLevel struct {
	Price      float64 `json:"price"`       // 价格位
	Strength   float64 `json:"strength"`    // 强度（0-100）
	Distance   float64 `json:"distance"`    // 距离当前价格百分比
	Source     string  `json:"source"`      // 来源：high_cluster, low_cluster, volume_profile, fibonacci, bollinger
	TouchCount int     `json:"touch_count"` // 触及次数
}

// FibonacciLevels 斐波那契回撤位
type FibonacciLevels struct {
	SwingHigh float64            `json:"swing_high"` // 波段高点
	SwingLow  float64            `json:"swing_low"`  // 波段低点
	Trend     string             `json:"trend"`      // 趋势方向: uptrend, downtrend
	Levels    map[string]float64 `json:"levels"`     // 回撤位：0.236, 0.382, 0.5, 0.618, 0.786
}

// VolumeProfile 成交量分布
type VolumeProfile struct {
	POC             float64     `json:"poc"`               // Point of Control (最大成交量价格)
	VAH             float64     `json:"vah"`               // Value Area High (70%成交量上界)
	VAL             float64     `json:"val"`               // Value Area Low (70%成交量下界)
	HighVolumeZones []PriceZone `json:"high_volume_zones"` // 高成交量区域
}

// BollingerBands 布林带
type BollingerBands struct {
	Upper    float64 `json:"upper"`    // 上轨
	Middle   float64 `json:"middle"`   // 中轨（MA）
	Lower    float64 `json:"lower"`    // 下轨
	Width    float64 `json:"width"`    // 带宽（波动率）
	Position string  `json:"position"` // 价格位置: above_upper, in_band, below_lower
	Squeeze  bool    `json:"squeeze"`  // 是否处于收缩状态（低波动）
}

// PriceZone 价格区域
type PriceZone struct {
	HighPrice float64 `json:"high_price"` // 区域高价
	LowPrice  float64 `json:"low_price"`  // 区域低价
	Volume    float64 `json:"volume"`     // 区域成交量
	Strength  float64 `json:"strength"`   // 区域强度
	Type      string  `json:"type"`       // 类型: support_zone, resistance_zone, high_volume_zone
}

// BreakoutSignal 突破信号
type BreakoutSignal struct {
	Type      string  `json:"type"`      // 类型: support_break, resistance_break
	Price     float64 `json:"price"`     // 突破价格
	Volume    float64 `json:"volume"`    // 突破时成交量
	Strength  float64 `json:"strength"`  // 突破强度（成交量确认）
	Direction string  `json:"direction"` // 方向: bullish, bearish
	Confirmed bool    `json:"confirmed"` // 是否确认（回踩测试）
}

// Binance API 响应结构
type ExchangeInfo struct {
	Symbols []SymbolInfo `json:"symbols"`
}

type SymbolInfo struct {
	Symbol            string `json:"symbol"`
	Status            string `json:"status"`
	BaseAsset         string `json:"baseAsset"`
	QuoteAsset        string `json:"quoteAsset"`
	ContractType      string `json:"contractType"`
	PricePrecision    int    `json:"pricePrecision"`
	QuantityPrecision int    `json:"quantityPrecision"`
}

type Kline struct {
	OpenTime            int64   `json:"openTime"`
	Open                float64 `json:"open"`
	High                float64 `json:"high"`
	Low                 float64 `json:"low"`
	Close               float64 `json:"close"`
	Volume              float64 `json:"volume"`
	CloseTime           int64   `json:"closeTime"`
	QuoteVolume         float64 `json:"quoteVolume"`
	Trades              int     `json:"trades"`
	TakerBuyBaseVolume  float64 `json:"takerBuyBaseVolume"`
	TakerBuyQuoteVolume float64 `json:"takerBuyQuoteVolume"`
}

type KlineResponse []interface{}

type PriceTicker struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type Ticker24hr struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
}

// 特征数据结构
type SymbolFeatures struct {
	Symbol           string    `json:"symbol"`
	Timestamp        time.Time `json:"timestamp"`
	Price            float64   `json:"price"`
	PriceChange15Min float64   `json:"price_change_15min"`
	PriceChange1H    float64   `json:"price_change_1h"`
	PriceChange4H    float64   `json:"price_change_4h"`
	Volume           float64   `json:"volume"`
	VolumeRatio5     float64   `json:"volume_ratio_5"`
	VolumeRatio20    float64   `json:"volume_ratio_20"`
	VolumeTrend      float64   `json:"volume_trend"`
	RSI14            float64   `json:"rsi_14"`
	SMA5             float64   `json:"sma_5"`
	SMA10            float64   `json:"sma_10"`
	SMA20            float64   `json:"sma_20"`
	HighLowRatio     float64   `json:"high_low_ratio"`
	Volatility20     float64   `json:"volatility_20"`
	PositionInRange  float64   `json:"position_in_range"`
}

// 警报数据结构
type Alert struct {
	Type      string    `json:"type"`
	Symbol    string    `json:"symbol"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type Config struct {
	AlertThresholds AlertThresholds `json:"alert_thresholds"`
	UpdateInterval  int             `json:"update_interval"` // seconds
	CleanupConfig   CleanupConfig   `json:"cleanup_config"`
}

type AlertThresholds struct {
	VolumeSpike      float64 `json:"volume_spike"`
	PriceChange15Min float64 `json:"price_change_15min"`
	VolumeTrend      float64 `json:"volume_trend"`
	RSIOverbought    float64 `json:"rsi_overbought"`
	RSIOversold      float64 `json:"rsi_oversold"`
}
type CleanupConfig struct {
	InactiveTimeout   time.Duration `json:"inactive_timeout"`    // 不活跃超时时间
	MinScoreThreshold float64       `json:"min_score_threshold"` // 最低评分阈值
	NoAlertTimeout    time.Duration `json:"no_alert_timeout"`    // 无警报超时时间
	CheckInterval     time.Duration `json:"check_interval"`      // 检查间隔
}

var config = Config{
	AlertThresholds: AlertThresholds{
		VolumeSpike:      3.0,
		PriceChange15Min: 0.05,
		VolumeTrend:      2.0,
		RSIOverbought:    70,
		RSIOversold:      30,
	},
	CleanupConfig: CleanupConfig{
		InactiveTimeout:   30 * time.Minute,
		MinScoreThreshold: 15.0,
		NoAlertTimeout:    20 * time.Minute,
		CheckInterval:     5 * time.Minute,
	},
	UpdateInterval: 60, // 1 minute
}
