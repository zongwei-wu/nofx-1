package indicator

import (
	"math"
	"testing"
)

func generateTestKlines(count int) []Kline {
	klines := make([]Kline, count)
	for i := 0; i < count; i++ {
		basePrice := 100.0
		variance := float64(i%10) * 0.5
		open := basePrice + variance
		klines[i] = Kline{
			OpenTime: int64(i * 180000),
			Open:     open,
			High:     open + 1.0,
			Low:      open - 0.5,
			Close:    open + 0.3,
			Volume:   1000.0 + float64(i*100),
		}
	}
	return klines
}

func TestEMA_InsufficientData(t *testing.T) {
	klines := generateTestKlines(5)
	result := EMA(klines, 20)
	if result.Current != 0 {
		t.Errorf("EMA with insufficient data = %.3f, want 0", result.Current)
	}
}

func TestEMA_Normal(t *testing.T) {
	klines := generateTestKlines(30)
	result := EMA(klines, 20)
	if result.Current <= 0 {
		t.Errorf("EMA = %.3f, expected > 0", result.Current)
	}
	if len(result.Series) != 11 {
		t.Errorf("EMA series length = %d, want 11", len(result.Series))
	}
}

func TestSMA_Normal(t *testing.T) {
	klines := generateTestKlines(25)
	result := SMA(klines, 20)
	if result.Current <= 0 {
		t.Errorf("SMA = %.3f, expected > 0", result.Current)
	}
}

func TestRSI_Bounds(t *testing.T) {
	klines := generateTestKlines(30)
	result := RSI(klines, 14)
	if result.Current < 0 || result.Current > 100 {
		t.Errorf("RSI = %.3f, expected 0-100", result.Current)
	}
}

func TestATR(t *testing.T) {
	klines := []Kline{
		{High: 50.0, Low: 48.0, Close: 49.0},
		{High: 51.0, Low: 49.0, Close: 50.0},
		{High: 52.0, Low: 50.0, Close: 51.0},
		{High: 53.0, Low: 51.0, Close: 52.0},
		{High: 54.0, Low: 52.0, Close: 53.0},
	}
	result := ATR(klines, 3)
	expectedATR := 2.0
	if math.Abs(result.Current-expectedATR) > 0.01 {
		t.Errorf("ATR = %.3f, want approximately %.3f", result.Current, expectedATR)
	}
}

func TestATR_InsufficientData(t *testing.T) {
	klines := []Kline{{High: 102.0, Low: 100.0, Close: 101.0}}
	result := ATR(klines, 14)
	if result.Current != 0 {
		t.Errorf("ATR with insufficient data = %.3f, want 0", result.Current)
	}
}

func TestMACD(t *testing.T) {
	klines := generateTestKlines(50)
	result := MACD(klines, 12, 26, 9)
	if result.Cross != "none" && result.Cross != "up" && result.Cross != "down" {
		t.Errorf("MACD cross = %q, invalid", result.Cross)
	}
}

func TestMACDLine(t *testing.T) {
	klines := generateTestKlines(30)
	line := MACDLine(klines)
	macd := MACD(klines, 12, 26, 9)
	if line != macd.MACD {
		t.Errorf("MACDLine = %.4f, MACD.MACD = %.4f", line, macd.MACD)
	}
}

func TestBollinger(t *testing.T) {
	klines := generateTestKlines(25)
	result := Bollinger(klines, 20, 2.0)
	if result.Upper <= result.Middle || result.Middle <= result.Lower {
		t.Errorf("Bollinger bands invalid: upper=%.3f middle=%.3f lower=%.3f",
			result.Upper, result.Middle, result.Lower)
	}
}

func TestStochastic(t *testing.T) {
	klines := generateTestKlines(20)
	result := Stochastic(klines, 14, 3)
	if result.K < 0 || result.K > 100 {
		t.Errorf("Stochastic K = %.3f, expected 0-100", result.K)
	}
}

func TestOBV(t *testing.T) {
	klines := []Kline{
		{Close: 100, Volume: 1000},
		{Close: 101, Volume: 1100},
		{Close: 99, Volume: 900},
	}
	result := OBV(klines)
	if len(result) != 3 {
		t.Fatalf("OBV length = %d, want 3", len(result))
	}
	if result[2] != 1200 {
		t.Errorf("OBV[2] = %.0f, want 1200", result[2])
	}
}

func TestVWAP(t *testing.T) {
	klines := generateTestKlines(10)
	result := VWAP(klines)
	if len(result) != 10 || result[9] <= 0 {
		t.Errorf("VWAP invalid: len=%d last=%.3f", len(result), result[9])
	}
}

func TestADX(t *testing.T) {
	klines := generateTestKlines(60)
	result := ADX(klines, 14)
	if result.ADX < 0 || result.ADX > 100 {
		t.Errorf("ADX = %.3f, expected 0-100", result.ADX)
	}
}

func TestIchimoku(t *testing.T) {
	klines := generateTestKlines(60)
	result := Ichimoku(klines)
	if result.Tenkan <= 0 || result.Kijun <= 0 {
		t.Errorf("Ichimoku invalid: tenkan=%.3f kijun=%.3f", result.Tenkan, result.Kijun)
	}
}

func TestMFI(t *testing.T) {
	klines := generateTestKlines(30)
	result := MFI(klines, 14)
	if result.Current < 0 || result.Current > 100 {
		t.Errorf("MFI = %.3f, expected 0-100", result.Current)
	}
}
