package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"nofx/config"

	"github.com/gin-gonic/gin"
)

// adminSystemConfigKeys 管理端可读写系统配置白名单
var adminSystemConfigKeys = []string{
	"registration_enabled",
	"beta_mode",
	"default_coins",
	"use_default_coins",
	"btc_eth_leverage",
	"altcoin_leverage",
	"max_daily_loss",
	"max_drawdown",
	"stop_trading_minutes",
}

// handleAdminMe 返回当前管理员信息
func (s *Server) handleAdminMe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"user_id": c.GetString("user_id"),
		"email":   c.GetString("email"),
		"role":    c.GetString("role"),
	})
}

// handleAdminListUsers 分页查询用户列表
func (s *Server) handleAdminListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current", c.DefaultQuery("page", "1")))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	emailFilter := strings.TrimSpace(c.Query("email"))

	users, total, err := s.database.ListUsers(page, pageSize, emailFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "查询用户失败"})
		return
	}

	data := make([]gin.H, 0, len(users))
	for _, u := range users {
		data = append(data, gin.H{
			"id":           u.ID,
			"email":        u.Email,
			"role":         u.Role,
			"plan":         u.Plan,
			"otp_verified": u.OTPVerified,
			"created_at":   u.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    data,
		"total":   total,
		"success": true,
	})
}

// handleAdminCopyTradeRecords 跨用户分页查询跟单记录
func (s *Server) handleAdminCopyTradeRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current", c.DefaultQuery("page", "1")))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	userID := strings.TrimSpace(c.Query("user_id"))
	status := strings.TrimSpace(c.Query("status"))
	symbol := strings.TrimSpace(c.Query("symbol"))

	where := []string{"1=1"}
	args := []interface{}{}

	if userID != "" {
		where = append(where, "user_id = ?")
		args = append(args, userID)
	}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	if symbol != "" {
		where = append(where, "symbol LIKE ?")
		args = append(args, "%"+symbol+"%")
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := "SELECT COUNT(*) FROM copy_trade_records WHERE " + whereClause
	if err := s.database.DB().QueryRow(countQuery, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "查询失败"})
		return
	}

	offset := (page - 1) * pageSize
	query := `
		SELECT id, user_id, portfolio_id, nickname, order_id, symbol, side, position_side,
		       executed_qty, avg_price, total_pnl, status, lead_order_time, copy_time, close_time,
		       IFNULL(close_price,0) as close_price, IFNULL(error_message,'') as error_message
		FROM copy_trade_records WHERE ` + whereClause + ` ORDER BY copy_time DESC LIMIT ? OFFSET ?`
	queryArgs := append(args, pageSize, offset)

	rows, err := s.database.DB().Query(query, queryArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "查询失败"})
		return
	}
	defer rows.Close()

	data := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int
		var recordUserID, portfolioID, nickname, orderID, symbolVal, side, posSide, statusVal, errorMessage string
		var qty, price, pnl, closePrice float64
		var leadTime int64
		var copyTime, closeTime interface{}
		if err := rows.Scan(&id, &recordUserID, &portfolioID, &nickname, &orderID, &symbolVal, &side, &posSide,
			&qty, &price, &pnl, &statusVal, &leadTime, &copyTime, &closeTime, &closePrice, &errorMessage); err != nil {
			continue
		}
		data = append(data, map[string]interface{}{
			"id": id, "user_id": recordUserID, "portfolio_id": portfolioID, "nickname": nickname,
			"order_id": orderID, "symbol": symbolVal, "side": side,
			"position_side": posSide, "executed_qty": qty, "avg_price": price,
			"total_pnl": pnl, "pnl_kind": copyTradePnLKind(statusVal), "status": statusVal, "lead_order_time": leadTime,
			"copy_time": copyTime, "close_time": closeTime,
			"close_price": closePrice, "error_message": errorMessage,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    data,
		"total":   total,
		"success": true,
	})
}

// handleAdminGetSystemConfig 获取运营相关系统配置
func (s *Server) handleAdminGetSystemConfig(c *gin.Context) {
	result := make(map[string]string)
	for _, key := range adminSystemConfigKeys {
		value, err := s.database.GetSystemConfig(key)
		if err != nil {
			result[key] = ""
			continue
		}
		result[key] = value
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// handleAdminPutSystemConfig 更新系统配置（白名单 key）
func (s *Server) handleAdminPutSystemConfig(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	allowed := make(map[string]bool, len(adminSystemConfigKeys))
	for _, key := range adminSystemConfigKeys {
		allowed[key] = true
	}

	for key, value := range req {
		if !allowed[key] {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "不允许修改的配置项: " + key})
			return
		}
		if key == "default_coins" {
			var coins []string
			if err := json.Unmarshal([]byte(value), &coins); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "default_coins 必须是 JSON 数组"})
				return
			}
		}
		if err := s.database.SetSystemConfig(key, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "保存配置失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "配置已更新"})
}

// handleAdminListPlans 返回套餐列表及功能绑定
func (s *Server) handleAdminListPlans(c *gin.Context) {
	plans, err := s.database.ListPlans()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "查询套餐失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": plans})
}

// handleAdminUpdateUser 更新用户套餐等信息
func (s *Server) handleAdminUpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "缺少用户 ID"})
		return
	}

	var req struct {
		Plan string `json:"plan"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if req.Plan == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "plan 不能为空"})
		return
	}

	if _, err := s.database.GetUserByID(userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "用户不存在"})
		return
	}
	if err := s.database.UpdateUserPlan(userID, req.Plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	user, _ := s.database.GetUserByID(userID)
	features, _ := s.database.GetUserFeatures(userID, user.Role)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户已更新",
		"data": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"role":     user.Role,
			"plan":     user.Plan,
			"features": features,
		},
	})
}

// adminMiddleware 校验管理员角色
func (s *Server) adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != config.UserRoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}
