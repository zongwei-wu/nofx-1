package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"nofx/auth"
	"nofx/config"

	"github.com/gin-gonic/gin"
)

func setupAdminTestDB(t *testing.T) (*config.Database, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth.SetJWTSecret("test-jwt-secret-for-admin")

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "admin_test.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	adminHash, err := auth.HashPassword("admin123")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.DB().Exec(`UPDATE users SET password_hash = ?, role = ? WHERE id = 'admin'`,
		adminHash, config.UserRoleAdmin)
	if err != nil {
		t.Fatal(err)
	}

	userHash, err := auth.HashPassword("user123")
	if err != nil {
		t.Fatal(err)
	}
	user := &config.User{
		ID:           "user-1",
		Email:        "user@test.com",
		PasswordHash: userHash,
		OTPVerified:  true,
		Role:         config.UserRoleUser,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatal(err)
	}

	_, err = db.DB().Exec(`
		INSERT INTO copy_trade_records
		(user_id, portfolio_id, nickname, order_id, symbol, side, position_side, executed_qty, avg_price, total_pnl, status)
		VALUES ('user-1', 'p1', 'leader1', 'o1', 'BTCUSDT', 'BUY', 'LONG', 1, 50000, 10, 'OPEN')
	`)
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}
	return db, cleanup
}

func newAdminTestRouter(db *config.Database) *gin.Engine {
	s := &Server{database: db}
	r := gin.New()
	api := r.Group("/api")
	protected := api.Group("/", s.authMiddleware())
	admin := protected.Group("/admin", s.adminMiddleware())
	{
		admin.GET("/me", s.handleAdminMe)
		admin.GET("/users", s.handleAdminListUsers)
		admin.GET("/copy-trade/records", s.handleAdminCopyTradeRecords)
		admin.GET("/system-config", s.handleAdminGetSystemConfig)
		admin.PUT("/system-config", s.handleAdminPutSystemConfig)
	}
	return r
}

func TestAdminAPI_NonAdminForbidden(t *testing.T) {
	db, cleanup := setupAdminTestDB(t)
	defer cleanup()

	token, err := auth.GenerateJWT("user-1", "user@test.com", config.UserRoleUser, config.PlanStandard)
	if err != nil {
		t.Fatal(err)
	}

	r := newAdminTestRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAdminAPI_ListUsers(t *testing.T) {
	db, cleanup := setupAdminTestDB(t)
	defer cleanup()

	token, err := auth.GenerateJWT("admin", "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	if err != nil {
		t.Fatal(err)
	}

	r := newAdminTestRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/users?current=1&pageSize=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data    []map[string]interface{} `json:"data"`
		Total   int                      `json:"total"`
		Success bool                     `json:"success"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.Total < 2 {
		t.Fatalf("expected at least 2 users, got total=%d", resp.Total)
	}
}

func TestAdminAPI_CopyTradeRecords(t *testing.T) {
	db, cleanup := setupAdminTestDB(t)
	defer cleanup()

	token, err := auth.GenerateJWT("admin", "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	if err != nil {
		t.Fatal(err)
	}

	r := newAdminTestRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/copy-trade/records?user_id=user-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Data    []map[string]interface{} `json:"data"`
		Total   int                      `json:"total"`
		Success bool                     `json:"success"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total < 1 || len(resp.Data) < 1 {
		t.Fatalf("expected copy trade records, got %+v", resp)
	}
}

func TestAdminAPI_SystemConfig(t *testing.T) {
	db, cleanup := setupAdminTestDB(t)
	defer cleanup()

	token, err := auth.GenerateJWT("admin", "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	if err != nil {
		t.Fatal(err)
	}
	r := newAdminTestRouter(db)

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/system-config", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET expected 200, got %d", getW.Code)
	}

	body, _ := json.Marshal(map[string]string{"registration_enabled": "false"})
	putReq := httptest.NewRequest(http.MethodPut, "/api/admin/system-config", bytes.NewReader(body))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	r.ServeHTTP(putW, putReq)
	if putW.Code != http.StatusOK {
		t.Fatalf("PUT expected 200, got %d body=%s", putW.Code, putW.Body.String())
	}

	val, err := db.GetSystemConfig("registration_enabled")
	if err != nil {
		t.Fatal(err)
	}
	if val != "false" {
		t.Fatalf("expected registration_enabled=false, got %s", val)
	}
}
