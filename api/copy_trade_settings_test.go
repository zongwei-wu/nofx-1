package api

import (
	"os"
	"path/filepath"
	"testing"

	"nofx/config"
)

func setupCopyTradeSettingsDB(t *testing.T) (*config.Database, string, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	userID := "user-copy-ai"
	user := &config.User{
		ID:           userID,
		Email:        "copy@test.com",
		PasswordHash: "hash",
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	deepseekID := userID + "_deepseek"
	qwenID := userID + "_qwen"
	if err := db.UpdateAIModel(userID, deepseekID, true, "key1", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateAIModel(userID, qwenID, true, "key2", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateExchange(userID, "binance", true, "ak", "sk", false, "", "", "", ""); err != nil {
		t.Fatal(err)
	}
	trader := &config.TraderRecord{
		ID:                  "trader-copy-1",
		UserID:              userID,
		Name:                "跟单分析员",
		AIModelID:           qwenID,
		ExchangeID:          "binance",
		InitialBalance:      1000,
		ScanIntervalMinutes: 5,
		SystemPromptTemplate: "default",
	}
	if err := db.CreateTrader(trader); err != nil {
		t.Fatal(err)
	}

	return db, userID, qwenID
}

func TestResolveCopyTradeAI_SelectedTrader(t *testing.T) {
	db, userID, qwenID := setupCopyTradeSettingsDB(t)
	defer db.Close()

	if err := db.UpsertCopyTradeSettings(userID, "trader-copy-1"); err != nil {
		t.Fatal(err)
	}

	s := &Server{database: db}
	result, err := s.resolveCopyTradeAI(userID)
	if err != nil {
		t.Fatal(err)
	}
	if result.FallbackUsed {
		t.Fatal("expected selected trader, got fallback")
	}
	if result.AITraderID != "trader-copy-1" {
		t.Fatalf("unexpected trader id: %s", result.AITraderID)
	}
	if result.AITraderName != "跟单分析员" {
		t.Fatalf("unexpected trader name: %s", result.AITraderName)
	}
	if result.AICfg == nil || result.AICfg.ID != qwenID {
		t.Fatalf("expected %s model, got %+v", qwenID, result.AICfg)
	}
}

func TestResolveCopyTradeAI_Fallback(t *testing.T) {
	db, userID, _ := setupCopyTradeSettingsDB(t)
	defer db.Close()

	s := &Server{database: db}
	result, err := s.resolveCopyTradeAI(userID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.FallbackUsed {
		t.Fatal("expected fallback")
	}
	if result.AICfg == nil {
		t.Fatal("expected ai cfg")
	}
}

func TestResolveCopyTradeAI_NoModel(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dbPath)
	defer db.Close()

	userID := "user-no-ai"
	user := &config.User{
		ID:           userID,
		Email:        "noai@test.com",
		PasswordHash: "hash",
	}
	if err := db.CreateUser(user); err != nil {
		t.Fatal(err)
	}

	s := &Server{database: db}
	_, err = s.resolveCopyTradeAI(userID)
	if err == nil {
		t.Fatal("expected error when no ai model")
	}
}
