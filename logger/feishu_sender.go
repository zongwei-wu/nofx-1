package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// FeishuSender 飞书消息发送器（异步，通过 Webhook 机器人）
type FeishuSender struct {
	webhookURL    string
	httpClient    *http.Client
	msgChan       chan string
	retryCount    int
	retryInterval time.Duration
	wg            sync.WaitGroup
	stopChan      chan struct{}
	once          sync.Once
}

// feishuMessage 飞书消息体
type feishuMessage struct {
	MsgType string        `json:"msg_type"`
	Content feishuContent `json:"content"`
}

type feishuContent struct {
	Text string `json:"text"`
}

// feishuResponse 飞书响应
type feishuResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// NewFeishuSender 创建飞书发送器
// webhookURL: 飞书机器人 Webhook 地址，格式 https://open.feishu.cn/open-apis/bot/v2/hook/xxx
func NewFeishuSender(webhookURL string) (*FeishuSender, error) {
	if webhookURL == "" {
		return nil, fmt.Errorf("飞书webhook地址不能为空")
	}

	sender := &FeishuSender{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		msgChan:       make(chan string, 20),    // 缓冲区大小: 20
		retryCount:    3,                         // 重试次数: 3
		retryInterval: 3 * time.Second,           // 重试间隔: 3秒
		stopChan:      make(chan struct{}),
	}

	// 启动异步发送协程
	sender.Start()

	return sender, nil
}

// Start 启动异步发送协程
func (s *FeishuSender) Start() {
	s.wg.Add(1)
	go s.listenAndSend()
}

// SendAsync 异步发送消息（非阻塞）
func (s *FeishuSender) SendAsync(message string) {
	select {
	case s.msgChan <- message:
		// 成功写入缓冲区
	default:
		// 缓冲区满，丢弃消息
		fmt.Printf("[Feishu] 消息缓冲区已满，消息被丢弃\n")
	}
}

// listenAndSend 监听channel并发送消息
func (s *FeishuSender) listenAndSend() {
	defer s.wg.Done()

	for {
		select {
		case msg := <-s.msgChan:
			s.sendWithRetry(msg)
		case <-s.stopChan:
			// 清空缓冲区后退出
			for len(s.msgChan) > 0 {
				msg := <-s.msgChan
				s.sendWithRetry(msg)
			}
			return
		}
	}
}

// sendWithRetry 发送消息（带重试）
func (s *FeishuSender) sendWithRetry(message string) {
	var err error
	for i := 0; i < s.retryCount; i++ {
		err = s.send(message)
		if err == nil {
			return // 发送成功
		}

		// 重试前等待
		if i < s.retryCount-1 {
			time.Sleep(s.retryInterval)
		}
	}

	// 所有重试都失败
	if err != nil {
		fmt.Printf("[Feishu] 发送消息失败（已重试%d次）: %v\n", s.retryCount, err)
	}
}

// send 发送单条消息到飞书
func (s *FeishuSender) send(message string) error {
	msg := feishuMessage{
		MsgType: "text",
		Content: feishuContent{
			Text: message,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	req, err := http.NewRequest("POST", s.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("飞书返回非200状态码: %d", resp.StatusCode)
	}

	// 解析飞书响应
	var feishuResp feishuResponse
	if err := json.NewDecoder(resp.Body).Decode(&feishuResp); err != nil {
		// 响应解析失败不算错误，HTTP 200就认为发送成功
		return nil
	}

	if feishuResp.Code != 0 {
		return fmt.Errorf("飞书返回错误: code=%d, msg=%s", feishuResp.Code, feishuResp.Msg)
	}

	return nil
}

// Stop 停止发送器（优雅关闭）
func (s *FeishuSender) Stop() {
	s.once.Do(func() {
		close(s.stopChan)
		s.wg.Wait()
	})
}
