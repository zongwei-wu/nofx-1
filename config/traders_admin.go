package config

import (
	"fmt"
	"strings"
)

// AdminTraderItem 管理端交易员列表项（不含密钥）
type AdminTraderItem struct {
	ID                  string  `json:"id"`
	UserID              string  `json:"user_id"`
	UserEmail           string  `json:"user_email"`
	Name                string  `json:"name"`
	AIModelID           string  `json:"ai_model_id"`
	AIModelName         string  `json:"ai_model_name"`
	ExchangeID          string  `json:"exchange_id"`
	ExchangeName        string  `json:"exchange_name"`
	ExchangeType        string  `json:"exchange_type"`
	InitialBalance      float64 `json:"initial_balance"`
	ScanIntervalMinutes int     `json:"scan_interval_minutes"`
	IsRunning           bool    `json:"is_running"`
	BTCETHLeverage      int     `json:"btc_eth_leverage"`
	AltcoinLeverage     int     `json:"altcoin_leverage"`
	TradingSymbols      string  `json:"trading_symbols"`
	SystemPromptTemplate string `json:"system_prompt_template"`
	IsCrossMargin       bool    `json:"is_cross_margin"`
	UseCoinPool         bool    `json:"use_coin_pool"`
	UseOITop            bool    `json:"use_oi_top"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

// ListTradersAdmin 跨用户分页查询交易员
func (d *Database) ListTradersAdmin(page, pageSize int, userID, email string, isRunning *bool, nameFilter string) ([]AdminTraderItem, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	where := []string{"1=1"}
	args := []interface{}{}

	if userID != "" {
		where = append(where, "t.user_id = ?")
		args = append(args, userID)
	}
	if email != "" {
		where = append(where, "u.email LIKE ?")
		args = append(args, "%"+email+"%")
	}
	if isRunning != nil {
		where = append(where, "t.is_running = ?")
		args = append(args, *isRunning)
	}
	if nameFilter != "" {
		where = append(where, "t.name LIKE ?")
		args = append(args, "%"+nameFilter+"%")
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM traders t
		JOIN users u ON t.user_id = u.id
		WHERE ` + whereClause
	if err := d.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query := `
		SELECT
			t.id, t.user_id, u.email, t.name,
			t.ai_model_id, COALESCE(a.name, '') as ai_model_name,
			t.exchange_id, COALESCE(e.name, '') as exchange_name, COALESCE(e.type, '') as exchange_type,
			t.initial_balance, t.scan_interval_minutes, t.is_running,
			COALESCE(t.btc_eth_leverage, 5), COALESCE(t.altcoin_leverage, 5),
			COALESCE(t.trading_symbols, ''),
			COALESCE(t.system_prompt_template, 'default'),
			COALESCE(t.is_cross_margin, 1),
			COALESCE(t.use_coin_pool, 0), COALESCE(t.use_oi_top, 0),
			t.created_at, t.updated_at
		FROM traders t
		JOIN users u ON t.user_id = u.id
		LEFT JOIN ai_models a ON t.ai_model_id = a.id AND t.user_id = a.user_id
		LEFT JOIN exchanges e ON t.exchange_id = e.id AND t.user_id = e.user_id
		WHERE ` + whereClause + `
		ORDER BY t.created_at DESC
		LIMIT ? OFFSET ?`
	queryArgs := append(args, pageSize, offset)

	rows, err := d.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []AdminTraderItem
	for rows.Next() {
		var item AdminTraderItem
		var createdAt, updatedAt interface{}
		var isCross int
		var usePool, useOI int
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.UserEmail, &item.Name,
			&item.AIModelID, &item.AIModelName,
			&item.ExchangeID, &item.ExchangeName, &item.ExchangeType,
			&item.InitialBalance, &item.ScanIntervalMinutes, &item.IsRunning,
			&item.BTCETHLeverage, &item.AltcoinLeverage,
			&item.TradingSymbols,
			&item.SystemPromptTemplate,
			&isCross,
			&usePool, &useOI,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, 0, err
		}
		item.IsCrossMargin = isCross != 0
		item.UseCoinPool = usePool != 0
		item.UseOITop = useOI != 0
		item.CreatedAt = fmt.Sprint(createdAt)
		item.UpdatedAt = fmt.Sprint(updatedAt)
		items = append(items, item)
	}
	return items, total, nil
}

// GetTraderByID 按 ID 获取交易员（管理端，不校验归属）
func (d *Database) GetTraderByID(traderID string) (*TraderRecord, error) {
	var trader TraderRecord
	err := d.db.QueryRow(`
		SELECT id, user_id, name, ai_model_id, exchange_id, initial_balance, scan_interval_minutes, is_running,
		       COALESCE(btc_eth_leverage, 5), COALESCE(altcoin_leverage, 5),
		       COALESCE(trading_symbols, ''),
		       COALESCE(use_coin_pool, 0), COALESCE(use_oi_top, 0),
		       COALESCE(custom_prompt, ''), COALESCE(override_base_prompt, 0),
		       COALESCE(system_prompt_template, 'default'),
		       COALESCE(is_cross_margin, 1), created_at, updated_at
		FROM traders WHERE id = ?
	`, traderID).Scan(
		&trader.ID, &trader.UserID, &trader.Name, &trader.AIModelID, &trader.ExchangeID,
		&trader.InitialBalance, &trader.ScanIntervalMinutes, &trader.IsRunning,
		&trader.BTCETHLeverage, &trader.AltcoinLeverage, &trader.TradingSymbols,
		&trader.UseCoinPool, &trader.UseOITop,
		&trader.CustomPrompt, &trader.OverrideBasePrompt, &trader.SystemPromptTemplate,
		&trader.IsCrossMargin, &trader.CreatedAt, &trader.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &trader, nil
}
