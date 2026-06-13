package strategy

import (
	"encoding/json"
	"fmt"
	"log"
	"nofx/config"
	"nofx/trader"
	"time"

	"github.com/google/uuid"
)

// ExecutorDeps 执行器依赖
type ExecutorDeps struct {
	Trader   trader.Trader
	Database *config.Database
}

// Executor 策略自动下单执行器
type Executor struct {
	deps ExecutorDeps
}

// NewExecutor 创建执行器
func NewExecutor(deps ExecutorDeps) *Executor {
	return &Executor{deps: deps}
}

// Execute 执行策略信号
func (ex *Executor) Execute(signal Signal, cfg StrategyConfig) error {
	if signal.Type == SignalHold {
		return ex.logSignal(signal, cfg, nil, nil)
	}

	if err := ex.riskCheck(cfg); err != nil {
		return ex.logSignal(signal, cfg, nil, err)
	}

	qty, err := ex.calculatePositionSize(cfg)
	if err != nil {
		return ex.logSignal(signal, cfg, nil, err)
	}
	if qty <= 0 {
		return ex.logSignal(signal, cfg, nil, fmt.Errorf("计算下单量为 0"))
	}

	leverage := cfg.Risk.Leverage
	if leverage <= 0 {
		leverage = 5
	}

	var result map[string]interface{}
	var execErr error

	switch signal.Type {
	case SignalBuy:
		if cfg.Direction == "short_only" {
			execErr = fmt.Errorf("策略方向为 short_only，忽略 BUY 信号")
		} else {
			result, execErr = ex.deps.Trader.OpenLong(cfg.Symbol, qty, leverage)
		}
	case SignalSell:
		if cfg.Direction == "long_only" {
			// 平多
			result, execErr = ex.deps.Trader.CloseLong(cfg.Symbol, 0)
		} else if cfg.Direction == "short_only" || cfg.Direction == "both" {
			result, execErr = ex.deps.Trader.OpenShort(cfg.Symbol, qty, leverage)
		}
	}

	if execErr != nil {
		return ex.logSignal(signal, cfg, result, execErr)
	}

	// 设置止盈止损（仅开仓）
	if signal.Type == SignalBuy || (signal.Type == SignalSell && cfg.Direction != "long_only") {
		entryPrice := signal.Price
		if ep, ok := result["entry_price"].(float64); ok && ep > 0 {
			entryPrice = ep
		}
		ex.setExitOrders(cfg, entryPrice, qty, signal.Type)
	}

	return ex.logSignal(signal, cfg, result, nil)
}

func (ex *Executor) riskCheck(cfg StrategyConfig) error {
	if cfg.Risk.MaxPositions <= 0 {
		return nil
	}
	positions, err := ex.deps.Trader.GetPositions()
	if err != nil {
		return fmt.Errorf("获取持仓失败: %w", err)
	}
	if len(positions) >= cfg.Risk.MaxPositions {
		return fmt.Errorf("已达最大持仓数 %d", cfg.Risk.MaxPositions)
	}
	return nil
}

func (ex *Executor) calculatePositionSize(cfg StrategyConfig) (float64, error) {
	balance, err := ex.deps.Trader.GetBalance()
	if err != nil {
		return 0, err
	}
	available, _ := balance["availableBalance"].(float64)
	if available <= 0 {
		if eq, ok := balance["totalWalletBalance"].(float64); ok {
			available = eq
		}
	}
	if available <= 0 {
		return 0, fmt.Errorf("可用余额不足")
	}

	riskPct := cfg.Risk.RiskPerTrade
	if riskPct <= 0 {
		riskPct = 0.02
	}
	leverage := cfg.Risk.Leverage
	if leverage <= 0 {
		leverage = 5
	}

	price, err := ex.deps.Trader.GetMarketPrice(cfg.Symbol)
	if err != nil || price <= 0 {
		return 0, fmt.Errorf("获取市价失败: %w", err)
	}

	notional := available * riskPct * float64(leverage)
	qty := notional / price
	formatted, err := ex.deps.Trader.FormatQuantity(cfg.Symbol, qty)
	if err != nil {
		return qty, nil
	}
	var parsed float64
	fmt.Sscanf(formatted, "%f", &parsed)
	if parsed > 0 {
		return parsed, nil
	}
	return qty, nil
}

func (ex *Executor) setExitOrders(cfg StrategyConfig, entryPrice, qty float64, signalType SignalType) {
	positionSide := "LONG"
	if signalType == SignalSell && cfg.Direction != "long_only" {
		positionSide = "SHORT"
	}

	if cfg.ExitRules.TakeProfitPct > 0 {
		var tpPrice float64
		if positionSide == "LONG" {
			tpPrice = entryPrice * (1 + cfg.ExitRules.TakeProfitPct/100)
		} else {
			tpPrice = entryPrice * (1 - cfg.ExitRules.TakeProfitPct/100)
		}
		if err := ex.deps.Trader.SetTakeProfit(cfg.Symbol, positionSide, qty, tpPrice); err != nil {
			log.Printf("[STRATEGY] 设置止盈失败 %s: %v", cfg.Symbol, err)
		}
	}

	if cfg.ExitRules.StopLossPct > 0 {
		var slPrice float64
		if positionSide == "LONG" {
			slPrice = entryPrice * (1 - cfg.ExitRules.StopLossPct/100)
		} else {
			slPrice = entryPrice * (1 + cfg.ExitRules.StopLossPct/100)
		}
		if err := ex.deps.Trader.SetStopLoss(cfg.Symbol, positionSide, qty, slPrice); err != nil {
			log.Printf("[STRATEGY] 设置止损失败 %s: %v", cfg.Symbol, err)
		}
	}
}

func (ex *Executor) logSignal(signal Signal, cfg StrategyConfig, result map[string]interface{}, execErr error) error {
	if ex.deps.Database == nil {
		if execErr != nil {
			return execErr
		}
		return nil
	}

	indJSON, _ := json.Marshal(signal.Indicators)
	orderJSON, _ := json.Marshal(map[string]interface{}{
		"result": result,
		"error":  errString(execErr),
	})

	record := &config.StrategySignalRecord{
		ID:          uuid.New().String(),
		StrategyID:  cfg.ID,
		Symbol:      signal.Symbol,
		Signal:      string(signal.Type),
		Price:       signal.Price,
		Indicators:  string(indJSON),
		OrderResult: string(orderJSON),
		CreatedAt:   time.Now(),
	}
	if err := ex.deps.Database.CreateStrategySignal(record); err != nil {
		log.Printf("[STRATEGY] 记录信号失败: %v", err)
	}
	return execErr
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
