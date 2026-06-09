package api

import (
	"log"
	"net/http"

	"nofx/auth"
	"nofx/config"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// handleCreateAPIKey creates a new API key for the authenticated user.
// The plaintext key is returned ONCE and cannot be retrieved later.
func (s *Server) handleCreateAPIKey(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// name is optional, allow empty body
		req.Name = ""
	}

	plaintext, hashed, err := auth.GenerateAPIKey()
	if err != nil {
		log.Printf("❌ 生成 API Key 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 API Key 失败"})
		return
	}

	id := uuid.New().String()
	prefix := auth.MaskAPIKey(plaintext)

	if err := s.database.CreateAPIKey(id, userID, hashed, prefix, req.Name); err != nil {
		log.Printf("❌ 存储 API Key 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "存储 API Key 失败"})
		return
	}

	log.Printf("✅ 用户 %s 创建了 API Key: %s (%s)", userID, prefix, req.Name)
	c.JSON(http.StatusCreated, gin.H{
		"id":         id,
		"api_key":    plaintext, // Only time the plaintext is returned
		"key_prefix": prefix,
		"name":       req.Name,
		"message":    "API Key 创建成功，请立即复制保存，关闭后无法再次查看",
	})
}

// handleListAPIKeys returns all API keys for the authenticated user.
// Plaintext keys are never returned.
func (s *Server) handleListAPIKeys(c *gin.Context) {
	userID := c.GetString("user_id")

	keys, err := s.database.GetAPIKeysByUser(userID)
	if err != nil {
		log.Printf("❌ 获取 API Keys 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 API Keys 失败"})
		return
	}

	if keys == nil {
		keys = []*config.APIKeyRecord{}
	}

	c.JSON(http.StatusOK, keys)
}

// handleRevokeAPIKey deletes an API key. Users can only delete their own keys.
func (s *Server) handleRevokeAPIKey(c *gin.Context) {
	userID := c.GetString("user_id")
	keyID := c.Param("id")

	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 API Key ID"})
		return
	}

	if err := s.database.DeleteAPIKey(userID, keyID); err != nil {
		log.Printf("❌ 撤销 API Key 失败: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	log.Printf("✅ 用户 %s 撤销了 API Key: %s", userID, keyID)
	c.JSON(http.StatusOK, gin.H{"message": "API Key 已撤销"})
}
