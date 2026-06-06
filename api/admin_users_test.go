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

	"github.com/gin-gonic/gin"
)

func setupAdminUsersTestDB(t *testing.T) (*config.Database, string, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth.SetJWTSecret("test-jwt-admin-users")

	dir := t.TempDir()
	db, err := config.NewDatabase(filepath.Join(dir, "admin_users.db"))
	if err != nil {
		t.Fatal(err)
	}

	adminHash, _ := auth.HashPassword("admin123")
	_, _ = db.DB().Exec(`UPDATE users SET password_hash = ? WHERE id = 'admin'`, adminHash)

	userHash, _ := auth.HashPassword("user123")
	user := &config.User{
		ID: "user-normal", Email: "normal@test.com",
		PasswordHash: userHash, OTPVerified: true, Role: config.UserRoleUser, Plan: config.PlanStandard,
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	return db, "admin", user.ID
}

func newAdminUsersRouter(db *config.Database) *gin.Engine {
	s := &Server{database: db, traderManager: nil}
	r := gin.New()
	api := r.Group("/api")
	protected := api.Group("/", s.authMiddleware())
	admin := protected.Group("/admin", s.adminMiddleware())
	{
		admin.GET("/admins", s.handleAdminListAdmins)
		admin.POST("/admins", s.handleAdminCreateAdmin)
		admin.PUT("/users/:id/role", s.handleAdminUpdateUserRole)
		admin.PUT("/users/:id/password", s.handleAdminResetUserPassword)
		admin.DELETE("/users/:id", s.handleAdminDeleteUser)
	}
	return r
}

func TestAdminUsers_CreateAndList(t *testing.T) {
	db, adminID, _ := setupAdminUsersTestDB(t)
	defer db.Close()

	token, _ := auth.GenerateJWT(adminID, "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	r := newAdminUsersRouter(db)

	body, _ := json.Marshal(map[string]string{
		"email": "newadmin@test.com", "password": "pass1234",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/admin/admins", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("create admin: %d %s", w.Code, w.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/admins", nil)
	listReq.Header.Set("Authorization", "Bearer "+token)
	listW := httptest.NewRecorder()
	r.ServeHTTP(listW, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("list admins: %d", listW.Code)
	}
	var resp struct {
		Total int `json:"total"`
	}
	json.Unmarshal(listW.Body.Bytes(), &resp)
	if resp.Total < 2 {
		t.Fatalf("expected >=2 admins, got %d", resp.Total)
	}
}

func TestAdminUsers_PromoteUser(t *testing.T) {
	db, adminID, userID := setupAdminUsersTestDB(t)
	defer db.Close()

	token, _ := auth.GenerateJWT(adminID, "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	r := newAdminUsersRouter(db)

	body, _ := json.Marshal(map[string]string{"role": "admin"})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+userID+"/role", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("promote: %d %s", w.Code, w.Body.String())
	}

	user, _ := db.GetUserByID(userID)
	if user.Role != config.UserRoleAdmin {
		t.Fatalf("expected admin role, got %s", user.Role)
	}
}

func TestAdminUsers_CannotDeleteSelf(t *testing.T) {
	db, adminID, _ := setupAdminUsersTestDB(t)
	defer db.Close()

	token, _ := auth.GenerateJWT(adminID, "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	r := newAdminUsersRouter(db)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/"+adminID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestAdminUsers_CannotDemoteLastAdmin(t *testing.T) {
	db, adminID, _ := setupAdminUsersTestDB(t)
	defer db.Close()

	token, _ := auth.GenerateJWT(adminID, "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	r := newAdminUsersRouter(db)

	body, _ := json.Marshal(map[string]string{"role": "user"})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+adminID+"/role", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for self demote, got %d", w.Code)
	}
}

func TestAdminUsers_ResetPassword(t *testing.T) {
	db, adminID, userID := setupAdminUsersTestDB(t)
	defer db.Close()

	token, _ := auth.GenerateJWT(adminID, "admin@localhost", config.UserRoleAdmin, config.PlanVIP)
	r := newAdminUsersRouter(db)

	body, _ := json.Marshal(map[string]string{"password": "newpass99"})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/users/"+userID+"/password", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("reset password: %d %s", w.Code, w.Body.String())
	}

	user, _ := db.GetUserByID(userID)
	if !auth.CheckPassword("newpass99", user.PasswordHash) {
		t.Fatal("password not updated")
	}
}
