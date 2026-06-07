package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// handleAdminListPromptTemplates 分页查询提示词模板
func (s *Server) handleAdminListPromptTemplates(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("current", c.DefaultQuery("page", "1")))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	nameFilter := strings.TrimSpace(c.Query("name"))

	list, total, err := s.database.ListPromptTemplates(page, pageSize, nameFilter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "查询提示词模板失败"})
		return
	}

	data := make([]gin.H, 0, len(list))
	for _, item := range list {
		preview := item.Content
		if len(preview) > 120 {
			preview = preview[:120] + "..."
		}
		data = append(data, gin.H{
			"name":            item.Name,
			"description":     item.Description,
			"content_preview": preview,
			"content_length":  len(item.Content),
			"created_at":      item.CreatedAt,
			"updated_at":      item.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    data,
		"total":   total,
		"success": true,
	})
}

// handleAdminGetPromptTemplate 获取单个提示词模板
func (s *Server) handleAdminGetPromptTemplate(c *gin.Context) {
	name := c.Param("name")
	item, err := s.database.GetPromptTemplate(name)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "模板不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "查询模板失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":        item.Name,
			"content":     item.Content,
			"description": item.Description,
			"created_at":  item.CreatedAt,
			"updated_at":  item.UpdatedAt,
		},
	})
}

// handleAdminCreatePromptTemplate 创建提示词模板
func (s *Server) handleAdminCreatePromptTemplate(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Content     string `json:"content"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "请求参数无效"})
		return
	}

	if err := s.database.CreatePromptTemplate(req.Name, req.Content, req.Description); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "模板名称已存在"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := s.reloadPromptTemplatesFromDB(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "模板已创建但重新加载失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "创建成功"})
}

// handleAdminUpdatePromptTemplate 更新提示词模板
func (s *Server) handleAdminUpdatePromptTemplate(c *gin.Context) {
	name := c.Param("name")
	var req struct {
		Content     string `json:"content"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "请求参数无效"})
		return
	}

	if err := s.database.UpdatePromptTemplate(name, req.Content, req.Description); err != nil {
		if strings.Contains(err.Error(), "模板不存在") {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := s.reloadPromptTemplatesFromDB(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "模板已更新但重新加载失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "更新成功"})
}

// handleAdminDeletePromptTemplate 删除提示词模板
func (s *Server) handleAdminDeletePromptTemplate(c *gin.Context) {
	name := c.Param("name")

	if err := s.database.DeletePromptTemplate(name); err != nil {
		if strings.Contains(err.Error(), "模板不存在") {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "正在使用该模板") {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := s.reloadPromptTemplatesFromDB(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "模板已删除但重新加载失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}
