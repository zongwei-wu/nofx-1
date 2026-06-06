package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"nofx/auth"
	"nofx/config"

	"github.com/gin-gonic/gin"
)

func setupAdminTradersTestDB(t *testing.T) (*config.Database, string, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth.SetJWTSecret("test-jwt-admin-traders")

	dir := t.TempDir()
	db, err := config.NewDatabase(filepath.Join(dir, "admin_traders.db"))
	if err != nil {
		t.Fatal(err)
	}

	adminHash, _ := auth.HashPassword("admin123")
	_, _ = db.DB().Exec(`UPDATE users SET password_hash = ? WHERE id = 'admin'`, adminHash)

	userID := "trader-owner"
	user := &config.User{
		ID: userID, Email: "owner@test.com",
		PasswordHash: "hash", OTPVerified: true, Role: config.UserRoleUser, Plan: config.PlanPro,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatal(err)
	}

	if err := db.UpdateAIModel(userID, userID+"_deepseek", true, "key", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateExchange(userID, "binance", true, "ak", "sk", false, "", "", "", ""); err != nil {
		t.Fatal(err)
	}

	trader := &config.TraderRecord{
		ID: "trader-admin-1", UserID: userID, Name: "测试员",
		AIModelID: userID + "_deepseek", ExchangeID: "binance",
		InitialBalance: 1000, ScanIntervalMinutes: 5,
	}
	if err := db.CreateTrader(trader); err != nil {
		t.Fatal(err)
	}

	return db, "admin", trader.ID
}

func newAdminTradersRouter(db *config.Database) *gin.Engine {
	s := &Server{database: db}
	r := gin.New()
	api := r.Group("/api")
	protected := api.Group("/", s.authMiddleware())
	admin := protected.Group("/admin", s.adminMiddleware())
	{
		admin.GET("/traders", s.handleAdminListTraders)
		admin.GET("/traders/:id", s.handleAdminGetTrader)
		admin.POST("/traders/:id/stop", s.handleAdminStopTrader)
	}
	return r
}

func TestAdminTraders_ListAndDetail(t *testing.T) {
	db, adminID, traderID := setupAdminTradersTestDB(t)
	defer db.Close()

	token, _ := auth.GenerateJWT(adminID, "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	r := newAdminTradersRouter(db)

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/traders?email=owner@test.com", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list: %d %s", listW.Code, listW.Body.String())
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/api/admin/traders/"+traderID, nil)
	detailReq.Header.Set("Authorization", "Bearer "+token)
	detailW := httptest.NewRecorder()
	r.ServeHTTP(detailW, detailReq)
	if detailW.Code != http.StatusOK {
		t.Fatalf("detail: %d %s", detailW.Code, detailW.Body.String())
	}
}

func TestAdminTraders_StopNotRunning(t *testing.T) {
	db, adminID, traderID := setupAdminTradersTestDB(t)
	defer db.Close()

	token, _ := auth.GenerateJWT(adminID, "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	r := newAdminTradersRouter(db)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/traders/"+traderID+"/stop", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// 交易员未在内存中加载，预期 404 或 400
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		t.Fatalf("stop without runtime: expected 404/400, got %d %s", w.Code, w.Body.String())
	}
}
