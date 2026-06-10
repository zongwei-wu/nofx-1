package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Capabilities(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "ApiKey test-key" {
			t.Fatalf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/api/gateway/capabilities" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Capabilities{Version: "1"})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-key")
	resp, err := c.Capabilities(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resp.Version != "1" {
		t.Fatalf("got version %s", resp.Version)
	}
}

func TestClient_Trade(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req TradeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.Action != "open_long" {
			t.Fatalf("unexpected action: %s", req.Action)
		}
		_ = json.NewEncoder(w).Encode(TradeResponse{
			ExchangeID: "binance",
			Action:     req.Action,
			Symbol:     "BTCUSDT",
		})
	}))
	defer srv.Close()

	c := New(srv.URL, "test-key")
	resp, err := c.Trade(context.Background(), TradeRequest{
		ExchangeID: "binance",
		Action:     "open_long",
		Symbol:     "BTCUSDT",
		Quantity:   0.01,
		Leverage:   5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Symbol != "BTCUSDT" {
		t.Fatalf("got %s", resp.Symbol)
	}
}
