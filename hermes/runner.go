package hermes

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"nofx/config"
	"nofx/decision"
	"nofx/logger"
	"nofx/mcp"
	"nofx/trader"
)

// RunnerDeps Hermes Runner 依赖
type RunnerDeps struct {
	CreateTrader func(userID string, ex *config.ExchangeConfig) (trader.Trader, error)
}

// Runner Hermes 自主交易运行器（独立于 AI 交易员）
type Runner struct {
	userID         string
	database       *config.Database
	deps           RunnerDeps
	trader         trader.Trader
	mcpClient      mcp.AIClient
	decisionLogger logger.IDecisionLogger
	settings       *config.HermesSettings

	mu        sync.Mutex
	isRunning bool
	startTime time.Time
	callCount int
	stopCh    chan struct{}
}

// NewRunner 创建 Hermes Runner
func NewRunner(userID string, database *config.Database, deps RunnerDeps) *Runner {
	logDir := fmt.Sprintf("decision_logs/hermes/%s", userID)
	return &Runner{
		userID:         userID,
		database:       database,
		deps:           deps,
		decisionLogger: logger.NewDecisionLogger(logDir),
		stopCh:         make(chan struct{}),
	}
}

func (r *Runner) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.isRunning
}

func (r *Runner) Start() error {
	r.mu.Lock()
	if r.isRunning {
		r.mu.Unlock()
		return fmt.Errorf("Hermes Runner 已在运行")
	}
	r.isRunning = true
	r.startTime = time.Now()
	r.callCount = 0
	r.stopCh = make(chan struct{})
	r.mu.Unlock()

	if err := r.reloadRuntime(); err != nil {
		r.mu.Lock()
		r.isRunning = false
		r.mu.Unlock()
		return err
	}

	go r.loop()
	log.Printf("🚀 [Hermes] 用户 %s Runner 已启动", r.userID)
	return nil
}

func (r *Runner) Stop() {
	r.mu.Lock()
	if !r.isRunning {
		r.mu.Unlock()
		return
	}
	r.isRunning = false
	close(r.stopCh)
	r.mu.Unlock()
	log.Printf("⏹ [Hermes] 用户 %s Runner 已停止", r.userID)
}

func (r *Runner) reloadRuntime() error {
	settings, err := r.database.GetHermesSettings(r.userID)
	if err != nil {
		return err
	}
	exchanges, err := r.database.GetExchanges(r.userID)
	if err != nil {
		return err
	}
	models, err := r.database.GetAIModels(r.userID)
	if err != nil {
		return err
	}
	if err := ValidateReadyToStart(settings, exchanges, models); err != nil {
		return err
	}

	var ex *config.ExchangeConfig
	for _, e := range exchanges {
		if e.ID == settings.ExchangeID {
			ex = e
			break
		}
	}
	t, err := r.deps.CreateTrader(r.userID, ex)
	if err != nil {
		return err
	}

	var model *config.AIModelConfig
	for _, m := range models {
		if m.ID == settings.AIModelID {
			model = m
			break
		}
	}
	ai, err := NewAIClientFromModel(model)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.settings = settings
	r.trader = t
	r.mcpClient = ai
	r.mu.Unlock()
	return nil
}

func (r *Runner) loop() {
	interval := time.Duration(r.settings.ScanIntervalMinutes) * time.Minute
	if interval < time.Minute {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if err := r.runCycle(); err != nil {
		log.Printf("⚠️ [Hermes] %s 首周期错误: %v", r.userID, err)
	}

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			if err := r.reloadRuntime(); err != nil {
				log.Printf("❌ [Hermes] %s 重载配置失败: %v", r.userID, err)
				continue
			}
			if err := r.runCycle(); err != nil {
				log.Printf("⚠️ [Hermes] %s 周期错误: %v", r.userID, err)
			}
		}
	}
}

func (r *Runner) runCycle() error {
	r.mu.Lock()
	r.callCount++
	callCount := r.callCount
	startTime := r.startTime
	settings := r.settings
	t := r.trader
	ai := r.mcpClient
	r.mu.Unlock()

	record := &logger.DecisionRecord{
		ExecutionLog: []string{},
		Success:      true,
		CycleNumber:  callCount,
	}

	ctx, err := BuildContext(t, settings, callCount, startTime, r.decisionLogger)
	if err != nil {
		record.Success = false
		record.ErrorMessage = err.Error()
		r.decisionLogger.LogDecision(record)
		return err
	}

	if SetInitialBalanceIfZero(settings, ctx.Account.TotalEquity) {
		_ = r.database.UpsertHermesSettings(settings)
	}

	template := settings.SystemPromptTemplate
	if template == "" {
		template = "hermes"
	}

	full, err := decision.GetFullDecisionWithCustomPrompt(ctx, ai, "", false, template)
	if full != nil && full.AIRequestDurationMs > 0 {
		record.AIRequestDurationMs = full.AIRequestDurationMs
	}
	if full != nil {
		record.SystemPrompt = full.SystemPrompt
		record.InputPrompt = full.UserPrompt
		record.CoTTrace = full.CoTTrace
		if len(full.Decisions) > 0 {
			b, _ := json.MarshalIndent(full.Decisions, "", "  ")
			record.DecisionJSON = string(b)
		}
	}
	if err != nil {
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("AI 决策失败: %v", err)
		r.decisionLogger.LogDecision(record)
		return err
	}

	for _, d := range full.Decisions {
		actionRecord := logger.DecisionAction{
			Action:    d.Action,
			Symbol:    d.Symbol,
			Leverage:  d.Leverage,
			Timestamp: time.Now(),
			Success:   true,
		}
		result, execErr := ExecuteDecision(t, &d)
		if execErr != nil {
			actionRecord.Success = false
			actionRecord.Error = execErr.Error()
			record.Success = false
			record.ErrorMessage = execErr.Error()
		} else if result != nil {
			if oid, ok := result["orderId"].(int64); ok {
				actionRecord.OrderID = oid
			}
		}
		record.Decisions = append(record.Decisions, actionRecord)
		record.ExecutionLog = append(record.ExecutionLog,
			fmt.Sprintf("%s %s: %v", d.Action, d.Symbol, execErr == nil))
		log.Printf("[HERMES] user=%s action=%s symbol=%s ok=%v",
			r.userID, d.Action, d.Symbol, execErr == nil)
	}

	r.decisionLogger.LogDecision(record)
	return nil
}

// Status 返回运行状态摘要
func (r *Runner) Status() map[string]interface{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	exchangeID := ""
	modelID := ""
	auto := false
	scan := 5
	if r.settings != nil {
		exchangeID = r.settings.ExchangeID
		modelID = r.settings.AIModelID
		auto = r.settings.AutonomousEnabled
		scan = r.settings.ScanIntervalMinutes
	}
	return map[string]interface{}{
		"user_id":               r.userID,
		"is_running":            r.isRunning,
		"call_count":            r.callCount,
		"exchange_id":           exchangeID,
		"ai_model_id":           modelID,
		"autonomous_enabled":    auto,
		"scan_interval_minutes": scan,
		"runtime_minutes":       int(time.Since(r.startTime).Minutes()),
	}
}

// GetRecentDecisions 最近决策记录
func (r *Runner) GetRecentDecisions(limit int) ([]*logger.DecisionRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	return r.decisionLogger.GetLatestRecords(limit)
}

// UserID 返回用户 ID
func (r *Runner) UserID() string {
	return r.userID
}

// LogPrefix 日志前缀
func (r *Runner) LogPrefix() string {
	return strings.TrimSpace("hermes:" + r.userID)
}
