package strategy

import "time"

// StrategyConfig 策略配置
type StrategyConfig struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	ExchangeID  string `json:"exchange_id"`
	Symbol      string `json:"symbol"`
	Timeframe   string `json:"timeframe"`
	Direction   string `json:"direction"` // long_only / short_only / both

	EntryConditions       []ConditionGroup `json:"entry_conditions"`
	AddPositionConditions []ConditionGroup `json:"add_position_conditions,omitempty"`
	ExitConditions        []ConditionGroup `json:"exit_conditions,omitempty"`
	ExitRules             ExitRules        `json:"exit_rules"`
	Risk                  RiskConfig       `json:"risk"`
}

// ConditionGroup 条件组（组内 AND）
type ConditionGroup []Condition

// Condition 单个条件
type Condition struct {
	Indicator string  `json:"indicator"`
	Operator  string  `json:"operator"`
	Value     float64 `json:"value"`
	Timeframe string  `json:"timeframe,omitempty"`
}

// ExitRules 止盈止损规则
type ExitRules struct {
	TakeProfitPct   float64 `json:"take_profit_pct"`
	StopLossPct     float64 `json:"stop_loss_pct"`
	TrailingStop    bool    `json:"trailing_stop"`
	TrailingDistPct float64 `json:"trailing_dist_pct,omitempty"`
	TimeStop        string  `json:"time_stop,omitempty"`
}

// RiskConfig 风控配置
type RiskConfig struct {
	MaxPositions   int     `json:"max_positions"`
	Leverage       int     `json:"leverage"`
	RiskPerTrade   float64 `json:"risk_per_trade"`
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
}

// SignalType 信号类型
type SignalType string

const (
	SignalBuy  SignalType = "BUY"
	SignalSell SignalType = "SELL"
	SignalHold SignalType = "HOLD"
)

// Signal 策略信号
type Signal struct {
	Type       SignalType         `json:"type"`
	Symbol     string             `json:"symbol"`
	Price      float64            `json:"price"`
	Indicators map[string]float64 `json:"indicators"`
	Timestamp  time.Time          `json:"timestamp"`
	Reason     string             `json:"reason,omitempty"`
}

// StrategyStatus 策略状态
const (
	StatusDraft    = "draft"
	StatusActive   = "active"
	StatusPaused   = "paused"
	StatusArchived = "archived"
)
