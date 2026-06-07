package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUserSymbolPreferencesCRUD(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(dbPath)
	defer db.Close()

	userID := "user-test-1"
	prefs, err := db.GetUserSymbolPreferences(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(prefs.Symbols) != 0 || prefs.UseCustomOrder {
		t.Fatalf("expected empty prefs: %+v", prefs)
	}

	symbols := []string{"ETHUSDT", "BTCUSDT"}
	starred := []string{"BTCUSDT"}
	if err := db.UpsertUserSymbolPreferences(userID, symbols, true, starred); err != nil {
		t.Fatal(err)
	}
	prefs, err = db.GetUserSymbolPreferences(userID)
	if err != nil {
		t.Fatal(err)
	}
	if !prefs.UseCustomOrder || len(prefs.Symbols) != 2 || prefs.Symbols[0] != "ETHUSDT" {
		t.Fatalf("unexpected prefs: %+v", prefs)
	}
	if len(prefs.StarredSymbols) != 1 || prefs.StarredSymbols[0] != "BTCUSDT" {
		t.Fatalf("unexpected starred: %+v", prefs.StarredSymbols)
	}

	if err := db.UpsertUserSymbolPreferences(userID, symbols, false, starred); err != nil {
		t.Fatal(err)
	}
	prefs, err = db.GetUserSymbolPreferences(userID)
	if err != nil {
		t.Fatal(err)
	}
	if prefs.UseCustomOrder {
		t.Fatal("expected use_custom_order false")
	}
}
