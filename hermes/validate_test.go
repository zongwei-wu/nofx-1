package hermes

import (
	"testing"

	"nofx/config"
)

func TestValidateReadyToStart_MissingExchange(t *testing.T) {
	err := ValidateReadyToStart(&config.HermesSettings{
		AutonomousEnabled: true,
		AIModelID:         "deepseek",
	}, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateReadyToStart_AutonomousDisabled(t *testing.T) {
	err := ValidateReadyToStart(&config.HermesSettings{
		ExchangeID: "binance",
		AIModelID:  "deepseek",
	}, []*config.ExchangeConfig{
		{ID: "binance", Enabled: true, APIKey: "k", SecretKey: "s"},
	}, []*config.AIModelConfig{
		{ID: "deepseek", Enabled: true, APIKey: "key"},
	})
	if err == nil {
		t.Fatal("expected autonomous error")
	}
}
