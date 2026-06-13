package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"nofx/config"
	"nofx/indicator"
	"nofx/market"
	"nofx/strategy"
	"nofx/trader"
	"strings"
	"sync"
	"time"
)

// TraderFactory 创建交易器工厂函数
type TraderFactory func(userID, exchangeID string) (trader.Trader, error)

// StrategyRunner 单个策略运行实例
type StrategyRunner struct {
	ID       string
	UserID   string
	Config   strategy.StrategyConfig
	stopCh   chan struct{}
	factory  TraderFactory
	database *config.Database
	engine   *strategy.Engine
}

// StrategyManager 策略管理器
type StrategyManager struct {
	runners map[string]*StrategyRunner
	mu      sync.RWMutex
	db      *config.Database
	factory TraderFactory
}

// NewStrategyManager 创建策略管理器
func NewStrategyManager(db *config.Database, factory TraderFactory) *StrategyManager {
	return &StrategyManager{
		runners: make(map[string]*StrategyRunner),
		db:      db,
		factory: factory,
	}
}

// LoadStrategiesFromDatabase 加载并启动所有 active 策略
func (sm *StrategyManager) LoadStrategiesFromDatabase() error {
	records, err := sm.db.GetActiveStrategies()
	if err != nil {
		return fmt.Errorf("加载策略失败: %w", err)
	}
	for _, rec := range records {
		cfg, err := parseStrategyConfig(rec)
		if err != nil {
			log.Printf("⚠️ 跳过策略 %s: %v", rec.ID, err)
			continue
		}
		if err := sm.StartStrategy(rec.UserID, cfg); err != nil {
			log.Printf("⚠️ 启动策略 %s 失败: %v", rec.ID, err)
		}
	}
	log.Printf("📋 已加载 %d 个激活策略", len(records))
	return nil
}

// StartStrategy 启动策略
func (sm *StrategyManager) StartStrategy(userID string, cfg strategy.StrategyConfig) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if r, ok := sm.runners[cfg.ID]; ok {
		close(r.stopCh)
		delete(sm.runners, cfg.ID)
	}

	runner := &StrategyRunner{
		ID:       cfg.ID,
		UserID:   userID,
		Config:   cfg,
		stopCh:   make(chan struct{}),
		factory:  sm.factory,
		database: sm.db,
		engine:   strategy.NewEngine(),
	}
	sm.runners[cfg.ID] = runner
	go runner.run()
	return nil
}

// PauseStrategy 暂停策略
func (sm *StrategyManager) PauseStrategy(strategyID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if r, ok := sm.runners[strategyID]; ok {
		close(r.stopCh)
		delete(sm.runners, strategyID)
	}
}

// IsRunning 策略是否在运行
func (sm *StrategyManager) IsRunning(strategyID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, ok := sm.runners[strategyID]
	return ok
}

func (r *StrategyRunner) run() {
	interval := timeframeToDuration(r.Config.Timeframe)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[STRATEGY] 启动策略 %s (%s %s %s)", r.Config.Name, r.Config.Symbol, r.Config.Timeframe, r.Config.ExchangeID)

	r.tick()
	for {
		select {
		case <-ticker.C:
			r.tick()
		case <-r.stopCh:
			log.Printf("[STRATEGY] 停止策略 %s", r.Config.Name)
			return
		}
	}
}

func (r *StrategyRunner) tick() {
	symbol := market.Normalize(r.Config.Symbol)
	client := market.NewAPIClient()
	mkKlines, err := client.GetKlines(symbol, r.Config.Timeframe, 200)
	if err != nil {
		log.Printf("[STRATEGY] %s 获取K线失败: %v", r.Config.Name, err)
		return
	}

	ik := toIndicatorKlines(mkKlines)
	signal := r.engine.Evaluate(r.Config, ik)
	signal.Timestamp = time.Now()

	if signal.Type == strategy.SignalHold {
		return
	}

	t, err := r.factory(r.UserID, r.Config.ExchangeID)
	if err != nil {
		log.Printf("[STRATEGY] %s 创建交易器失败: %v", r.Config.Name, err)
		return
	}

	executor := strategy.NewExecutor(strategy.ExecutorDeps{
		Trader:   t,
		Database: r.database,
	})
	if err := executor.Execute(signal, r.Config); err != nil {
		log.Printf("[STRATEGY] %s 执行信号失败: %v", r.Config.Name, err)
	}
}

func toIndicatorKlines(klines []market.Kline) []indicator.Kline {
	out := make([]indicator.Kline, len(klines))
	for i, k := range klines {
		out[i] = indicator.Kline{
			OpenTime: k.OpenTime,
			Open:     k.Open,
			High:     k.High,
			Low:      k.Low,
			Close:    k.Close,
			Volume:   k.Volume,
		}
	}
	return out
}

func parseStrategyConfig(rec *config.StrategyRecord) (strategy.StrategyConfig, error) {
	var cfg strategy.StrategyConfig
	if err := json.Unmarshal([]byte(rec.Config), &cfg); err != nil {
		return cfg, err
	}
	cfg.ID = rec.ID
	cfg.UserID = rec.UserID
	if cfg.Name == "" {
		cfg.Name = rec.Name
	}
	return cfg, nil
}

func timeframeToDuration(tf string) time.Duration {
	switch strings.TrimSpace(tf) {
	case "1m":
		return time.Minute
	case "3m":
		return 3 * time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "2h":
		return 2 * time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return time.Hour
	}
}
