package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"nofx/config"
	"nofx/trader"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type copyTradeRunState struct {
	mu                      sync.RWMutex
	lastAutoFollowFinished  time.Time
	autoFollowRunning       bool
}

var globalCopyTradeRunState copyTradeRunState

func (s *copyTradeRunState) markAutoFollowFinished() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAutoFollowFinished = time.Now()
	s.autoFollowRunning = false
}

func (s *copyTradeRunState) markAutoFollowStarted() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autoFollowRunning = true
}

func (s *copyTradeRunState) lastFinished() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastAutoFollowFinished
}

func (s *copyTradeRunState) isRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.autoFollowRunning
}

func (s *Server) pruneCopyTradeRuns(userID string) {
	s.database.DB().Exec(`
		DELETE FROM copy_trade_runs WHERE user_id = ? AND id NOT IN (
			SELECT id FROM copy_trade_runs WHERE user_id = ? ORDER BY started_at DESC LIMIT 200
		)`, userID, userID)
}

func (s *Server) startCopyTradeRun(userID, trigger string) (int64, error) {
	s.pruneCopyTradeRuns(userID)
	res, err := s.database.DB().Exec(`
		INSERT INTO copy_trade_runs (user_id, trigger, status, started_at)
		VALUES (?, ?, 'running', CURRENT_TIMESTAMP)`, userID, trigger)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

type copyTradeRunCounters struct {
	tradersChecked int
	opened         int
	closed         int
	skipped        int
	failed         int
}

func (s *Server) insertCopyTradeRunEvent(runID int64, userID, portfolioID, nickname, symbol, action, leadSide, leadPosSide, ourStatus, detail string, leadOrderTime int64) {
	_, err := s.database.DB().Exec(`
		INSERT INTO copy_trade_run_events
		(run_id, user_id, portfolio_id, nickname, symbol, action, lead_side, lead_position_side, our_status, detail, lead_order_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		runID, userID, portfolioID, nickname, symbol, action, leadSide, leadPosSide, ourStatus, detail, leadOrderTime)
	if err != nil {
		log.Printf("⚠️ 写入跟单监控事件失败: %v", err)
	}
}

func (s *Server) finishCopyTradeRun(runID int64, status string, c copyTradeRunCounters, message string) {
	_, err := s.database.DB().Exec(`
		UPDATE copy_trade_runs SET
			finished_at = CURRENT_TIMESTAMP,
			status = ?,
			traders_checked = ?,
			actions_opened = ?,
			actions_skipped = ?,
			actions_failed = ?,
			message = ?
		WHERE id = ?`,
		status, c.tradersChecked, c.opened, c.skipped, c.failed, message, runID)
	if err != nil {
		log.Printf("⚠️ 更新跟单 run 失败: %v", err)
	}
}

func (s *Server) ourStatusForSymbol(userID, portfolioID, symbol string) string {
	var status string
	err := s.database.DB().QueryRow(`
		SELECT status FROM copy_trade_records
		WHERE user_id = ? AND portfolio_id = ? AND symbol = ?
		ORDER BY copy_time DESC LIMIT 1`,
		userID, portfolioID, symbol).Scan(&status)
	if err == sql.ErrNoRows {
		return "NONE"
	}
	if err != nil {
		return "NONE"
	}
	return status
}

// handleGetCopyTradeMonitor 跟单自动监控聚合数据
func (s *Server) handleGetCopyTradeMonitor(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	timerSec := copyTradeTimerIntervalSec
	lastFinished := globalCopyTradeRunState.lastFinished()
	var nextRunAt time.Time
	if lastFinished.IsZero() {
		nextRunAt = time.Now()
	} else {
		nextRunAt = lastFinished.Add(time.Duration(timerSec) * time.Second)
	}

	// 最近一次 run
	var lastRun map[string]interface{}
	row := s.database.DB().QueryRow(`
		SELECT id, trigger, started_at, finished_at, status,
		       traders_checked, actions_opened, actions_skipped, actions_failed, message
		FROM copy_trade_runs WHERE user_id = ?
		ORDER BY started_at DESC LIMIT 1`, userID)
	var runID int64
	var trigger, status, message sql.NullString
	var startedAt, finishedAt sql.NullString
	var tradersChecked, opened, skipped, failed int
	if err := row.Scan(&runID, &trigger, &startedAt, &finishedAt, &status,
		&tradersChecked, &opened, &skipped, &failed, &message); err == nil {
		lastRun = map[string]interface{}{
			"id":               runID,
			"trigger":          trigger.String,
			"started_at":       startedAt.String,
			"finished_at":      finishedAt.String,
			"status":           status.String,
			"traders_checked":  tradersChecked,
			"actions_opened":   opened,
			"actions_skipped":  skipped,
			"actions_failed":   failed,
			"message":          message.String,
		}
	}

	// 监控中的交易员配置
	cfgRows, err := s.database.DB().Query(`
		SELECT id, portfolio_id, nickname, enabled, auto_follow, last_order_time
		FROM copy_trade_config
		WHERE user_id = ? AND enabled = 1 AND auto_follow = 1
		ORDER BY nickname`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询配置失败"})
		return
	}
	defer cfgRows.Close()

	type monitoredTrader struct {
		ConfigID          int                      `json:"config_id"`
		PortfolioID       string                   `json:"portfolio_id"`
		Nickname          string                   `json:"nickname"`
		AutoFollow        bool                     `json:"auto_follow"`
		LastOrderTime     int64                    `json:"last_order_time"`
		LatestLeadActions []map[string]interface{} `json:"latest_lead_actions"`
		OurPositions      []map[string]interface{} `json:"our_positions"`
		LastEvent         map[string]interface{}   `json:"last_event"`
	}

	var monitored []monitoredTrader
	for cfgRows.Next() {
		var mt monitoredTrader
		var enabled, autoFollow int
		if err := cfgRows.Scan(&mt.ConfigID, &mt.PortfolioID, &mt.Nickname, &enabled, &autoFollow, &mt.LastOrderTime); err != nil {
			continue
		}
		mt.AutoFollow = autoFollow != 0

		if history, err := getLeadOrdersForMonitor(mt.PortfolioID, false); err == nil && history.Data != nil {
			for _, o := range history.Data.List {
				if o.Symbol == "" {
					continue
				}
				mt.LatestLeadActions = append(mt.LatestLeadActions, map[string]interface{}{
					"symbol":        o.Symbol,
					"action":        leadActionLabel(o.PositionSide, o.Side),
					"display":       leadActionDisplay(o.PositionSide, o.Side),
					"side":          o.Side,
					"position_side": o.PositionSide,
					"order_time":    o.OrderTime,
					"avg_price":     o.AvgPrice,
					"executed_qty":  o.ExecutedQty,
				})
				if len(mt.LatestLeadActions) >= 5 {
					break
				}
			}
		}

		// 按 symbol + position_side 聚合 OPEN 持仓，避免同币种同方向多条跟单记录重复展示
		posRows, _ := s.database.DB().Query(`
			SELECT symbol, position_side, side, status,
			       SUM(executed_qty) AS executed_qty,
			       CASE WHEN SUM(executed_qty) > 0
			            THEN SUM(avg_price * executed_qty) / SUM(executed_qty)
			            ELSE 0 END AS avg_price,
			       SUM(total_pnl) AS total_pnl,
			       MAX(lead_order_time) AS lead_order_time
			FROM copy_trade_records
			WHERE user_id = ? AND portfolio_id = ? AND status = 'OPEN'
			GROUP BY symbol, position_side
			ORDER BY MAX(copy_time) DESC
			LIMIT 10`, userID, mt.PortfolioID)
		if posRows != nil {
			for posRows.Next() {
				var sym, posSide, side, st string
				var qty, price, pnl float64
				var lot int64
				if posRows.Scan(&sym, &posSide, &side, &st, &qty, &price, &pnl, &lot) == nil {
					mt.OurPositions = append(mt.OurPositions, map[string]interface{}{
						"symbol":          sym,
						"position_side":   posSide,
						"side":            side,
						"status":          st,
						"executed_qty":    qty,
						"avg_price":       price,
						"total_pnl":       pnl,
						"pnl_kind":        copyTradePnLKind(st),
						"lead_order_time": lot,
					})
				}
			}
			posRows.Close()
		}

		evRow := s.database.DB().QueryRow(`
			SELECT action, symbol, detail, our_status, created_at
			FROM copy_trade_run_events
			WHERE user_id = ? AND portfolio_id = ?
			ORDER BY id DESC LIMIT 1`, userID, mt.PortfolioID)
		var action, sym, detail, ourSt, createdAt sql.NullString
		if evRow.Scan(&action, &sym, &detail, &ourSt, &createdAt) == nil {
			mt.LastEvent = map[string]interface{}{
				"action":     action.String,
				"symbol":     sym.String,
				"detail":     detail.String,
				"our_status": ourSt.String,
				"created_at": createdAt.String,
			}
		}

		monitored = append(monitored, mt)
	}

	// 最近 runs
	runRows, _ := s.database.DB().Query(`
		SELECT id, trigger, started_at, finished_at, status,
		       traders_checked, actions_opened, actions_skipped, actions_failed, message
		FROM copy_trade_runs WHERE user_id = ?
		ORDER BY started_at DESC LIMIT 50`, userID)
	var recentRuns []map[string]interface{}
	if runRows != nil {
		defer runRows.Close()
		for runRows.Next() {
			var id int64
			var tr, st, msg, started, finished sql.NullString
			var tc, op, sk, fl int
			if runRows.Scan(&id, &tr, &started, &finished, &st, &tc, &op, &sk, &fl, &msg) == nil {
				recentRuns = append(recentRuns, map[string]interface{}{
					"id": id, "trigger": tr.String, "started_at": started.String,
					"finished_at": finished.String, "status": st.String,
					"traders_checked": tc, "actions_opened": op,
					"actions_skipped": sk, "actions_failed": fl, "message": msg.String,
				})
			}
		}
	}

	// 最近 events
	evRows, _ := s.database.DB().Query(`
		SELECT e.id, e.run_id, e.portfolio_id, e.nickname, e.symbol, e.action,
		       e.lead_side, e.lead_position_side, e.our_status, e.detail, e.lead_order_time, e.created_at
		FROM copy_trade_run_events e
		WHERE e.user_id = ?
		ORDER BY e.id DESC LIMIT 100`, userID)
	var recentEvents []map[string]interface{}
	if evRows != nil {
		defer evRows.Close()
		for evRows.Next() {
			var eid, rid int64
			var pid, nick, sym, act, ls, lp, os, det, cat sql.NullString
			var lot int64
			if evRows.Scan(&eid, &rid, &pid, &nick, &sym, &act, &ls, &lp, &os, &det, &lot, &cat) == nil {
				recentEvents = append(recentEvents, map[string]interface{}{
					"id": eid, "run_id": rid, "portfolio_id": pid.String, "nickname": nick.String,
					"symbol": sym.String, "action": act.String, "lead_side": ls.String,
					"lead_position_side": lp.String, "our_status": os.String,
					"detail": det.String, "lead_order_time": lot, "created_at": cat.String,
				})
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"timer_interval_sec":      timerSec,
		"auto_follow_running":     globalCopyTradeRunState.isRunning(),
		"last_auto_follow_finished": lastFinished.Format(time.RFC3339),
		"next_run_estimated_at":   nextRunAt.Format(time.RFC3339),
		"last_run":                lastRun,
		"monitored_traders":       monitored,
		"recent_runs":             recentRuns,
		"recent_events":           recentEvents,
	})
}

func runStatusFromCounters(c copyTradeRunCounters) string {
	if c.failed > 0 && c.opened == 0 && c.closed == 0 {
		return "failed"
	}
	if c.failed > 0 {
		return "partial"
	}
	return "success"
}

func formatRunMessage(c copyTradeRunCounters) string {
	return fmt.Sprintf("检查 %d 位交易员，开仓 %d，平仓 %d，跳过 %d，失败 %d",
		c.tradersChecked, c.opened, c.closed, c.skipped, c.failed)
}

type copyTradePnLRefreshResult struct {
	Updated            int
	TotalUnrealizedPnL float64
}

func copyTradePosFloat(v interface{}) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case string:
		var f float64
		fmt.Sscanf(x, "%f", &f)
		return f
	default:
		return 0
	}
}

func copyTradePosSideFromExchange(pos map[string]interface{}) string {
	if ps, ok := pos["positionSide"].(string); ok && ps != "" && ps != "BOTH" {
		return strings.ToUpper(ps)
	}
	if side, ok := pos["side"].(string); ok {
		if strings.EqualFold(side, "short") {
			return "SHORT"
		}
		return "LONG"
	}
	qty := copyTradePosFloat(pos["positionAmt"])
	if qty < 0 {
		return "SHORT"
	}
	return "LONG"
}

func (s *Server) getBinanceExchangeForUser(userID string) (*config.ExchangeConfig, error) {
	exchanges, err := s.database.GetExchanges(userID)
	if err != nil || len(exchanges) == 0 {
		return nil, fmt.Errorf("请先配置交易所")
	}
	for _, ex := range exchanges {
		if ex.ID == "binance" && ex.Enabled {
			return ex, nil
		}
	}
	return nil, fmt.Errorf("请先启用币安交易所")
}

func (s *Server) refreshCopyTradeUnrealizedPnL(userID string, fTrader *trader.FuturesTrader) (*copyTradePnLRefreshResult, error) {
	positions, err := fTrader.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("获取持仓失败: %w", err)
	}
	result := &copyTradePnLRefreshResult{}
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
		pnl := copyTradePosFloat(pos["unRealizedProfit"])
		mark := copyTradePosFloat(pos["markPrice"])

		rows, err := s.database.DB().Query(`
			SELECT id, executed_qty FROM copy_trade_records
			WHERE user_id = ? AND symbol = ? AND position_side = ? AND status = 'OPEN'`,
			userID, symbol, posSide)
		if err != nil {
			continue
		}
		var ids []int
		var qtys []float64
		var totalQty float64
		for rows.Next() {
			var id int
			var eqty float64
			if err := rows.Scan(&id, &eqty); err != nil {
				continue
			}
			ids = append(ids, id)
			qtys = append(qtys, eqty)
			totalQty += eqty
		}
		rows.Close()

		for i, id := range ids {
			recordPnL := pnl
			if len(ids) > 1 && totalQty > 0 {
				recordPnL = pnl * (qtys[i] / totalQty)
			}
			res, err := s.database.DB().Exec(
				"UPDATE copy_trade_records SET total_pnl=? WHERE id=?",
				recordPnL, id)
			if err == nil {
				if n, _ := res.RowsAffected(); n > 0 {
					result.Updated++
					result.TotalUnrealizedPnL += recordPnL
				}
			}
			log.Printf("  ✓ 更新 %s %s 未实现盈亏 ID=%d: $%.2f (当前价: $%.2f)", symbol, posSide, id, recordPnL, mark)
		}
	}
	return result, nil
}

func (s *Server) handleRefreshCopyTradePnL(c *gin.Context) {
	userID := c.GetString("user_id")
	exchangeCfg, err := s.getBinanceExchangeForUser(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fTrader := trader.NewFuturesTrader(exchangeCfg.APIKey, exchangeCfg.SecretKey, userID, exchangeCfg.Testnet)
	result, err := s.refreshCopyTradeUnrealizedPnL(userID, fTrader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":              "盈亏已刷新",
		"updated":              result.Updated,
		"total_unrealized_pnl": result.TotalUnrealizedPnL,
	})
}
