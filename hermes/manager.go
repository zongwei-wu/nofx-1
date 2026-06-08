package hermes

import (
	"fmt"
	"log"
	"sync"

	"nofx/config"
	"nofx/trader"
)

// Manager 管理每用户 Hermes Runner（与 TraderManager 独立）
type Manager struct {
	runners map[string]*Runner
	deps    RunnerDeps
	mu      sync.RWMutex
}

// NewManager 创建 Hermes 管理器
func NewManager(deps RunnerDeps) *Manager {
	if deps.CreateTrader == nil {
		deps.CreateTrader = defaultCreateTrader
	}
	return &Manager{
		runners: make(map[string]*Runner),
		deps:    deps,
	}
}

func defaultCreateTrader(userID string, ex *config.ExchangeConfig) (trader.Trader, error) {
	if ex == nil {
		return nil, fmt.Errorf("交易所配置为空")
	}
	switch ex.ID {
	case "binance":
		return trader.NewFuturesTrader(ex.APIKey, ex.SecretKey, userID, ex.Testnet), nil
	case "hyperliquid":
		return trader.NewHyperliquidTrader(ex.APIKey, ex.HyperliquidWalletAddr, ex.Testnet)
	case "aster":
		return trader.NewAsterTrader(ex.AsterUser, ex.AsterSigner, ex.AsterPrivateKey)
	case "okx":
		return trader.NewOKXTrader(ex.APIKey, ex.SecretKey, ex.Passphrase, ex.Testnet)
	default:
		return nil, fmt.Errorf("不支持的交易所: %s", ex.ID)
	}
}

// LoadFromDatabase 恢复 runner_enabled 的用户
func (m *Manager) LoadFromDatabase(database *config.Database) error {
	ids, err := database.ListHermesRunnerEnabledUserIDs()
	if err != nil {
		return err
	}
	for _, userID := range ids {
		if err := m.Start(database, userID); err != nil {
			log.Printf("⚠️ [Hermes] 恢复用户 %s Runner 失败: %v", userID, err)
		}
	}
	return nil
}

// GetRunner 获取 Runner
func (m *Manager) GetRunner(userID string) (*Runner, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runners[userID]
	return r, ok
}

// Start 启动用户 Hermes Runner
func (m *Manager) Start(database *config.Database, userID string) error {
	m.mu.Lock()
	if existing, ok := m.runners[userID]; ok && existing.IsRunning() {
		m.mu.Unlock()
		return fmt.Errorf("Hermes Runner 已在运行")
	}
	runner := NewRunner(userID, database, m.deps)
	m.runners[userID] = runner
	m.mu.Unlock()

	if err := runner.Start(); err != nil {
		m.mu.Lock()
		delete(m.runners, userID)
		m.mu.Unlock()
		return err
	}

	settings, err := database.GetHermesSettings(userID)
	if err == nil {
		settings.RunnerEnabled = true
		_ = database.UpsertHermesSettings(settings)
	}
	return nil
}

// Stop 停止用户 Hermes Runner
func (m *Manager) Stop(database *config.Database, userID string) error {
	m.mu.Lock()
	runner, ok := m.runners[userID]
	if ok {
		runner.Stop()
		delete(m.runners, userID)
	}
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("Hermes Runner 未运行")
	}

	settings, err := database.GetHermesSettings(userID)
	if err == nil {
		settings.RunnerEnabled = false
		_ = database.UpsertHermesSettings(settings)
	}
	return nil
}
