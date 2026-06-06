package config

import "fmt"

// 功能权限 key（可扩展）
const (
	FeatureCompetition = "competition"
	FeatureAITrader    = "ai_trader"
	FeatureLeaderboard = "leaderboard"
	FeatureCopyTrade   = "copy_trade"
	FeatureSymbols     = "symbols"
)

// 用户套餐
const (
	PlanBasic    = "basic"
	PlanStandard = "standard"
	PlanPro      = "pro"
	PlanVIP      = "vip"
)

// AllFeatures 全部功能列表
var AllFeatures = []string{
	FeatureCompetition,
	FeatureAITrader,
	FeatureLeaderboard,
	FeatureCopyTrade,
	FeatureSymbols,
}

// PlanInfo 套餐信息
type PlanInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	SortOrder  int      `json:"sort_order"`
	Features   []string `json:"features"`
}

// initPlanData 初始化套餐与功能绑定
func (d *Database) initPlanData() error {
	plans := []struct {
		id, name string
		order    int
	}{
		{PlanBasic, "基础版", 1},
		{PlanStandard, "标准版", 2},
		{PlanPro, "专业版", 3},
		{PlanVIP, "VIP", 4},
	}
	for _, p := range plans {
		_, err := d.db.Exec(`
			INSERT OR IGNORE INTO plans (id, name, sort_order) VALUES (?, ?, ?)
		`, p.id, p.name, p.order)
		if err != nil {
			return fmt.Errorf("初始化套餐 %s 失败: %w", p.id, err)
		}
	}

	planFeatures := map[string][]string{
		PlanBasic:    {FeatureCompetition},
		PlanStandard: {FeatureCompetition, FeatureAITrader, FeatureSymbols},
		PlanPro:      {FeatureCompetition, FeatureAITrader, FeatureSymbols, FeatureLeaderboard, FeatureCopyTrade},
		PlanVIP:      {FeatureCompetition, FeatureAITrader, FeatureSymbols, FeatureLeaderboard, FeatureCopyTrade},
	}
	for planID, features := range planFeatures {
		for _, f := range features {
			_, err := d.db.Exec(`
				INSERT OR IGNORE INTO plan_features (plan_id, feature_key) VALUES (?, ?)
			`, planID, f)
			if err != nil {
				return fmt.Errorf("初始化套餐功能 %s/%s 失败: %w", planID, f, err)
			}
		}
	}
	return nil
}

// GetPlanFeatures 获取套餐绑定的功能列表
func (d *Database) GetPlanFeatures(planID string) ([]string, error) {
	rows, err := d.db.Query(`
		SELECT feature_key FROM plan_features WHERE plan_id = ? ORDER BY feature_key
	`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		features = append(features, f)
	}
	return features, nil
}

// GetUserFeatures 获取用户当前可用功能（admin 返回全部）
func (d *Database) GetUserFeatures(userID, role string) ([]string, error) {
	if role == UserRoleAdmin {
		return append([]string(nil), AllFeatures...), nil
	}
	plan, err := d.GetUserPlan(userID)
	if err != nil {
		return nil, err
	}
	return d.GetPlanFeatures(plan)
}

// GetUserPlan 获取用户套餐
func (d *Database) GetUserPlan(userID string) (string, error) {
	var plan string
	err := d.db.QueryRow(`SELECT plan FROM users WHERE id = ?`, userID).Scan(&plan)
	if err != nil {
		return "", err
	}
	if plan == "" {
		return PlanStandard, nil
	}
	return plan, nil
}

// UserHasFeature 检查用户是否拥有某功能
func (d *Database) UserHasFeature(userID, role, feature string) (bool, error) {
	if role == UserRoleAdmin {
		return true, nil
	}
	features, err := d.GetUserFeatures(userID, role)
	if err != nil {
		return false, err
	}
	for _, f := range features {
		if f == feature {
			return true, nil
		}
	}
	return false, nil
}

// PlanExists 检查套餐是否存在
func (d *Database) PlanExists(planID string) (bool, error) {
	var count int
	err := d.db.QueryRow(`SELECT COUNT(*) FROM plans WHERE id = ?`, planID).Scan(&count)
	return count > 0, err
}

// ListPlans 列出所有套餐及功能
func (d *Database) ListPlans() ([]PlanInfo, error) {
	rows, err := d.db.Query(`SELECT id, name, sort_order FROM plans ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []PlanInfo
	for rows.Next() {
		var p PlanInfo
		if err := rows.Scan(&p.ID, &p.Name, &p.SortOrder); err != nil {
			return nil, err
		}
		p.Features, err = d.GetPlanFeatures(p.ID)
		if err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, nil
}

// UpdateUserPlan 更新用户套餐
func (d *Database) UpdateUserPlan(userID, plan string) error {
	exists, err := d.PlanExists(plan)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("套餐不存在: %s", plan)
	}
	_, err = d.db.Exec(`UPDATE users SET plan = ? WHERE id = ?`, plan, userID)
	return err
}
