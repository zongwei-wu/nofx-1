package backtest

import "time"

// BacktestConfig 回测配置
type BacktestConfig struct {
	Symbol         string    `json:"symbol"`
	Timeframe      string    `json:"timeframe"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	InitialCapital float64   `json:"initial_capital"`
	FeeRate        float64   `json:"fee_rate"`   // 默认 0.0004
	SlippageRate   float64   `json:"slippage_rate"` // 默认 0.0002
}

// BacktestMetrics 绩效指标
type BacktestMetrics struct {
	TotalReturnPct       float64 `json:"total_return_pct"`
	AnnualizedReturnPct  float64 `json:"annualized_return_pct"`
	MaxDrawdownPct       float64 `json:"max_drawdown_pct"`
	SharpeRatio          float64 `json:"sharpe_ratio"`
	WinRatePct           float64 `json:"win_rate_pct"`
	ProfitFactor         float64 `json:"profit_factor"`
	AvgWinPct            float64 `json:"avg_win_pct"`
	AvgLossPct           float64 `json:"avg_loss_pct"`
	MaxConsecutiveLosses int     `json:"max_consecutive_losses"`
	TotalTrades          int     `json:"total_trades"`
}

// EquityPoint 权益曲线点
type EquityPoint struct {
	Time   int64   `json:"time"`
	Equity float64 `json:"equity"`
}

// TradeRecord 交易记录
type TradeRecord struct {
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	EntryTime  time.Time `json:"entry_time"`
	ExitTime   time.Time `json:"exit_time"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	Quantity   float64   `json:"quantity"`
	PnL        float64   `json:"pnl"`
	PnLPct     float64   `json:"pnl_pct"`
	Fee        float64   `json:"fee"`
}

// BacktestResult 回测结果
type BacktestResult struct {
	StrategyName   string          `json:"strategy_name"`
	Symbol         string          `json:"symbol"`
	Timeframe      string          `json:"timeframe"`
	StartTime      time.Time       `json:"start_time"`
	EndTime        time.Time       `json:"end_time"`
	InitialCapital float64         `json:"initial_capital"`
	FinalEquity    float64         `json:"final_equity"`
	Metrics        BacktestMetrics `json:"metrics"`
	EquityCurve    []EquityPoint   `json:"equity_curve"`
	Trades         []TradeRecord   `json:"trades"`
}
