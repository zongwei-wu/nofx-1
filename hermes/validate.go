package hermes

import (
	"fmt"
	"strings"

	"nofx/config"
)

// ValidateReadyToStart 校验 Hermes Runner 启动前置条件
func ValidateReadyToStart(settings *config.HermesSettings, exchanges []*config.ExchangeConfig, models []*config.AIModelConfig) error {
	if settings == nil {
		return fmt.Errorf("Hermes 设置为空")
	}
	if strings.TrimSpace(settings.ExchangeID) == "" {
		return fmt.Errorf("请先配置 exchange_id")
	}
	if !settings.AutonomousEnabled {
		return fmt.Errorf("请先开启 autonomous_enabled（用户授权半自动交易）")
	}

	var ex *config.ExchangeConfig
	for _, e := range exchanges {
		if e.ID == settings.ExchangeID {
			ex = e
			break
		}
	}
	if ex == nil || !ex.Enabled {
		return fmt.Errorf("交易所 %s 未启用", settings.ExchangeID)
	}
	if err := validateExchangeCreds(ex); err != nil {
		return err
	}

	if strings.TrimSpace(settings.AIModelID) == "" {
		return fmt.Errorf("请先配置 ai_model_id")
	}
	var model *config.AIModelConfig
	for _, m := range models {
		if m.ID == settings.AIModelID {
			model = m
			break
		}
	}
	if model == nil || !model.Enabled || model.APIKey == "" {
		return fmt.Errorf("AI 模型 %s 未配置或未启用", settings.AIModelID)
	}
	return nil
}

func validateExchangeCreds(ex *config.ExchangeConfig) error {
	switch ex.ID {
	case "binance":
		if ex.APIKey == "" || ex.SecretKey == "" {
			return fmt.Errorf("币安 API 凭证不完整")
		}
	case "okx":
		if ex.APIKey == "" || ex.SecretKey == "" || ex.Passphrase == "" {
			return fmt.Errorf("OKX API 凭证不完整")
		}
	case "hyperliquid":
		if ex.APIKey == "" || ex.HyperliquidWalletAddr == "" {
			return fmt.Errorf("Hyperliquid 凭证不完整")
		}
	case "aster":
		if ex.AsterUser == "" || ex.AsterSigner == "" || ex.AsterPrivateKey == "" {
			return fmt.Errorf("Aster 凭证不完整")
		}
	default:
		return fmt.Errorf("不支持的交易所: %s", ex.ID)
	}
	return nil
}
