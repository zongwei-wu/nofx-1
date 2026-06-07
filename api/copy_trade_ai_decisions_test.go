package api

import (
	"testing"
	"time"
)

func TestCopyTradeRowToDecisionResponse_AITraderMeta(t *testing.T) {
	row := copyTradeAIDecisionRow{
		ID:             1,
		UserID:         "user-1",
		Nickname:       "LeadA",
		PortfolioID:    "pf-1",
		Symbol:         "BTCUSDT",
		Reasoning:      "资金充足",
		ActionTaken:    "pending_confirm",
		Feasible:       true,
		RecommendedQty: 0.5,
		Success:        true,
		AITraderID:     "trader-copy-1",
		AITraderName:   "跟单分析员",
		CreatedAt:      time.Now(),
	}

	resp := copyTradeRowToDecisionResponse(row)
	if resp.Source != "copy_trade" {
		t.Fatalf("expected copy_trade source, got %s", resp.Source)
	}
	if resp.CopyTradeMeta["ai_trader_id"] != "trader-copy-1" {
		t.Fatalf("missing ai_trader_id: %+v", resp.CopyTradeMeta)
	}
	if resp.CopyTradeMeta["ai_trader_name"] != "跟单分析员" {
		t.Fatalf("missing ai_trader_name: %+v", resp.CopyTradeMeta)
	}
	if resp.Decisions[0].Action != "copy_pending" {
		t.Fatalf("expected copy_pending action, got %s", resp.Decisions[0].Action)
	}
}

func TestParseLeadOperationFromPrompt(t *testing.T) {
	prompt := "## 跟单请求\n- 带单员: LeadA\n- 操作: 买入开多\n"
	if got := parseLeadOperationFromPrompt(prompt); got != "开多" {
		t.Fatalf("expected 开多, got %q", got)
	}
}

func TestCopyTradeRowToDecisionResponse_LeadOperationMeta(t *testing.T) {
	row := copyTradeAIDecisionRow{
		Nickname:      "LeadA",
		Symbol:        "ETHUSDT",
		ActionTaken:   "ai_error",
		LeadOperation: "开多",
		CreatedAt:     time.Now(),
	}
	resp := copyTradeRowToDecisionResponse(row)
	if resp.CopyTradeMeta["lead_operation"] != "开多" {
		t.Fatalf("expected lead_operation 开多, got %+v", resp.CopyTradeMeta)
	}
	if len(resp.ExecutionLog) == 0 || resp.ExecutionLog[0] != "跟随带单操作: 开多" {
		t.Fatalf("expected lead op in execution log, got %v", resp.ExecutionLog)
	}
}

func TestCopyTradeRowToDecisionResponse_RejectedAction(t *testing.T) {
	row := copyTradeAIDecisionRow{
		ActionTaken: "ai_rejected",
		Success:     false,
		CreatedAt:   time.Now(),
	}
	resp := copyTradeRowToDecisionResponse(row)
	if resp.Decisions[0].Action != "copy_reject" {
		t.Fatalf("expected copy_reject, got %s", resp.Decisions[0].Action)
	}
}

func TestInsertAndGetCopyTradeAIDecisionWithAITrader(t *testing.T) {
	db, userID, _ := setupCopyTradeSettingsDB(t)
	defer db.Close()

	s := &Server{database: db}
	s.insertCopyTradeAIDecision(copyTradeAIDecisionParams{
		UserID:         userID,
		RunID:          0,
		PortfolioID:    "pf-1",
		Nickname:       "LeadB",
		Symbol:         "ETHUSDT",
		InputPrompt:    "test prompt",
		AIResponseRaw:  `{"feasible":true}`,
		DecisionJSON:   `{"feasible":true}`,
		Feasible:       true,
		RecommendedQty: 0.2,
		Reasoning:      "ok",
		ActionTaken:    "pending_confirm",
		Success:        true,
		AITraderID:     "trader-copy-1",
		AITraderName:   "跟单分析员",
	})

	rows, err := s.getLatestCopyTradeAIDecisions(userID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("expected at least one row")
	}
	if rows[0].AITraderID != "trader-copy-1" {
		t.Fatalf("expected ai_trader_id, got %q", rows[0].AITraderID)
	}
	if rows[0].AITraderName != "跟单分析员" {
		t.Fatalf("expected ai_trader_name, got %q", rows[0].AITraderName)
	}
}
