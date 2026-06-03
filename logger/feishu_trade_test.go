package logger

import (
	"strings"
	"testing"
)

func TestFormatTradeNotify(t *testing.T) {
	msg := FormatTradeNotify(TradeNotifyParams{
		Source:           "跟单-自动监控",
		Status:           TradeStatusSuccess,
		Action:           "跟单开仓",
		Symbol:           "BTCUSDT",
		Side:             "BUY",
		PositionSide:     "LONG",
		Qty:              0.01,
		Price:            50000,
		TraderOrNickname: "测试带单员",
	})
	if !strings.Contains(msg, "[NOFX 交易通知]") {
		t.Fatalf("missing title: %s", msg)
	}
	if !strings.Contains(msg, "跟单-自动监控") || !strings.Contains(msg, "BTCUSDT") {
		t.Fatalf("missing fields: %s", msg)
	}
}

func TestNotifyTradeNoOpWhenDisabled(t *testing.T) {
	feishuTradeNotifyEnabled = false
	feishuHook = nil
	NotifyTrade(TradeNotifyParams{Source: "test", Status: TradeStatusSuccess})
}
