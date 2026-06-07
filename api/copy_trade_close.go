package api

import (
	"fmt"
	"log"
	"nofx/logger"
	"nofx/trader"
)

type copyCloseOrder struct {
	Symbol       string
	Side         string
	PositionSide string
	ExecutedQty  float64
	AvgPrice     float64
	OrderTime    int64
}

type copyCloseResult struct {
	success bool
	skipped bool
	failed  bool
}

// leadCloseOurPositionSide 将带单平仓订单映射为跟单账户应平的持仓方向
func leadCloseOurPositionSide(positionSide, side string) string {
	if positionSide == "BOTH" {
		if side == "BUY" {
			return "SHORT"
		}
		return "LONG"
	}
	return positionSide
}

func (s *Server) sumOpenCopyQty(userID, portfolioID, symbol, positionSide string) float64 {
	var total float64
	_ = s.database.DB().QueryRow(`
		SELECT COALESCE(SUM(executed_qty), 0) FROM copy_trade_records
		WHERE user_id=? AND portfolio_id=? AND symbol=? AND position_side=? AND status='OPEN'`,
		userID, portfolioID, symbol, positionSide).Scan(&total)
	return total
}

func (s *Server) alreadyCopiedClose(userID string, leadOrderTime int64) bool {
	var count int
	_ = s.database.DB().QueryRow(
		`SELECT COUNT(*) FROM copy_trade_run_events
		 WHERE user_id=? AND lead_order_time=? AND action='copied_close'`,
		userID, leadOrderTime).Scan(&count)
	return count > 0
}

func (s *Server) closeCopyTradeRecords(userID, portfolioID, symbol, positionSide string, closePrice float64) {
	rows, err := s.database.DB().Query(`
		SELECT id, executed_qty, avg_price FROM copy_trade_records
		WHERE user_id=? AND portfolio_id=? AND symbol=? AND position_side=? AND status='OPEN'`,
		userID, portfolioID, symbol, positionSide)
	if err != nil {
		log.Printf("⚠️ 查询跟单平仓记录失败 %s %s: %v", symbol, positionSide, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var qty, entryPrice float64
		if err := rows.Scan(&id, &qty, &entryPrice); err != nil {
			continue
		}
		realizedPnL := computeCopyTradeRealizedPnL(positionSide, qty, entryPrice, closePrice)
		_, err := s.database.DB().Exec(`
			UPDATE copy_trade_records
			SET status='CLOSED', close_time=CURRENT_TIMESTAMP, close_price=?, total_pnl=?
			WHERE id=?`,
			closePrice, realizedPnL, id)
		if err != nil {
			log.Printf("⚠️ 更新跟单平仓记录失败 %s %s id=%d: %v", symbol, positionSide, id, err)
		}
	}
}

func (s *Server) processCopyTradeClose(
	fTrader *trader.FuturesTrader,
	userID, portfolioID, nickname string,
	runID int64,
	order copyCloseOrder,
	source string,
) copyCloseResult {
	posSide := leadCloseOurPositionSide(order.PositionSide, order.Side)
	ourSt := s.ourStatusForSymbol(userID, portfolioID, order.Symbol)

	if s.alreadyCopiedClose(userID, order.OrderTime) {
		if runID > 0 {
			s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, order.Symbol, "already_copied",
				order.Side, order.PositionSide, ourSt, "该平仓订单已跟过", order.OrderTime)
		}
		return copyCloseResult{skipped: true}
	}

	openQty := s.sumOpenCopyQty(userID, portfolioID, order.Symbol, posSide)
	if openQty < 0.001 {
		if runID > 0 {
			s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, order.Symbol, "skipped",
				order.Side, order.PositionSide, ourSt, "无对应跟单持仓，跳过平仓", order.OrderTime)
		}
		return copyCloseResult{skipped: true}
	}

	var tradeResult map[string]interface{}
	var tradeErr error
	if posSide == "LONG" {
		tradeResult, tradeErr = fTrader.CloseLong(order.Symbol, openQty)
	} else {
		tradeResult, tradeErr = fTrader.CloseShort(order.Symbol, openQty)
	}

	if tradeErr != nil {
		if runID > 0 {
			s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, order.Symbol, "close_failed",
				order.Side, order.PositionSide, ourSt, tradeErr.Error(), order.OrderTime)
		}
		logger.NotifyTrade(logger.TradeNotifyParams{
			Source: source, Status: logger.TradeStatusFailed, Action: "跟单平仓",
			Symbol: order.Symbol, Side: order.Side, PositionSide: posSide,
			Qty: openQty, TraderOrNickname: nickname, Error: tradeErr.Error(),
		})
		return copyCloseResult{failed: true}
	}

	actualQty := openQty
	actualPrice := order.AvgPrice
	if tradeResult != nil {
		if q, ok := tradeResult["executedQty"].(float64); ok && q > 0 {
			actualQty = q
		}
		if p, ok := tradeResult["avgPrice"].(float64); ok && p > 0 {
			actualPrice = p
		}
	}

	s.closeCopyTradeRecords(userID, portfolioID, order.Symbol, posSide, actualPrice)

	if runID > 0 {
		s.insertCopyTradeRunEvent(runID, userID, portfolioID, nickname, order.Symbol, "copied_close",
			order.Side, order.PositionSide, "CLOSED",
			fmt.Sprintf("已跟单平仓 %.4f张 @ $%.2f", actualQty, actualPrice), order.OrderTime)
	}

	logger.NotifyTrade(logger.TradeNotifyParams{
		Source: source, Status: logger.TradeStatusSuccess, Action: "跟单平仓",
		Symbol: order.Symbol, Side: order.Side, PositionSide: posSide,
		Qty: actualQty, Price: actualPrice, TraderOrNickname: nickname,
	})
	log.Printf("  ✓ [%s] 已跟单平仓 %s %s %.4f张 @ $%.2f", nickname, order.Symbol, posSide, actualQty, actualPrice)
	return copyCloseResult{success: true}
}
