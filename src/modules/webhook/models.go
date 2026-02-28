// Package webhook Webhook事件通知模块
// 提供镜像仓库事件推送和外部系统集成功能
package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// 事件类型常量
const (
	EventTypePush      = "push"       // 镜像推送
	EventTypePull      = "pull"       // 镜像拉取
	EventTypeDelete    = "delete"     // 镜像删除
	EventTypeScan      = "scan"       // 漏洞扫描完成
	EventTypeScanFail  = "scan_fail"  // 漏洞扫描失败
	EventTypePolicy    = "policy"     // 策略变更
	EventTypeMember    = "member"     // 成员变更
)

// 事件状态常量
const (
	EventStatusPending   = "pending"    // 待处理
	EventStatusSent      = "sent"       // 已发送
	EventStatusFailed    = "failed"     // 发送失败
	EventStatusRetrying  = "retrying"   // 重试中
)

// Webhook Webhook配置
type Webhook struct {
	WebhookID       string            `json:"webhookId"`
	ProjectID       string            `json:"projectId"`
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	URL             string            `json:"url"`
	Secret          string            `json:"secret"`          // 签名密钥
	Events          []string          `json:"events"`          // 订阅的事件类型
	IsActive        bool              `json:"isActive"`        // 是否启用
	Headers         map[string]string `json:"headers"`         // 自定义请求头
	RetryPolicy     *RetryPolicy      `json:"retryPolicy"`     // 重试策略
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
	CreatedBy       string            `json:"createdBy"`
	LastTriggeredAt *time.Time        `json:"lastTriggeredAt,omitempty"`
	// 统计字段：根据 webhook_deliveries 实时汇总，不直接持久化到 webhooks 表
	SuccessCount    int64             `json:"successCount,omitempty"`
	FailedCount     int64             `json:"failedCount,omitempty"`
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries    int           `json:"maxRetries"`    // 最大重试次数
	RetryDelay    time.Duration `json:"retryDelay"`    // 重试间隔
	BackoffMultiplier float64   `json:"backoffMultiplier"` // 退避乘数
	MaxDelay      time.Duration `json:"maxDelay"`      // 最大延迟
}

// DefaultRetryPolicy 默认重试策略
var DefaultRetryPolicy = &RetryPolicy{
	MaxRetries:         3,
	RetryDelay:         5 * time.Second,
	BackoffMultiplier:  2.0,
	MaxDelay:           1 * time.Minute,
}

// WebhookEvent Webhook事件
type WebhookEvent struct {
	EventID       string          `json:"eventId"`
	WebhookID     string          `json:"webhookId"`
	EventType     string          `json:"eventType"`
	ProjectID     string          `json:"projectId"`
	Repository    string          `json:"repository"`
	Tag           string          `json:"tag,omitempty"`
	Digest        string          `json:"digest,omitempty"`
	UserID        string          `json:"userId,omitempty"`
	Username      string          `json:"username,omitempty"`
	Payload       json.RawMessage `json:"payload"`
	Timestamp     time.Time       `json:"timestamp"`
	Status        string          `json:"status"`
	Attempts      int             `json:"attempts"`
	LastAttemptAt *time.Time      `json:"lastAttemptAt,omitempty"`
	NextRetryAt   *time.Time      `json:"nextRetryAt,omitempty"`
	Error         string          `json:"error,omitempty"`
}

// WebhookDelivery Webhook投递记录
type WebhookDelivery struct {
	DeliveryID    string          `json:"deliveryId"`
	EventID       string          `json:"eventId"`
	WebhookID     string          `json:"webhookId"`
	RequestURL    string          `json:"requestUrl"`
	RequestMethod string          `json:"requestMethod"`
	RequestHeaders map[string]string `json:"requestHeaders"`
	RequestBody   json.RawMessage `json:"requestBody"`
	ResponseStatus int            `json:"responseStatus"`
	ResponseHeaders map[string]string `json:"responseHeaders"`
	ResponseBody  string          `json:"responseBody"`
	Duration      int64           `json:"duration"` // 耗时(毫秒)
	DeliveredAt   time.Time       `json:"deliveredAt"`
	Error         string          `json:"error,omitempty"`
}

// WebhookStatistics Webhook统计
type WebhookStatistics struct {
	TotalWebhooks     int64 `json:"totalWebhooks"`
	ActiveWebhooks    int64 `json:"activeWebhooks"`
	TotalEvents       int64 `json:"totalEvents"`
	DeliveredEvents   int64 `json:"deliveredEvents"`
	FailedEvents      int64 `json:"failedEvents"`
	SuccessRate       float64 `json:"successRate"`
}

// EventPayload 事件载荷基类
type EventPayload struct {
	Action       string    `json:"action"`
	Timestamp    time.Time `json:"timestamp"`
	Actor        *Actor    `json:"actor"`
}

// Actor 事件执行者
type Actor struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

// PushEventPayload 推送事件载荷
type PushEventPayload struct {
	EventPayload
	Repository   string `json:"repository"`
	Tag          string `json:"tag"`
	Digest       string `json:"digest"`
	ImageSize    int64  `json:"imageSize"`
}

// DeleteEventPayload 删除事件载荷
type DeleteEventPayload struct {
	EventPayload
	Repository   string `json:"repository"`
	Tag          string `json:"tag"`
	Digest       string `json:"digest"`
}

// ScanEventPayload 扫描事件载荷
type ScanEventPayload struct {
	EventPayload
	Repository      string  `json:"repository"`
	Tag             string  `json:"tag"`
	Digest          string  `json:"digest"`
	ScanStatus      string  `json:"scanStatus"`
	CriticalCount   int     `json:"criticalCount"`
	HighCount       int     `json:"highCount"`
	ReportURL       string  `json:"reportUrl"`
}

// CreateWebhookRequest 创建Webhook请求
type CreateWebhookRequest struct {
	ProjectID   string            `json:"projectId"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	URL         string            `json:"url"`
	Secret      string            `json:"secret,omitempty"`
	Events      []string          `json:"events"`
	Headers     map[string]string `json:"headers,omitempty"`
	RetryPolicy *RetryPolicy      `json:"retryPolicy,omitempty"`
}

// UpdateWebhookRequest 更新Webhook请求
type UpdateWebhookRequest struct {
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	URL         string            `json:"url,omitempty"`
	Secret      string            `json:"secret,omitempty"`
	Events      []string          `json:"events,omitempty"`
	IsActive    *bool             `json:"isActive,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	RetryPolicy *RetryPolicy      `json:"retryPolicy,omitempty"`
}

// TestWebhookRequest 测试Webhook请求
type TestWebhookRequest struct {
	EventType string          `json:"eventType"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

// Validate 验证Webhook配置
func (w *Webhook) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("webhook name is required")
	}

	if w.URL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	// 验证URL格式
	if !strings.HasPrefix(w.URL, "http://") && !strings.HasPrefix(w.URL, "https://") {
		return fmt.Errorf("webhook URL must start with http:// or https://")
	}

	// 验证事件类型
	validEvents := map[string]bool{
		EventTypePush: true, EventTypePull: true, EventTypeDelete: true,
		EventTypeScan: true, EventTypeScanFail: true, EventTypePolicy: true,
		EventTypeMember: true,
	}

	for _, event := range w.Events {
		if !validEvents[event] {
			return fmt.Errorf("invalid event type: %s", event)
		}
	}

	return nil
}

// GenerateSignature 生成HMAC-SHA256签名
func (w *Webhook) GenerateSignature(body []byte) string {
	if w.Secret == "" {
		return ""
	}

	h := hmac.New(sha256.New, []byte(w.Secret))
	h.Write(body)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

// VerifySignature 验证签名
func VerifySignature(body []byte, signature, secret string) bool {
	if secret == "" || signature == "" {
		return false
	}

	// 去掉前缀 "sha256="
	expectedSig := strings.TrimPrefix(signature, "sha256=")

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	actualSig := hex.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(expectedSig), []byte(actualSig))
}

// NewWebhook 创建新的Webhook
func NewWebhook(req *CreateWebhookRequest, userID string) (*Webhook, error) {
	now := time.Now()
	webhook := &Webhook{
		WebhookID:   uuid.New().String(),
		ProjectID:   req.ProjectID,
		Name:        req.Name,
		Description: req.Description,
		URL:         req.URL,
		Secret:      req.Secret,
		Events:      req.Events,
		IsActive:    true,
		Headers:     req.Headers,
		RetryPolicy: req.RetryPolicy,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   userID,
	}

	// 设置默认重试策略
	if webhook.RetryPolicy == nil {
		webhook.RetryPolicy = DefaultRetryPolicy
	}

	// 验证配置
	if err := webhook.Validate(); err != nil {
		return nil, err
	}

	return webhook, nil
}

// CreateEvent 创建新的Webhook事件
func CreateEvent(webhook *Webhook, eventType, projectID, repository string, payload json.RawMessage, actor *Actor) *WebhookEvent {
	event := &WebhookEvent{
		EventID:   uuid.New().String(),
		WebhookID: webhook.WebhookID,
		EventType: eventType,
		ProjectID: projectID,
		Repository: repository,
		Payload:   payload,
		Timestamp: time.Now(),
		Status:    EventStatusPending,
		Attempts:  0,
	}

	if actor != nil {
		event.UserID = actor.UserID
		event.Username = actor.Username
	}

	return event
}

// ShouldRetry 判断是否应该重试
func (e *WebhookEvent) ShouldRetry() bool {
	if e.Status != EventStatusFailed {
		return false
	}

	// 检查是否超过最大重试次数
	if e.Attempts >= 3 { // 默认最大重试次数
		return false
	}

	// 检查是否到达重试时间
	if e.NextRetryAt != nil && time.Now().Before(*e.NextRetryAt) {
		return false
	}

	return true
}

// CalculateNextRetry 计算下次重试时间
func (e *WebhookEvent) CalculateNextRetry(backoffMultiplier float64, maxDelay time.Duration) time.Duration {
	baseDelay := 5 * time.Second
	delay := time.Duration(float64(baseDelay) * pow(backoffMultiplier, float64(e.Attempts)))

	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// pow 计算幂
func pow(base, exp float64) float64 {
	result := 1.0
	for exp > 0 {
		if exp == 1 {
			result *= base
		}
		exp--
	}
	return result
}

// BuildRequest 构建HTTP请求
func (d *WebhookDelivery) BuildRequest(event *WebhookEvent, webhook *Webhook) (*http.Request, error) {
	req, err := http.NewRequest("POST", webhook.URL, strings.NewReader(string(event.Payload)))
	if err != nil {
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CYP-Registry-Webhook/1.0")
	req.Header.Set("X-Webhook-Event", event.EventType)
	req.Header.Set("X-Webhook-ID", event.EventID)
	req.Header.Set("X-Webhook-Delivery", d.DeliveryID)

	// 添加签名
	if webhook.Secret != "" {
		signature := webhook.GenerateSignature([]byte(event.Payload))
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// 添加自定义头
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	d.RequestURL = webhook.URL
	d.RequestMethod = "POST"
	d.RequestHeaders = make(map[string]string)
	for key := range req.Header {
		d.RequestHeaders[key] = req.Header.Get(key)
	}
	d.RequestBody = event.Payload

	return req, nil
}

// SendDelivery 发送Webhook并记录结果
func SendDelivery(req *http.Request, delivery *WebhookDelivery) error {
	startTime := time.Now()

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		delivery.Error = err.Error()
		delivery.Duration = time.Since(startTime).Milliseconds()
		return err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// 读取响应体失败，记录错误但继续处理
		delivery.Error = fmt.Sprintf("failed to read response body: %v", err)
	}
	delivery.ResponseStatus = resp.StatusCode
	delivery.ResponseBody = string(body)
	delivery.Duration = time.Since(startTime).Milliseconds()

	// 设置响应头
	delivery.ResponseHeaders = make(map[string]string)
	for key := range resp.Header {
		delivery.ResponseHeaders[key] = resp.Header.Get(key)
	}

	// 判断是否成功
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		delivery.Error = ""
		return nil
	}

	delivery.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	return fmt.Errorf("webhook delivery failed: %d", resp.StatusCode)
}

