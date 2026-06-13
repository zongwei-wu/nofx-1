package backtest

import (
	"math"
	"time"

	"nofx/indicator"
	"nofx/strategy"
)

// Engine 回测引擎
type Engine struct {
	cfg      BacktestConfig
	strategy strategy.StrategyConfig
	engine   *strategy.Engine
}

// NewEngine 创建回测引擎
func NewEngine(btCfg BacktestConfig, stratCfg strategy.StrategyConfig) *Engine {
	if btCfg.FeeRate <= 0 {
		btCfg.FeeRate = 0.0004
	}
	if btCfg.SlippageRate <= 0 {
		btCfg.SlippageRate = 0.0002
	}
	if btCfg.InitialCapital <= 0 {
		btCfg.InitialCapital = 1000
	}
	return &Engine{
		cfg:      btCfg,
		strategy: stratCfg,
		engine:   strategy.NewEngine(),
	}
}

type position struct {
	side       string
	entryPrice float64
	entryTime  time.Time
	quantity   float64
	stopLoss   float64
	takeProfit float64
}

// Run 执行回测
func (e *Engine) Run(klines []indicator.Kline) *BacktestResult {
	result := &BacktestResult{
		StrategyName:   e.strategy.Name,
		Symbol:         e.cfg.Symbol,
		Timeframe:      e.cfg.Timeframe,
		InitialCapital: e.cfg.InitialCapital,
		FinalEquity:    e.cfg.InitialCapital,
		EquityCurve:    []EquityPoint{},
		Trades:         []TradeRecord{},
	}

	if len(klines) < 30 {
		return result
	}

	cash := e.cfg.InitialCapital
	var pos *position
	equityCurve := make([]EquityPoint, 0, len(klines))
	dailyReturns := make([]float64, 0)
	prevEquity := cash

	for i := 30; i < len(klines); i++ {
		slice := klines[:i+1]
		k := klines[i]
		ts := k.OpenTime
		if ts > 1e12 {
			ts = ts / 1000
		}

		// 检查止盈止损
		if pos != nil {
			closed, trade := e.checkExit(pos, k, ts)
			if closed {
				cash += trade.PnL - trade.Fee
				result.Trades = append(result.Trades, trade)
				pos = nil
			}
		}

		sig := e.engine.Evaluate(e.strategy, slice)

		if pos == nil && sig.Type == strategy.SignalBuy && e.strategy.Direction != "short_only" {
			qty, cost := e.calcEntry(cash, k.Close)
			if qty > 0 && cost <= cash {
				fee := cost * e.cfg.FeeRate
				cash -= cost + fee
				pos = &position{
					side:       "LONG",
					entryPrice: k.Close * (1 + e.cfg.SlippageRate),
					entryTime:  time.Unix(ts, 0),
					quantity:   qty,
				}
				e.setExitLevels(pos)
			}
		} else if pos == nil && sig.Type == strategy.SignalSell && e.strategy.Direction != "long_only" {
			qty, cost := e.calcEntry(cash, k.Close)
			if qty > 0 && cost <= cash {
				fee := cost * e.cfg.FeeRate
				cash -= cost + fee
				pos = &position{
					side:       "SHORT",
					entryPrice: k.Close * (1 - e.cfg.SlippageRate),
					entryTime:  time.Unix(ts, 0),
					quantity:   qty,
				}
				e.setExitLevels(pos)
			}
		} else if pos != nil && sig.Type == strategy.SignalSell && pos.side == "LONG" {
			trade := e.closePosition(pos, k, ts)
			cash += pos.quantity*pos.entryPrice + trade.PnL - trade.Fee
			result.Trades = append(result.Trades, trade)
			pos = nil
		}

		equity := cash
		if pos != nil {
			if pos.side == "LONG" {
				equity += pos.quantity * k.Close
			} else {
				equity += pos.quantity * (2*pos.entryPrice - k.Close)
			}
		}
		equityCurve = append(equityCurve, EquityPoint{Time: ts, Equity: equity})
		if prevEquity > 0 {
			dailyReturns = append(dailyReturns, (equity-prevEquity)/prevEquity)
		}
		prevEquity = equity
	}

	// 强制平仓
	if pos != nil {
		last := klines[len(klines)-1]
		ts := last.OpenTime
		if ts > 1e12 {
			ts = ts / 1000
		}
		trade := e.closePosition(pos, last, ts)
		cash += pos.quantity*pos.entryPrice + trade.PnL - trade.Fee
		result.Trades = append(result.Trades, trade)
	}

	result.FinalEquity = cash
	if len(equityCurve) > 0 {
		result.EquityCurve = equityCurve
	}
	result.Metrics = computeMetrics(e.cfg.InitialCapital, cash, equityCurve, result.Trades, dailyReturns, e.cfg.Timeframe)
	return result
}

func (e *Engine) calcEntry(cash, price float64) (qty, cost float64) {
	riskPct := e.strategy.Risk.RiskPerTrade
	if riskPct <= 0 {
		riskPct = 0.1
	}
	leverage := float64(e.strategy.Risk.Leverage)
	if leverage <= 0 {
		leverage = 1
	}
	notional := cash * riskPct * leverage
	if price <= 0 {
		return 0, 0
	}
	qty = notional / price
	cost = qty * price
	return qty, cost
}

func (e *Engine) setExitLevels(pos *position) {
	if e.strategy.ExitRules.TakeProfitPct > 0 {
		if pos.side == "LONG" {
			pos.takeProfit = pos.entryPrice * (1 + e.strategy.ExitRules.TakeProfitPct/100)
		} else {
			pos.takeProfit = pos.entryPrice * (1 - e.strategy.ExitRules.TakeProfitPct/100)
		}
	}
	if e.strategy.ExitRules.StopLossPct > 0 {
		if pos.side == "LONG" {
			pos.stopLoss = pos.entryPrice * (1 - e.strategy.ExitRules.StopLossPct/100)
		} else {
			pos.stopLoss = pos.entryPrice * (1 + e.strategy.ExitRules.StopLossPct/100)
		}
	}
}

func (e *Engine) checkExit(pos *position, k indicator.Kline, ts int64) (bool, TradeRecord) {
	exitPrice := 0.0
	if pos.side == "LONG" {
		if pos.stopLoss > 0 && k.Low <= pos.stopLoss {
			exitPrice = pos.stopLoss
		} else if pos.takeProfit > 0 && k.High >= pos.takeProfit {
			exitPrice = pos.takeProfit
		}
	} else {
		if pos.stopLoss > 0 && k.High >= pos.stopLoss {
			exitPrice = pos.stopLoss
		} else if pos.takeProfit > 0 && k.Low <= pos.takeProfit {
			exitPrice = pos.takeProfit
		}
	}
	if exitPrice <= 0 {
		return false, TradeRecord{}
	}
	trade := e.buildTrade(pos, exitPrice, time.Unix(ts, 0))
	return true, trade
}

func (e *Engine) closePosition(pos *position, k indicator.Kline, ts int64) TradeRecord {
	exitPrice := k.Close
	if pos.side == "LONG" {
		exitPrice *= (1 - e.cfg.SlippageRate)
	} else {
		exitPrice *= (1 + e.cfg.SlippageRate)
	}
	return e.buildTrade(pos, exitPrice, time.Unix(ts, 0))
}

func (e *Engine) buildTrade(pos *position, exitPrice float64, exitTime time.Time) TradeRecord {
	var pnl float64
	if pos.side == "LONG" {
		pnl = (exitPrice - pos.entryPrice) * pos.quantity
	} else {
		pnl = (pos.entryPrice - exitPrice) * pos.quantity
	}
	fee := (pos.entryPrice + exitPrice) * pos.quantity * e.cfg.FeeRate
	pnlPct := 0.0
	if pos.entryPrice > 0 {
		pnlPct = (exitPrice - pos.entryPrice) / pos.entryPrice * 100
		if pos.side == "SHORT" {
			pnlPct = -pnlPct
		}
	}
	return TradeRecord{
		Symbol:     e.cfg.Symbol,
		Side:       pos.side,
		EntryTime:  pos.entryTime,
		ExitTime:   exitTime,
		EntryPrice: pos.entryPrice,
		ExitPrice:  exitPrice,
		Quantity:   pos.quantity,
		PnL:        pnl,
		PnLPct:     pnlPct,
		Fee:        fee,
	}
}

func computeMetrics(initial, final float64, curve []EquityPoint, trades []TradeRecord, returns []float64, timeframe string) BacktestMetrics {
	m := BacktestMetrics{TotalTrades: len(trades)}
	if initial > 0 {
		m.TotalReturnPct = (final - initial) / initial * 100
	}

	// 最大回撤
	peak := 0.0
	maxDD := 0.0
	for _, p := range curve {
		if p.Equity > peak {
			peak = p.Equity
		}
		if peak > 0 {
			dd := (peak - p.Equity) / peak * 100
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	m.MaxDrawdownPct = maxDD

	// 胜率 / 盈亏比
	wins, losses := 0, 0
	totalWin, totalLoss := 0.0, 0.0
	maxConsecLoss, consecLoss := 0, 0
	for _, t := range trades {
		net := t.PnL - t.Fee
		if net > 0 {
			wins++
			totalWin += t.PnLPct
			consecLoss = 0
		} else if net < 0 {
			losses++
			totalLoss += math.Abs(t.PnLPct)
			consecLoss++
			if consecLoss > maxConsecLoss {
				maxConsecLoss = consecLoss
			}
		}
	}
	m.MaxConsecutiveLosses = maxConsecLoss
	if len(trades) > 0 {
		m.WinRatePct = float64(wins) / float64(len(trades)) * 100
	}
	if wins > 0 {
		m.AvgWinPct = totalWin / float64(wins)
	}
	if losses > 0 {
		m.AvgLossPct = totalLoss / float64(losses)
	}
	if totalLoss > 0 {
		m.ProfitFactor = totalWin / totalLoss
	}

	// 夏普比率
	if len(returns) > 1 {
		mean := 0.0
		for _, r := range returns {
			mean += r
		}
		mean /= float64(len(returns))
		variance := 0.0
		for _, r := range returns {
			variance += (r - mean) * (r - mean)
		}
		stddev := math.Sqrt(variance / float64(len(returns)))
		if stddev > 0 {
			m.SharpeRatio = mean / stddev * math.Sqrt(252)
		}
	}

	// 年化收益（粗略估算）
	if len(curve) >= 2 {
		days := float64(len(curve)) / barsPerDay(timeframe)
		if days > 0 && initial > 0 {
			m.AnnualizedReturnPct = (math.Pow(final/initial, 365/days) - 1) * 100
		}
	}

	return m
}

func barsPerDay(tf string) float64 {
	switch tf {
	case "1m":
		return 1440
	case "5m":
		return 288
	case "15m":
		return 96
	case "1h":
		return 24
	case "4h":
		return 6
	case "1d":
		return 1
	default:
		return 24
	}
}
