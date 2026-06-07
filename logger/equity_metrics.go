package logger

// EquityMetrics 从账户快照推导的净值与盈亏指标。
type EquityMetrics struct {
	TotalEquity   float64
	TotalPnL      float64
	TotalPnLPct   float64
	InitialBase   float64
	WalletBalance float64
	UnrealizedPnL float64
}

// ComputeEquityMetrics 根据快照计算账户净值与相对初始余额的盈亏。
// TotalBalance 存钱包余额；TotalUnrealizedProfit 存未实现盈亏。
func ComputeEquityMetrics(s AccountSnapshot, fallbackInitial float64) EquityMetrics {
	wallet := s.TotalBalance
	unrealized := s.TotalUnrealizedProfit
	totalEquity := wallet + unrealized

	base := fallbackInitial
	if s.InitialBalance > 0 {
		base = s.InitialBalance
	}

	totalPnL := 0.0
	totalPnLPct := 0.0
	if base > 0 {
		totalPnL = totalEquity - base
		totalPnLPct = (totalPnL / base) * 100
	}

	return EquityMetrics{
		TotalEquity:   totalEquity,
		TotalPnL:      totalPnL,
		TotalPnLPct:   totalPnLPct,
		InitialBase:   base,
		WalletBalance: wallet,
		UnrealizedPnL: unrealized,
	}
}
