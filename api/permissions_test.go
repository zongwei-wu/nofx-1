package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"nofx/auth"
	"nofx/config"

	"github.com/gin-gonic/gin"
)

func setupPermissionsTestDB(t *testing.T, plan string) (*config.Database, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth.SetJWTSecret("test-jwt-secret-permissions")

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "perm_test.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	userHash, err := auth.HashPassword("pass123")
	if err != nil {
		t.Fatal(err)
	}
	user := &config.User{
		ID:           "user-perm",
		Email:        "perm@test.com",
		PasswordHash: userHash,
		OTPVerified:  true,
		Role:         config.UserRoleUser,
		Plan:         plan,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	return db, user.ID
}

func newPermissionsTestRouter(db *config.Database) *gin.Engine {
	s := &Server{database: db}
	r := gin.New()
	api := r.Group("/api")
	protected := api.Group("/", s.authMiddleware())
	protected.GET("/me", s.handleMe)
	copyTrade := protected.Group("/", s.requireFeature(config.FeatureCopyTrade))
	{
		copyTrade.GET("/copy-trade/configs", s.handleGetCopyTradeConfigs)
	}
	return r
}

func TestPermissions_BasicUserCopyTradeForbidden(t *testing.T) {
	db, userID := setupPermissionsTestDB(t, config.PlanBasic)
	defer db.Close()

	token, err := auth.GenerateJWT(userID, "perm@test.com", config.UserRoleUser, config.PlanBasic)
	if err != nil {
		t.Fatal(err)
	}

	r := newPermissionsTestRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/api/copy-trade/configs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestPermissions_ProUserCopyTradeAllowed(t *testing.T) {
	db, userID := setupPermissionsTestDB(t, config.PlanPro)
	defer db.Close()

	token, err := auth.GenerateJWT(userID, "perm@test.com", config.UserRoleUser, config.PlanPro)
	if err != nil {
		t.Fatal(err)
	}

	r := newPermissionsTestRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/api/copy-trade/configs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestPermissions_GetMeFeatures(t *testing.T) {
	db, userID := setupPermissionsTestDB(t, config.PlanStandard)
	defer db.Close()

	token, err := auth.GenerateJWT(userID, "perm@test.com", config.UserRoleUser, config.PlanStandard)
	if err != nil {
		t.Fatal(err)
	}

	r := newPermissionsTestRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Plan     string   `json:"plan"`
		Features []string `json:"features"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Plan != config.PlanStandard {
		t.Fatalf("expected plan standard, got %s", resp.Plan)
	}
	hasAITrader := false
	for _, f := range resp.Features {
		if f == config.FeatureAITrader {
			hasAITrader = true
		}
		if f == config.FeatureCopyTrade {
			t.Fatal("standard plan should not include copy_trade")
		}
	}
	if !hasAITrader {
		t.Fatal("standard plan should include ai_trader")
	}
}

func TestPermissions_PlanChangeImmediateEffect(t *testing.T) {
	db, userID := setupPermissionsTestDB(t, config.PlanPro)
	defer db.Close()

	token, err := auth.GenerateJWT(userID, "perm@test.com", config.UserRoleUser, config.PlanPro)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.UpdateUserPlan(userID, config.PlanBasic); err != nil {
		t.Fatal(err)
	}

	r := newPermissionsTestRouter(db)
	req := httptest.NewRequest(http.MethodGet, "/api/copy-trade/configs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 after plan downgrade, got %d", w.Code)
	}
}
