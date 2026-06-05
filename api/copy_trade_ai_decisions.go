package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"nofx/logger"
	"sort"
	"time"
)

// copyTradeAIDecisionParams 跟单 AI 风控分析写入参数
type copyTradeAIDecisionParams struct {
	UserID         string
	RunID          int64
	PortfolioID    string
	Nickname       string
	Symbol         string
	InputPrompt    string
	AIResponseRaw  string
	DecisionJSON   string
	Feasible       bool
	RecommendedQty float64
	Reasoning      string
	Suggestion     string
	ActionTaken    string
	Success        bool
	AITraderID     string
	AITraderName   string
}

func (s *Server) insertCopyTradeAIDecision(p copyTradeAIDecisionParams) {
	feasible := 0
	if p.Feasible {
		feasible = 1
	}
	success := 0
	if p.Success {
		success = 1
	}
	_, err := s.database.DB().Exec(`
		INSERT INTO copy_trade_ai_decisions
		(user_id, run_id, portfolio_id, nickname, symbol, input_prompt, ai_response_raw,
		 decision_json, feasible, recommended_qty, reasoning, suggestion, action_taken, success,
		 ai_trader_id, ai_trader_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.UserID, p.RunID, p.PortfolioID, p.Nickname, p.Symbol, p.InputPrompt, p.AIResponseRaw,
		p.DecisionJSON, feasible, p.RecommendedQty, p.Reasoning, p.Suggestion, p.ActionTaken, success,
		p.AITraderID, p.AITraderName)
	if err != nil {
		log.Printf("⚠️ 写入跟单 AI 决策失败: %v", err)
	}
}

type copyTradeAIDecisionRow struct {
	ID             int64
	UserID         string
	RunID          int64
	PortfolioID    string
	Nickname       string
	Symbol         string
	InputPrompt    string
	AIResponseRaw  string
	DecisionJSON   string
	Feasible       bool
	RecommendedQty float64
	Reasoning      string
	Suggestion     string
	ActionTaken    string
	Success        bool
	AITraderID     string
	AITraderName   string
	CreatedAt      time.Time
}

func (s *Server) getLatestCopyTradeAIDecisions(userID string, limit int) ([]copyTradeAIDecisionRow, error) {
	rows, err := s.database.DB().Query(`
		SELECT id, user_id, run_id, portfolio_id, nickname, symbol, input_prompt, ai_response_raw,
		       decision_json, feasible, recommended_qty, reasoning, suggestion, action_taken, success,
		       ai_trader_id, ai_trader_name, created_at
		FROM copy_trade_ai_decisions
		WHERE user_id = ?
		ORDER BY id DESC
		LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []copyTradeAIDecisionRow
	for rows.Next() {
		var r copyTradeAIDecisionRow
		var feasible, success int
		var createdAt sql.NullString
		if err := rows.Scan(&r.ID, &r.UserID, &r.RunID, &r.PortfolioID, &r.Nickname, &r.Symbol,
			&r.InputPrompt, &r.AIResponseRaw, &r.DecisionJSON, &feasible, &r.RecommendedQty,
			&r.Reasoning, &r.Suggestion, &r.ActionTaken, &success,
			&r.AITraderID, &r.AITraderName, &createdAt); err != nil {
			continue
		}
		r.Feasible = feasible != 0
		r.Success = success != 0
		if createdAt.Valid {
			if t, err := time.Parse(time.RFC3339, createdAt.String); err == nil {
				r.CreatedAt = t
			} else if t, err := time.Parse("2006-01-02 15:04:05", createdAt.String); err == nil {
				r.CreatedAt = t
			}
		}
		if r.CreatedAt.IsZero() {
			r.CreatedAt = time.Now()
		}
		out = append(out, r)
	}
	return out, nil
}

type decisionRecordResponse struct {
	logger.DecisionRecord
	Source        string                 `json:"source,omitempty"`
	CopyTradeMeta map[string]interface{} `json:"copy_trade_meta,omitempty"`
}

func copyTradeRowToDecisionResponse(row copyTradeAIDecisionRow) decisionRecordResponse {
	action := "copy_reject"
	switch row.ActionTaken {
	case "copied_open":
		if row.Symbol != "" {
			action = "copy_follow"
		}
	case "open_failed":
		action = "copy_follow_failed"
	case "ai_error", "parse_failed":
		action = "copy_ai_error"
	case "pending_confirm", "pending_open":
		action = "copy_pending"
	case "ai_rejected":
		action = "copy_reject"
	}

	cotTrace := row.AIResponseRaw
	if cotTrace == "" && row.Reasoning != "" {
		cotTrace = row.Reasoning
		if row.Suggestion != "" {
			cotTrace += "\n\n建议: " + row.Suggestion
		}
	}

	decisionJSON := row.DecisionJSON
	if decisionJSON == "" && row.Reasoning != "" {
		b, _ := json.Marshal(map[string]interface{}{
			"feasible":        row.Feasible,
			"reasoning":       row.Reasoning,
			"suggestion":      row.Suggestion,
			"recommended_qty": row.RecommendedQty,
			"action_taken":    row.ActionTaken,
		})
		decisionJSON = string(b)
	}

	execLog := []string{fmt.Sprintf("跟单风控: %s", row.ActionTaken)}
	if row.Reasoning != "" {
		execLog = append(execLog, row.Reasoning)
	}

	meta := map[string]interface{}{
		"nickname":     row.Nickname,
		"portfolio_id": row.PortfolioID,
		"action_taken": row.ActionTaken,
		"feasible":     row.Feasible,
		"run_id":       row.RunID,
	}
	if row.AITraderID != "" {
		meta["ai_trader_id"] = row.AITraderID
	}
	if row.AITraderName != "" {
		meta["ai_trader_name"] = row.AITraderName
	}

	return decisionRecordResponse{
		DecisionRecord: logger.DecisionRecord{
			Timestamp:    row.CreatedAt,
			CycleNumber:  0,
			InputPrompt:  row.InputPrompt,
			CoTTrace:     cotTrace,
			DecisionJSON: decisionJSON,
			Decisions: []logger.DecisionAction{
				{
					Action:    action,
					Symbol:    row.Symbol,
					Quantity:  row.RecommendedQty,
					Timestamp: row.CreatedAt,
					Success:   row.Success,
				},
			},
			ExecutionLog: execLog,
			Success:      row.Success,
			ErrorMessage: func() string {
				if row.Success {
					return ""
				}
				if row.Reasoning != "" {
					return row.Reasoning
				}
				return row.ActionTaken
			}(),
		},
		Source:        "copy_trade",
		CopyTradeMeta: meta,
	}
}

func mergeLatestDecisionResponses(fileRecords []*logger.DecisionRecord, copyRows []copyTradeAIDecisionRow, limit int) []decisionRecordResponse {
	merged := make([]decisionRecordResponse, 0, len(fileRecords)+len(copyRows))

	for _, rec := range fileRecords {
		if rec == nil {
			continue
		}
		merged = append(merged, decisionRecordResponse{
			DecisionRecord: *rec,
			Source:         "auto_trader",
		})
	}
	for _, row := range copyRows {
		merged = append(merged, copyTradeRowToDecisionResponse(row))
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Timestamp.After(merged[j].Timestamp)
	})

	if len(merged) > limit {
		merged = merged[:limit]
	}
	return merged
}
