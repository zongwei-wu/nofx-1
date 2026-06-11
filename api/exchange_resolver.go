package api

import (
	"fmt"
	"strings"

	"nofx/config"
	"nofx/trader"
)

// getExecutionExchangeForUser 获取跟单执行所用的交易所配置
func (s *Server) getExecutionExchangeForUser(userID string) (*config.ExchangeConfig, error) {
	settings, err := s.database.GetCopyTradeSettings(userID)
	if err != nil {
		return nil, err
	}

	exchanges, err := s.database.GetExchanges(userID)
	if err != nil {
		return nil, err
	}

	preferred := strings.TrimSpace(settings.ExecutionExchangeID)
	if preferred != "" {
		for _, ex := range exchanges {
			if ex.ID == preferred && ex.Enabled {
				if err := validateExchangeReady(ex); err != nil {
					return nil, err
				}
				return ex, nil
			}
		}
		return nil, fmt.Errorf("跟单执行交易所 %s 未启用或凭证不完整", preferred)
	}

	// 向后兼容：优先 binance，其次 okx
	for _, id := range []string{"binance", "okx"} {
		for _, ex := range exchanges {
			if ex.ID == id && ex.Enabled {
				if err := validateExchangeReady(ex); err != nil {
					continue
				}
				return ex, nil
			}
		}
	}

	return nil, fmt.Errorf("请先启用币安或 OKX 交易所并完成 API 配置")
}

func validateExchangeReady(ex *config.ExchangeConfig) error {
	switch ex.ID {
	case "binance":
		if ex.APIKey == "" || ex.SecretKey == "" {
			return fmt.Errorf("币安 API 凭证不完整")
		}
	case "okx":
		if ex.APIKey == "" || ex.SecretKey == "" || ex.Passphrase == "" {
			return fmt.Errorf("OKX API 凭证不完整")
		}
	default:
		return fmt.Errorf("不支持的跟单执行交易所: %s", ex.ID)
	}
	return nil
}

// createTraderForExchange 按交易所配置创建 Trader
func createTraderForExchange(userID string, ex *config.ExchangeConfig) (trader.Trader, error) {
	if ex == nil {
		return nil, fmt.Errorf("交易所配置为空")
	}
	switch ex.ID {
	case "binance":
		return trader.NewFuturesTrader(ex.APIKey, ex.SecretKey, userID, ex.Testnet), nil
	case "okx":
		return trader.NewOKXTrader(ex.APIKey, ex.SecretKey, ex.Passphrase, ex.Testnet)
	case "gate":
		return trader.NewGateTrader(ex.APIKey, ex.SecretKey, ex.Testnet)
	default:
		return nil, fmt.Errorf("不支持的交易所: %s", ex.ID)
	}
}

// contractQtyConverter 张数→标的币换算接口
type contractQtyConverter interface {
	GetLotStepSize(symbol string) (float64, error)
}

func leadContractsToBaseQtyGeneric(conv contractQtyConverter, symbol string, contracts float64) (float64, error) {
	if conv == nil || contracts <= 0 {
		return contracts, nil
	}
	step, err := conv.GetLotStepSize(symbol)
	if err != nil || step <= 0 {
		return contracts, err
	}
	return contracts * step, nil
}

func invalidateTraderAccountCache(t trader.Trader) {
	switch v := t.(type) {
	case *trader.FuturesTrader:
		v.InvalidateAccountCache()
	case *trader.OKXTrader:
		v.InvalidateAccountCache()
	case *trader.GateTrader:
		v.InvalidateAccountCache()
	}
}

func asContractQtyConverter(t trader.Trader) contractQtyConverter {
	switch v := t.(type) {
	case *trader.FuturesTrader:
		return v
	case *trader.OKXTrader:
		return v
	case *trader.GateTrader:
		return v
	default:
		return nil
	}
}

func exchangeOrderQtyLabel(exchangeID string) string {
	switch exchangeID {
	case "okx":
		return "OKX 合约"
	case "binance":
		return "币安合约"
	case "gate":
		return "Gate 合约"
	default:
		return "合约"
	}
}
