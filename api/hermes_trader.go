package api

import (
	"fmt"
	"strings"

	"nofx/config"
	"nofx/trader"
)

func exchangeConfigToCreds(ex *config.ExchangeConfig) resolvedExchangeCredentials {
	return resolvedExchangeCredentials{
		ExchangeID:            ex.ID,
		APIKey:                ex.APIKey,
		SecretKey:             ex.SecretKey,
		Testnet:               ex.Testnet,
		HyperliquidWalletAddr: ex.HyperliquidWalletAddr,
		AsterUser:             ex.AsterUser,
		AsterSigner:           ex.AsterSigner,
		AsterPrivateKey:       ex.AsterPrivateKey,
		Passphrase:            ex.Passphrase,
	}
}

// createTraderFromExchangeConfig 根据用户交易所配置创建 Trader（支持 4 家）
func createTraderFromExchangeConfig(userID string, ex *config.ExchangeConfig) (trader.Trader, error) {
	if ex == nil {
		return nil, fmt.Errorf("交易所配置为空")
	}
	creds := exchangeConfigToCreds(ex)
	if err := validateExchangeCredentials(creds); err != nil {
		return nil, err
	}
	return createTraderFromCredentials(userID, creds)
}

func createTraderFromCredentials(userID string, creds resolvedExchangeCredentials) (trader.Trader, error) {
	switch creds.ExchangeID {
	case "binance":
		return trader.NewFuturesTrader(creds.APIKey, creds.SecretKey, userID, creds.Testnet), nil
	case "hyperliquid":
		return trader.NewHyperliquidTrader(creds.APIKey, creds.HyperliquidWalletAddr, creds.Testnet)
	case "aster":
		return trader.NewAsterTrader(creds.AsterUser, creds.AsterSigner, creds.AsterPrivateKey)
	case "okx":
		return trader.NewOKXTrader(creds.APIKey, creds.SecretKey, creds.Passphrase, creds.Testnet)
	default:
		return nil, fmt.Errorf("不支持的交易所: %s", creds.ExchangeID)
	}
}

func createTempTraderForTest(userID string, creds resolvedExchangeCredentials) (trader.Trader, error) {
	return createTraderFromCredentials(userID, creds)
}

func (s *Server) getHermesExchangeConfig(userID, exchangeID string) (*config.ExchangeConfig, error) {
	exchangeID = strings.TrimSpace(exchangeID)
	if exchangeID == "" {
		return nil, fmt.Errorf("exchange_id 不能为空")
	}

	exchanges, err := s.database.GetExchanges(userID)
	if err != nil {
		return nil, fmt.Errorf("获取交易所配置失败: %w", err)
	}

	for _, ex := range exchanges {
		if ex.ID == exchangeID {
			if !ex.Enabled {
				return nil, fmt.Errorf("交易所 %s 未启用", exchangeID)
			}
			creds := exchangeConfigToCreds(ex)
			if err := validateExchangeCredentials(creds); err != nil {
				return nil, err
			}
			return ex, nil
		}
	}
	return nil, fmt.Errorf("未找到交易所配置: %s", exchangeID)
}

func (s *Server) getHermesTrader(userID, exchangeID string) (trader.Trader, *config.ExchangeConfig, error) {
	ex, err := s.getHermesExchangeConfig(userID, exchangeID)
	if err != nil {
		return nil, nil, err
	}
	t, err := createTraderFromExchangeConfig(userID, ex)
	if err != nil {
		return nil, nil, err
	}
	return t, ex, nil
}
