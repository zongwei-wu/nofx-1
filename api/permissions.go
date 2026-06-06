package api

import (
	"net/http"

	"nofx/config"

	"github.com/gin-gonic/gin"
)

// handleMe 返回当前用户信息与功能权限
func (s *Server) handleMe(c *gin.Context) {
	userID := c.GetString("user_id")
	role := c.GetString("role")

	user, err := s.database.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	features, err := s.database.GetUserFeatures(userID, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取权限失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":  user.ID,
		"email":    user.Email,
		"role":     user.Role,
		"plan":     user.Plan,
		"features": features,
	})
}

func (s *Server) userAuthPayload(user *config.User, token, message string) gin.H {
	role := user.Role
	if role == "" {
		role = config.UserRoleUser
	}
	plan := user.Plan
	if plan == "" {
		plan = config.PlanStandard
	}
	features, _ := s.database.GetUserFeatures(user.ID, role)
	return gin.H{
		"token":    token,
		"user_id":  user.ID,
		"email":    user.Email,
		"role":     role,
		"plan":     plan,
		"features": features,
		"message":  message,
	}
}

// requireFeature 校验用户是否拥有指定功能（实时查 DB plan）
func (s *Server) requireFeature(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		role := c.GetString("role")
		ok, err := s.database.UserHasFeature(userID, role, feature)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "权限校验失败"})
			c.Abort()
			return
		}
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "无此功能使用权限"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// requireAnyFeature 拥有任一功能即可访问
func (s *Server) requireAnyFeature(features ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		role := c.GetString("role")
		for _, feature := range features {
			ok, err := s.database.UserHasFeature(userID, role, feature)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "权限校验失败"})
				c.Abort()
				return
			}
			if ok {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "无此功能使用权限"})
		c.Abort()
	}
}
