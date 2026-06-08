package api

import (
	"testing"

	"nofx/config"
)

func TestCreateTraderFromExchangeConfig_Unsupported(t *testing.T) {
	_, err := createTraderFromExchangeConfig("user-1", &config.ExchangeConfig{ID: "unknown"})
	if err == nil {
		t.Fatal("expected error for unknown exchange")
	}
}

func TestCreateTraderFromExchangeConfig_BinanceMissingCreds(t *testing.T) {
	_, err := createTraderFromExchangeConfig("user-1", &config.ExchangeConfig{
		ID:      "binance",
		Enabled: true,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestExecuteHermesTrade_RequiresConfirmed(t *testing.T) {
	s := &Server{}
	// executeHermesTrade is tested via handler binding; validate action routing on mock-free level
	req := hermesTradeRequest{
		ExchangeID: "binance",
		Action:     "unknown_action",
		Symbol:     "BTCUSDT",
		Quantity:   1,
	}
	_, err := s.executeHermesTrade(nil, req)
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
}

func TestNormalizeHermesSymbol(t *testing.T) {
	if got := normalizeHermesSymbol("btc"); got != "BTCUSDT" {
		t.Fatalf("got %s", got)
	}
	if got := normalizeHermesSymbol("ETHUSDT"); got != "ETHUSDT" {
		t.Fatalf("got %s", got)
	}
}

func TestExchangeConfigToCreds(t *testing.T) {
	ex := &config.ExchangeConfig{
		ID:         "okx",
		APIKey:     "k",
		SecretKey:  "s",
		Passphrase: "p",
		Testnet:    true,
	}
	creds := exchangeConfigToCreds(ex)
	if creds.ExchangeID != "okx" || creds.Passphrase != "p" || !creds.Testnet {
		t.Fatalf("unexpected creds: %+v", creds)
	}
}
