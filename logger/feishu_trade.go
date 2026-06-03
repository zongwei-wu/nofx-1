package logger

import (
	"fmt"
	"strings"
	"time"
)

// TradeNotifyStatus 交易通知状态
type TradeNotifyStatus string

const (
	TradeStatusSuccess  TradeNotifyStatus = "成功"
	TradeStatusFailed   TradeNotifyStatus = "失败"
	TradeStatusAIReject TradeNotifyStatus = "AI拒绝"
)

// TradeNotifyParams 飞书交易通知参数
type TradeNotifyParams struct {
	Source            string
	Status            TradeNotifyStatus
	Action            string
	Symbol            string
	Side              string
	PositionSide      string
	Qty               float64
	Price             float64
	Leverage          int
	OrderID           int64
	TraderOrNickname  string
	Reason            string
	Error             string
}

// FormatTradeNotify 格式化为飞书文本消息
func FormatTradeNotify(p TradeNotifyParams) string {
	var b strings.Builder
	b.WriteString("[NOFX 交易通知]\n")
	b.WriteString(fmt.Sprintf("时间: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	if p.Source != "" {
		b.WriteString(fmt.Sprintf("来源: %s\n", p.Source))
	}
	b.WriteString(fmt.Sprintf("状态: %s\n", p.Status))
	if p.TraderOrNickname != "" {
		b.WriteString(fmt.Sprintf("交易员/带单员: %s\n", p.TraderOrNickname))
	}
	if p.Action != "" {
		b.WriteString(fmt.Sprintf("动作: %s\n", p.Action))
	}
	if p.Symbol != "" {
		b.WriteString(fmt.Sprintf("交易对: %s\n", p.Symbol))
	}
	if p.Side != "" {
		b.WriteString(fmt.Sprintf("方向: %s\n", p.Side))
	}
	if p.PositionSide != "" {
		b.WriteString(fmt.Sprintf("持仓侧: %s\n", p.PositionSide))
	}
	if p.Qty > 0 {
		b.WriteString(fmt.Sprintf("数量: %.4f\n", p.Qty))
	}
	if p.Price > 0 {
		b.WriteString(fmt.Sprintf("价格: $%.4f\n", p.Price))
	}
	if p.Leverage > 0 {
		b.WriteString(fmt.Sprintf("杠杆: %dx\n", p.Leverage))
	}
	if p.OrderID > 0 {
		b.WriteString(fmt.Sprintf("订单ID: %d\n", p.OrderID))
	}
	if p.Reason != "" {
		b.WriteString(fmt.Sprintf("说明: %s\n", p.Reason))
	}
	if p.Error != "" {
		b.WriteString(fmt.Sprintf("错误: %s\n", p.Error))
	}
	return strings.TrimRight(b.String(), "\n")
}

// NotifyTrade 异步发送交易通知到飞书（非阻塞，未启用时 no-op）
func NotifyTrade(p TradeNotifyParams) {
	if !feishuTradeNotifyEnabled || feishuHook == nil || !feishuHook.enabled || feishuHook.sender == nil {
		return
	}
	feishuHook.sender.SendAsync(FormatTradeNotify(p))
}
