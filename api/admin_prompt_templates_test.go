package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"nofx/auth"
	"nofx/config"
	"nofx/decision"

	"github.com/gin-gonic/gin"
)

func setupAdminPromptTemplatesTestDB(t *testing.T) (*config.Database, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth.SetJWTSecret("test-jwt-admin-prompts")

	dir := t.TempDir()
	db, err := config.NewDatabase(filepath.Join(dir, "admin_prompts.db"))
	if err != nil {
		t.Fatal(err)
	}

	adminHash, _ := auth.HashPassword("admin123")
	_, _ = db.DB().Exec(`UPDATE users SET password_hash = ? WHERE id = 'admin'`, adminHash)

	if err := db.CreatePromptTemplate("default", "base content", "default template"); err != nil {
		t.Fatal(err)
	}
	return db, "admin"
}

func newAdminPromptTemplatesRouter(db *config.Database) *gin.Engine {
	s := &Server{database: db, traderManager: nil}
	_ = s.reloadPromptTemplatesFromDB()
	r := gin.New()
	api := r.Group("/api")
	protected := api.Group("/", s.authMiddleware())
	admin := protected.Group("/admin", s.adminMiddleware())
	{
		admin.GET("/prompt-templates", s.handleAdminListPromptTemplates)
		admin.POST("/prompt-templates", s.handleAdminCreatePromptTemplate)
		admin.GET("/prompt-templates/:name", s.handleAdminGetPromptTemplate)
		admin.PUT("/prompt-templates/:name", s.handleAdminUpdatePromptTemplate)
		admin.DELETE("/prompt-templates/:name", s.handleAdminDeletePromptTemplate)
	}
	return r
}

func TestAdminPromptTemplates_CRUD(t *testing.T) {
	db, adminID := setupAdminPromptTemplatesTestDB(t)
	defer db.Close()

	token, _ := auth.GenerateJWT(adminID, "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	r := newAdminPromptTemplatesRouter(db)

	createBody, _ := json.Marshal(map[string]string{
		"name": "nof1", "content": "nof1 prompt body", "description": "nof1 desc",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/prompt-templates", bytes.NewReader(createBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}

	tmpl, err := decision.GetPromptTemplate("nof1")
	if err != nil || tmpl.Content != "nof1 prompt body" {
		t.Fatalf("memory cache not updated after create: %v", err)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/prompt-templates/nof1", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("get: %d %s", getW.Code, getW.Body.String())
	}

	updateBody, _ := json.Marshal(map[string]string{
		"content": "updated body", "description": "updated desc",
	})
	putReq := httptest.NewRequest(http.MethodPut, "/api/admin/prompt-templates/nof1", bytes.NewReader(updateBody))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("update: %d %s", putW.Code, putW.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/prompt-templates", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listW.Code, listW.Body.String())
	}

	delReq := httptest.NewRequest(http.MethodDelete, "/api/admin/prompt-templates/nof1", nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", delW.Code, delW.Body.String())
	}
}

func TestAdminPromptTemplates_DeleteInUse(t *testing.T) {
	db, adminID := setupAdminPromptTemplatesTestDB(t)
	defer db.Close()

	_, _ = db.DB().Exec(`INSERT INTO traders (id, user_id, name, ai_model_id, exchange_id, initial_balance, system_prompt_template)
		VALUES ('t1', 'admin', 'test', 'm1', 'e1', 1000, 'default')`)

	token, _ := auth.GenerateJWT(adminID, "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	r := newAdminPromptTemplatesRouter(db)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/admin/prompt-templates/default", nil)
	delReq.Header.Set("Authorization", "Bearer "+token)
	delW := httptest.NewRecorder()
	r.ServeHTTP(delW, delReq)
	if delW.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when template in use, got %d %s", delW.Code, delW.Body.String())
	}
}
