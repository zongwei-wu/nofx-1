package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"nofx/config"
	"nofx/crypto"
	"nofx/trader"

	"github.com/gin-gonic/gin"
)

// TestExchangeConnectionRequest 测试交易所连接请求（解密后）
type TestExchangeConnectionRequest struct {
	ExchangeID            string `json:"exchange_id"`
	APIKey                string `json:"api_key"`
	SecretKey             string `json:"secret_key"`
	Testnet               bool   `json:"testnet"`
	HyperliquidWalletAddr string `json:"hyperliquid_wallet_addr"`
	AsterUser             string `json:"aster_user"`
	AsterSigner           string `json:"aster_signer"`
	AsterPrivateKey       string `json:"aster_private_key"`
}

// TestExchangeConnectionResponse 测试交易所连接响应
type TestExchangeConnectionResponse struct {
	Success          bool    `json:"success"`
	ExchangeID       string  `json:"exchange_id,omitempty"`
	Testnet          bool    `json:"testnet,omitempty"`
	TotalEquity      float64 `json:"total_equity,omitempty"`
	AvailableBalance float64 `json:"available_balance,omitempty"`
	Message          string  `json:"message,omitempty"`
	Error            string  `json:"error,omitempty"`
}

type resolvedExchangeCredentials struct {
	ExchangeID            string
	APIKey                string
	SecretKey             string
	Testnet               bool
	HyperliquidWalletAddr string
	AsterUser             string
	AsterSigner           string
	AsterPrivateKey       string
}

// resolveExchangeCredentials 合并表单与数据库已保存的敏感字段
func resolveExchangeCredentials(req TestExchangeConnectionRequest, saved *config.ExchangeConfig) resolvedExchangeCredentials {
	creds := resolvedExchangeCredentials{
		ExchangeID:            strings.TrimSpace(req.ExchangeID),
		APIKey:                strings.TrimSpace(req.APIKey),
		SecretKey:             strings.TrimSpace(req.SecretKey),
		Testnet:               req.Testnet,
		HyperliquidWalletAddr: strings.TrimSpace(req.HyperliquidWalletAddr),
		AsterUser:             strings.TrimSpace(req.AsterUser),
		AsterSigner:           strings.TrimSpace(req.AsterSigner),
		AsterPrivateKey:       strings.TrimSpace(req.AsterPrivateKey),
	}

	if saved == nil {
		return creds
	}

	if creds.APIKey == "" {
		creds.APIKey = saved.APIKey
	}
	if creds.SecretKey == "" {
		creds.SecretKey = saved.SecretKey
	}
	if creds.AsterPrivateKey == "" {
		creds.AsterPrivateKey = saved.AsterPrivateKey
	}

	return creds
}

func validateExchangeCredentials(creds resolvedExchangeCredentials) error {
	switch creds.ExchangeID {
	case "binance":
		if creds.APIKey == "" || creds.SecretKey == "" {
			return fmt.Errorf("币安 API Key 和 Secret Key 不能为空")
		}
	case "hyperliquid":
		if creds.APIKey == "" {
			return fmt.Errorf("Hyperliquid 私钥不能为空")
		}
		if creds.HyperliquidWalletAddr == "" {
			return fmt.Errorf("Hyperliquid 主钱包地址不能为空")
		}
	case "aster":
		if creds.AsterUser == "" || creds.AsterSigner == "" || creds.AsterPrivateKey == "" {
			return fmt.Errorf("Aster 用户、签名者和私钥不能为空")
		}
	case "okx":
		return fmt.Errorf("暂不支持该交易所连接测试")
	default:
		return fmt.Errorf("暂不支持该交易所连接测试")
	}
	return nil
}

type balanceSnapshot struct {
	TotalEquity      float64
	AvailableBalance float64
}

// extractBalanceSnapshot 从 GetBalance 返回值解析净值与可用余额
func extractBalanceSnapshot(balanceInfo map[string]interface{}) balanceSnapshot {
	var totalWalletBalance float64
	var totalUnrealizedProfit float64
	var availableBalance float64

	if wb, ok := balanceInfo["totalWalletBalance"].(float64); ok {
		totalWalletBalance = wb
	} else if wb, ok := balanceInfo["wallet_balance"].(float64); ok {
		totalWalletBalance = wb
	} else if wb, ok := balanceInfo["balance"].(float64); ok {
		totalWalletBalance = wb
	}

	if up, ok := balanceInfo["totalUnrealizedProfit"].(float64); ok {
		totalUnrealizedProfit = up
	} else if up, ok := balanceInfo["unrealized_profit"].(float64); ok {
		totalUnrealizedProfit = up
	}

	if ab, ok := balanceInfo["availableBalance"].(float64); ok {
		availableBalance = ab
	} else if ab, ok := balanceInfo["available_balance"].(float64); ok {
		availableBalance = ab
	}

	return balanceSnapshot{
		TotalEquity:      totalWalletBalance + totalUnrealizedProfit,
		AvailableBalance: availableBalance,
	}
}

func createTempTraderForTest(userID string, creds resolvedExchangeCredentials) (trader.Trader, error) {
	switch creds.ExchangeID {
	case "binance":
		return trader.NewFuturesTrader(creds.APIKey, creds.SecretKey, userID, creds.Testnet), nil
	case "hyperliquid":
		return trader.NewHyperliquidTrader(creds.APIKey, creds.HyperliquidWalletAddr, creds.Testnet)
	case "aster":
		return trader.NewAsterTrader(creds.AsterUser, creds.AsterSigner, creds.AsterPrivateKey)
	default:
		return nil, fmt.Errorf("暂不支持该交易所连接测试")
	}
}

func testExchangeConnection(userID string, creds resolvedExchangeCredentials) TestExchangeConnectionResponse {
	if err := validateExchangeCredentials(creds); err != nil {
		return TestExchangeConnectionResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	tempTrader, err := createTempTraderForTest(userID, creds)
	if err != nil {
		return TestExchangeConnectionResponse{
			Success:    false,
			ExchangeID: creds.ExchangeID,
			Testnet:    creds.Testnet,
			Error:      err.Error(),
		}
	}

	balanceInfo, balanceErr := tempTrader.GetBalance()
	if balanceErr != nil {
		log.Printf("❌ 交易所连接测试失败 [%s] testnet=%v: %v", creds.ExchangeID, creds.Testnet, balanceErr)
		return TestExchangeConnectionResponse{
			Success:    false,
			ExchangeID: creds.ExchangeID,
			Testnet:    creds.Testnet,
			Error:      balanceErr.Error(),
		}
	}

	snapshot := extractBalanceSnapshot(balanceInfo)
	log.Printf("✅ 交易所连接测试成功 [%s] testnet=%v equity=%.2f", creds.ExchangeID, creds.Testnet, snapshot.TotalEquity)

	return TestExchangeConnectionResponse{
		Success:          true,
		ExchangeID:       creds.ExchangeID,
		Testnet:          creds.Testnet,
		TotalEquity:      snapshot.TotalEquity,
		AvailableBalance: snapshot.AvailableBalance,
		Message:          "连接成功",
	}
}

// handleTestExchangeConnection 测试交易所 API 密钥连接（仅支持加密数据）
func (s *Server) handleTestExchangeConnection(c *gin.Context) {
	userID := c.GetString("user_id")

	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	var encryptedPayload crypto.EncryptedPayload
	if err := json.Unmarshal(bodyBytes, &encryptedPayload); err != nil {
		log.Printf("❌ 解析加密载荷失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误，必须使用加密传输"})
		return
	}

	if encryptedPayload.WrappedKey == "" {
		log.Printf("❌ 检测到非加密请求 (UserID: %s)", userID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "此接口仅支持加密传输，请使用加密客户端",
			"code":    "ENCRYPTION_REQUIRED",
			"message": "Encrypted transmission is required for security reasons",
		})
		return
	}

	decrypted, err := s.cryptoHandler.cryptoService.DecryptSensitiveData(&encryptedPayload)
	if err != nil {
		log.Printf("❌ 解密交易所测试请求失败 (UserID: %s): %v", userID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "解密数据失败"})
		return
	}

	var req TestExchangeConnectionRequest
	if err := json.Unmarshal([]byte(decrypted), &req); err != nil {
		log.Printf("❌ 解析解密数据失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析解密数据失败"})
		return
	}

	if strings.TrimSpace(req.ExchangeID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "exchange_id 不能为空"})
		return
	}

	var saved *config.ExchangeConfig
	exchanges, err := s.database.GetExchanges(userID)
	if err != nil {
		log.Printf("⚠️ 获取用户交易所配置失败 (UserID: %s): %v", userID, err)
	} else {
		for _, ex := range exchanges {
			if ex.ID == req.ExchangeID {
				saved = ex
				break
			}
		}
	}

	creds := resolveExchangeCredentials(req, saved)
	result := testExchangeConnection(userID, creds)
	c.JSON(http.StatusOK, result)
}
