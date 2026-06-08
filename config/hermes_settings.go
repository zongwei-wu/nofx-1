package config

import (
	"database/sql"
	"encoding/json"
	"time"
)

// HermesSettings Hermes 自主交易配置（每用户一条）
type HermesSettings struct {
	UserID               string    `json:"user_id"`
	RunnerEnabled        bool      `json:"runner_enabled"`
	AutonomousEnabled    bool      `json:"autonomous_enabled"`
	ExchangeID           string    `json:"exchange_id"`
	AIModelID            string    `json:"ai_model_id"`
	ScanIntervalMinutes  int       `json:"scan_interval_minutes"`
	SystemPromptTemplate string    `json:"system_prompt_template"`
	BTCETHLeverage       int       `json:"btc_eth_leverage"`
	AltcoinLeverage      int       `json:"altcoin_leverage"`
	TradingCoins         []string  `json:"trading_coins"`
	InitialBalance       float64   `json:"initial_balance"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func defaultHermesSettings(userID string) *HermesSettings {
	return &HermesSettings{
		UserID:               userID,
		ScanIntervalMinutes:  5,
		SystemPromptTemplate: "hermes",
		BTCETHLeverage:       5,
		AltcoinLeverage:      3,
		TradingCoins:         []string{},
	}
}

// GetHermesSettings 获取 Hermes 配置
func (d *Database) GetHermesSettings(userID string) (*HermesSettings, error) {
	var (
		runnerEnabled, autonomousEnabled int
		exchangeID, aiModelID, template  string
		scanInterval                     int
		btcLev, altLev                   int
		tradingCoinsJSON                 string
		initialBalance                   float64
		updatedAt                        time.Time
	)
	err := d.db.QueryRow(`
		SELECT runner_enabled, autonomous_enabled, exchange_id, ai_model_id,
		       scan_interval_minutes, system_prompt_template,
		       btc_eth_leverage, altcoin_leverage, trading_coins,
		       initial_balance, updated_at
		FROM hermes_settings WHERE user_id = ?
	`, userID).Scan(
		&runnerEnabled, &autonomousEnabled, &exchangeID, &aiModelID,
		&scanInterval, &template, &btcLev, &altLev, &tradingCoinsJSON,
		&initialBalance, &updatedAt,
	)
	if err == sql.ErrNoRows {
		return defaultHermesSettings(userID), nil
	}
	if err != nil {
		return nil, err
	}
	coins, _ := parseSymbolJSONArray(tradingCoinsJSON)
	if coins == nil {
		coins = []string{}
	}
	return &HermesSettings{
		UserID:               userID,
		RunnerEnabled:        runnerEnabled != 0,
		AutonomousEnabled:    autonomousEnabled != 0,
		ExchangeID:           exchangeID,
		AIModelID:            aiModelID,
		ScanIntervalMinutes:  scanInterval,
		SystemPromptTemplate: template,
		BTCETHLeverage:       btcLev,
		AltcoinLeverage:      altLev,
		TradingCoins:         coins,
		InitialBalance:       initialBalance,
		UpdatedAt:            updatedAt,
	}, nil
}

// UpsertHermesSettings 保存 Hermes 配置
func (d *Database) UpsertHermesSettings(s *HermesSettings) error {
	if s == nil {
		return nil
	}
	coinsJSON, _ := json.Marshal(s.TradingCoins)
	if s.TradingCoins == nil {
		coinsJSON = []byte("[]")
	}
	if s.ScanIntervalMinutes <= 0 {
		s.ScanIntervalMinutes = 5
	}
	if s.SystemPromptTemplate == "" {
		s.SystemPromptTemplate = "hermes"
	}
	if s.BTCETHLeverage <= 0 {
		s.BTCETHLeverage = 5
	}
	if s.AltcoinLeverage <= 0 {
		s.AltcoinLeverage = 3
	}
	runner := 0
	if s.RunnerEnabled {
		runner = 1
	}
	auto := 0
	if s.AutonomousEnabled {
		auto = 1
	}
	_, err := d.db.Exec(`
		INSERT INTO hermes_settings (
			user_id, runner_enabled, autonomous_enabled, exchange_id, ai_model_id,
			scan_interval_minutes, system_prompt_template,
			btc_eth_leverage, altcoin_leverage, trading_coins, initial_balance, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			runner_enabled = excluded.runner_enabled,
			autonomous_enabled = excluded.autonomous_enabled,
			exchange_id = excluded.exchange_id,
			ai_model_id = excluded.ai_model_id,
			scan_interval_minutes = excluded.scan_interval_minutes,
			system_prompt_template = excluded.system_prompt_template,
			btc_eth_leverage = excluded.btc_eth_leverage,
			altcoin_leverage = excluded.altcoin_leverage,
			trading_coins = excluded.trading_coins,
			initial_balance = excluded.initial_balance,
			updated_at = CURRENT_TIMESTAMP
	`, s.UserID, runner, auto, s.ExchangeID, s.AIModelID,
		s.ScanIntervalMinutes, s.SystemPromptTemplate,
		s.BTCETHLeverage, s.AltcoinLeverage, string(coinsJSON), s.InitialBalance)
	return err
}

// ListHermesRunnerEnabledUserIDs 列出需恢复运行的用户
func (d *Database) ListHermesRunnerEnabledUserIDs() ([]string, error) {
	rows, err := d.db.Query(`SELECT user_id FROM hermes_settings WHERE runner_enabled = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// EnsureHermesPlanFeature 为已有库补充 hermes 功能绑定
func (d *Database) EnsureHermesPlanFeature() error {
	for _, planID := range []string{PlanStandard, PlanPro, PlanVIP} {
		_, err := d.db.Exec(`
			INSERT OR IGNORE INTO plan_features (plan_id, feature_key) VALUES (?, ?)
		`, planID, FeatureHermes)
		if err != nil {
			return err
		}
	}
	return nil
}
