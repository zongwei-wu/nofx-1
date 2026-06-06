package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"nofx/auth"
	"nofx/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) adminGuardSelfAndLastAdmin(c *gin.Context, targetUserID string, isDemoteOrDelete bool) bool {
	currentUserID := c.GetString("user_id")
	if targetUserID == currentUserID && isDemoteOrDelete {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "不能对自己执行此操作"})
		return false
	}
	if isDemoteOrDelete {
		target, err := s.database.GetUserByID(targetUserID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "用户不存在"})
			return false
		}
		if target.Role == config.UserRoleAdmin {
			count, err := s.database.CountAdmins()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "校验管理员数量失败"})
				return false
			}
			if count <= 1 {
				c.JSON(http.StatusForbidden, gin.H{"success": false, "error": "不能删除或降权最后一个管理员"})
				return false
			}
		}
	}
	return true
}

// handleAdminListAdmins 分页查询管理员列表
func (s *Server) handleAdminListAdmins(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current", c.DefaultQuery("page", "1")))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	emailFilter := strings.TrimSpace(c.Query("email"))

	users, total, err := s.database.ListUsersByRole(config.UserRoleAdmin, page, pageSize, emailFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "查询管理员失败"})
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
	c.JSON(http.StatusOK, gin.H{"data": data, "total": total, "success": true})
}

// handleAdminCreateAdmin 创建管理员账号
func (s *Server) handleAdminCreateAdmin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if _, err := s.database.GetUserByEmail(req.Email); err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "邮箱已被注册"})
		return
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "查询用户失败"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "密码加密失败"})
		return
	}

	user := &config.User{
		ID:           uuid.New().String(),
		Email:        req.Email,
		PasswordHash: hash,
		OTPSecret:    "",
		OTPVerified:  true,
		Role:         config.UserRoleAdmin,
		Plan:         config.PlanVIP,
	}
	if err := s.database.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "创建管理员失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "管理员已创建",
		"data": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
			"plan":  user.Plan,
		},
	})
}

// handleAdminUpdateUserRole 提权/降权
func (s *Server) handleAdminUpdateUserRole(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	if req.Role != config.UserRoleUser && req.Role != config.UserRoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "无效的角色"})
		return
	}

	target, err := s.database.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "用户不存在"})
		return
	}

	if req.Role == config.UserRoleUser && target.Role == config.UserRoleAdmin {
		if !s.adminGuardSelfAndLastAdmin(c, userID, true) {
			return
		}
	}
	if req.Role == config.UserRoleAdmin && target.Role != config.UserRoleAdmin {
		// 提权为管理员，自动升级套餐
		_ = s.database.UpdateUserPlan(userID, config.PlanVIP)
	}

	if err := s.database.UpdateUserRole(userID, req.Role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	user, _ := s.database.GetUserByID(userID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "角色已更新",
		"data": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role,
			"plan":  user.Plan,
		},
	})
}

// handleAdminResetUserPassword 管理员重置密码
func (s *Server) handleAdminResetUserPassword(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if _, err := s.database.GetUserByID(userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "用户不存在"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "密码加密失败"})
		return
	}
	if err := s.database.UpdateUserPassword(userID, hash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "密码更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "密码已重置"})
}

// handleAdminDeleteUser 删除用户/管理员
func (s *Server) handleAdminDeleteUser(c *gin.Context) {
	userID := c.Param("id")
	if !s.adminGuardSelfAndLastAdmin(c, userID, true) {
		return
	}

	target, err := s.database.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "用户不存在"})
		return
	}

	// 停止该用户所有运行中的交易员
	traders, _ := s.database.GetTraders(userID)
	for _, t := range traders {
		if at, err := s.traderManager.GetTrader(t.ID); err == nil {
			status := at.GetStatus()
			if running, ok := status["is_running"].(bool); ok && running {
				at.Stop()
			}
		}
		_ = s.database.UpdateTraderStatus(userID, t.ID, false)
	}

	if err := s.database.DeleteUser(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "删除用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户已删除",
		"data":    gin.H{"id": target.ID, "email": target.Email},
	})
}
