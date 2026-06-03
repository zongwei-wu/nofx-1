package logger

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// FeishuHook 实现logrus.Hook接口，将日志推送到飞书
type FeishuHook struct {
	sender  *FeishuSender
	levels  []logrus.Level
	enabled bool
}

// NewFeishuHook 创建飞书 Hook
func NewFeishuHook(config *FeishuConfig) (*FeishuHook, error) {
	if !config.Enabled {
		return &FeishuHook{enabled: false}, nil
	}

	if config.WebhookURL == "" {
		return nil, fmt.Errorf("飞书配置不完整: webhook_url不能为空")
	}

	// 创建发送器（使用默认参数）
	sender, err := NewFeishuSender(config.WebhookURL)
	if err != nil {
		return nil, fmt.Errorf("创建飞书发送器失败: %w", err)
	}

	hook := &FeishuHook{
		sender:  sender,
		levels:  config.GetLogrusLevels(),
		enabled: true,
	}

	return hook, nil
}

// Levels 返回需要触发的日志级别
func (h *FeishuHook) Levels() []logrus.Level {
	if !h.enabled {
		return []logrus.Level{}
	}
	return h.levels
}

// Fire 当日志触发时调用
func (h *FeishuHook) Fire(entry *logrus.Entry) error {
	if !h.enabled {
		return nil
	}

	// 格式化消息
	message := h.formatMessage(entry)

	// 异步发送（非阻塞）
	h.sender.SendAsync(message)

	return nil
}

// formatMessage 格式化日志消息为飞书格式
func (h *FeishuHook) formatMessage(entry *logrus.Entry) string {
	levelEmoji := h.getLevelEmoji(entry.Level)

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s [%s] NOFX 系统日志警报\n", levelEmoji, strings.ToUpper(entry.Level.String())))
	builder.WriteString(fmt.Sprintf("📝 消息: %s\n", entry.Message))

	// 字段信息
	if len(entry.Data) > 0 {
		builder.WriteString("📊 字段:\n")
		for key, value := range entry.Data {
			builder.WriteString(fmt.Sprintf("  · %s: %v\n", key, value))
		}
	}

	// 调用位置
	if entry.HasCaller() {
		file := entry.Caller.File
		if idx := strings.Index(file, "nofx/"); idx >= 0 {
			file = file[idx:]
		}
		builder.WriteString(fmt.Sprintf("📍 位置: %s:%d\n", file, entry.Caller.Line))
	} else {
		if _, file, line, ok := runtime.Caller(8); ok {
			if idx := strings.Index(file, "nofx/"); idx >= 0 {
				file = file[idx:]
			}
			builder.WriteString(fmt.Sprintf("📍 位置: %s:%d\n", file, line))
		}
	}

	// 时间戳
	builder.WriteString(fmt.Sprintf("🕐 时间: %s", entry.Time.Format("2006-01-02 15:04:05")))

	return builder.String()
}

// getLevelEmoji 获取日志级别对应的emoji
func (h *FeishuHook) getLevelEmoji(level logrus.Level) string {
	switch level {
	case logrus.PanicLevel:
		return "🔴"
	case logrus.FatalLevel:
		return "🔴"
	case logrus.ErrorLevel:
		return "🟠"
	case logrus.WarnLevel:
		return "🟡"
	case logrus.InfoLevel:
		return "🟢"
	case logrus.DebugLevel:
		return "🔵"
	default:
		return "⚪"
	}
}

// Stop 停止Hook（优雅关闭）
func (h *FeishuHook) Stop() {
	if h.enabled && h.sender != nil {
		h.sender.Stop()
	}
}
