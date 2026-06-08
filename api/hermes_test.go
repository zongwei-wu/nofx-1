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
