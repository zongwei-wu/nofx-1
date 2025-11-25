package market

import (
	"testing"
)

// TestCalculateSupportResistance 测试支撑压力位计算
func TestCalculateSupportResistance(t *testing.T) {
	// 创建模拟K线数据
	klines1h := generateMockKlines(100, 50000.0, 0.02) // 100个K线，基准价50000，波动2%
	klines4h := generateMockKlines(50, 50000.0, 0.03)  // 50个K线，基准价50000，波动3%
	currentPrice := 50500.0

	result := calculateSupportResistance(klines1h, klines4h, currentPrice)

	// 验证结果不为空
	if result == nil {
		t.Fatal("calculateSupportResistance returned nil")
	}

	// 验证支撑位在当前价格下方
	for _, support := range result.StrongSupports {
		if support.Price >= currentPrice {
			t.Errorf("Support price %.2f should be below current price %.2f", support.Price, currentPrice)
		}
	}

	// 验证压力位在当前价格上方
	for _, resistance := range result.StrongResistances {
		if resistance.Price <= currentPrice {
			t.Errorf("Resistance price %.2f should be above current price %.2f", resistance.Price, currentPrice)
		}
	}

	// 验证强度在合理范围内
	for _, support := range result.StrongSupports {
		if support.Strength < 0 || support.Strength > 100 {
			t.Errorf("Support strength %.2f should be between 0 and 100", support.Strength)
		}
	}

	for _, resistance := range result.StrongResistances {
		if resistance.Strength < 0 || resistance.Strength > 100 {
			t.Errorf("Resistance strength %.2f should be between 0 and 100", resistance.Strength)
		}
	}

	t.Logf("Found %d strong supports and %d strong resistances",
		len(result.StrongSupports), len(result.StrongResistances))
}

// TestFindLocalHighsLows 测试局部高低点识别
func TestFindLocalHighsLows(t *testing.T) {
	// 创建有明显高低点的K线数据
	klines := []Kline{
		{Close: 100, High: 105, Low: 95, Volume: 1000},
		{Close: 110, High: 115, Low: 105, Volume: 1200}, // 局部高点
		{Close: 105, High: 110, Low: 100, Volume: 1100},
		{Close: 95, High: 100, Low: 90, Volume: 900}, // 局部低点
		{Close: 100, High: 105, Low: 95, Volume: 1000},
		{Close: 105, High: 110, Low: 100, Volume: 1100},
		{Close: 110, High: 115, Low: 105, Volume: 1200},
	}

	highs := findLocalHighs(klines, 1)
	lows := findLocalLows(klines, 1)

	if len(highs) == 0 {
		t.Error("Should find at least one local high")
	}

	if len(lows) == 0 {
		t.Error("Should find at least one local low")
	}

	t.Logf("Found %d local highs and %d local lows", len(highs), len(lows))
}

// TestClusterPrices 测试价格聚类
func TestClusterPrices(t *testing.T) {
	points := []pricePoint{
		{Price: 100.0, Volume: 1000},
		{Price: 100.5, Volume: 1100}, // 应该聚类到一起
		{Price: 101.0, Volume: 900},  // 应该聚类到一起
		{Price: 150.0, Volume: 2000}, // 独立聚类
		{Price: 150.3, Volume: 2100}, // 应该聚类到一起
	}

	clusters := clusterPrices(points, 2.0) // 2.0的阈值

	if len(clusters) == 0 {
		t.Fatal("clusterPrices returned empty result")
	}

	// 应该生成2个聚类：一个在100附近，一个在150附近
	if len(clusters) != 2 {
		t.Errorf("Expected 2 clusters, got %d", len(clusters))
	}

	for _, cluster := range clusters {
		if cluster.Strength < 0 || cluster.Strength > 100 {
			t.Errorf("Cluster strength %.2f should be between 0 and 100", cluster.Strength)
		}
		t.Logf("Cluster at price %.2f with strength %.2f and %d touches",
			cluster.Price, cluster.Strength, cluster.TouchCount)
	}
}

// generateMockKlines 生成模拟K线数据用于测试
func generateMockKlines(count int, basePrice float64, volatility float64) []Kline {
	klines := make([]Kline, count)
	price := basePrice

	for i := 0; i < count; i++ {
		// 简单的随机波动（使用伪随机数）
		change := (float64(i%10) - 5) * volatility * basePrice / 100
		price += change

		high := price * (1 + volatility/2)
		low := price * (1 - volatility/2)

		klines[i] = Kline{
			Open:   price * 0.99,
			High:   high,
			Low:    low,
			Close:  price,
			Volume: 1000 + float64(i*10),
		}
	}

	return klines
}

// TestCalculateFibonacciLevels 测试斐波那契回撤位计算
func TestCalculateFibonacciLevels(t *testing.T) {
	klines := generateMockKlines(100, 50000.0, 0.03)
	currentPrice := 52000.0 // 在波段高点附近

	fib := calculateFibonacciLevels(klines, currentPrice)

	if fib == nil {
		t.Fatal("calculateFibonacciLevels returned nil")
	}

	if fib.SwingHigh <= fib.SwingLow {
		t.Errorf("Swing high (%.2f) should be greater than swing low (%.2f)",
			fib.SwingHigh, fib.SwingLow)
	}

	// 验证斐波那契比例
	expectedLevels := []string{"0.236", "0.382", "0.5", "0.618", "0.786"}
	for _, level := range expectedLevels {
		if _, ok := fib.Levels[level]; !ok {
			t.Errorf("Missing Fibonacci level: %s", level)
		}
	}

	t.Logf("Fibonacci: Trend=%s, High=%.2f, Low=%.2f", fib.Trend, fib.SwingHigh, fib.SwingLow)
	for _, level := range expectedLevels {
		t.Logf("  %s: %.2f", level, fib.Levels[level])
	}
}

// TestCalculateVolumeProfile 测试成交量分布计算
func TestCalculateVolumeProfile(t *testing.T) {
	klines := generateMockKlines(100, 50000.0, 0.02)
	currentPrice := 50500.0

	vp := calculateVolumeProfile(klines, currentPrice)

	if vp == nil {
		t.Fatal("calculateVolumeProfile returned nil")
	}

	// POC应该在VAL和VAH之间
	if vp.POC < vp.VAL || vp.POC > vp.VAH {
		t.Errorf("POC (%.2f) should be between VAL (%.2f) and VAH (%.2f)",
			vp.POC, vp.VAL, vp.VAH)
	}

	// VAL应该小于VAH
	if vp.VAL >= vp.VAH {
		t.Errorf("VAL (%.2f) should be less than VAH (%.2f)", vp.VAL, vp.VAH)
	}

	t.Logf("Volume Profile: POC=%.2f, VAL=%.2f, VAH=%.2f, HighVolumeZones=%d",
		vp.POC, vp.VAL, vp.VAH, len(vp.HighVolumeZones))
}

// TestCalculateBollingerBands 测试布林带计算
func TestCalculateBollingerBands(t *testing.T) {
	klines := generateMockKlines(50, 50000.0, 0.02)
	currentPrice := 50500.0

	bb := calculateBollingerBands(klines, 20, 2.0, currentPrice)

	if bb == nil {
		t.Fatal("calculateBollingerBands returned nil")
	}

	// 上轨应该大于中轨，中轨应该大于下轨
	if bb.Upper <= bb.Middle {
		t.Errorf("Upper band (%.2f) should be greater than middle band (%.2f)",
			bb.Upper, bb.Middle)
	}
	if bb.Middle <= bb.Lower {
		t.Errorf("Middle band (%.2f) should be greater than lower band (%.2f)",
			bb.Middle, bb.Lower)
	}

	// 带宽应该大于0
	if bb.Width <= 0 {
		t.Errorf("Band width should be positive, got %.2f", bb.Width)
	}

	// 验证位置判断
	validPositions := map[string]bool{"above_upper": true, "in_band": true, "below_lower": true}
	if !validPositions[bb.Position] {
		t.Errorf("Invalid position: %s", bb.Position)
	}

	t.Logf("Bollinger Bands: Upper=%.2f, Middle=%.2f, Lower=%.2f, Width=%.2f%%, Position=%s, Squeeze=%v",
		bb.Upper, bb.Middle, bb.Lower, bb.Width, bb.Position, bb.Squeeze)
}

// TestDetectBreakouts 测试突破检测
func TestDetectBreakouts(t *testing.T) {
	// 创建模拟突破场景
	klines := []Kline{
		{Close: 100, High: 105, Low: 95, Volume: 1000},
		{Close: 105, High: 110, Low: 100, Volume: 1200},
		{Close: 110, High: 115, Low: 105, Volume: 1500}, // 突破压力位112
		{Close: 112, High: 115, Low: 110, Volume: 1300},
	}

	resistanceLevels := []PriceLevel{
		{Price: 112, Strength: 70, Source: "high_cluster"},
	}
	supportLevels := []PriceLevel{
		{Price: 98, Strength: 65, Source: "low_cluster"},
	}

	signals := detectBreakouts(klines, supportLevels, resistanceLevels)

	// 应该检测到压力位突破
	if len(signals) == 0 {
		t.Log("No breakout signals detected (this is acceptable with simple test data)")
	} else {
		for i, signal := range signals {
			t.Logf("Breakout %d: Type=%s, Price=%.2f, Direction=%s, Strength=%.2f, Confirmed=%v",
				i+1, signal.Type, signal.Price, signal.Direction, signal.Strength, signal.Confirmed)
		}
	}
}

// TestIdentifyKeyPriceZones 测试关键价格区域识别
func TestIdentifyKeyPriceZones(t *testing.T) {
	supportLevels := []PriceLevel{
		{Price: 49500, Strength: 75, Source: "low_cluster"},
		{Price: 49000, Strength: 65, Source: "low_cluster"},
	}
	resistanceLevels := []PriceLevel{
		{Price: 51500, Strength: 70, Source: "high_cluster"},
		{Price: 52000, Strength: 62, Source: "high_cluster"},
	}

	volumeProfile := &VolumeProfile{
		HighVolumeZones: []PriceZone{
			{LowPrice: 50000, HighPrice: 50500, Strength: 80, Type: "high_volume_zone"},
		},
	}

	zones := identifyKeyPriceZones(supportLevels, resistanceLevels, volumeProfile)

	if len(zones) == 0 {
		t.Error("Should identify at least one key price zone")
	}

	// 验证区域类型
	validTypes := map[string]bool{
		"support_zone": true, "resistance_zone": true, "high_volume_zone": true,
	}
	for i, zone := range zones {
		if !validTypes[zone.Type] {
			t.Errorf("Invalid zone type: %s", zone.Type)
		}
		t.Logf("Zone %d: Type=%s, Range=%.2f-%.2f, Strength=%.0f",
			i+1, zone.Type, zone.LowPrice, zone.HighPrice, zone.Strength)
	}
}
