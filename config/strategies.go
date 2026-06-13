package config

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// StrategyRecord 策略数据库记录
type StrategyRecord struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Config    string    `json:"config"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StrategySignalRecord 策略信号记录
type StrategySignalRecord struct {
	ID          string    `json:"id"`
	StrategyID  string    `json:"strategy_id"`
	Symbol      string    `json:"symbol"`
	Signal      string    `json:"signal"`
	Price       float64   `json:"price"`
	Indicators  string    `json:"indicators"`
	OrderResult string    `json:"order_result"`
	CreatedAt   time.Time `json:"created_at"`
}

// BacktestRecord 回测记录
type BacktestRecord struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	StrategyID     string     `json:"strategy_id"`
	Symbol         string     `json:"symbol"`
	Timeframe      string     `json:"timeframe"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        time.Time  `json:"end_time"`
	InitialCapital float64    `json:"initial_capital"`
	Result         string     `json:"result"`
	Status         string     `json:"status"`
	Error          string     `json:"error"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

func (d *Database) initStrategyTables() {
	d.db.Exec(`CREATE TABLE IF NOT EXISTS strategies (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		config TEXT NOT NULL,
		status TEXT DEFAULT 'draft',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	)`)
	d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_strategies_user ON strategies(user_id)`)
	d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_strategies_status ON strategies(status)`)

	d.db.Exec(`CREATE TABLE IF NOT EXISTS strategy_signals (
		id TEXT PRIMARY KEY,
		strategy_id TEXT NOT NULL,
		symbol TEXT NOT NULL,
		signal TEXT NOT NULL,
		price REAL,
		indicators TEXT,
		order_result TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (strategy_id) REFERENCES strategies(id)
	)`)
	d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_signals_strategy ON strategy_signals(strategy_id)`)

	d.db.Exec(`CREATE TABLE IF NOT EXISTS backtests (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		strategy_id TEXT NOT NULL,
		symbol TEXT NOT NULL,
		timeframe TEXT NOT NULL,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP NOT NULL,
		initial_capital REAL DEFAULT 1000,
		result TEXT,
		status TEXT DEFAULT 'pending',
		error TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (strategy_id) REFERENCES strategies(id)
	)`)
	d.db.Exec(`CREATE INDEX IF NOT EXISTS idx_backtests_user ON backtests(user_id)`)
}

// CreateStrategy 创建策略
func (d *Database) CreateStrategy(record *StrategyRecord) error {
	_, err := d.db.Exec(`
		INSERT INTO strategies (id, user_id, name, config, status)
		VALUES (?, ?, ?, ?, ?)
	`, record.ID, record.UserID, record.Name, record.Config, record.Status)
	return err
}

// GetStrategies 获取用户策略列表
func (d *Database) GetStrategies(userID string) ([]*StrategyRecord, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, name, config, status, created_at, updated_at
		FROM strategies WHERE user_id = ? ORDER BY updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrategyRows(rows)
}

// GetActiveStrategies 获取所有激活策略
func (d *Database) GetActiveStrategies() ([]*StrategyRecord, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, name, config, status, created_at, updated_at
		FROM strategies WHERE status = 'active' ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrategyRows(rows)
}

func scanStrategyRows(rows *sql.Rows) ([]*StrategyRecord, error) {
	var out []*StrategyRecord
	for rows.Next() {
		var r StrategyRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.Config, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, nil
}

// GetStrategy 获取单个策略
func (d *Database) GetStrategy(userID, id string) (*StrategyRecord, error) {
	var r StrategyRecord
	err := d.db.QueryRow(`
		SELECT id, user_id, name, config, status, created_at, updated_at
		FROM strategies WHERE id = ? AND user_id = ?
	`, id, userID).Scan(&r.ID, &r.UserID, &r.Name, &r.Config, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// UpdateStrategy 更新策略
func (d *Database) UpdateStrategy(record *StrategyRecord) error {
	_, err := d.db.Exec(`
		UPDATE strategies SET name = ?, config = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`, record.Name, record.Config, record.Status, record.ID, record.UserID)
	return err
}

// UpdateStrategyStatus 更新策略状态
func (d *Database) UpdateStrategyStatus(userID, id, status string) error {
	_, err := d.db.Exec(`
		UPDATE strategies SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND user_id = ?
	`, status, id, userID)
	return err
}

// DeleteStrategy 删除策略
func (d *Database) DeleteStrategy(userID, id string) error {
	_, err := d.db.Exec(`DELETE FROM strategies WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

// CreateStrategySignal 记录策略信号
func (d *Database) CreateStrategySignal(record *StrategySignalRecord) error {
	_, err := d.db.Exec(`
		INSERT INTO strategy_signals (id, strategy_id, symbol, signal, price, indicators, order_result)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, record.ID, record.StrategyID, record.Symbol, record.Signal, record.Price, record.Indicators, record.OrderResult)
	return err
}

// GetStrategySignals 获取策略信号历史
func (d *Database) GetStrategySignals(strategyID string, limit int) ([]*StrategySignalRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := d.db.Query(`
		SELECT id, strategy_id, symbol, signal, price, indicators, order_result, created_at
		FROM strategy_signals WHERE strategy_id = ?
		ORDER BY created_at DESC LIMIT ?
	`, strategyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*StrategySignalRecord
	for rows.Next() {
		var r StrategySignalRecord
		if err := rows.Scan(&r.ID, &r.StrategyID, &r.Symbol, &r.Signal, &r.Price, &r.Indicators, &r.OrderResult, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, nil
}

// ParseStrategyConfig 解析策略 JSON 配置
func ParseStrategyConfig(record *StrategyRecord) (map[string]interface{}, error) {
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(record.Config), &cfg); err != nil {
		return nil, fmt.Errorf("解析策略配置失败: %w", err)
	}
	return cfg, nil
}

// CreateBacktest 创建回测记录
func (d *Database) CreateBacktest(record *BacktestRecord) error {
	_, err := d.db.Exec(`
		INSERT INTO backtests (id, user_id, strategy_id, symbol, timeframe, start_time, end_time, initial_capital, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.ID, record.UserID, record.StrategyID, record.Symbol, record.Timeframe,
		record.StartTime, record.EndTime, record.InitialCapital, record.Status)
	return err
}

// GetBacktest 获取回测记录
func (d *Database) GetBacktest(userID, id string) (*BacktestRecord, error) {
	var r BacktestRecord
	var completedAt sql.NullTime
	err := d.db.QueryRow(`
		SELECT id, user_id, strategy_id, symbol, timeframe, start_time, end_time,
		       initial_capital, result, status, error, created_at, completed_at
		FROM backtests WHERE id = ? AND user_id = ?
	`, id, userID).Scan(
		&r.ID, &r.UserID, &r.StrategyID, &r.Symbol, &r.Timeframe,
		&r.StartTime, &r.EndTime, &r.InitialCapital, &r.Result, &r.Status, &r.Error,
		&r.CreatedAt, &completedAt,
	)
	if err != nil {
		return nil, err
	}
	if completedAt.Valid {
		r.CompletedAt = &completedAt.Time
	}
	return &r, nil
}

// GetBacktests 获取用户回测列表
func (d *Database) GetBacktests(userID string) ([]*BacktestRecord, error) {
	rows, err := d.db.Query(`
		SELECT id, user_id, strategy_id, symbol, timeframe, start_time, end_time,
		       initial_capital, result, status, error, created_at, completed_at
		FROM backtests WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*BacktestRecord
	for rows.Next() {
		var r BacktestRecord
		var completedAt sql.NullTime
		if err := rows.Scan(
			&r.ID, &r.UserID, &r.StrategyID, &r.Symbol, &r.Timeframe,
			&r.StartTime, &r.EndTime, &r.InitialCapital, &r.Result, &r.Status, &r.Error,
			&r.CreatedAt, &completedAt,
		); err != nil {
			return nil, err
		}
		if completedAt.Valid {
			r.CompletedAt = &completedAt.Time
		}
		out = append(out, &r)
	}
	return out, nil
}

// UpdateBacktestResult 更新回测结果
func (d *Database) UpdateBacktestResult(id, status, result, errMsg string) error {
	_, err := d.db.Exec(`
		UPDATE backtests SET status = ?, result = ?, error = ?, completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, status, result, errMsg, id)
	return err
}
