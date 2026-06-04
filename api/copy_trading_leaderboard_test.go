package api

import (
	"encoding/json"
	"testing"
)

func TestBinanceLeadListPayloadPNL(t *testing.T) {
	p := binanceLeadListPayloadPNL()
	if p["dataType"] != "PNL" || p["timeRange"] != "30D" || p["pageSize"] != 20 {
		t.Fatalf("unexpected payload: %+v", p)
	}
	if p["PAGE_SIZE"] != 20 || p["useAiRecommended"] != true {
		t.Fatal("PNL payload fields mismatch")
	}
}

func TestBinanceLeadListPayloadROI(t *testing.T) {
	p := binanceLeadListPayloadROI()
	if p["dataType"] != "ROI" || p["timeRange"] != "30D" || p["pageSize"] != 20 {
		t.Fatalf("unexpected payload: %+v", p)
	}
	if _, has := p["PAGE_SIZE"]; has {
		t.Fatal("ROI payload should not include PAGE_SIZE")
	}
	if p["useAiRecommended"] != true {
		t.Fatal("useAiRecommended should be true")
	}
}

func TestLeadsFromListData(t *testing.T) {
	raw := json.RawMessage(`{"list":[{"leadPortfolioId":"abc","nickname":"t1"}]}`)
	leads, err := leadsFromListData(raw)
	if err != nil || len(leads) != 1 {
		t.Fatalf("list parse: leads=%v err=%v", leads, err)
	}

	rec := json.RawMessage(`{"highestPnlLeads":[{"leadPortfolioId":"x"}],"highestRoiLeads":[{"leadPortfolioId":"y"}]}`)
	pnl, roi, ok := mergeLeaderboardFromRecommend(rec)
	if !ok || len(pnl) != 1 || len(roi) != 1 {
		t.Fatalf("recommend merge: pnl=%v roi=%v ok=%v", pnl, roi, ok)
	}
}

func TestBuildLeaderboardJSON(t *testing.T) {
	out, err := buildLeaderboardJSON(
		[]interface{}{map[string]interface{}{"leadPortfolioId": "1"}},
		[]interface{}{map[string]interface{}{"leadPortfolioId": "2"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		HighestPnlLeads []map[string]interface{} `json:"highestPnlLeads"`
		HighestRoiLeads []map[string]interface{} `json:"highestRoiLeads"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.HighestPnlLeads) != 1 || len(parsed.HighestRoiLeads) != 1 {
		t.Fatalf("unexpected: %+v", parsed)
	}
}
