// Package webhook 数据库模型定义
// 提供Webhook相关的数据库持久化模型
package webhook

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/cyp-registry/registry/src/pkg/database"
	"gorm.io/gorm"
)

// WebhookModel Webhook数据库模型
type WebhookModel struct {
	WebhookID       string         `gorm:"type:varchar(36);primaryKey" json:"webhookId"`
	ProjectID       string         `gorm:"type:varchar(36);index;not null" json:"projectId"`
	Name            string         `gorm:"type:varchar(255);not null" json:"name"`
	Description     string         `gorm:"type:text" json:"description"`
	URL             string         `gorm:"type:varchar(512);not null" json:"url"`
	Secret          string         `gorm:"type:varchar(255)" json:"secret"`
	Events          StringArray    `gorm:"type:text" json:"events"`
	IsActive        bool           `gorm:"default:true;index" json:"isActive"`
	Headers         StringMap      `gorm:"type:text" json:"headers"`
	RetryPolicyJSON string         `gorm:"type:text" json:"-"` // 存储重试策略JSON
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	CreatedBy       string         `gorm:"type:varchar(36);index" json:"createdBy"`
	LastTriggeredAt *time.Time     `json:"lastTriggeredAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

// TableName 指定表名
func (WebhookModel) TableName() string {
	return "webhooks"
}

// WebhookEventModel Webhook事件数据库模型
type WebhookEventModel struct {
	EventID       string         `gorm:"type:varchar(36);primaryKey" json:"eventId"`
	WebhookID     string         `gorm:"type:varchar(36);index;not null" json:"webhookId"`
	EventType     string         `gorm:"type:varchar(50);index;not null" json:"eventType"`
	ProjectID     string         `gorm:"type:varchar(36);index" json:"projectId"`
	Repository    string         `gorm:"type:varchar(512)" json:"repository"`
	Tag           string         `gorm:"type:varchar(255)" json:"tag"`
	Digest        string         `gorm:"type:varchar(128)" json:"digest"`
	UserID        string         `gorm:"type:varchar(36)" json:"userId"`
	Username      string         `gorm:"type:varchar(255)" json:"username"`
	Payload       string         `gorm:"type:text" json:"payload"`
	Timestamp     time.Time      `gorm:"autoCreateTime;index" json:"timestamp"`
	Status        string         `gorm:"type:varchar(50);default:'pending';index" json:"status"`
	Attempts      int            `gorm:"default:0" json:"attempts"`
	LastAttemptAt *time.Time     `json:"lastAttemptAt"`
	NextRetryAt   *time.Time     `json:"nextRetryAt"`
	Error         string         `gorm:"type:text" json:"error"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

// TableName 指定表名
func (WebhookEventModel) TableName() string {
	return "webhook_events"
}

// WebhookDeliveryModel Webhook投递记录数据库模型
type WebhookDeliveryModel struct {
	DeliveryID      string         `gorm:"type:varchar(36);primaryKey" json:"deliveryId"`
	EventID         string         `gorm:"type:varchar(36);index;not null" json:"eventId"`
	WebhookID       string         `gorm:"type:varchar(36);index;not null" json:"webhookId"`
	RequestURL      string         `gorm:"type:varchar(512)" json:"requestUrl"`
	RequestMethod   string         `gorm:"type:varchar(10)" json:"requestMethod"`
	RequestHeaders  StringMap      `gorm:"type:text" json:"requestHeaders"`
	RequestBody     string         `gorm:"type:text" json:"requestBody"`
	ResponseStatus  int            `gorm:"default:0" json:"responseStatus"`
	ResponseHeaders StringMap      `gorm:"type:text" json:"responseHeaders"`
	ResponseBody    string         `gorm:"type:text" json:"responseBody"`
	Duration        int64          `json:"duration"`
	DeliveredAt     time.Time      `gorm:"autoCreateTime;index" json:"deliveredAt"`
	Error           string         `gorm:"type:text" json:"error"`
	CreatedAt       time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deletedAt"`
}

// TableName 指定表名
func (WebhookDeliveryModel) TableName() string {
	return "webhook_deliveries"
}

// StringArray 字符串数组类型
type StringArray []string

// Value 实现driver.Valuer接口
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "[]", nil
	}
	return json.Marshal(a)
}

// Scan 实现sql.Scanner接口
func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	return json.Unmarshal(bytes, a)
}

// StringMap 字符串映射类型
type StringMap map[string]string

// Value 实现driver.Valuer接口
func (m StringMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	return json.Marshal(m)
}

// Scan 实现sql.Scanner接口
func (m *StringMap) Scan(value interface{}) error {
	if value == nil {
		*m = make(map[string]string)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// ToWebhook 转换为业务模型
func (m *WebhookModel) ToWebhook() (*Webhook, error) {
	webhook := &Webhook{
		WebhookID:       m.WebhookID,
		ProjectID:       m.ProjectID,
		Name:            m.Name,
		Description:     m.Description,
		URL:             m.URL,
		Secret:          m.Secret,
		Events:          []string(m.Events),
		IsActive:        m.IsActive,
		Headers:         map[string]string(m.Headers),
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		CreatedBy:       m.CreatedBy,
		LastTriggeredAt: m.LastTriggeredAt,
	}

	// 解析重试策略
	if m.RetryPolicyJSON != "" {
		var retryPolicy RetryPolicy
		if err := json.Unmarshal([]byte(m.RetryPolicyJSON), &retryPolicy); err == nil {
			webhook.RetryPolicy = &retryPolicy
		} else {
			webhook.RetryPolicy = DefaultRetryPolicy
		}
	} else {
		webhook.RetryPolicy = DefaultRetryPolicy
	}

	return webhook, nil
}

// FromWebhook 从业务模型创建数据库模型
func (m *WebhookModel) FromWebhook(webhook *Webhook) error {
	m.WebhookID = webhook.WebhookID
	m.ProjectID = webhook.ProjectID
	m.Name = webhook.Name
	m.Description = webhook.Description
	m.URL = webhook.URL
	m.Secret = webhook.Secret
	m.Events = StringArray(webhook.Events)
	m.IsActive = webhook.IsActive
	m.Headers = StringMap(webhook.Headers)
	m.CreatedAt = webhook.CreatedAt
	m.UpdatedAt = webhook.UpdatedAt
	m.CreatedBy = webhook.CreatedBy
	m.LastTriggeredAt = webhook.LastTriggeredAt

	// 序列化重试策略
	if webhook.RetryPolicy != nil {
		policyBytes, err := json.Marshal(webhook.RetryPolicy)
		if err == nil {
			m.RetryPolicyJSON = string(policyBytes)
		}
	}

	return nil
}

// ToWebhookEvent 转换为业务模型
func (m *WebhookEventModel) ToWebhookEvent() *WebhookEvent {
	return &WebhookEvent{
		EventID:       m.EventID,
		WebhookID:     m.WebhookID,
		EventType:     m.EventType,
		ProjectID:     m.ProjectID,
		Repository:    m.Repository,
		Tag:           m.Tag,
		Digest:        m.Digest,
		UserID:        m.UserID,
		Username:      m.Username,
		Payload:       json.RawMessage(m.Payload),
		Timestamp:     m.Timestamp,
		Status:        m.Status,
		Attempts:      m.Attempts,
		LastAttemptAt: m.LastAttemptAt,
		NextRetryAt:   m.NextRetryAt,
		Error:         m.Error,
	}
}

// FromWebhookEvent 从业务模型创建数据库模型
func (m *WebhookEventModel) FromWebhookEvent(event *WebhookEvent) {
	m.EventID = event.EventID
	m.WebhookID = event.WebhookID
	m.EventType = event.EventType
	m.ProjectID = event.ProjectID
	m.Repository = event.Repository
	m.Tag = event.Tag
	m.Digest = event.Digest
	m.UserID = event.UserID
	m.Username = event.Username
	m.Payload = string(event.Payload)
	m.Timestamp = event.Timestamp
	m.Status = event.Status
	m.Attempts = event.Attempts
	m.LastAttemptAt = event.LastAttemptAt
	m.NextRetryAt = event.NextRetryAt
	m.Error = event.Error
}

// ToWebhookDelivery 转换为业务模型
func (m *WebhookDeliveryModel) ToWebhookDelivery() *WebhookDelivery {
	return &WebhookDelivery{
		DeliveryID:      m.DeliveryID,
		EventID:         m.EventID,
		WebhookID:       m.WebhookID,
		RequestURL:      m.RequestURL,
		RequestMethod:   m.RequestMethod,
		RequestHeaders:  map[string]string(m.RequestHeaders),
		RequestBody:     json.RawMessage(m.RequestBody),
		ResponseStatus:  m.ResponseStatus,
		ResponseHeaders: map[string]string(m.ResponseHeaders),
		ResponseBody:    m.ResponseBody,
		Duration:        m.Duration,
		DeliveredAt:     m.DeliveredAt,
		Error:           m.Error,
	}
}

// FromWebhookDelivery 从业务模型创建数据库模型
func (m *WebhookDeliveryModel) FromWebhookDelivery(delivery *WebhookDelivery) {
	m.DeliveryID = delivery.DeliveryID
	m.EventID = delivery.EventID
	m.WebhookID = delivery.WebhookID
	m.RequestURL = delivery.RequestURL
	m.RequestMethod = delivery.RequestMethod
	m.RequestHeaders = StringMap(delivery.RequestHeaders)
	m.RequestBody = string(delivery.RequestBody)
	m.ResponseStatus = delivery.ResponseStatus
	m.ResponseHeaders = StringMap(delivery.ResponseHeaders)
	m.ResponseBody = delivery.ResponseBody
	m.Duration = delivery.Duration
	m.DeliveredAt = delivery.DeliveredAt
	m.Error = delivery.Error
}

// InitWebhookDatabase 初始化Webhook数据库表
func InitWebhookDatabase() error {
	db := database.GetDB()
	return db.AutoMigrate(
		&WebhookModel{},
		&WebhookEventModel{},
		&WebhookDeliveryModel{},
	)
}
