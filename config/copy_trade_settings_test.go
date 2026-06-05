package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTradeSettingsCRUD(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dbPath)
	defer db.Close()

	userID := "user-copy-trade-1"
	settings, err := db.GetCopyTradeSettings(userID)
	if err != nil {
		t.Fatal(err)
	}
	if settings.AITraderID != "" {
		t.Fatalf("expected empty ai_trader_id, got %q", settings.AITraderID)
	}

	if err := db.UpsertCopyTradeSettings(userID, "trader-abc"); err != nil {
		t.Fatal(err)
	}
	settings, err = db.GetCopyTradeSettings(userID)
	if err != nil {
		t.Fatal(err)
	}
	if settings.AITraderID != "trader-abc" {
		t.Fatalf("expected trader-abc, got %q", settings.AITraderID)
	}

	if err := db.UpsertCopyTradeSettings(userID, ""); err != nil {
		t.Fatal(err)
	}
	settings, err = db.GetCopyTradeSettings(userID)
	if err != nil {
		t.Fatal(err)
	}
	if settings.AITraderID != "" {
		t.Fatalf("expected cleared ai_trader_id, got %q", settings.AITraderID)
	}
}
