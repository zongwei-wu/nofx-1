package api

import (
	"strings"
	"testing"

	"nofx/config"
)

func TestResolveExchangeCredentials_MergesSavedSecrets(t *testing.T) {
	saved := &config.ExchangeConfig{
		ID:              "binance",
		APIKey:          "saved-api-key",
		SecretKey:       "saved-secret-key",
		AsterPrivateKey: "saved-aster-private-key",
		Testnet:         true,
	}

	req := TestExchangeConnectionRequest{
		ExchangeID: "binance",
		Testnet:      false,
	}

	creds := resolveExchangeCredentials(req, saved)

	if creds.APIKey != "saved-api-key" {
		t.Errorf("APIKey = %q, want saved-api-key", creds.APIKey)
	}
	if creds.SecretKey != "saved-secret-key" {
		t.Errorf("SecretKey = %q, want saved-secret-key", creds.SecretKey)
	}
	if creds.Testnet {
		t.Error("Testnet should use form value false")
	}
}

func TestResolveExchangeCredentials_FormOverridesSaved(t *testing.T) {
	saved := &config.ExchangeConfig{
		ID:        "binance",
		APIKey:    "saved-api-key",
		SecretKey: "saved-secret-key",
		Testnet:   true,
	}

	req := TestExchangeConnectionRequest{
		ExchangeID: "binance",
		APIKey:     "form-api-key",
		SecretKey:  "form-secret-key",
		Testnet:    false,
	}

	creds := resolveExchangeCredentials(req, saved)

	if creds.APIKey != "form-api-key" {
		t.Errorf("APIKey = %q, want form-api-key", creds.APIKey)
	}
	if creds.SecretKey != "form-secret-key" {
		t.Errorf("SecretKey = %q, want form-secret-key", creds.SecretKey)
	}
}

func TestResolveExchangeCredentials_NoSaved(t *testing.T) {
	req := TestExchangeConnectionRequest{
		ExchangeID: "binance",
		APIKey:     "only-form-key",
		SecretKey:  "only-form-secret",
		Testnet:    true,
	}

	creds := resolveExchangeCredentials(req, nil)

	if creds.APIKey != "only-form-key" {
		t.Errorf("APIKey = %q, want only-form-key", creds.APIKey)
	}
	if !creds.Testnet {
		t.Error("Testnet should be true")
	}
}

func TestExtractBalanceSnapshot_BinanceFields(t *testing.T) {
	snapshot := extractBalanceSnapshot(map[string]interface{}{
		"totalWalletBalance":    1000.0,
		"totalUnrealizedProfit": 50.5,
		"availableBalance":      800.0,
	})

	if snapshot.TotalEquity != 1050.5 {
		t.Errorf("TotalEquity = %v, want 1050.5", snapshot.TotalEquity)
	}
	if snapshot.AvailableBalance != 800.0 {
		t.Errorf("AvailableBalance = %v, want 800.0", snapshot.AvailableBalance)
	}
}

func TestExtractBalanceSnapshot_AltFieldNames(t *testing.T) {
	snapshot := extractBalanceSnapshot(map[string]interface{}{
		"wallet_balance":    500.0,
		"unrealized_profit": 25.0,
		"available_balance": 400.0,
	})

	if snapshot.TotalEquity != 525.0 {
		t.Errorf("TotalEquity = %v, want 525.0", snapshot.TotalEquity)
	}
	if snapshot.AvailableBalance != 400.0 {
		t.Errorf("AvailableBalance = %v, want 400.0", snapshot.AvailableBalance)
	}
}

func TestValidateExchangeCredentials(t *testing.T) {
	tests := []struct {
		name      string
		creds     resolvedExchangeCredentials
		wantError string
	}{
		{
			name: "binance missing secret",
			creds: resolvedExchangeCredentials{
				ExchangeID: "binance",
				APIKey:     "key",
			},
			wantError: "Secret Key 不能为空",
		},
		{
			name: "hyperliquid missing wallet",
			creds: resolvedExchangeCredentials{
				ExchangeID: "hyperliquid",
				APIKey:     "priv",
			},
			wantError: "主钱包地址不能为空",
		},
		{
			name: "okx unsupported",
			creds: resolvedExchangeCredentials{
				ExchangeID: "okx",
				APIKey:     "key",
				SecretKey:  "secret",
			},
			wantError: "暂不支持",
		},
		{
			name: "binance valid",
			creds: resolvedExchangeCredentials{
				ExchangeID: "binance",
				APIKey:     "key",
				SecretKey:  "secret",
			},
			wantError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExchangeCredentials(tt.creds)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantError)
			}
		})
	}
}

func TestTestExchangeConnection_UnsupportedExchange(t *testing.T) {
	result := testExchangeConnection("user-1", resolvedExchangeCredentials{
		ExchangeID: "unknown-exchange",
		APIKey:     "key",
		SecretKey:  "secret",
	})

	if result.Success {
		t.Fatal("expected failure for unsupported exchange")
	}
	if result.Error == "" {
		t.Fatal("expected error message")
	}
}
