// Package service Webhook服务层
// 提供Webhook管理和事件分发功能
package service

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cyp-registry/registry/src/modules/webhook"
	"github.com/cyp-registry/registry/src/pkg/database"
	"gorm.io/gorm"
)

// WebhookService Webhook服务
type WebhookService struct {
	httpClient  *http.Client // HTTP客户端
	workerCount int          // 并发Worker数量
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	WorkerCount int           // 并发Worker数量
	SendTimeout time.Duration // 发送超时时间
}

// NewWebhookService 创建新的Webhook服务
func NewWebhookService(cfg *ServiceConfig) *WebhookService {
	workerCount := cfg.WorkerCount
	if workerCount <= 0 {
		workerCount = 5
	}

	sendTimeout := cfg.SendTimeout
	if sendTimeout == 0 {
		sendTimeout = 30 * time.Second
	}

	return &WebhookService{
		httpClient: &http.Client{
			Timeout: sendTimeout,
		},
		workerCount: workerCount,
	}
}

// GetLastPushMeta 查询指定仓库/标签最近一次 push 事件的时间与用户名。
// 该方法用于在 Registry 标签列表中补充“推送时间/推送用户”信息，修复前端始终显示“未知时间 / 未知用户”的问题。
func (s *WebhookService) GetLastPushMeta(repository, tag string) (*time.Time, string, error) {
	if repository == "" || tag == "" {
		return nil, "", nil
	}

	var eventModel webhook.WebhookEventModel
	err := database.GetDB().
		Where("repository = ? AND tag = ? AND event_type = ?", repository, tag, webhook.EventTypePush).
		Order("timestamp DESC").
		First(&eventModel).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 没有找到 push 事件时视为无额外信息，不报错，交给前端展示“未知”
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("failed to query last push event: %w", err)
	}

	// 使用事件时间作为推送时间，用户名来自事件记录
	t := eventModel.Timestamp
	return &t, eventModel.Username, nil
}

// CreateWebhook 创建Webhook
func (s *WebhookService) CreateWebhook(req *webhook.CreateWebhookRequest, userID string) (*webhook.Webhook, error) {
	// 创建Webhook
	webhookObj, err := webhook.NewWebhook(req, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}

	// 保存到数据库
	var model webhook.WebhookModel
	if err := model.FromWebhook(webhookObj); err != nil {
		return nil, fmt.Errorf("failed to convert webhook: %w", err)
	}

	if err := database.GetDB().Create(&model).Error; err != nil {
		return nil, fmt.Errorf("failed to save webhook: %w", err)
	}

	log.Printf("Webhook created: %s (%s)", webhookObj.Name, webhookObj.WebhookID)
	return webhookObj, nil
}

// UpdateWebhook 更新Webhook
func (s *WebhookService) UpdateWebhook(webhookID string, req *webhook.UpdateWebhookRequest) (*webhook.Webhook, error) {
	var model webhook.WebhookModel
	if err := database.GetDB().Where("webhook_id = ?", webhookID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("webhook not found: %s", webhookID)
		}
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}

	webhookObj, err := model.ToWebhook()
	if err != nil {
		return nil, fmt.Errorf("failed to convert webhook: %w", err)
	}

	// 更新字段
	if req.Name != "" {
		webhookObj.Name = req.Name
	}
	if req.Description != "" {
		webhookObj.Description = req.Description
	}
	if req.URL != "" {
		webhookObj.URL = req.URL
	}
	if req.Secret != "" {
		webhookObj.Secret = req.Secret
	}
	if len(req.Events) > 0 {
		webhookObj.Events = req.Events
	}
	if req.IsActive != nil {
		webhookObj.IsActive = *req.IsActive
	}
	if req.Headers != nil {
		webhookObj.Headers = req.Headers
	}
	if req.RetryPolicy != nil {
		webhookObj.RetryPolicy = req.RetryPolicy
	}

	webhookObj.UpdatedAt = time.Now()

	// 保存到数据库
	if err := model.FromWebhook(webhookObj); err != nil {
		return nil, fmt.Errorf("failed to convert webhook: %w", err)
	}
	if err := database.GetDB().Save(&model).Error; err != nil {
		return nil, fmt.Errorf("failed to update webhook: %w", err)
	}

	log.Printf("Webhook updated: %s", webhookID)
	return webhookObj, nil
}

// DeleteWebhook 删除Webhook
func (s *WebhookService) DeleteWebhook(webhookID string) error {
	if err := database.GetDB().Where("webhook_id = ?", webhookID).Delete(&webhook.WebhookModel{}).Error; err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	log.Printf("Webhook deleted: %s", webhookID)
	return nil
}

// GetWebhook 获取Webhook
func (s *WebhookService) GetWebhook(webhookID string) (*webhook.Webhook, error) {
	var model webhook.WebhookModel
	if err := database.GetDB().Where("webhook_id = ?", webhookID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("webhook not found: %s", webhookID)
		}
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	wh, err := model.ToWebhook()
	if err != nil {
		return nil, fmt.Errorf("failed to convert webhook: %w", err)
	}

	// 为单个 Webhook 附带统计信息（成功/失败次数），用于详情弹窗展示真实触发统计
	var successCount, failedCount int64
	db := database.GetDB()
	db.Model(&webhook.WebhookDeliveryModel{}).
		Where("webhook_id = ? AND response_status >= ? AND response_status < ?", webhookID, 200, 300).
		Count(&successCount)
	db.Model(&webhook.WebhookDeliveryModel{}).
		Where("webhook_id = ? AND (response_status < ? OR response_status >= ?)", webhookID, 200, 300).
		Count(&failedCount)
	wh.SuccessCount = successCount
	wh.FailedCount = failedCount

	return wh, nil
}

// ListWebhooks 列出项目所有Webhook
func (s *WebhookService) ListWebhooks(projectID string) ([]*webhook.Webhook, error) {
	var models []webhook.WebhookModel
	if err := database.GetDB().Where("project_id = ?", projectID).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	result := make([]*webhook.Webhook, 0, len(models))
	if len(models) == 0 {
		return result, nil
	}

	// 预先按 webhook_id 聚合统计成功/失败次数，避免在循环中多次查询数据库
	type aggRow struct {
		WebhookID    string
		SuccessCount int64
		FailedCount  int64
	}
	var aggResults []aggRow
	db := database.GetDB()
	if err := db.
		Model(&webhook.WebhookDeliveryModel{}).
		Select("webhook_id, "+
			"SUM(CASE WHEN response_status >= 200 AND response_status < 300 THEN 1 ELSE 0 END) AS success_count, "+
			"SUM(CASE WHEN response_status < 200 OR response_status >= 300 THEN 1 ELSE 0 END) AS failed_count").
		Where("webhook_id IN (?)", db.Model(&webhook.WebhookModel{}).Select("webhook_id").Where("project_id = ?", projectID)).
		Group("webhook_id").
		Scan(&aggResults).Error; err != nil {
		// 统计失败不影响主流程，仅记录日志
		log.Printf("failed to aggregate webhook delivery stats: %v", err)
	}

	statsByID := make(map[string]aggRow, len(aggResults))
	for _, row := range aggResults {
		statsByID[row.WebhookID] = row
	}

	for _, model := range models {
		wh, err := model.ToWebhook()
		if err != nil {
			continue
		}
		if stat, ok := statsByID[model.WebhookID]; ok {
			wh.SuccessCount = stat.SuccessCount
			wh.FailedCount = stat.FailedCount
		}
		result = append(result, wh)
	}

	return result, nil
}
