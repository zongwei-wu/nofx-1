package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"nofx/auth"
	"nofx/config"
	"nofx/crypto"
	"nofx/decision"
	"nofx/hook"
	"nofx/logger"
	"nofx/manager"
	"nofx/mcp"
	"nofx/trader"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Server HTTP API服务器
type Server struct {
	router        *gin.Engine
	httpServer    *http.Server
	traderManager *manager.TraderManager
	database      *config.Database
	cryptoHandler *CryptoHandler
	port          int
}

// NewServer 创建API服务器
func NewServer(traderManager *manager.TraderManager, database *config.Database, cryptoService *crypto.CryptoService, port int) *Server {
	// 设置为Release模式（减少日志输出）
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	// 启用CORS
	router.Use(corsMiddleware())

	// 创建加密处理器
	cryptoHandler := NewCryptoHandler(cryptoService)

	s := &Server{
		router:        router,
		traderManager: traderManager,
		database:      database,
		cryptoHandler: cryptoHandler,
		port:          port,
	}

	// 设置路由
	s.setupRoutes()

	return s
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// API路由组
	api := s.router.Group("/api")
	{
		// 健康检查
		api.Any("/health", s.handleHealth)

		// 管理员登录（管理员模式下使用，公共）

		// 系统支持的模型和交易所（无需认证）
		api.GET("/supported-models", s.handleGetSupportedModels)
		api.GET("/supported-exchanges", s.handleGetSupportedExchanges)

		// 系统配置（无需认证，用于前端判断是否管理员模式/注册是否开启）
		api.GET("/config", s.handleGetSystemConfig)

		// 加密相关接口（无需认证）
		api.GET("/crypto/public-key", s.cryptoHandler.HandleGetPublicKey)
		api.POST("/crypto/decrypt", s.cryptoHandler.HandleDecryptSensitiveData)

		// 系统提示词模板管理（无需认证）
		api.GET("/prompt-templates", s.handleGetPromptTemplates)
		api.GET("/prompt-templates/:name", s.handleGetPromptTemplate)

		// 公开的竞赛数据（无需认证）
		api.GET("/traders", s.handlePublicTraderList)
		api.GET("/competition", s.handlePublicCompetition)
		api.GET("/top-traders", s.handleTopTraders)
		api.GET("/equity-history", s.handleEquityHistory)
		api.POST("/equity-history-batch", s.handleEquityHistoryBatch)
		api.GET("/traders/:id/public-config", s.handleGetPublicTraderConfig)

		api.GET("/market/klines", s.handleMarketKlines)

		// 认证相关路由（无需认证）
		api.POST("/register", s.handleRegister)
		api.POST("/login", s.handleLogin)
		api.POST("/verify-otp", s.handleVerifyOTP)
		api.POST("/complete-registration", s.handleCompleteRegistration)

		// 需要认证的路由
		protected := api.Group("/", s.authMiddleware())
		{
			protected.GET("/me", s.handleMe)
			protected.POST("/logout", s.handleLogout)
			protected.GET("/server-ip", s.handleGetServerIP)

			// 排行榜（需 leaderboard 权限）
			leaderboard := protected.Group("/", s.requireFeature(config.FeatureLeaderboard))
			{
				leaderboard.GET("/copy-trading/leaderboard", s.handleCopyTradingLeaderboard)
				leaderboard.GET("/copy-trading/orders", s.handleCopyTradingOrders)
			}

			// AI 交易员（需 ai_trader 权限）
			aiTrader := protected.Group("/", s.requireFeature(config.FeatureAITrader))
			{
				aiTrader.GET("/my-traders", s.handleTraderList)
				aiTrader.GET("/traders/:id/config", s.handleGetTraderConfig)
				aiTrader.POST("/traders", s.handleCreateTrader)
				aiTrader.PUT("/traders/:id", s.handleUpdateTrader)
				aiTrader.DELETE("/traders/:id", s.handleDeleteTrader)
				aiTrader.POST("/traders/:id/start", s.handleStartTrader)
				aiTrader.POST("/traders/:id/stop", s.handleStopTrader)
				aiTrader.PUT("/traders/:id/prompt", s.handleUpdateTraderPrompt)
				aiTrader.GET("/models", s.handleGetModelConfigs)
				aiTrader.PUT("/models", s.handleUpdateModelConfigs)
				aiTrader.GET("/exchanges", s.handleGetExchangeConfigs)
				aiTrader.PUT("/exchanges", s.handleUpdateExchangeConfigs)
				aiTrader.GET("/user/signal-sources", s.handleGetUserSignalSource)
				aiTrader.POST("/user/signal-sources", s.handleSaveUserSignalSource)
				aiTrader.GET("/status", s.handleStatus)
				aiTrader.GET("/account", s.handleAccount)
				aiTrader.GET("/positions", s.handlePositions)
				aiTrader.GET("/decisions", s.handleDecisions)
				aiTrader.GET("/decisions/latest", s.handleLatestDecisions)
				aiTrader.GET("/statistics", s.handleStatistics)
				aiTrader.GET("/performance", s.handlePerformance)
			}

			// 跟单管理（需 copy_trade 权限）
			copyTrade := protected.Group("/", s.requireFeature(config.FeatureCopyTrade))
			{
				copyTrade.GET("/copy-trade/configs", s.handleGetCopyTradeConfigs)
				copyTrade.POST("/copy-trade/configs", s.handleUpdateCopyTradeConfig)
				copyTrade.DELETE("/copy-trade/configs/:id", s.handleDeleteCopyTradeConfig)
				copyTrade.GET("/copy-trade/settings", s.handleGetCopyTradeSettings)
				copyTrade.PUT("/copy-trade/settings", s.handleUpdateCopyTradeSettings)
				copyTrade.GET("/copy-trade/records", s.handleGetCopyTradeRecords)
				copyTrade.GET("/copy-trade/monitor", s.handleGetCopyTradeMonitor)
				copyTrade.POST("/copy-trade/refresh-pnl", s.handleRefreshCopyTradePnL)
				copyTrade.POST("/copy-trade/sync", s.handleSyncCopyTrade)
				copyTrade.POST("/copy-trade/copy-order", s.handleCopyOrder)
			}

			// 交易事件图（AI 交易员或跟单任一权限）
			tradeEvents := protected.Group("/", s.requireAnyFeature(config.FeatureAITrader, config.FeatureCopyTrade))
			{
				tradeEvents.GET("/trade-events", s.handleTradeEvents)
			}

			// 币种管理（需 symbols 权限）
			symbols := protected.Group("/", s.requireFeature(config.FeatureSymbols))
			{
				symbols.GET("/symbol-preferences", s.handleGetSymbolPreferences)
				symbols.PUT("/symbol-preferences", s.handlePutSymbolPreferences)
				symbols.POST("/symbol-preferences/reset", s.handleResetSymbolPreferences)
				symbols.GET("/symbol-values", s.handleGetSymbolValues)
			}

			// 管理端 API（需 admin 角色）
			admin := protected.Group("/admin", s.adminMiddleware())
			{
				admin.GET("/me", s.handleAdminMe)
				admin.GET("/admins", s.handleAdminListAdmins)
				admin.POST("/admins", s.handleAdminCreateAdmin)
				admin.GET("/users", s.handleAdminListUsers)
				admin.PUT("/users/:id", s.handleAdminUpdateUser)
				admin.PUT("/users/:id/role", s.handleAdminUpdateUserRole)
				admin.PUT("/users/:id/password", s.handleAdminResetUserPassword)
				admin.DELETE("/users/:id", s.handleAdminDeleteUser)
				admin.GET("/plans", s.handleAdminListPlans)
				admin.GET("/traders", s.handleAdminListTraders)
				admin.GET("/traders/:id", s.handleAdminGetTrader)
				admin.GET("/traders/:id/account", s.handleAdminGetTraderAccount)
				admin.GET("/traders/:id/positions", s.handleAdminGetTraderPositions)
				admin.POST("/traders/:id/stop", s.handleAdminStopTrader)
				admin.GET("/copy-trade/records", s.handleAdminCopyTradeRecords)
				admin.GET("/system-config", s.handleAdminGetSystemConfig)
				admin.PUT("/system-config", s.handleAdminPutSystemConfig)
			}
		}
	}
}

// handleHealth 健康检查
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   c.Request.Context().Value("time"),
	})
}

// handleGetSystemConfig 获取系统配置（客户端需要知道的配置）
func (s *Server) handleGetSystemConfig(c *gin.Context) {
	// 获取默认币种
	defaultCoinsStr, _ := s.database.GetSystemConfig("default_coins")
	var defaultCoins []string
	if defaultCoinsStr != "" {
		json.Unmarshal([]byte(defaultCoinsStr), &defaultCoins)
	}
	if len(defaultCoins) == 0 {
		// 使用硬编码的默认币种
		defaultCoins = []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT", "XRPUSDT", "DOGEUSDT", "ADAUSDT", "HYPEUSDT"}
	}

	// 获取杠杆配置
	btcEthLeverageStr, _ := s.database.GetSystemConfig("btc_eth_leverage")
	altcoinLeverageStr, _ := s.database.GetSystemConfig("altcoin_leverage")

	btcEthLeverage := 5
	if val, err := strconv.Atoi(btcEthLeverageStr); err == nil && val > 0 {
		btcEthLeverage = val
	}

	altcoinLeverage := 5
	if val, err := strconv.Atoi(altcoinLeverageStr); err == nil && val > 0 {
		altcoinLeverage = val
	}

	// 获取内测模式配置
	betaModeStr, _ := s.database.GetSystemConfig("beta_mode")
	betaMode := betaModeStr == "true"

	regEnabledStr, err := s.database.GetSystemConfig("registration_enabled")
	registrationEnabled := true
	if err == nil {
		registrationEnabled = strings.ToLower(regEnabledStr) != "false"
	}

	c.JSON(http.StatusOK, gin.H{
		"beta_mode":            betaMode,
		"default_coins":        defaultCoins,
		"btc_eth_leverage":     btcEthLeverage,
		"altcoin_leverage":     altcoinLeverage,
		"registration_enabled": registrationEnabled,
	})
}

// handleGetServerIP 获取服务器IP地址（用于白名单配置）
func (s *Server) handleGetServerIP(c *gin.Context) {

	// 首先尝试从Hook获取用户专用IP
	userIP := hook.HookExec[hook.IpResult](hook.GETIP, c.GetString("user_id"))
	if userIP != nil && userIP.Error() == nil {
		c.JSON(http.StatusOK, gin.H{
			"public_ip": userIP.GetResult(),
			"message":   "请将此IP地址添加到白名单中",
		})
		return
	}

	// 尝试通过第三方API获取公网IP
	publicIP := getPublicIPFromAPI()

	// 如果第三方API失败，从网络接口获取第一个公网IP
	if publicIP == "" {
		publicIP = getPublicIPFromInterface()
	}

	// 如果还是没有获取到，返回错误
	if publicIP == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取公网IP地址"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"public_ip": publicIP,
		"message":   "请将此IP地址添加到白名单中",
	})
}

// getPublicIPFromAPI 通过第三方API获取公网IP
func getPublicIPFromAPI() string {
	// 尝试多个公网IP查询服务
	services := []string{
		"https://api.ipify.org?format=text",
		"https://icanhazip.com",
		"https://ifconfig.me",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body := make([]byte, 128)
			n, err := resp.Body.Read(body)
			if err != nil && err.Error() != "EOF" {
				continue
			}

			ip := strings.TrimSpace(string(body[:n]))
			// 验证是否为有效的IP地址
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	return ""
}

// getPublicIPFromInterface 从网络接口获取第一个公网IP
func getPublicIPFromInterface() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		// 跳过未启用的接口和回环接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			// 只考虑IPv4地址
			if ip.To4() != nil {
				ipStr := ip.String()
				// 排除私有IP地址范围
				if !isPrivateIP(ip) {
					return ipStr
				}
			}
		}
	}

	return ""
}

// isPrivateIP 判断是否为私有IP地址
func isPrivateIP(ip net.IP) bool {
	// 私有IP地址范围：
	// 10.0.0.0/8
	// 172.16.0.0/12
	// 192.168.0.0/16
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range privateRanges {
		_, subnet, _ := net.ParseCIDR(cidr)
		if subnet.Contains(ip) {
			return true
		}
	}

	return false
}

// getTraderFromQuery 从query参数获取trader
func (s *Server) getTraderFromQuery(c *gin.Context) (*manager.TraderManager, string, error) {
	userID := c.GetString("user_id")
	traderID := c.Query("trader_id")

	// 确保用户的交易员已加载到内存中
	err := s.traderManager.LoadUserTraders(s.database, userID)
	if err != nil {
		log.Printf("⚠️ 加载用户 %s 的交易员失败: %v", userID, err)
	}

	if traderID == "" {
		// 如果没有指定trader_id，返回该用户的第一个trader
		ids := s.traderManager.GetTraderIDs()
		if len(ids) == 0 {
			return nil, "", fmt.Errorf("没有可用的trader")
		}

		// 获取用户的交易员列表，优先返回用户自己的交易员
		userTraders, err := s.database.GetTraders(userID)
		if err == nil && len(userTraders) > 0 {
			traderID = userTraders[0].ID
		} else {
			traderID = ids[0]
		}
	}

	return s.traderManager, traderID, nil
}

// AI交易员管理相关结构体
type CreateTraderRequest struct {
	Name                 string  `json:"name" binding:"required"`
	AIModelID            string  `json:"ai_model_id" binding:"required"`
	ExchangeID           string  `json:"exchange_id" binding:"required"`
	InitialBalance       float64 `json:"initial_balance"`
	ScanIntervalMinutes  int     `json:"scan_interval_minutes"`
	BTCETHLeverage       int     `json:"btc_eth_leverage"`
	AltcoinLeverage      int     `json:"altcoin_leverage"`
	TradingSymbols       string  `json:"trading_symbols"`
	CustomPrompt         string  `json:"custom_prompt"`
	OverrideBasePrompt   bool    `json:"override_base_prompt"`
	SystemPromptTemplate string  `json:"system_prompt_template"` // 系统提示词模板名称
	IsCrossMargin        *bool   `json:"is_cross_margin"`        // 指针类型，nil表示使用默认值true
	UseCoinPool          bool    `json:"use_coin_pool"`
	UseOITop             bool    `json:"use_oi_top"`
}

type ModelConfig struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Enabled      bool   `json:"enabled"`
	APIKey       string `json:"apiKey,omitempty"`
	CustomAPIURL string `json:"customApiUrl,omitempty"`
}

// SafeModelConfig 安全的模型配置结构（不包含敏感信息）
type SafeModelConfig struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Provider        string `json:"provider"`
	Enabled         bool   `json:"enabled"`
	APIKey          string `json:"apiKey,omitempty"`
	CustomAPIURL    string `json:"customApiUrl"`    // 自定义API URL（通常不敏感）
	CustomModelName string `json:"customModelName"` // 自定义模型名（不敏感）
}

type ExchangeConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "cex" or "dex"
	Enabled   bool   `json:"enabled"`
	APIKey    string `json:"apiKey,omitempty"`
	SecretKey string `json:"secretKey,omitempty"`
	Testnet   bool   `json:"testnet,omitempty"`
}

// SafeExchangeConfig 安全的交易所配置结构（不包含敏感信息）
type SafeExchangeConfig struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Type                  string `json:"type"` // "cex" or "dex"
	Enabled               bool   `json:"enabled"`
	APIKey                string `json:"apiKey,omitempty"`
	SecretKey             string `json:"secretKey,omitempty"`
	Testnet               bool   `json:"testnet,omitempty"`
	HyperliquidWalletAddr string `json:"hyperliquidWalletAddr"` // Hyperliquid钱包地址（不敏感）
	AsterUser             string `json:"asterUser"`             // Aster用户名（不敏感）
	AsterSigner           string `json:"asterSigner"`           // Aster签名者（不敏感）
}

type UpdateModelConfigRequest struct {
	Models map[string]struct {
		Enabled         bool   `json:"enabled"`
		APIKey          string `json:"api_key"`
		CustomAPIURL    string `json:"custom_api_url"`
		CustomModelName string `json:"custom_model_name"`
	} `json:"models"`
}

type UpdateExchangeConfigRequest struct {
	Exchanges map[string]struct {
		Enabled               bool   `json:"enabled"`
		APIKey                string `json:"api_key"`
		SecretKey             string `json:"secret_key"`
		Testnet               bool   `json:"testnet"`
		HyperliquidWalletAddr string `json:"hyperliquid_wallet_addr"`
		AsterUser             string `json:"aster_user"`
		AsterSigner           string `json:"aster_signer"`
		AsterPrivateKey       string `json:"aster_private_key"`
	} `json:"exchanges"`
}

// handleCreateTrader 创建新的AI交易员
func (s *Server) handleCreateTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	var req CreateTraderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 校验杠杆值
	if req.BTCETHLeverage < 0 || req.BTCETHLeverage > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BTC/ETH杠杆必须在1-50倍之间"})
		return
	}
	if req.AltcoinLeverage < 0 || req.AltcoinLeverage > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "山寨币杠杆必须在1-20倍之间"})
		return
	}

	// 校验交易币种格式
	if req.TradingSymbols != "" {
		symbols := strings.Split(req.TradingSymbols, ",")
		for _, symbol := range symbols {
			symbol = strings.TrimSpace(symbol)
			if symbol != "" && !strings.HasSuffix(strings.ToUpper(symbol), "USDT") {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("无效的币种格式: %s，必须以USDT结尾", symbol)})
				return
			}
		}
	}

	// 生成交易员ID (使用 UUID 确保唯一性，解决 Issue #893)
	// 保留前缀以便调试和日志追踪
	traderID := fmt.Sprintf("%s_%s_%s", req.ExchangeID, req.AIModelID, uuid.New().String())

	// 设置默认值
	isCrossMargin := true // 默认为全仓模式
	if req.IsCrossMargin != nil {
		isCrossMargin = *req.IsCrossMargin
	}

	// 设置杠杆默认值（从系统配置获取）
	btcEthLeverage := 5
	altcoinLeverage := 5
	if req.BTCETHLeverage > 0 {
		btcEthLeverage = req.BTCETHLeverage
	} else {
		// 从系统配置获取默认值
		if btcEthLeverageStr, _ := s.database.GetSystemConfig("btc_eth_leverage"); btcEthLeverageStr != "" {
			if val, err := strconv.Atoi(btcEthLeverageStr); err == nil && val > 0 {
				btcEthLeverage = val
			}
		}
	}
	if req.AltcoinLeverage > 0 {
		altcoinLeverage = req.AltcoinLeverage
	} else {
		// 从系统配置获取默认值
		if altcoinLeverageStr, _ := s.database.GetSystemConfig("altcoin_leverage"); altcoinLeverageStr != "" {
			if val, err := strconv.Atoi(altcoinLeverageStr); err == nil && val > 0 {
				altcoinLeverage = val
			}
		}
	}

	// 设置系统提示词模板默认值
	systemPromptTemplate := "default"
	if req.SystemPromptTemplate != "" {
		systemPromptTemplate = req.SystemPromptTemplate
	}

	// 设置扫描间隔默认值
	scanIntervalMinutes := req.ScanIntervalMinutes
	if scanIntervalMinutes < 3 {
		scanIntervalMinutes = 3 // 默认3分钟，且不允许小于3
	}

	// ✨ 查询交易所实际余额，覆盖用户输入
	actualBalance := req.InitialBalance // 默认使用用户输入
	exchanges, err := s.database.GetExchanges(userID)
	if err != nil {
		log.Printf("⚠️ 获取交易所配置失败，使用用户输入的初始资金: %v", err)
	}

	// 查找匹配的交易所配置
	var exchangeCfg *config.ExchangeConfig
	for _, ex := range exchanges {
		if ex.ID == req.ExchangeID {
			exchangeCfg = ex
			break
		}
	}

	if exchangeCfg == nil {
		log.Printf("⚠️ 未找到交易所 %s 的配置，使用用户输入的初始资金", req.ExchangeID)
	} else if !exchangeCfg.Enabled {
		log.Printf("⚠️ 交易所 %s 未启用，使用用户输入的初始资金", req.ExchangeID)
	} else {
		// 根据交易所类型创建临时 trader 查询余额
		var tempTrader trader.Trader
		var createErr error

		switch req.ExchangeID {
		case "binance":
			tempTrader = trader.NewFuturesTrader(exchangeCfg.APIKey, exchangeCfg.SecretKey, userID, exchangeCfg.Testnet)
		case "hyperliquid":
			tempTrader, createErr = trader.NewHyperliquidTrader(
				exchangeCfg.APIKey, // private key
				exchangeCfg.HyperliquidWalletAddr,
				exchangeCfg.Testnet,
			)
		case "aster":
			tempTrader, createErr = trader.NewAsterTrader(
				exchangeCfg.AsterUser,
				exchangeCfg.AsterSigner,
				exchangeCfg.AsterPrivateKey,
			)
		default:
			log.Printf("⚠️ 不支持的交易所类型: %s，使用用户输入的初始资金", req.ExchangeID)
		}

		if createErr != nil {
			log.Printf("⚠️ 创建临时 trader 失败，使用用户输入的初始资金: %v", createErr)
		} else if tempTrader != nil {
			// 查询实际余额
			balanceInfo, balanceErr := tempTrader.GetBalance()
			if balanceErr != nil {
				log.Printf("⚠️ 查询交易所余额失败，使用用户输入的初始资金: %v", balanceErr)
			} else {
				// 🔧 计算Total Equity = Wallet Balance + Unrealized Profit
				// 这是账户的真实净值，用作Initial Balance的基准
				var totalWalletBalance float64
				var totalUnrealizedProfit float64

				// 提取钱包余额
				if wb, ok := balanceInfo["totalWalletBalance"].(float64); ok {
					totalWalletBalance = wb
				} else if wb, ok := balanceInfo["wallet_balance"].(float64); ok {
					totalWalletBalance = wb
				} else if wb, ok := balanceInfo["balance"].(float64); ok {
					totalWalletBalance = wb
				}

				// 提取未实现盈亏
				if up, ok := balanceInfo["totalUnrealizedProfit"].(float64); ok {
					totalUnrealizedProfit = up
				} else if up, ok := balanceInfo["unrealized_profit"].(float64); ok {
					totalUnrealizedProfit = up
				}

				// 计算总净值
				totalEquity := totalWalletBalance + totalUnrealizedProfit

				if totalEquity > 0 {
					actualBalance = totalEquity
					log.Printf("✅ 查询到交易所实际净值: %.2f USDT (钱包: %.2f + 未实现: %.2f, 用户输入: %.2f)",
						actualBalance, totalWalletBalance, totalUnrealizedProfit, req.InitialBalance)
				} else {
					log.Printf("⚠️ 无法从余额信息中计算净值，使用用户输入的初始资金")
				}
			}
		}
	}

	// 创建交易员配置（数据库实体）
	trader := &config.TraderRecord{
		ID:                   traderID,
		UserID:               userID,
		Name:                 req.Name,
		AIModelID:            req.AIModelID,
		ExchangeID:           req.ExchangeID,
		InitialBalance:       actualBalance, // 使用实际查询的余额
		BTCETHLeverage:       btcEthLeverage,
		AltcoinLeverage:      altcoinLeverage,
		TradingSymbols:       req.TradingSymbols,
		UseCoinPool:          req.UseCoinPool,
		UseOITop:             req.UseOITop,
		CustomPrompt:         req.CustomPrompt,
		OverrideBasePrompt:   req.OverrideBasePrompt,
		SystemPromptTemplate: systemPromptTemplate,
		IsCrossMargin:        isCrossMargin,
		ScanIntervalMinutes:  scanIntervalMinutes,
		IsRunning:            false,
	}

	// 保存到数据库
	err = s.database.CreateTrader(trader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建交易员失败: %v", err)})
		return
	}

	// 立即将新交易员加载到TraderManager中
	err = s.traderManager.LoadTraderByID(s.database, userID, traderID)
	if err != nil {
		log.Printf("⚠️ 加载交易员到内存失败: %v", err)
		// 这里不返回错误，因为交易员已经成功创建到数据库
	}

	log.Printf("✓ 创建交易员成功: %s (模型: %s, 交易所: %s)", req.Name, req.AIModelID, req.ExchangeID)

	c.JSON(http.StatusCreated, gin.H{
		"trader_id":   traderID,
		"trader_name": req.Name,
		"ai_model":    req.AIModelID,
		"is_running":  false,
	})
}

// UpdateTraderRequest 更新交易员请求
type UpdateTraderRequest struct {
	Name                 string  `json:"name" binding:"required"`
	AIModelID            string  `json:"ai_model_id" binding:"required"`
	ExchangeID           string  `json:"exchange_id" binding:"required"`
	InitialBalance       float64 `json:"initial_balance"`
	ScanIntervalMinutes  int     `json:"scan_interval_minutes"`
	BTCETHLeverage       int     `json:"btc_eth_leverage"`
	AltcoinLeverage      int     `json:"altcoin_leverage"`
	TradingSymbols       string  `json:"trading_symbols"`
	CustomPrompt         string  `json:"custom_prompt"`
	OverrideBasePrompt   bool    `json:"override_base_prompt"`
	SystemPromptTemplate string  `json:"system_prompt_template"`
	IsCrossMargin        *bool   `json:"is_cross_margin"`
}

// handleUpdateTrader 更新交易员配置
func (s *Server) handleUpdateTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	traderID := c.Param("id")

	var req UpdateTraderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查交易员是否存在且属于当前用户
	traders, err := s.database.GetTraders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取交易员列表失败"})
		return
	}

	var existingTrader *config.TraderRecord
	for _, trader := range traders {
		if trader.ID == traderID {
			existingTrader = trader
			break
		}
	}

	if existingTrader == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "交易员不存在"})
		return
	}

	// 设置默认值
	isCrossMargin := existingTrader.IsCrossMargin // 保持原值
	if req.IsCrossMargin != nil {
		isCrossMargin = *req.IsCrossMargin
	}

	// 设置杠杆默认值
	btcEthLeverage := req.BTCETHLeverage
	altcoinLeverage := req.AltcoinLeverage
	if btcEthLeverage <= 0 {
		btcEthLeverage = existingTrader.BTCETHLeverage // 保持原值
	}
	if altcoinLeverage <= 0 {
		altcoinLeverage = existingTrader.AltcoinLeverage // 保持原值
	}

	// 设置扫描间隔，允许更新
	scanIntervalMinutes := req.ScanIntervalMinutes
	if scanIntervalMinutes <= 0 {
		scanIntervalMinutes = existingTrader.ScanIntervalMinutes // 保持原值
	} else if scanIntervalMinutes < 3 {
		scanIntervalMinutes = 3
	}

	// 设置提示词模板，允许更新
	systemPromptTemplate := req.SystemPromptTemplate
	if systemPromptTemplate == "" {
		systemPromptTemplate = existingTrader.SystemPromptTemplate // 如果请求中没有提供，保持原值
	}

	// 更新交易员配置
	trader := &config.TraderRecord{
		ID:                   traderID,
		UserID:               userID,
		Name:                 req.Name,
		AIModelID:            req.AIModelID,
		ExchangeID:           req.ExchangeID,
		InitialBalance:       req.InitialBalance,
		BTCETHLeverage:       btcEthLeverage,
		AltcoinLeverage:      altcoinLeverage,
		TradingSymbols:       req.TradingSymbols,
		CustomPrompt:         req.CustomPrompt,
		OverrideBasePrompt:   req.OverrideBasePrompt,
		SystemPromptTemplate: systemPromptTemplate,
		IsCrossMargin:        isCrossMargin,
		ScanIntervalMinutes:  scanIntervalMinutes,
		IsRunning:            existingTrader.IsRunning, // 保持原值
	}

	// 更新数据库
	err = s.database.UpdateTrader(trader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新交易员失败: %v", err)})
		return
	}

	// 如果请求中包含initial_balance且与现有值不同，单独更新它
	// UpdateTrader不会更新initial_balance，需要使用专门的方法
	if req.InitialBalance > 0 && math.Abs(req.InitialBalance-existingTrader.InitialBalance) > 0.1 {
		err = s.database.UpdateTraderInitialBalance(userID, traderID, req.InitialBalance)
		if err != nil {
			log.Printf("⚠️ 更新初始余额失败: %v", err)
			// 不返回错误，因为主要配置已更新成功
		} else {
			log.Printf("✓ 初始余额已更新: %.2f -> %.2f", existingTrader.InitialBalance, req.InitialBalance)
		}
	}

	// 🔄 从内存中移除旧的trader实例，以便重新加载最新配置
	s.traderManager.RemoveTrader(traderID)

	// 重新加载交易员到内存
	err = s.traderManager.LoadTraderByID(s.database, userID, traderID)
	if err != nil {
		log.Printf("⚠️ 重新加载交易员到内存失败: %v", err)
	}

	log.Printf("✓ 更新交易员成功: %s (模型: %s, 交易所: %s)", req.Name, req.AIModelID, req.ExchangeID)

	c.JSON(http.StatusOK, gin.H{
		"trader_id":   traderID,
		"trader_name": req.Name,
		"ai_model":    req.AIModelID,
		"message":     "交易员更新成功",
	})
}

// handleDeleteTrader 删除交易员
func (s *Server) handleDeleteTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	traderID := c.Param("id")

	// 从数据库删除
	err := s.database.DeleteTrader(userID, traderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("删除交易员失败: %v", err)})
		return
	}

	// 如果交易员正在运行，先停止它
	if trader, err := s.traderManager.GetTrader(traderID); err == nil {
		status := trader.GetStatus()
		if isRunning, ok := status["is_running"].(bool); ok && isRunning {
			trader.Stop()
			log.Printf("⏹  已停止运行中的交易员: %s", traderID)
		}
	}

	log.Printf("✓ 交易员已删除: %s", traderID)
	c.JSON(http.StatusOK, gin.H{"message": "交易员已删除"})
}

// handleStartTrader 启动交易员
func (s *Server) handleStartTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	traderID := c.Param("id")

	// 校验交易员是否属于当前用户
	traderRecord, _, _, err := s.database.GetTraderConfig(userID, traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "交易员不存在或无访问权限"})
		return
	}

	// 获取模板名称
	templateName := traderRecord.SystemPromptTemplate

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "交易员不存在"})
		return
	}

	// 检查交易员是否已经在运行
	status := trader.GetStatus()
	if isRunning, ok := status["is_running"].(bool); ok && isRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "交易员已在运行中"})
		return
	}

	// 重新加载系统提示词模板（确保使用最新的硬盘文件）
	s.reloadPromptTemplatesWithLog(templateName)

	// 启动交易员
	go func() {
		log.Printf("▶️  启动交易员 %s (%s)", traderID, trader.GetName())
		if err := trader.Run(); err != nil {
			log.Printf("❌ 交易员 %s 运行错误: %v", trader.GetName(), err)
		}
	}()

	// 更新数据库中的运行状态
	err = s.database.UpdateTraderStatus(userID, traderID, true)
	if err != nil {
		log.Printf("⚠️  更新交易员状态失败: %v", err)
	}

	log.Printf("✓ 交易员 %s 已启动", trader.GetName())
	c.JSON(http.StatusOK, gin.H{"message": "交易员已启动"})
}

// handleStopTrader 停止交易员
func (s *Server) handleStopTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	traderID := c.Param("id")

	if err := s.stopTraderForUser(userID, traderID); err != nil {
		if err.Error() == "交易员已停止" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "交易员已停止"})
}

// handleUpdateTraderPrompt 更新交易员自定义Prompt
func (s *Server) handleUpdateTraderPrompt(c *gin.Context) {
	traderID := c.Param("id")
	userID := c.GetString("user_id")

	var req struct {
		CustomPrompt       string `json:"custom_prompt"`
		OverrideBasePrompt bool   `json:"override_base_prompt"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新数据库
	err := s.database.UpdateTraderCustomPrompt(userID, traderID, req.CustomPrompt, req.OverrideBasePrompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新自定义prompt失败: %v", err)})
		return
	}

	// 如果trader在内存中，更新其custom prompt和override设置
	trader, err := s.traderManager.GetTrader(traderID)
	if err == nil {
		trader.SetCustomPrompt(req.CustomPrompt)
		trader.SetOverrideBasePrompt(req.OverrideBasePrompt)
		log.Printf("✓ 已更新交易员 %s 的自定义prompt (覆盖基础=%v)", trader.GetName(), req.OverrideBasePrompt)
	}

	c.JSON(http.StatusOK, gin.H{"message": "自定义prompt已更新"})
}

// handleGetModelConfigs 获取AI模型配置
func (s *Server) handleGetModelConfigs(c *gin.Context) {
	userID := c.GetString("user_id")
	log.Printf("🔍 查询用户 %s 的AI模型配置", userID)
	models, err := s.database.GetAIModels(userID)
	if err != nil {
		log.Printf("❌ 获取AI模型配置失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取AI模型配置失败: %v", err)})
		return
	}
	log.Printf("✅ 找到 %d 个AI模型配置", len(models))

	// 转换为安全的响应结构，移除敏感信息
	safeModels := make([]SafeModelConfig, len(models))
	for i, model := range models {
		safeModels[i] = SafeModelConfig{
			ID:              model.ID,
			Name:            model.Name,
			Provider:        model.Provider,
			Enabled:         model.Enabled,
			APIKey:          model.APIKey,
			CustomAPIURL:    model.CustomAPIURL,
			CustomModelName: model.CustomModelName,
		}
	}

	c.JSON(http.StatusOK, safeModels)
}

// handleUpdateModelConfigs 更新AI模型配置（仅支持加密数据）
func (s *Server) handleUpdateModelConfigs(c *gin.Context) {
	userID := c.GetString("user_id")

	// 读取原始请求体
	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	// 解析加密的 payload
	var encryptedPayload crypto.EncryptedPayload
	if err := json.Unmarshal(bodyBytes, &encryptedPayload); err != nil {
		log.Printf("❌ 解析加密载荷失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误，必须使用加密传输"})
		return
	}

	// 验证是否为加密数据
	if encryptedPayload.WrappedKey == "" {
		log.Printf("❌ 检测到非加密请求 (UserID: %s)", userID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "此接口仅支持加密传输，请使用加密客户端",
			"code":    "ENCRYPTION_REQUIRED",
			"message": "Encrypted transmission is required for security reasons",
		})
		return
	}

	// 解密数据
	decrypted, err := s.cryptoHandler.cryptoService.DecryptSensitiveData(&encryptedPayload)
	if err != nil {
		log.Printf("❌ 解密模型配置失败 (UserID: %s): %v", userID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "解密数据失败"})
		return
	}

	// 解析解密后的数据
	var req UpdateModelConfigRequest
	if err := json.Unmarshal([]byte(decrypted), &req); err != nil {
		log.Printf("❌ 解析解密数据失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析解密数据失败"})
		return
	}
	log.Printf("🔓 已解密模型配置数据 (UserID: %s)", userID)

	// 更新每个模型的配置
	for modelID, modelData := range req.Models {
		err := s.database.UpdateAIModel(userID, modelID, modelData.Enabled, modelData.APIKey, modelData.CustomAPIURL, modelData.CustomModelName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新模型 %s 失败: %v", modelID, err)})
			return
		}
	}

	// 重新加载该用户的所有交易员，使新配置立即生效
	err = s.traderManager.LoadUserTraders(s.database, userID)
	if err != nil {
		log.Printf("⚠️ 重新加载用户交易员到内存失败: %v", err)
		// 这里不返回错误，因为模型配置已经成功更新到数据库
	}

	log.Printf("✓ AI模型配置已更新: %+v", SanitizeModelConfigForLog(req.Models))
	c.JSON(http.StatusOK, gin.H{"message": "模型配置已更新"})
}

// handleGetExchangeConfigs 获取交易所配置
func (s *Server) handleGetExchangeConfigs(c *gin.Context) {
	userID := c.GetString("user_id")
	log.Printf("🔍 查询用户 %s 的交易所配置", userID)
	exchanges, err := s.database.GetExchanges(userID)
	if err != nil {
		log.Printf("❌ 获取交易所配置失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取交易所配置失败: %v", err)})
		return
	}
	log.Printf("✅ 找到 %d 个交易所配置", len(exchanges))

	// 转换为安全的响应结构，移除敏感信息
	safeExchanges := make([]SafeExchangeConfig, len(exchanges))
	for i, exchange := range exchanges {
		safeExchanges[i] = SafeExchangeConfig{
			ID:                    exchange.ID,
			Name:                  exchange.Name,
			Type:                  exchange.Type,
			Enabled:               exchange.Enabled,
			APIKey:                exchange.APIKey,
			SecretKey:             exchange.SecretKey,
			Testnet:               exchange.Testnet,
			HyperliquidWalletAddr: exchange.HyperliquidWalletAddr,
			AsterUser:             exchange.AsterUser,
			AsterSigner:           exchange.AsterSigner,
		}
	}

	c.JSON(http.StatusOK, safeExchanges)
}

// handleUpdateExchangeConfigs 更新交易所配置（仅支持加密数据）
func (s *Server) handleUpdateExchangeConfigs(c *gin.Context) {
	userID := c.GetString("user_id")

	// 读取原始请求体
	bodyBytes, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取请求体失败"})
		return
	}

	// 解析加密的 payload
	var encryptedPayload crypto.EncryptedPayload
	if err := json.Unmarshal(bodyBytes, &encryptedPayload); err != nil {
		log.Printf("❌ 解析加密载荷失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误，必须使用加密传输"})
		return
	}

	// 验证是否为加密数据
	if encryptedPayload.WrappedKey == "" {
		log.Printf("❌ 检测到非加密请求 (UserID: %s)", userID)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "此接口仅支持加密传输，请使用加密客户端",
			"code":    "ENCRYPTION_REQUIRED",
			"message": "Encrypted transmission is required for security reasons",
		})
		return
	}

	// 解密数据
	decrypted, err := s.cryptoHandler.cryptoService.DecryptSensitiveData(&encryptedPayload)
	if err != nil {
		log.Printf("❌ 解密交易所配置失败 (UserID: %s): %v", userID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "解密数据失败"})
		return
	}

	// 解析解密后的数据
	var req UpdateExchangeConfigRequest
	if err := json.Unmarshal([]byte(decrypted), &req); err != nil {
		log.Printf("❌ 解析解密数据失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "解析解密数据失败"})
		return
	}
	log.Printf("🔓 已解密交易所配置数据 (UserID: %s)", userID)

	// 更新每个交易所的配置
	for exchangeID, exchangeData := range req.Exchanges {
		err := s.database.UpdateExchange(userID, exchangeID, exchangeData.Enabled, exchangeData.APIKey, exchangeData.SecretKey, exchangeData.Testnet, exchangeData.HyperliquidWalletAddr, exchangeData.AsterUser, exchangeData.AsterSigner, exchangeData.AsterPrivateKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新交易所 %s 失败: %v", exchangeID, err)})
			return
		}
	}

	// 重新加载该用户的所有交易员，使新配置立即生效
	err = s.traderManager.LoadUserTraders(s.database, userID)
	if err != nil {
		log.Printf("⚠️ 重新加载用户交易员到内存失败: %v", err)
		// 这里不返回错误，因为交易所配置已经成功更新到数据库
	}

	log.Printf("✓ 交易所配置已更新: %+v", SanitizeExchangeConfigForLog(req.Exchanges))
	c.JSON(http.StatusOK, gin.H{"message": "交易所配置已更新"})
}

// handleGetUserSignalSource 获取用户信号源配置
func (s *Server) handleGetUserSignalSource(c *gin.Context) {
	userID := c.GetString("user_id")
	source, err := s.database.GetUserSignalSource(userID)
	if err != nil {
		// 如果配置不存在，返回空配置而不是404错误
		c.JSON(http.StatusOK, gin.H{
			"coin_pool_url": "",
			"oi_top_url":    "",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"coin_pool_url": source.CoinPoolURL,
		"oi_top_url":    source.OITopURL,
	})
}

// handleSaveUserSignalSource 保存用户信号源配置
func (s *Server) handleSaveUserSignalSource(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		CoinPoolURL string `json:"coin_pool_url"`
		OITopURL    string `json:"oi_top_url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.database.CreateUserSignalSource(userID, req.CoinPoolURL, req.OITopURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("保存用户信号源配置失败: %v", err)})
		return
	}

	log.Printf("✓ 用户信号源配置已保存: user=%s, coin_pool=%s, oi_top=%s", userID, req.CoinPoolURL, req.OITopURL)
	c.JSON(http.StatusOK, gin.H{"message": "用户信号源配置已保存"})
}

// handleTraderList trader列表
func (s *Server) handleTraderList(c *gin.Context) {
	userID := c.GetString("user_id")
	traders, err := s.database.GetTraders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取交易员列表失败: %v", err)})
		return
	}

	result := make([]map[string]interface{}, 0, len(traders))
	for _, trader := range traders {
		// 获取实时运行状态
		isRunning := trader.IsRunning
		if at, err := s.traderManager.GetTrader(trader.ID); err == nil {
			status := at.GetStatus()
			if running, ok := status["is_running"].(bool); ok {
				isRunning = running
			}
		}

		// 返回完整的 AIModelID（如 "admin_deepseek"），不要截断
		// 前端需要完整 ID 来验证模型是否存在（与 handleGetTraderConfig 保持一致）
		result = append(result, map[string]interface{}{
			"trader_id":              trader.ID,
			"trader_name":            trader.Name,
			"ai_model":               trader.AIModelID, // 使用完整 ID
			"exchange_id":            trader.ExchangeID,
			"is_running":             isRunning,
			"initial_balance":        trader.InitialBalance,
			"system_prompt_template": trader.SystemPromptTemplate,
		})
	}

	c.JSON(http.StatusOK, result)
}

// handleGetTraderConfig 获取交易员详细配置
func (s *Server) handleGetTraderConfig(c *gin.Context) {
	userID := c.GetString("user_id")
	traderID := c.Param("id")

	if traderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "交易员ID不能为空"})
		return
	}

	traderConfig, _, _, err := s.database.GetTraderConfig(userID, traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("获取交易员配置失败: %v", err)})
		return
	}

	// 获取实时运行状态
	isRunning := traderConfig.IsRunning
	if at, err := s.traderManager.GetTrader(traderID); err == nil {
		status := at.GetStatus()
		if running, ok := status["is_running"].(bool); ok {
			isRunning = running
		}
	}

	// 返回完整的模型ID，不做转换，保持与前端模型列表一致
	aiModelID := traderConfig.AIModelID

	result := map[string]interface{}{
		"trader_id":              traderConfig.ID,
		"trader_name":            traderConfig.Name,
		"ai_model":               aiModelID,
		"exchange_id":            traderConfig.ExchangeID,
		"initial_balance":        traderConfig.InitialBalance,
		"scan_interval_minutes":  traderConfig.ScanIntervalMinutes,
		"btc_eth_leverage":       traderConfig.BTCETHLeverage,
		"altcoin_leverage":       traderConfig.AltcoinLeverage,
		"trading_symbols":        traderConfig.TradingSymbols,
		"custom_prompt":          traderConfig.CustomPrompt,
		"override_base_prompt":   traderConfig.OverrideBasePrompt,
		"system_prompt_template": traderConfig.SystemPromptTemplate,
		"is_cross_margin":        traderConfig.IsCrossMargin,
		"use_coin_pool":          traderConfig.UseCoinPool,
		"use_oi_top":             traderConfig.UseOITop,
		"is_running":             isRunning,
	}

	c.JSON(http.StatusOK, result)
}

// handleStatus 系统状态
func (s *Server) handleStatus(c *gin.Context) {
	_, traderID, err := s.getTraderFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	status := trader.GetStatus()
	c.JSON(http.StatusOK, status)
}

// handleAccount 账户信息
func (s *Server) handleAccount(c *gin.Context) {
	_, traderID, err := s.getTraderFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	log.Printf("📊 收到账户信息请求 [%s]", trader.GetName())
	account, err := trader.GetAccountInfo()
	if err != nil {
		log.Printf("❌ 获取账户信息失败 [%s]: %v", trader.GetName(), err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取账户信息失败: %v", err),
		})
		return
	}

	log.Printf("✓ 返回账户信息 [%s]: 净值=%.2f, 可用=%.2f, 盈亏=%.2f (%.2f%%)",
		trader.GetName(),
		account["total_equity"],
		account["available_balance"],
		account["total_pnl"],
		account["total_pnl_pct"])
	c.JSON(http.StatusOK, account)
}

// handlePositions 持仓列表
func (s *Server) handlePositions(c *gin.Context) {
	_, traderID, err := s.getTraderFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	positions, err := trader.GetPositions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取持仓列表失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, positions)
}

// handleDecisions 决策日志列表
func (s *Server) handleDecisions(c *gin.Context) {
	_, traderID, err := s.getTraderFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 获取所有历史决策记录（无限制）
	records, err := trader.GetDecisionLogger().GetLatestRecords(10000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取决策日志失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, records)
}

// handleLatestDecisions 最新决策日志（最近5条，最新的在前）
func (s *Server) handleLatestDecisions(c *gin.Context) {
	userID := c.GetString("user_id")
	_, traderID, err := s.getTraderFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	records, err := trader.GetDecisionLogger().GetLatestRecords(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取决策日志失败: %v", err),
		})
		return
	}

	var copyRows []copyTradeAIDecisionRow
	traderCfg, _, exchangeCfg, cfgErr := s.database.GetTraderConfig(userID, traderID)
	if cfgErr == nil && traderCfg != nil && exchangeCfg != nil && exchangeCfg.ID == "binance" {
		copyRows, _ = s.getLatestCopyTradeAIDecisions(userID, limit)
	}

	merged := mergeLatestDecisionResponses(records, copyRows, limit)
	c.JSON(http.StatusOK, merged)
}

// handleStatistics 统计信息
func (s *Server) handleStatistics(c *gin.Context) {
	_, traderID, err := s.getTraderFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	stats, err := trader.GetDecisionLogger().GetStatistics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取统计信息失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleCompetition 竞赛总览（对比所有trader）
func (s *Server) handleCompetition(c *gin.Context) {
	userID := c.GetString("user_id")

	// 确保用户的交易员已加载到内存中
	err := s.traderManager.LoadUserTraders(s.database, userID)
	if err != nil {
		log.Printf("⚠️ 加载用户 %s 的交易员失败: %v", userID, err)
	}

	competition, err := s.traderManager.GetCompetitionData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取竞赛数据失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, competition)
}

// handleEquityHistory 收益率历史数据
func (s *Server) handleEquityHistory(c *gin.Context) {
	_, traderID, err := s.getTraderFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 获取尽可能多的历史数据（几天的数据）
	// 每3分钟一个周期：10000条 = 约20天的数据
	records, err := trader.GetDecisionLogger().GetLatestRecords(10000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取历史数据失败: %v", err),
		})
		return
	}

	// 构建收益率历史数据点
	type EquityPoint struct {
		Timestamp        string  `json:"timestamp"`
		TotalEquity      float64 `json:"total_equity"`      // 账户净值（wallet + unrealized）
		AvailableBalance float64 `json:"available_balance"` // 可用余额
		TotalPnL         float64 `json:"total_pnl"`         // 总盈亏（相对初始余额）
		TotalPnLPct      float64 `json:"total_pnl_pct"`     // 总盈亏百分比
		PositionCount    int     `json:"position_count"`    // 持仓数量
		MarginUsedPct    float64 `json:"margin_used_pct"`   // 保证金使用率
		CycleNumber      int     `json:"cycle_number"`
	}

	// 从AutoTrader获取当前初始余额（用作旧数据的fallback）
	base := 0.0
	if status := trader.GetStatus(); status != nil {
		if ib, ok := status["initial_balance"].(float64); ok && ib > 0 {
			base = ib
		}
	}

	// 如果还是无法获取，返回错误
	if base == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "无法获取初始余额",
		})
		return
	}

	var history []EquityPoint
	for _, record := range records {
		// TotalBalance字段实际存储的是TotalEquity
		// totalEquity := record.AccountState.TotalBalance
		// TotalUnrealizedProfit字段实际存储的是TotalPnL（相对初始余额）
		// totalPnL := record.AccountState.TotalUnrealizedProfit
		walletBalance := record.AccountState.TotalBalance
		unrealizedPnL := record.AccountState.TotalUnrealizedProfit
		totalEquity := walletBalance + unrealizedPnL

		// 🔄 使用历史记录中保存的initial_balance（如果有）
		// 这样可以保持历史PNL%的准确性，即使用户后来更新了initial_balance
		if record.AccountState.InitialBalance > 0 {
			base = record.AccountState.InitialBalance
		}

		totalPnL := totalEquity - base
		// 计算盈亏百分比
		totalPnLPct := 0.0
		if base > 0 {
			totalPnLPct = (totalPnL / base) * 100
		}

		history = append(history, EquityPoint{
			Timestamp:        record.Timestamp.Format("2006-01-02 15:04:05"),
			TotalEquity:      totalEquity,
			AvailableBalance: record.AccountState.AvailableBalance,
			TotalPnL:         totalPnL,
			TotalPnLPct:      totalPnLPct,
			PositionCount:    record.AccountState.PositionCount,
			MarginUsedPct:    record.AccountState.MarginUsedPct,
			CycleNumber:      record.CycleNumber,
		})
	}

	c.JSON(http.StatusOK, history)
}

// handlePerformance AI历史表现分析（用于展示AI学习和反思）
func (s *Server) handlePerformance(c *gin.Context) {
	_, traderID, err := s.getTraderFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// 分析最近100个周期的交易表现（避免长期持仓的交易记录丢失）
	// 假设每3分钟一个周期，100个周期 = 5小时，足够覆盖大部分交易
	performance, err := trader.GetDecisionLogger().AnalyzePerformance(100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("分析历史表现失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, performance)
}

// authMiddleware JWT认证中间件
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少Authorization头"})
			c.Abort()
			return
		}

		// 检查Bearer token格式
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Authorization格式"})
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// 黑名单检查
		if auth.IsTokenBlacklisted(tokenString) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token已失效，请重新登录"})
			c.Abort()
			return
		}

		// 验证JWT token
		claims, err := auth.ValidateJWT(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token: " + err.Error()})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文中
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		role := claims.Role
		if role == "" {
			role = config.UserRoleUser
		}
		c.Set("role", role)
		plan := claims.Plan
		if plan == "" {
			plan = config.PlanStandard
		}
		c.Set("plan", plan)
		c.Next()
	}
}

// handleLogout 将当前token加入黑名单
func (s *Server) handleLogout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少Authorization头"})
		return
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的Authorization格式"})
		return
	}
	tokenString := parts[1]
	claims, err := auth.ValidateJWT(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token"})
		return
	}
	var exp time.Time
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Time
	} else {
		exp = time.Now().Add(24 * time.Hour)
	}
	auth.BlacklistToken(tokenString, exp)
	c.JSON(http.StatusOK, gin.H{"message": "已登出"})
}

// handleRegister 处理用户注册请求
func (s *Server) handleRegister(c *gin.Context) {
	regEnabled := true
	if regStr, err := s.database.GetSystemConfig("registration_enabled"); err == nil {
		regEnabled = strings.ToLower(regStr) != "false"
	}
	if !regEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "注册已关闭"})
		return
	}

	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		BetaCode string `json:"beta_code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 检查是否开启了内测模式
	betaModeStr, _ := s.database.GetSystemConfig("beta_mode")
	if betaModeStr == "true" {
		// 内测模式下必须提供有效的内测码
		if req.BetaCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "内测期间，注册需要提供内测码"})
			return
		}

		// 验证内测码
		isValid, err := s.database.ValidateBetaCode(req.BetaCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "验证内测码失败"})
			return
		}
		if !isValid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "内测码无效或已被使用"})
			return
		}
	}

	// 检查邮箱是否已存在
	existingUser, err := s.database.GetUserByEmail(req.Email)
	if err == nil {
		// 如果用户未完成OTP验证，允许重新获取OTP（支持中断后恢复注册）
		if !existingUser.OTPVerified {
			qrCodeURL := auth.GetOTPQRCodeURL(existingUser.OTPSecret, req.Email)
			c.JSON(http.StatusOK, gin.H{
				"user_id":     existingUser.ID,
				"email":       req.Email,
				"otp_secret":  existingUser.OTPSecret,
				"qr_code_url": qrCodeURL,
				"message":     "检测到未完成的注册，请继续完成OTP设置",
			})
			return
		}
		// 用户已完成验证，拒绝重复注册
		c.JSON(http.StatusConflict, gin.H{"error": "邮箱已被注册"})
		return
	}

	// 生成密码哈希
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败"})
		return
	}

	// 生成OTP密钥
	otpSecret, err := auth.GenerateOTPSecret()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "OTP密钥生成失败"})
		return
	}

	// 创建用户（未验证OTP状态）
	userID := uuid.New().String()
	user := &config.User{
		ID:           userID,
		Email:        req.Email,
		PasswordHash: passwordHash,
		OTPSecret:    otpSecret,
		OTPVerified:  false,
	}

	err = s.database.CreateUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败: " + err.Error()})
		return
	}

	// 如果是内测模式，标记内测码为已使用
	betaModeStr2, _ := s.database.GetSystemConfig("beta_mode")
	if betaModeStr2 == "true" && req.BetaCode != "" {
		err := s.database.UseBetaCode(req.BetaCode, req.Email)
		if err != nil {
			log.Printf("⚠️ 标记内测码为已使用失败: %v", err)
			// 这里不返回错误，因为用户已经创建成功
		} else {
			log.Printf("✓ 内测码 %s 已被用户 %s 使用", req.BetaCode, req.Email)
		}
	}

	// 返回OTP设置信息
	qrCodeURL := auth.GetOTPQRCodeURL(otpSecret, req.Email)
	c.JSON(http.StatusOK, gin.H{
		"user_id":     userID,
		"email":       req.Email,
		"otp_secret":  otpSecret,
		"qr_code_url": qrCodeURL,
		"message":     "请使用Google Authenticator扫描二维码并验证OTP",
	})
}

// handleCompleteRegistration 完成注册（验证OTP）
func (s *Server) handleCompleteRegistration(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id" binding:"required"`
		OTPCode string `json:"otp_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户信息
	user, err := s.database.GetUserByID(req.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 验证OTP
	if !auth.VerifyOTP(user.OTPSecret, req.OTPCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OTP验证码错误"})
		return
	}

	// 更新用户OTP验证状态
	err = s.database.UpdateUserOTPVerified(req.UserID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户状态失败"})
		return
	}

	// 生成JWT token
	role := user.Role
	if role == "" {
		role = config.UserRoleUser
	}
	plan := user.Plan
	if plan == "" {
		plan = config.PlanStandard
	}
	token, err := auth.GenerateJWT(user.ID, user.Email, role, plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	// 初始化用户的默认模型和交易所配置
	err = s.initUserDefaultConfigs(user.ID)
	if err != nil {
		log.Printf("初始化用户默认配置失败: %v", err)
	}

	c.JSON(http.StatusOK, s.userAuthPayload(user, token, "注册完成"))
}

// handleLogin 处理用户登录请求
func (s *Server) handleLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户信息
	user, err := s.database.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}

	// 验证密码
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}

	// 管理员账号跳过 OTP，直接签发 token
	if user.Role == config.UserRoleAdmin {
		plan := user.Plan
		if plan == "" {
			plan = config.PlanVIP
		}
		token, err := auth.GenerateJWT(user.ID, user.Email, user.Role, plan)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
			return
		}
		c.JSON(http.StatusOK, s.userAuthPayload(user, token, "登录成功"))
		return
	}

	// 检查OTP是否已验证
	if !user.OTPVerified {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":              "账户未完成OTP设置",
			"user_id":            user.ID,
			"requires_otp_setup": true,
		})
		return
	}

	// 返回需要OTP验证的状态
	c.JSON(http.StatusOK, gin.H{
		"user_id":      user.ID,
		"email":        user.Email,
		"message":      "请输入Google Authenticator验证码",
		"requires_otp": true,
	})
}

// handleVerifyOTP 验证OTP并完成登录
func (s *Server) handleVerifyOTP(c *gin.Context) {
	var req struct {
		UserID  string `json:"user_id" binding:"required"`
		OTPCode string `json:"otp_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户信息
	user, err := s.database.GetUserByID(req.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 验证OTP
	if !auth.VerifyOTP(user.OTPSecret, req.OTPCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "验证码错误"})
		return
	}

	role := user.Role
	if role == "" {
		role = config.UserRoleUser
	}
	plan := user.Plan
	if plan == "" {
		plan = config.PlanStandard
	}
	token, err := auth.GenerateJWT(user.ID, user.Email, role, plan)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	c.JSON(http.StatusOK, s.userAuthPayload(user, token, "登录成功"))
}

// handleResetPassword 重置密码（通过邮箱 + OTP 验证）
func (s *Server) handleResetPassword(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		NewPassword string `json:"new_password" binding:"required,min=6"`
		OTPCode     string `json:"otp_code" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 查询用户
	user, err := s.database.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "邮箱不存在"})
		return
	}

	// 验证 OTP
	if !auth.VerifyOTP(user.OTPSecret, req.OTPCode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google Authenticator 验证码错误"})
		return
	}

	// 生成新密码哈希
	newPasswordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码处理失败"})
		return
	}

	// 更新密码
	err = s.database.UpdateUserPassword(user.ID, newPasswordHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码更新失败"})
		return
	}

	log.Printf("✓ 用户 %s 密码已重置", user.Email)
	c.JSON(http.StatusOK, gin.H{"message": "密码重置成功，请使用新密码登录"})
}

// initUserDefaultConfigs 为新用户初始化默认的模型和交易所配置
func (s *Server) initUserDefaultConfigs(userID string) error {
	// 注释掉自动创建默认配置，让用户手动添加
	// 这样新用户注册后不会自动有配置项
	log.Printf("用户 %s 注册完成，等待手动配置AI模型和交易所", userID)
	return nil
}

// handleGetSupportedModels 获取系统支持的AI模型列表
func (s *Server) handleGetSupportedModels(c *gin.Context) {
	// 返回系统支持的AI模型（从default用户获取）
	models, err := s.database.GetAIModels("default")
	if err != nil {
		log.Printf("❌ 获取支持的AI模型失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取支持的AI模型失败"})
		return
	}

	c.JSON(http.StatusOK, models)
}

// handleGetSupportedExchanges 获取系统支持的交易所列表
func (s *Server) handleGetSupportedExchanges(c *gin.Context) {
	// 返回系统支持的交易所（从default用户获取）
	exchanges, err := s.database.GetExchanges("default")
	if err != nil {
		log.Printf("❌ 获取支持的交易所失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取支持的交易所失败"})
		return
	}

	// 转换为安全的响应结构，移除敏感信息
	safeExchanges := make([]SafeExchangeConfig, len(exchanges))
	for i, exchange := range exchanges {
		safeExchanges[i] = SafeExchangeConfig{
			ID:                    exchange.ID,
			Name:                  exchange.Name,
			Type:                  exchange.Type,
			Enabled:               exchange.Enabled,
			APIKey:                exchange.APIKey,
			SecretKey:             exchange.SecretKey,
			Testnet:               exchange.Testnet,
			HyperliquidWalletAddr: "", // 默认配置不包含钱包地址
			AsterUser:             "", // 默认配置不包含用户信息
			AsterSigner:           "",
		}
	}

	c.JSON(http.StatusOK, safeExchanges)
}

// Start 启动服务器
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("🌐 API服务器启动在 http://localhost%s", addr)
	log.Printf("📊 API文档:")
	log.Printf("  • GET  /api/health           - 健康检查")
	log.Printf("  • GET  /api/traders          - 公开的AI交易员排行榜前50名（无需认证）")
	log.Printf("  • GET  /api/competition      - 公开的竞赛数据（无需认证）")
	log.Printf("  • GET  /api/top-traders      - 前5名交易员数据（无需认证，表现对比用）")
	log.Printf("  • GET  /api/equity-history?trader_id=xxx - 公开的收益率历史数据（无需认证，竞赛用）")
	log.Printf("  • GET  /api/equity-history-batch?trader_ids=a,b,c - 批量获取历史数据（无需认证，表现对比优化）")
	log.Printf("  • GET  /api/traders/:id/public-config - 公开的交易员配置（无需认证，不含敏感信息）")
	log.Printf("  • POST /api/traders          - 创建新的AI交易员")
	log.Printf("  • DELETE /api/traders/:id    - 删除AI交易员")
	log.Printf("  • POST /api/traders/:id/start - 启动AI交易员")
	log.Printf("  • POST /api/traders/:id/stop  - 停止AI交易员")
	log.Printf("  • GET  /api/models           - 获取AI模型配置")
	log.Printf("  • PUT  /api/models           - 更新AI模型配置")
	log.Printf("  • GET  /api/exchanges        - 获取交易所配置")
	log.Printf("  • PUT  /api/exchanges        - 更新交易所配置")
	log.Printf("  • GET  /api/status?trader_id=xxx     - 指定trader的系统状态")
	log.Printf("  • GET  /api/account?trader_id=xxx    - 指定trader的账户信息")
	log.Printf("  • GET  /api/positions?trader_id=xxx  - 指定trader的持仓列表")
	log.Printf("  • GET  /api/decisions?trader_id=xxx  - 指定trader的决策日志")
	log.Printf("  • GET  /api/decisions/latest?trader_id=xxx - 指定trader的最新决策")
	log.Printf("  • GET  /api/statistics?trader_id=xxx - 指定trader的统计信息")
	log.Printf("  • GET  /api/performance?trader_id=xxx - 指定trader的AI学习表现分析")
	log.Println()

	// 启动自动跟单定时器（每5分钟）
	go s.startAutoFollowTimer()

	// 创建 http.Server 以支持 graceful shutdown
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	return s.httpServer.ListenAndServe()
}

// Shutdown 优雅关闭 API 服务器
func (s *Server) Shutdown() error {
	if s.httpServer == nil {
		return nil
	}

	// 设置 5 秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// startAutoFollowTimer 启动自动跟单定时器
func (s *Server) startAutoFollowTimer() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// 启动后首次执行：不应用30分钟过期检查，处理停机期间遗漏的带单操作
	log.Println("⏰ 自动跟单定时器已启动（每5分钟）")
	log.Println("📋 首次启动：跳过30分钟过期检查，处理停机期间遗漏的带单")
	s.autoFollow(true)

	for range ticker.C {
		s.autoFollow(false)
	}
}

// autoFollow 自动跟单：读取开启了auto_follow的配置，调用AI分析后执行，并写入监控日志
// isFirstRun: 首次启动时不应用30分钟过期检查
func (s *Server) autoFollow(isFirstRun bool) {
	log.Println("⏰ 执行自动跟单检查...")
	globalCopyTradeRunState.markAutoFollowStarted()

	rows, err := s.database.DB().Query(`
		SELECT DISTINCT user_id FROM copy_trade_config WHERE enabled = 1 AND auto_follow = 1
	`)
	if err != nil {
		log.Printf("⚠️ 自动跟单查询失败: %v", err)
		globalCopyTradeRunState.markAutoFollowFinished()
		return
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var uid string
		rows.Scan(&uid)
		userIDs = append(userIDs, uid)
	}

	if len(userIDs) == 0 {
		globalCopyTradeRunState.markAutoFollowFinished()
		return
	}

	for _, userID := range userIDs {
		s.runAutoFollowForUser(userID, isFirstRun)
	}
	globalCopyTradeRunState.markAutoFollowFinished()
}

func (s *Server) runAutoFollowForUser(userID string, isFirstRun bool) {
	runID, err := s.startCopyTradeRun(userID, "auto")
	if err != nil {
		log.Printf("⚠️ [自动跟单] 创建 run 失败 user=%s: %v", userID, err)
		return
	}
	counters := copyTradeRunCounters{}

	defer func() {
		s.finishCopyTradeRun(runID, runStatusFromCounters(counters), counters, formatRunMessage(counters))
	}()

	exchanges, err := s.database.GetExchanges(userID)
	if err != nil || len(exchanges) == 0 {
		s.insertCopyTradeRunEvent(runID, userID, "", "", "", "no_exchange", "", "", "NONE", "未配置交易所", 0)
		counters.failed++
		return
	}
	var exchangeCfg *config.ExchangeConfig
	for _, ex := range exchanges {
		if ex.ID == "binance" && ex.Enabled {
			exchangeCfg = ex
			break
		}
	}
	if exchangeCfg == nil {
		s.insertCopyTradeRunEvent(runID, userID, "", "", "", "no_exchange", "", "", "NONE", "未启用币安交易所", 0)
		counters.failed++
		return
	}

	aiResult, aiErr := s.resolveCopyTradeAI(userID)
	if aiErr != nil {
		s.insertCopyTradeRunEvent(runID, userID, "", "", "", "no_ai", "", "", "NONE", aiErr.Error(), 0)
		counters.failed++
		return
	}
	aiCfg := aiResult.AICfg

	fTrader := trader.NewFuturesTrader(exchangeCfg.APIKey, exchangeCfg.SecretKey, userID, exchangeCfg.Testnet)

	cfgRows, err := s.database.DB().Query(`
		SELECT id, portfolio_id, nickname, max_copy_size, size_multiplier, copy_open_only, last_order_time
		FROM copy_trade_config WHERE user_id = ? AND enabled = 1 AND auto_follow = 1`, userID)
	if err != nil {
		return
	}
	defer cfgRows.Close()

	for cfgRows.Next() {
		var id int
		var portfolioID, nickname string
		var maxCopySize, sizeMultiplier float64
		var lastOrderTime int64
		var copyOpen int
		cfgRows.Scan(&id, &portfolioID, &nickname, &maxCopySize, &sizeMultiplier, &copyOpen, &lastOrderTime)
		copyOpenOnly := copyOpen != 0
		counters.tradersChecked++

		cached, err := refreshOrderCache(portfolioID, 50)
		if err != nil {
			s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, "", "fetch_failed", "", "", "NONE", err.Error(), 0)
			counters.failed++
			continue
		}
		if cached.Data == nil || len(cached.Data.List) == 0 {
			s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, "", "holding", "", "", "NONE", "带单员暂无订单记录", 0)
			counters.skipped++
			continue
		}

		type symOrder struct {
			Symbol       string
			Side         string
			PositionSide string
			ExecutedQty  float64
			AvgPrice     float64
			OrderTime    int64
		}
		var newOpens []symOrder
		var newCloses []copyCloseOrder
		seenOrderKey := make(map[string]bool)
		watermark := lastOrderTime
		bumpWatermark := func(orderTime int64) {
			if orderTime > watermark {
				watermark = orderTime
			}
		}

		autoFollowNow := time.Now()
		for _, o := range cached.Data.List {
			if o.OrderTime <= lastOrderTime {
				continue
			}
			if isLeadOrderStaleForAutoFollow(o.OrderTime, autoFollowNow) && !isFirstRun {
				bumpWatermark(o.OrderTime)
				ourSt := s.ourStatusForSymbol(userID, portfolioID, o.Symbol)
				s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, o.Symbol, "stale_skipped",
					o.Side, o.PositionSide, ourSt, "带单操作已超过30分钟，跳过跟单", o.OrderTime)
				counters.skipped++
				continue
			}
			if isLeadCloseOrder(o.PositionSide, o.Side) {
				if copyOpenOnly {
					bumpWatermark(o.OrderTime)
					continue
				}
				key := fmt.Sprintf("close_%s_%d_%s_%s", o.Symbol, o.OrderTime, o.PositionSide, o.Side)
				if seenOrderKey[key] {
					continue
				}
				seenOrderKey[key] = true
				ourSt := s.ourStatusForSymbol(userID, portfolioID, o.Symbol)
				s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, o.Symbol, "lead_close",
					o.Side, o.PositionSide, ourSt, leadActionDisplay(o.PositionSide, o.Side), o.OrderTime)
				newCloses = append(newCloses, copyCloseOrder{
					Symbol: o.Symbol, Side: o.Side, PositionSide: o.PositionSide,
					ExecutedQty: o.ExecutedQty, AvgPrice: o.AvgPrice, OrderTime: o.OrderTime,
				})
				continue
			}
			if !isLeadOpenOrder(o.PositionSide, o.Side) {
				continue
			}
			key := fmt.Sprintf("open_%s_%d_%s_%s", o.Symbol, o.OrderTime, o.PositionSide, o.Side)
			if seenOrderKey[key] {
				continue
			}
			seenOrderKey[key] = true
			newOpens = append(newOpens, symOrder{
				Symbol: o.Symbol, Side: o.Side, PositionSide: o.PositionSide,
				ExecutedQty: o.ExecutedQty, AvgPrice: o.AvgPrice, OrderTime: o.OrderTime,
			})
		}

		for _, co := range newCloses {
			res := s.processCopyTradeClose(fTrader, userID, portfolioID, nickname, runID, co, "跟单-自动监控")
			switch {
			case res.success:
				counters.closed++
				bumpWatermark(co.OrderTime)
				time.Sleep(500 * time.Millisecond)
			case res.failed:
				counters.failed++
			default:
				counters.skipped++
				bumpWatermark(co.OrderTime)
			}
		}

		if len(newOpens) == 0 {
			if len(newCloses) == 0 {
				s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, "", "holding", "", "", "NONE", "无新带单操作", 0)
				counters.skipped++
			}
			if watermark > lastOrderTime {
				s.database.DB().Exec("UPDATE copy_trade_config SET last_order_time=? WHERE id=?", watermark, id)
			}
			continue
		}

		for _, order := range newOpens {
			symbol := order.Symbol
			ourSt := s.ourStatusForSymbol(userID, portfolioID, symbol)

			s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, symbol, "lead_open",
				order.Side, order.PositionSide, ourSt, leadActionDisplay(order.PositionSide, order.Side), order.OrderTime)

			var count int
			s.database.DB().QueryRow("SELECT COUNT(*) FROM copy_trade_records WHERE user_id=? AND symbol=? AND lead_order_time=?",
				userID, symbol, order.OrderTime).Scan(&count)
			if count > 0 {
				s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, symbol, "already_copied",
					order.Side, order.PositionSide, ourSt, "该带单订单已跟过", order.OrderTime)
				counters.skipped++
				bumpWatermark(order.OrderTime)
				continue
			}

			qty := order.ExecutedQty * sizeMultiplier
			if maxCopySize > 0 && qty*order.AvgPrice > maxCopySize {
				qty = maxCopySize / order.AvgPrice
			}
			if qty < 0.001 {
				s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, symbol, "qty_too_small",
					order.Side, order.PositionSide, ourSt, fmt.Sprintf("计算数量过小: %.6f", qty), order.OrderTime)
				counters.skipped++
				continue
			}

			tradePosSide := order.PositionSide
			if tradePosSide == "BOTH" {
				tradePosSide = "SHORT"
			}
			dir := "卖出开空"
			if tradePosSide == "LONG" {
				dir = "买入开多"
			}

			mcpClient := mcp.New()
			if aiCfg.Provider == "deepseek" {
				mcpClient = mcp.NewDeepSeekClient()
			} else if aiCfg.Provider == "qwen" {
				mcpClient = mcp.NewQwenClient()
			}
			mcpClient.SetAPIKey(aiCfg.APIKey, aiCfg.CustomAPIURL, aiCfg.CustomModelName)

			accountCtx := fetchCopyTradeAIAccountContext(fTrader)
			marketSection := fetchCopyTradeAIMarketSection(symbol)
			prompt := buildCopyTradeRiskPrompt(accountCtx, marketSection, copyTradeAIPromptParams{
				Symbol: symbol, Direction: dir, LeadNickname: nickname,
				LeadPrice: order.AvgPrice, LeadQty: qty,
			})

			resp, aiErr := mcpClient.CallWithMessages("你是一个加密合约风控分析师。结合最新K线与账户资产输出JSON。", prompt)
			aiQty := qty
			aiFeasible := true
			aiReason := ""
			aiSuggestion := ""
			decisionJSON := ""
			actionTaken := "parse_failed"
			aiSuccess := false

			if aiErr != nil {
				actionTaken = "ai_error"
				aiFeasible = false
				aiReason = aiErr.Error()
			} else if resp == "" {
				actionTaken = "ai_error"
				aiFeasible = false
				aiReason = "AI 返回空响应"
			} else {
				cleaned := strings.TrimSpace(resp)
				if si := strings.Index(cleaned, "{"); si >= 0 {
					if e := strings.LastIndex(cleaned, "}"); e > si {
						decisionJSON = cleaned[si : e+1]
						cleaned = decisionJSON
					}
				}
				var parsedAI struct {
					Feasible       bool    `json:"feasible"`
					RecommendedQty float64 `json:"recommended_qty"`
					Reasoning      string  `json:"reasoning"`
					Suggestion     string  `json:"suggestion"`
				}
				if json.Unmarshal([]byte(cleaned), &parsedAI) == nil {
					aiFeasible = parsedAI.Feasible
					aiReason = parsedAI.Reasoning
					aiSuggestion = parsedAI.Suggestion
					if parsedAI.RecommendedQty > 0 {
						aiQty = parsedAI.RecommendedQty
					}
					if !aiFeasible {
						actionTaken = "ai_rejected"
					} else {
						actionTaken = "pending_open"
					}
				} else {
					aiFeasible = false
					aiReason = "AI 响应 JSON 解析失败"
				}
			}

			if !aiFeasible {
				s.insertCopyTradeAIDecision(copyTradeAIDecisionParams{
					UserID: userID, RunID: runID, PortfolioID: portfolioID, Nickname: nickname, Symbol: symbol,
					InputPrompt: prompt, AIResponseRaw: resp, DecisionJSON: decisionJSON,
					Feasible: aiFeasible, RecommendedQty: aiQty, Reasoning: aiReason, Suggestion: aiSuggestion,
					ActionTaken: actionTaken, Success: false,
					AITraderID: aiResult.AITraderID, AITraderName: aiResult.AITraderName,
				})
				s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, symbol, "ai_rejected",
					order.Side, tradePosSide, ourSt, aiReason, order.OrderTime)
				logger.NotifyTrade(logger.TradeNotifyParams{
					Source: "跟单-自动监控", Status: logger.TradeStatusAIReject, Action: "跟单开仓",
					Symbol: symbol, Side: order.Side, PositionSide: tradePosSide,
					TraderOrNickname: nickname, Reason: aiReason,
				})
				counters.skipped++
				continue
			}

			var tradeResult map[string]interface{}
			var tradeErr error
			if tradePosSide == "LONG" {
				tradeResult, tradeErr = fTrader.OpenLong(symbol, aiQty, 5)
			} else {
				tradeResult, tradeErr = fTrader.OpenShort(symbol, aiQty, 5)
			}

			if tradeErr != nil {
				s.insertCopyTradeAIDecision(copyTradeAIDecisionParams{
					UserID: userID, RunID: runID, PortfolioID: portfolioID, Nickname: nickname, Symbol: symbol,
					InputPrompt: prompt, AIResponseRaw: resp, DecisionJSON: decisionJSON,
					Feasible: true, RecommendedQty: aiQty, Reasoning: aiReason, Suggestion: aiSuggestion,
					ActionTaken: "open_failed", Success: false,
					AITraderID: aiResult.AITraderID, AITraderName: aiResult.AITraderName,
				})
				s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, symbol, "open_failed",
					order.Side, tradePosSide, ourSt, tradeErr.Error(), order.OrderTime)
				logger.NotifyTrade(logger.TradeNotifyParams{
					Source: "跟单-自动监控", Status: logger.TradeStatusFailed, Action: "跟单开仓",
					Symbol: symbol, Side: order.Side, PositionSide: tradePosSide, Qty: aiQty,
					TraderOrNickname: nickname, Error: tradeErr.Error(),
				})
				counters.failed++
				continue
			}

			actualQty := aiQty
			actualPrice := order.AvgPrice
			orderID := fmt.Sprintf("auto_%s_%d", nickname, order.OrderTime)
			if tradeResult != nil {
				if q, ok := tradeResult["executedQty"].(float64); ok && q > 0 {
					actualQty = q
				}
				if p, ok := tradeResult["avgPrice"].(float64); ok && p > 0 {
					actualPrice = p
				}
			}
			aiSuccess = true

			s.insertCopyTradeAIDecision(copyTradeAIDecisionParams{
				UserID: userID, RunID: runID, PortfolioID: portfolioID, Nickname: nickname, Symbol: symbol,
				InputPrompt: prompt, AIResponseRaw: resp, DecisionJSON: decisionJSON,
				Feasible: true, RecommendedQty: actualQty, Reasoning: aiReason, Suggestion: aiSuggestion,
				ActionTaken: "copied_open", Success: aiSuccess,
				AITraderID: aiResult.AITraderID, AITraderName: aiResult.AITraderName,
			})

			s.database.DB().Exec(`
				INSERT INTO copy_trade_records 
				(user_id, portfolio_id, nickname, order_id, symbol, side, position_side,
				 executed_qty, avg_price, total_pnl, status, lead_order_time)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 'OPEN', ?)`,
				userID, portfolioID, nickname, orderID, symbol,
				order.Side, tradePosSide, actualQty, actualPrice, order.OrderTime)

			s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, symbol, "copied_open",
				order.Side, tradePosSide, "OPEN",
				fmt.Sprintf("已跟单 %.4f张 @ $%.2f", actualQty, actualPrice), order.OrderTime)
			logger.NotifyTrade(logger.TradeNotifyParams{
				Source: "跟单-自动监控", Status: logger.TradeStatusSuccess, Action: "跟单开仓",
				Symbol: symbol, Side: order.Side, PositionSide: tradePosSide,
				Qty: actualQty, Price: actualPrice, TraderOrNickname: nickname,
			})
			counters.opened++
			bumpWatermark(order.OrderTime)
			log.Printf("  ✓ [自动跟单] %s %s %s %.4f张 @ $%.2f", nickname, symbol, tradePosSide, actualQty, actualPrice)
			time.Sleep(500 * time.Millisecond)
		}

		if watermark > lastOrderTime {
			s.database.DB().Exec("UPDATE copy_trade_config SET last_order_time=? WHERE id=?", watermark, id)
		}
	}

	if _, err := s.refreshCopyTradeUnrealizedPnL(userID, fTrader); err != nil {
		log.Printf("⚠️ [自动跟单] 刷新未实现盈亏失败: %v", err)
	}
}

// handleGetPromptTemplates 获取所有系统提示词模板列表
func (s *Server) handleGetPromptTemplates(c *gin.Context) {
	// 导入 decision 包
	templates := decision.GetAllPromptTemplates()

	// 转换为响应格式
	response := make([]map[string]interface{}, 0, len(templates))
	for _, tmpl := range templates {
		response = append(response, map[string]interface{}{
			"name": tmpl.Name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": response,
	})
}

// handleGetPromptTemplate 获取指定名称的提示词模板内容
func (s *Server) handleGetPromptTemplate(c *gin.Context) {
	templateName := c.Param("name")

	template, err := decision.GetPromptTemplate(templateName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("模板不存在: %s", templateName)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":    template.Name,
		"content": template.Content,
	})
}

// handlePublicTraderList 获取公开的交易员列表（无需认证）
func (s *Server) handlePublicTraderList(c *gin.Context) {
	// 从所有用户获取交易员信息
	competition, err := s.traderManager.GetCompetitionData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取交易员列表失败: %v", err),
		})
		return
	}

	// 获取traders数组
	tradersData, exists := competition["traders"]
	if !exists {
		c.JSON(http.StatusOK, []map[string]interface{}{})
		return
	}

	traders, ok := tradersData.([]map[string]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "交易员数据格式错误",
		})
		return
	}

	// 返回交易员基本信息，过滤敏感信息
	result := make([]map[string]interface{}, 0, len(traders))
	for _, trader := range traders {
		result = append(result, map[string]interface{}{
			"trader_id":              trader["trader_id"],
			"trader_name":            trader["trader_name"],
			"ai_model":               trader["ai_model"],
			"exchange":               trader["exchange"],
			"is_running":             trader["is_running"],
			"total_equity":           trader["total_equity"],
			"total_pnl":              trader["total_pnl"],
			"total_pnl_pct":          trader["total_pnl_pct"],
			"position_count":         trader["position_count"],
			"margin_used_pct":        trader["margin_used_pct"],
			"system_prompt_template": trader["system_prompt_template"],
		})
	}

	c.JSON(http.StatusOK, result)
}

// handlePublicCompetition 获取公开的竞赛数据（无需认证）
func (s *Server) handlePublicCompetition(c *gin.Context) {
	competition, err := s.traderManager.GetCompetitionData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取竞赛数据失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, competition)
}

// handleTopTraders 获取前5名交易员数据（无需认证，用于表现对比）
func (s *Server) handleTopTraders(c *gin.Context) {
	topTraders, err := s.traderManager.GetTopTradersData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取前10名交易员数据失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, topTraders)
}

// handleEquityHistoryBatch 批量获取多个交易员的收益率历史数据（无需认证，用于表现对比）
func (s *Server) handleEquityHistoryBatch(c *gin.Context) {
	var requestBody struct {
		TraderIDs []string `json:"trader_ids"`
	}

	// 尝试解析POST请求的JSON body
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		// 如果JSON解析失败，尝试从query参数获取（兼容GET请求）
		traderIDsParam := c.Query("trader_ids")
		if traderIDsParam == "" {
			// 如果没有指定trader_ids，则返回前5名的历史数据
			topTraders, err := s.traderManager.GetTopTradersData()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("获取前5名交易员失败: %v", err),
				})
				return
			}

			traders, ok := topTraders["traders"].([]map[string]interface{})
			if !ok {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "交易员数据格式错误"})
				return
			}

			// 提取trader IDs
			traderIDs := make([]string, 0, len(traders))
			for _, trader := range traders {
				if traderID, ok := trader["trader_id"].(string); ok {
					traderIDs = append(traderIDs, traderID)
				}
			}

			result := s.getEquityHistoryForTraders(traderIDs)
			c.JSON(http.StatusOK, result)
			return
		}

		// 解析逗号分隔的trader IDs
		requestBody.TraderIDs = strings.Split(traderIDsParam, ",")
		for i := range requestBody.TraderIDs {
			requestBody.TraderIDs[i] = strings.TrimSpace(requestBody.TraderIDs[i])
		}
	}

	// 限制最多20个交易员，防止请求过大
	if len(requestBody.TraderIDs) > 20 {
		requestBody.TraderIDs = requestBody.TraderIDs[:20]
	}

	result := s.getEquityHistoryForTraders(requestBody.TraderIDs)
	c.JSON(http.StatusOK, result)
}

// getEquityHistoryForTraders 获取多个交易员的历史数据
func (s *Server) getEquityHistoryForTraders(traderIDs []string) map[string]interface{} {
	result := make(map[string]interface{})
	histories := make(map[string]interface{})
	errors := make(map[string]string)

	for _, traderID := range traderIDs {
		if traderID == "" {
			continue
		}

		trader, err := s.traderManager.GetTrader(traderID)
		if err != nil {
			errors[traderID] = "交易员不存在"
			continue
		}

		// 获取历史数据（用于对比展示，限制数据量）
		records, err := trader.GetDecisionLogger().GetLatestRecords(500)
		if err != nil {
			errors[traderID] = fmt.Sprintf("获取历史数据失败: %v", err)
			continue
		}

		// 构建收益率历史数据
		history := make([]map[string]interface{}, 0, len(records))
		for _, record := range records {
			// 计算总权益（余额+未实现盈亏）
			totalEquity := record.AccountState.TotalBalance + record.AccountState.TotalUnrealizedProfit

			history = append(history, map[string]interface{}{
				"timestamp":    record.Timestamp,
				"total_equity": totalEquity,
				"total_pnl":    record.AccountState.TotalUnrealizedProfit,
				"balance":      record.AccountState.TotalBalance,
			})
		}

		histories[traderID] = history
	}

	result["histories"] = histories
	result["count"] = len(histories)
	if len(errors) > 0 {
		result["errors"] = errors
	}

	return result
}

// handleGetPublicTraderConfig 获取公开的交易员配置信息（无需认证，不包含敏感信息）
func (s *Server) handleGetPublicTraderConfig(c *gin.Context) {
	traderID := c.Param("id")
	if traderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "交易员ID不能为空"})
		return
	}

	trader, err := s.traderManager.GetTrader(traderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "交易员不存在"})
		return
	}

	// 获取交易员的状态信息
	status := trader.GetStatus()

	// 只返回公开的配置信息，不包含API密钥等敏感数据
	result := map[string]interface{}{
		"trader_id":   trader.GetID(),
		"trader_name": trader.GetName(),
		"ai_model":    trader.GetAIModel(),
		"exchange":    trader.GetExchange(),
		"is_running":  status["is_running"],
		"ai_provider": status["ai_provider"],
		"start_time":  status["start_time"],
	}

	c.JSON(http.StatusOK, result)
}

// reloadPromptTemplatesWithLog 重新加载提示词模板并记录日志
func (s *Server) reloadPromptTemplatesWithLog(templateName string) {
	if err := decision.ReloadPromptTemplates(); err != nil {
		log.Printf("⚠️  重新加载提示词模板失败: %v", err)
		return
	}

	if templateName == "" {
		log.Printf("✓ 已重新加载系统提示词模板 [当前使用: default (未指定，使用默认)]")
	} else {
		log.Printf("✓ 已重新加载系统提示词模板 [当前使用: %s]", templateName)
	}
}

// handleCopyTradingLeaderboard 从币安获取跟单交易员排行榜（失败时回退本地缓存）
func (s *Server) handleCopyTradingLeaderboard(c *gin.Context) {
	respondLeaderboard := func(data json.RawMessage, stale bool, warning string) {
		var parsed interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析排行榜数据失败"})
			return
		}
		resp := gin.H{
			"code":    "000000",
			"message": "success",
			"data":    parsed,
		}
		if stale {
			resp["stale"] = true
		}
		if warning != "" {
			resp["warning"] = warning
		}
		c.JSON(http.StatusOK, resp)
	}

	data, err := fetchBinanceLeaderboard()
	if err == nil {
		if werr := writeLeaderboardCache(data); werr != nil {
			log.Printf("⚠️ 写入排行榜缓存失败: %v", werr)
		}
		respondLeaderboard(data, false, "")
		return
	}

	log.Printf("⚠️ 币安跟单排行榜请求失败: %v", err)

	if cached, ts, cerr := readLeaderboardCache(); cerr == nil {
		age := time.Since(time.Unix(ts, 0)).Round(time.Minute)
		respondLeaderboard(cached, true,
			fmt.Sprintf("币安 API 暂不可用，已使用 %v 前的本地缓存（%v）", age, err))
		return
	}

	respondLeaderboard(emptyLeaderboardData(), true,
		"币安跟单 API 不可用（Docker/服务器可能受地区限制），且暂无本地缓存。可在能访问币安的环境先打开本页以生成缓存，或配置 HTTP_PROXY / BINANCE_BAPI_BASE_URL")
}

// handleCopyTradingOrders 获取币安跟单交易员最新操作记录
func (s *Server) handleCopyTradingOrders(c *gin.Context) {
	portfolioID := c.Query("portfolio_id")
	if portfolioID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 portfolio_id 参数"})
		return
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if pageSize <= 0 {
		pageSize = 10
	}

	stale := false
	warning := ""
	history, err := fetchBinanceLeadOrders(portfolioID, pageSize)
	if err != nil {
		log.Printf("⚠️ 币安跟单订单请求失败 portfolio=%s: %v", portfolioID, err)
		if cached, ok := getLeadOrdersCached(portfolioID, 7*24*time.Hour); ok {
			history = cached
			stale = true
			warning = "币安 API 暂不可用，已使用本地缓存"
		} else {
			c.JSON(http.StatusBadGateway, gin.H{"error": "请求币安API失败", "detail": err.Error()})
			return
		}
	} else if err := writeOrderCache(portfolioID, history); err != nil {
		log.Printf("⚠️ 写入跟单订单缓存失败: %v", err)
	}

	var data json.RawMessage
	if history.Data != nil {
		data, _ = json.Marshal(history.Data)
	}

	resp := gin.H{
		"code":    "000000",
		"message": "success",
		"data":    data,
	}
	if stale {
		resp["stale"] = true
		resp["warning"] = warning
	}
	c.JSON(http.StatusOK, resp)
}

// handleGetCopyTradeConfigs 获取跟单配置
func (s *Server) handleGetCopyTradeConfigs(c *gin.Context) {
	userID := c.GetString("user_id")
	rows, err := s.database.DB().Query(`
		SELECT id, portfolio_id, nickname, enabled, auto_follow, max_copy_size, size_multiplier, 
		       copy_open_only, last_order_time, created_at, updated_at 
		FROM copy_trade_config WHERE user_id = ? ORDER BY nickname`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	type ConfigResp struct {
		ID             int     `json:"id"`
		PortfolioID    string  `json:"portfolio_id"`
		Nickname       string  `json:"nickname"`
		Enabled        bool    `json:"enabled"`
		AutoFollow     bool    `json:"auto_follow"`
		MaxCopySize    float64 `json:"max_copy_size"`
		SizeMultiplier float64 `json:"size_multiplier"`
		CopyOpenOnly   bool    `json:"copy_open_only"`
		LastOrderTime  int64   `json:"last_order_time"`
	}

	var configs = make([]ConfigResp, 0)
	for rows.Next() {
		var cfg ConfigResp
		var enabled, copyOpenOnly, autoFollow int
		if err := rows.Scan(&cfg.ID, &cfg.PortfolioID, &cfg.Nickname, &enabled, &autoFollow,
			&cfg.MaxCopySize, &cfg.SizeMultiplier, &copyOpenOnly, &cfg.LastOrderTime,
			new(interface{}), new(interface{})); err != nil {
			continue
		}
		cfg.Enabled = enabled != 0
		cfg.CopyOpenOnly = copyOpenOnly != 0
		cfg.AutoFollow = autoFollow != 0
		configs = append(configs, cfg)
	}
	c.JSON(http.StatusOK, configs)
}

// handleUpdateCopyTradeConfig 更新跟单配置
func (s *Server) handleUpdateCopyTradeConfig(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		ID             *int    `json:"id"`
		PortfolioID    string  `json:"portfolio_id"`
		Nickname       string  `json:"nickname"`
		Enabled        bool    `json:"enabled"`
		AutoFollow     bool    `json:"auto_follow"`
		MaxCopySize    float64 `json:"max_copy_size"`
		SizeMultiplier float64 `json:"size_multiplier"`
		CopyOpenOnly   bool    `json:"copy_open_only"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	enabled := 0
	if req.Enabled {
		enabled = 1
	}
	copyOpenOnly := 0
	if req.CopyOpenOnly {
		copyOpenOnly = 1
	}
	autoFollow := 0
	if req.AutoFollow {
		autoFollow = 1
	}

	if req.ID != nil && *req.ID > 0 {
		// 更新
		_, err := s.database.DB().Exec(`
			UPDATE copy_trade_config SET enabled=?, auto_follow=?, max_copy_size=?, size_multiplier=?,
			copy_open_only=?, updated_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`,
			enabled, autoFollow, req.MaxCopySize, req.SizeMultiplier, copyOpenOnly, *req.ID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
			return
		}
	} else {
		// 插入
		_, err := s.database.DB().Exec(`
			INSERT OR REPLACE INTO copy_trade_config 
			(user_id, portfolio_id, nickname, enabled, auto_follow, max_copy_size, size_multiplier, copy_open_only)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, req.PortfolioID, req.Nickname, enabled, autoFollow, req.MaxCopySize, req.SizeMultiplier, copyOpenOnly)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleDeleteCopyTradeConfig 删除跟单配置
func (s *Server) handleDeleteCopyTradeConfig(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	_, err := s.database.DB().Exec("DELETE FROM copy_trade_config WHERE id=? AND user_id=?", id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleGetCopyTradeRecords 获取跟单记录
func (s *Server) handleGetCopyTradeRecords(c *gin.Context) {
	userID := c.GetString("user_id")
	rows, err := s.database.DB().Query(`
		SELECT id, portfolio_id, nickname, order_id, symbol, side, position_side,
		       executed_qty, avg_price, total_pnl, status, lead_order_time, copy_time, close_time,
		       IFNULL(close_price,0) as close_price, IFNULL(error_message,'') as error_message
		FROM copy_trade_records WHERE user_id = ? ORDER BY copy_time DESC LIMIT 100`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	var records = make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int
		var portfolioID, nickname, orderID, symbol, side, posSide, status, errorMessage string
		var qty, price, pnl, closePrice float64
		var leadTime int64
		var copyTime, closeTime interface{}
		if err := rows.Scan(&id, &portfolioID, &nickname, &orderID, &symbol, &side, &posSide,
			&qty, &price, &pnl, &status, &leadTime, &copyTime, &closeTime, &closePrice, &errorMessage); err != nil {
			continue
		}
		rec := map[string]interface{}{
			"id": id, "portfolio_id": portfolioID, "nickname": nickname,
			"order_id": orderID, "symbol": symbol, "side": side,
			"position_side": posSide, "executed_qty": qty, "avg_price": price,
			"total_pnl": pnl, "status": status, "lead_order_time": leadTime,
			"copy_time": copyTime, "close_time": closeTime,
			"close_price": closePrice, "error_message": errorMessage,
		}
		records = append(records, rec)
	}
	c.JSON(http.StatusOK, records)
}

// handleSyncCopyTrade 执行跟单同步（实际开仓）
func (s *Server) handleSyncCopyTrade(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ PANIC in handleSyncCopyTrade: %v", r)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("panic: %v", r)})
		}
	}()
	userID := c.GetString("user_id")

	// 获取用户交易所配置 - 查找binance
	exchanges, err := s.database.GetExchanges(userID)
	if err != nil || len(exchanges) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置交易所"})
		return
	}

	// 找到启用的binance交易所
	var exchangeCfg *config.ExchangeConfig
	for _, ex := range exchanges {
		if ex.ID == "binance" && ex.Enabled {
			exchangeCfg = ex
			break
		}
	}
	if exchangeCfg == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先启用币安交易所"})
		return
	}

	log.Printf("🔍 [跟单] Binance交易所已找到, APIKey=%s... testnet=%v", exchangeCfg.APIKey[:8], exchangeCfg.Testnet)

	// 读取跟单配置
	rows, err := s.database.DB().Query(`
		SELECT id, portfolio_id, nickname, enabled, max_copy_size, size_multiplier,
		       copy_open_only, last_order_time
		FROM copy_trade_config WHERE user_id = ? AND enabled = 1`, userID)
	if err != nil {
		log.Printf("❌ [跟单] 查询配置失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("查询失败: %v", err)})
		return
	}
	defer rows.Close()

	type cfgRow struct {
		ID             int
		PortfolioID    string
		Nickname       string
		MaxCopySize    float64
		SizeMultiplier float64
		CopyOpenOnly   bool
		LastOrderTime  int64
	}

	var configs []cfgRow
	for rows.Next() {
		var r cfgRow
		var enabled, copyOpenOnly int
		if err := rows.Scan(&r.ID, &r.PortfolioID, &r.Nickname, &enabled,
			&r.MaxCopySize, &r.SizeMultiplier, &copyOpenOnly, &r.LastOrderTime); err != nil {
			continue
		}
		r.CopyOpenOnly = copyOpenOnly != 0
		configs = append(configs, r)
	}

	if len(configs) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "没有启用的跟单配置", "copied": 0})
		return
	}

	// 创建币安交易器用于实际下单
	fTrader := trader.NewFuturesTrader(exchangeCfg.APIKey, exchangeCfg.SecretKey, userID, exchangeCfg.Testnet)

	runID, _ := s.startCopyTradeRun(userID, "manual")
	syncCounters := copyTradeRunCounters{}

	totalCopied := 0
	totalErrors := 0

	for _, cfg := range configs {
		log.Printf("🔍 [跟单] 处理配置: %s (ID=%d, portfolio=%s, last_time=%d)", cfg.Nickname, cfg.ID, cfg.PortfolioID, cfg.LastOrderTime)
		syncCounters.tradersChecked++

		cached, err := refreshOrderCache(cfg.PortfolioID, 50)
		if err != nil {
			log.Printf("⚠️  [%s] 拉取订单失败: %v", cfg.Nickname, err)
			s.insertCopyTradeRunEvent(runID, userID, cfg.PortfolioID, cfg.Nickname, "", "fetch_failed", "", "", "NONE", err.Error(), 0)
			totalErrors++
			continue
		}
		if cached.Data == nil {
			continue
		}

		latestTime := cfg.LastOrderTime

		type symbolOrder struct {
			Symbol       string
			Side         string
			PositionSide string
			ExecutedQty  float64
			AvgPrice     float64
			TotalPnl     float64
			OrderTime    int64
		}
		seenKey := make(map[string]bool)
		var allNewOrders []symbolOrder
		for _, order := range cached.Data.List {
			if order.OrderTime > latestTime {
				latestTime = order.OrderTime
			}
			if order.OrderTime <= cfg.LastOrderTime {
				continue
			}
			isOpen := isLeadOpenOrder(order.PositionSide, order.Side)
			isClose := isLeadCloseOrder(order.PositionSide, order.Side)
			if !isOpen && !isClose {
				continue
			}
			if cfg.CopyOpenOnly && !isOpen {
				continue // 仅跟开仓：不跟平仓
			}
			key := fmt.Sprintf("%s_%d_%s_%s", order.Symbol, order.OrderTime, order.PositionSide, order.Side)
			if seenKey[key] {
				continue
			}
			seenKey[key] = true
			allNewOrders = append(allNewOrders, symbolOrder{
				Symbol: order.Symbol, Side: order.Side, PositionSide: order.PositionSide,
				ExecutedQty: order.ExecutedQty, AvgPrice: order.AvgPrice,
				TotalPnl: order.TotalPnl, OrderTime: order.OrderTime,
			})
		}

		for _, order := range allNewOrders {
			if isLeadCloseOrder(order.PositionSide, order.Side) {
				res := s.processCopyTradeClose(fTrader, userID, cfg.PortfolioID, cfg.Nickname, runID, copyCloseOrder{
					Symbol: order.Symbol, Side: order.Side, PositionSide: order.PositionSide,
					ExecutedQty: order.ExecutedQty, AvgPrice: order.AvgPrice, OrderTime: order.OrderTime,
				}, "跟单-手动同步")
				if res.success {
					totalCopied++
					syncCounters.closed++
				} else if res.failed {
					totalErrors++
				}
				continue
			}

			// 检查是否已记录
			var count int
			s.database.DB().QueryRow("SELECT COUNT(*) FROM copy_trade_records WHERE user_id=? AND lead_order_time=?",
				userID, order.OrderTime).Scan(&count)
			if count > 0 {
				continue
			}

			// 计算跟单数量
			qty := order.ExecutedQty * cfg.SizeMultiplier
			if cfg.MaxCopySize > 0 && qty*order.AvgPrice > cfg.MaxCopySize {
				qty = cfg.MaxCopySize / order.AvgPrice
			}
			if qty < 0.001 {
				log.Printf("  ⚠️  [%s] 数量太小 %.4f，跳过", cfg.Nickname, qty)
				continue
			}

			// 实际执行交易
			var tradeResult map[string]interface{}
			var tradeErr error

			if isLeadOpenOrder(order.PositionSide, order.Side) {
				if order.PositionSide == "LONG" && order.Side == "BUY" {
					tradeResult, tradeErr = fTrader.OpenLong(order.Symbol, qty, 5)
				} else if order.PositionSide == "SHORT" && order.Side == "SELL" {
					tradeResult, tradeErr = fTrader.OpenShort(order.Symbol, qty, 5)
				} else if order.PositionSide == "BOTH" && order.Side == "SELL" {
					tradeResult, tradeErr = fTrader.OpenShort(order.Symbol, qty, 5)
				} else {
					log.Printf("  ℹ️  [%s] 跳过未知开仓: %s %s %s", cfg.Nickname, order.Symbol, order.Side, order.PositionSide)
					continue
				}
			} else {
				continue
			}

			actualQty := qty
			actualPrice := order.AvgPrice
			actualOrderID := fmt.Sprintf("copy_%s_%d", cfg.PortfolioID, order.OrderTime)

			if tradeErr != nil {
				log.Printf("❌ [%s] 开仓失败 %s %s: %v", cfg.Nickname, order.Symbol, order.PositionSide, tradeErr)
				totalErrors++
				logger.NotifyTrade(logger.TradeNotifyParams{
					Source: "跟单-手动同步", Status: logger.TradeStatusFailed, Action: "跟单开仓",
					Symbol: order.Symbol, Side: order.Side, PositionSide: order.PositionSide, Qty: qty,
					TraderOrNickname: cfg.Nickname, Error: tradeErr.Error(),
				})

				// 仍然记录失败状态
				s.database.DB().Exec(`
					INSERT OR IGNORE INTO copy_trade_records 
					(user_id, portfolio_id, nickname, order_id, symbol, side, position_side,
					 executed_qty, avg_price, total_pnl, status, lead_order_time)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'FAILED', ?)`,
					userID, cfg.PortfolioID, cfg.Nickname, actualOrderID, order.Symbol,
					order.Side, order.PositionSide, qty, order.AvgPrice, 0.0, order.OrderTime)
				continue
			}

			// 从交易结果中提取实际成交信息
			if tradeResult != nil {
				if q, ok := tradeResult["executedQty"].(float64); ok && q > 0 {
					actualQty = q
				}
				if p, ok := tradeResult["avgPrice"].(float64); ok && p > 0 {
					actualPrice = p
				}
				if id, ok := tradeResult["orderId"].(string); ok && id != "" {
					actualOrderID = id
				}
			}

			// 记录成功到数据库
			s.database.DB().Exec(`
				INSERT OR IGNORE INTO copy_trade_records 
				(user_id, portfolio_id, nickname, order_id, symbol, side, position_side,
				 executed_qty, avg_price, total_pnl, status, lead_order_time)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'OPEN', ?)`,
				userID, cfg.PortfolioID, cfg.Nickname, actualOrderID, order.Symbol,
				order.Side, order.PositionSide, actualQty, actualPrice, 0.0, order.OrderTime)

			totalCopied++
			log.Printf("  ✓ [%s] 已开仓 %s %s %.4f张 @ $%.2f",
				cfg.Nickname, order.Symbol, order.PositionSide, actualQty, actualPrice)
			logger.NotifyTrade(logger.TradeNotifyParams{
				Source: "跟单-手动同步", Status: logger.TradeStatusSuccess, Action: "跟单开仓",
				Symbol: order.Symbol, Side: order.Side, PositionSide: order.PositionSide,
				Qty: actualQty, Price: actualPrice, TraderOrNickname: cfg.Nickname,
			})
		}

		if latestTime > cfg.LastOrderTime {
			s.database.DB().Exec("UPDATE copy_trade_config SET last_order_time=?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
				latestTime, cfg.ID)
		}
	}

	// 同步当前交易所持仓到跟单记录 & 更新未实现盈亏
	log.Printf("📋 同步持仓盈亏数据...")
	if _, err := s.refreshCopyTradeUnrealizedPnL(userID, fTrader); err != nil {
		log.Printf("⚠️ 刷新未实现盈亏失败: %v", err)
	}
	positions, posErr := fTrader.GetPositions()
	if posErr == nil {
		for _, pos := range positions {
			symbol, _ := pos["symbol"].(string)
			if symbol == "" {
				continue
			}
			qty := copyTradePosFloat(pos["positionAmt"])
			if qty == 0 {
				continue
			}
			posSide := copyTradePosSideFromExchange(pos)
			side := "BUY"
			qtyAbs := qty
			if posSide == "SHORT" {
				side = "SELL"
				qtyAbs = -qty
			}
			price := copyTradePosFloat(pos["entryPrice"])
			pnl := copyTradePosFloat(pos["unRealizedProfit"])

			var count int
			s.database.DB().QueryRow(
				"SELECT COUNT(*) FROM copy_trade_records WHERE user_id=? AND symbol=? AND position_side=? AND status='OPEN'",
				userID, symbol, posSide).Scan(&count)
			if count > 0 {
				continue
			}

			orderID := fmt.Sprintf("sync_pos_%s_%d", symbol, time.Now().UnixMilli())
			s.database.DB().Exec(`
				INSERT OR IGNORE INTO copy_trade_records 
				(user_id, portfolio_id, nickname, order_id, symbol, side, position_side,
				 executed_qty, avg_price, total_pnl, status, lead_order_time)
				VALUES (?, 'exchange', '当前持仓', ?, ?, ?, ?, ?, ?, ?, 'OPEN', ?)`,
				userID, orderID, symbol, side, posSide, qtyAbs, price, pnl, time.Now().UnixMilli())
			log.Printf("  ✓ 已同步持仓: %s %s %.4f张 @ $%.2f PnL: $%.2f", symbol, posSide, qtyAbs, price, pnl)
			totalCopied++
		}

		activeKeys := make(map[string]bool)
		for _, pos := range positions {
			sym, _ := pos["symbol"].(string)
			qty := copyTradePosFloat(pos["positionAmt"])
			if sym != "" && qty != 0 {
				posSide := copyTradePosSideFromExchange(pos)
				activeKeys[sym+"|"+posSide] = true
			}
		}
		rows, err := s.database.DB().Query(
			"SELECT id, symbol, position_side FROM copy_trade_records WHERE user_id=? AND status='OPEN'",
			userID)
		if err == nil {
			for rows.Next() {
				var rid int
				var sym, posSide string
				if err := rows.Scan(&rid, &sym, &posSide); err != nil {
					continue
				}
				if !activeKeys[sym+"|"+posSide] {
					s.database.DB().Exec(
						"UPDATE copy_trade_records SET status='CLOSED', close_time=CURRENT_TIMESTAMP, close_price=avg_price WHERE id=?",
						rid)
					log.Printf("  🔒 %s %s 已平仓，标记为 CLOSED (ID=%d)", sym, posSide, rid)
				}
			}
			rows.Close()
		}
	} else {
		log.Printf("⚠️ 获取持仓失败: %v", posErr)
	}

	syncCounters.opened = totalCopied
	syncCounters.failed = totalErrors
	s.finishCopyTradeRun(runID, runStatusFromCounters(syncCounters), syncCounters, formatRunMessage(syncCounters))

	c.JSON(http.StatusOK, gin.H{
		"message": "同步完成",
		"copied":  totalCopied,
		"errors":  totalErrors,
		"run_id":  runID,
	})
}

// handleCopyOrder 手动跟单一笔操作（含AI分析）
func (s *Server) handleCopyOrder(c *gin.Context) {
	userID := c.GetString("user_id")

	var req struct {
		PortfolioID  string  `json:"portfolio_id"`
		Nickname     string  `json:"nickname"`
		Symbol       string  `json:"symbol"`
		Side         string  `json:"side"`
		PositionSide string  `json:"position_side"`
		ExecutedQty  float64 `json:"executed_qty"`
		AvgPrice     float64 `json:"avg_price"`
		OrderTime    int64   `json:"order_time"`
		SkipAI       bool    `json:"skip_ai"` // 前端确认后执行时跳过AI分析
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 只处理开仓操作
	isOpen := (req.PositionSide == "LONG" && req.Side == "BUY") ||
		(req.PositionSide == "SHORT" && req.Side == "SELL") ||
		(req.PositionSide == "BOTH" && req.Side == "SELL") // BOTH模式下SELL=开空
	if !isOpen {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只支持跟单开仓操作"})
		return
	}

	// 确定实际交易方向（BOTH模式转成SHORT/LONG）
	tradePosSide := req.PositionSide
	if tradePosSide == "BOTH" {
		tradePosSide = "SHORT" // BOTH+SELL=开空
	}

	// 获取交易所配置
	exchanges, err := s.database.GetExchanges(userID)
	if err != nil || len(exchanges) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先配置交易所"})
		return
	}
	var exchangeCfg *config.ExchangeConfig
	for _, ex := range exchanges {
		if ex.ID == "binance" && ex.Enabled {
			exchangeCfg = ex
			break
		}
	}
	if exchangeCfg == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先启用币安交易所"})
		return
	}

	fTrader := trader.NewFuturesTrader(exchangeCfg.APIKey, exchangeCfg.SecretKey, userID, exchangeCfg.Testnet)

	copyAI, copyAIErr := s.resolveCopyTradeAI(userID)
	aiTraderID := ""
	aiTraderName := ""
	var aiCfg *config.AIModelConfig
	if copyAIErr == nil {
		aiCfg = copyAI.AICfg
		aiTraderID = copyAI.AITraderID
		aiTraderName = copyAI.AITraderName
	}

	recordCopyDecision := func(p copyTradeAIDecisionParams) {
		p.UserID = userID
		p.RunID = 0
		p.PortfolioID = req.PortfolioID
		p.Nickname = req.Nickname
		p.Symbol = req.Symbol
		p.AITraderID = aiTraderID
		p.AITraderName = aiTraderName
		s.insertCopyTradeAIDecision(p)
	}

	// 如果skip_ai=true, 直接执行
	if req.SkipAI {
		aiQty := req.ExecutedQty
		var tradeResult map[string]interface{}
		var tradeErr error
		if tradePosSide == "LONG" {
			tradeResult, tradeErr = fTrader.OpenLong(req.Symbol, aiQty, 5)
		} else {
			tradeResult, tradeErr = fTrader.OpenShort(req.Symbol, aiQty, 5)
		}
		if tradeErr != nil {
			recordCopyDecision(copyTradeAIDecisionParams{
				RecommendedQty: aiQty,
				Reasoning:      tradeErr.Error(),
				ActionTaken:    "open_failed",
				Success:        false,
			})
			logger.NotifyTrade(logger.TradeNotifyParams{
				Source: "跟单-单笔", Status: logger.TradeStatusFailed, Action: "跟单开仓",
				Symbol: req.Symbol, Side: req.Side, PositionSide: tradePosSide, Qty: aiQty,
				TraderOrNickname: req.Nickname, Error: tradeErr.Error(),
			})
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("开仓失败: %v", tradeErr)})
			return
		}
		actualQty := aiQty
		actualPrice := req.AvgPrice
		actualOrderID := fmt.Sprintf("manual_%s_%d_%d", req.PortfolioID, req.OrderTime, time.Now().UnixNano()%100000)
		if tradeResult != nil {
			if q, ok := tradeResult["executedQty"].(float64); ok && q > 0 {
				actualQty = q
			}
			if p, ok := tradeResult["avgPrice"].(float64); ok && p > 0 {
				actualPrice = p
			}
		}
		recordCopyDecision(copyTradeAIDecisionParams{
			RecommendedQty: actualQty,
			ActionTaken:    "copied_open",
			Success:        true,
			Feasible:       true,
		})
		s.database.DB().Exec(`
			INSERT INTO copy_trade_records 
			(user_id, portfolio_id, nickname, order_id, symbol, side, position_side,
			 executed_qty, avg_price, total_pnl, status, lead_order_time)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, 'OPEN', ?)`,
			userID, req.PortfolioID, req.Nickname, actualOrderID, req.Symbol,
			req.Side, tradePosSide, actualQty, actualPrice, req.OrderTime)
		logger.NotifyTrade(logger.TradeNotifyParams{
			Source: "跟单-单笔", Status: logger.TradeStatusSuccess, Action: "跟单开仓",
			Symbol: req.Symbol, Side: req.Side, PositionSide: tradePosSide,
			Qty: actualQty, Price: actualPrice, TraderOrNickname: req.Nickname,
		})
		c.JSON(http.StatusOK, gin.H{"message": "跟单成功", "symbol": req.Symbol, "position": tradePosSide, "qty": actualQty, "price": actualPrice})
		return
	}

	// AI分析
	aiAnalysis := "AI分析不可用（无可用模型）"
	if copyAIErr != nil {
		aiAnalysis = copyAIErr.Error()
	}
	recommendedQty := req.ExecutedQty * 0.1

	var (
		inputPrompt   string
		aiResponseRaw string
		decisionJSON  string
		actionTaken   = "ai_error"
		feasible      bool
		reasoning     string
		suggestion    string
		decisionSuccess bool
	)

	if aiCfg == nil {
		reasoning = aiAnalysis
		actionTaken = "ai_error"
	} else {
		mcpClient := mcp.New()
		if aiCfg.Provider == "deepseek" {
			mcpClient = mcp.NewDeepSeekClient()
		} else if aiCfg.Provider == "qwen" {
			mcpClient = mcp.NewQwenClient()
		}
		mcpClient.SetAPIKey(aiCfg.APIKey, aiCfg.CustomAPIURL, aiCfg.CustomModelName)

		dir := req.Side
		if tradePosSide == "LONG" {
			dir = "买入开多"
		} else {
			dir = "卖出开空"
		}

		accountCtx := fetchCopyTradeAIAccountContext(fTrader)
		marketSection := fetchCopyTradeAIMarketSection(req.Symbol)
		inputPrompt = buildCopyTradeRiskPromptDetailed(accountCtx, marketSection, copyTradeAIPromptParams{
			Symbol: req.Symbol, Direction: dir, LeadNickname: req.Nickname,
			LeadPrice: req.AvgPrice, LeadQty: req.ExecutedQty,
		})

		systemPrompt := "你是一个专业的加密合约交易风控分析师，请结合最新K线与账户资产给出合理的跟单仓位建议。回答要简洁专业。"

		aiResponse, llmErr := mcpClient.CallWithMessages(systemPrompt, inputPrompt)
		if llmErr != nil {
			log.Printf("⚠️ AI分析失败: %v", llmErr)
			aiAnalysis = fmt.Sprintf("AI分析不可用: %v", llmErr)
			reasoning = aiAnalysis
			actionTaken = "ai_error"
		} else if aiResponse == "" {
			aiAnalysis = "AI分析不可用: 空响应"
			reasoning = aiAnalysis
			actionTaken = "ai_error"
		} else {
			aiResponseRaw = aiResponse
			cleaned := strings.TrimSpace(aiResponse)
			if jsonStart := strings.Index(cleaned, "{"); jsonStart >= 0 {
				if jsonEnd := strings.LastIndex(cleaned, "}"); jsonEnd > jsonStart {
					decisionJSON = cleaned[jsonStart : jsonEnd+1]
					cleaned = decisionJSON
				}
			}
			var parsedAI struct {
				Feasible         bool    `json:"feasible"`
				Reasoning        string  `json:"reasoning"`
				RecommendedRatio float64 `json:"recommended_ratio"`
				RecommendedQty   float64 `json:"recommended_qty"`
				Suggestion       string  `json:"suggestion"`
			}
			if jsonErr := json.Unmarshal([]byte(cleaned), &parsedAI); jsonErr == nil {
				feasible = parsedAI.Feasible
				reasoning = parsedAI.Reasoning
				suggestion = parsedAI.Suggestion
				if !feasible {
					rejectReason := parsedAI.Reasoning
					if parsedAI.Suggestion != "" {
						rejectReason = parsedAI.Suggestion
						if parsedAI.Reasoning != "" {
							rejectReason = parsedAI.Suggestion + "\n" + parsedAI.Reasoning
						}
					}
					aiAnalysis = rejectReason
					actionTaken = "ai_rejected"
					logger.NotifyTrade(logger.TradeNotifyParams{
						Source: "跟单-单笔", Status: logger.TradeStatusAIReject, Action: "跟单开仓",
						Symbol: req.Symbol, Side: req.Side, PositionSide: tradePosSide,
						TraderOrNickname: req.Nickname, Reason: rejectReason,
					})
				} else {
					aiAnalysis = parsedAI.Reasoning
					if parsedAI.Suggestion != "" {
						aiAnalysis = parsedAI.Suggestion + "\n" + parsedAI.Reasoning
					}
					if parsedAI.RecommendedQty > 0 {
						recommendedQty = parsedAI.RecommendedQty
					} else if parsedAI.RecommendedRatio > 0 {
						recommendedQty = req.ExecutedQty * parsedAI.RecommendedRatio
					}
					actionTaken = "pending_confirm"
					decisionSuccess = true
					log.Printf("  🤖 AI分析: 可行=%v 建议比例=%.2f 建议数量=%.4f", parsedAI.Feasible, parsedAI.RecommendedRatio, recommendedQty)
				}
			} else {
				aiAnalysis = aiResponse
				if len(aiAnalysis) > 500 {
					aiAnalysis = aiAnalysis[:500]
				}
				reasoning = "AI 响应 JSON 解析失败"
				actionTaken = "parse_failed"
			}
		}
	}

	recordCopyDecision(copyTradeAIDecisionParams{
		InputPrompt:    inputPrompt,
		AIResponseRaw:  aiResponseRaw,
		DecisionJSON:   decisionJSON,
		Feasible:       feasible,
		RecommendedQty: recommendedQty,
		Reasoning:      reasoning,
		Suggestion:     suggestion,
		ActionTaken:    actionTaken,
		Success:        decisionSuccess,
	})

	confirmAccount := fetchCopyTradeAIAccountContext(fTrader)
	c.JSON(http.StatusOK, gin.H{
		"need_confirm":    true,
		"ai_analysis":     aiAnalysis,
		"recommended_qty": recommendedQty,
		"account": gin.H{
			"total_equity":   confirmAccount.TotalEquity,
			"available":      confirmAccount.AvailableBalance,
			"open_positions": confirmAccount.OpenCount,
		},
		"trade": gin.H{
			"symbol":       req.Symbol,
			"side":         req.Side,
			"positionSide": req.PositionSide,
			"price":        req.AvgPrice,
			"original_qty": req.ExecutedQty,
		},
	})
}
