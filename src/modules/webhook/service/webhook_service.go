// Package service Webhook服务层
// 提供Webhook管理和事件分发功能
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cyp-registry/registry/src/modules/webhook"
	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/google/uuid"
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

// TriggerEvent 触发事件
func (s *WebhookService) TriggerEvent(eventType, projectID, repository string, payload interface{}, actor *webhook.Actor) error {
	// 从数据库查找订阅了该事件的项目Webhook
	var models []webhook.WebhookModel
	if err := database.GetDB().Where("project_id = ? AND is_active = ?", projectID, true).Find(&models).Error; err != nil {
		return fmt.Errorf("failed to find webhooks: %w", err)
	}

	var targetWebhooks []*webhook.Webhook
	for _, model := range models {
		wh, err := model.ToWebhook()
		if err != nil {
			continue
		}
		for _, event := range wh.Events {
			if event == eventType {
				targetWebhooks = append(targetWebhooks, wh)
				break
			}
		}
	}

	if len(targetWebhooks) == 0 {
		log.Printf("No active webhooks found for event %s in project %s", eventType, projectID)
		return nil
	}

	// 序列化载荷
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 为每个Webhook创建事件并保存到数据库
	for _, wh := range targetWebhooks {
		event := webhook.CreateEvent(wh, eventType, projectID, repository, payloadBytes, actor)

		// 保存事件到数据库
		var eventModel webhook.WebhookEventModel
		eventModel.FromWebhookEvent(event)
		if err := database.GetDB().Create(&eventModel).Error; err != nil {
			log.Printf("Failed to save webhook event: %v", err)
			continue
		}

		// 异步发送事件
		go s.sendEvent(context.Background(), event, wh)
	}

	log.Printf("Event %s triggered for %d webhooks", eventType, len(targetWebhooks))
	return nil
}

// PushPushEvent 推送镜像事件
func (s *WebhookService) PushPushEvent(projectID, repository, tag, digest string, imageSize int64, userID, username string) error {
	payload := &webhook.PushEventPayload{
		EventPayload: webhook.EventPayload{
			Action:    "push",
			Timestamp: time.Now(),
			Actor: &webhook.Actor{
				UserID:   userID,
				Username: username,
			},
		},
		Repository: repository,
		Tag:        tag,
		Digest:     digest,
		ImageSize:  imageSize,
	}

	actor := &webhook.Actor{
		UserID:   userID,
		Username: username,
	}

	// 向外部 Webhook 订阅者分发事件
	if err := s.TriggerEvent(webhook.EventTypePush, projectID, repository, payload, actor); err != nil {
		return err
	}

	// 同步推送一个简化事件给本地 SSE 订阅者（前端实时刷新镜像列表）
	webhook.PublishRegistryEvent(webhook.RegistryEvent{
		Type:       webhook.RegistryEventPush,
		Repository: repository,
		Tag:        tag,
		Digest:     digest,
		ProjectID:  projectID,
		Timestamp:  time.Now(),
	})

	return nil
}

// PushDeleteEvent 删除镜像事件
func (s *WebhookService) PushDeleteEvent(projectID, repository, tag, digest string, userID, username string) error {
	payload := &webhook.DeleteEventPayload{
		EventPayload: webhook.EventPayload{
			Action:    "delete",
			Timestamp: time.Now(),
			Actor: &webhook.Actor{
				UserID:   userID,
				Username: username,
			},
		},
		Repository: repository,
		Tag:        tag,
		Digest:     digest,
	}

	actor := &webhook.Actor{
		UserID:   userID,
		Username: username,
	}

	// 向外部 Webhook 订阅者分发事件
	if err := s.TriggerEvent(webhook.EventTypeDelete, projectID, repository, payload, actor); err != nil {
		return err
	}

	// 同步推送一个简化事件给本地 SSE 订阅者
	webhook.PublishRegistryEvent(webhook.RegistryEvent{
		Type:       webhook.RegistryEventDelete,
		Repository: repository,
		Tag:        tag,
		Digest:     digest,
		ProjectID:  projectID,
		Timestamp:  time.Now(),
	})

	return nil
}

// PushScanEvent 扫描完成事件
func (s *WebhookService) PushScanEvent(projectID, repository, tag, digest, scanStatus string, criticalCount, highCount int, userID, username string) error {
	payload := &webhook.ScanEventPayload{
		EventPayload: webhook.EventPayload{
			Action:    "scan_completed",
			Timestamp: time.Now(),
			Actor: &webhook.Actor{
				UserID:   userID,
				Username: username,
			},
		},
		Repository:    repository,
		Tag:           tag,
		Digest:        digest,
		ScanStatus:    scanStatus,
		CriticalCount: criticalCount,
		HighCount:     highCount,
		ReportURL:     fmt.Sprintf("/api/v1/scans/reports/%s", digest),
	}

	actor := &webhook.Actor{
		UserID:   userID,
		Username: username,
	}

	eventType := webhook.EventTypeScan
	if scanStatus == "failed" {
		eventType = webhook.EventTypeScanFail
	}

	return s.TriggerEvent(eventType, projectID, repository, payload, actor)
}

// SendEvents 发送所有待处理事件
func (s *WebhookService) SendEvents(ctx context.Context) {
	// 从数据库获取待处理事件
	var eventModels []webhook.WebhookEventModel
	if err := database.GetDB().Where("status = ? AND (next_retry_at IS NULL OR next_retry_at <= ?)", webhook.EventStatusPending, time.Now()).Find(&eventModels).Error; err != nil {
		log.Printf("Failed to get pending events: %v", err)
		return
	}

	for _, eventModel := range eventModels {
		event := eventModel.ToWebhookEvent()

		// 获取Webhook
		wh, err := s.GetWebhook(event.WebhookID)
		if err != nil {
			log.Printf("Failed to get webhook %s: %v", event.WebhookID, err)
			continue
		}

		s.sendEvent(ctx, event, wh)
	}
}

// sendEvent 发送单个事件
func (s *WebhookService) sendEvent(ctx context.Context, event *webhook.WebhookEvent, wh *webhook.Webhook) {
	if wh == nil {
		var err error
		wh, err = s.GetWebhook(event.WebhookID)
		if err != nil {
			log.Printf("Webhook not found: %s", event.WebhookID)
			return
		}
	}

	// 创建投递记录
	delivery := &webhook.WebhookDelivery{
		DeliveryID:  uuid.New().String(),
		EventID:     event.EventID,
		WebhookID:   event.WebhookID,
		DeliveredAt: time.Now(),
	}

	// 构建请求
	req, err := delivery.BuildRequest(event, wh)
	if err != nil {
		delivery.Error = fmt.Sprintf("failed to build request: %v", err)
		log.Printf("Failed to build webhook request: %v", err)
		s.recordDelivery(event, delivery)
		return
	}

	// 发送请求
	err = webhook.SendDelivery(req, delivery)
	if err != nil {
		event.Attempts++
		event.LastAttemptAt = &delivery.DeliveredAt

		// 检查是否需要重试
		if event.ShouldRetry() {
			event.Status = webhook.EventStatusRetrying
			delay := event.CalculateNextRetry(wh.RetryPolicy.BackoffMultiplier, wh.RetryPolicy.MaxDelay)
			nextRetry := time.Now().Add(delay)
			event.NextRetryAt = &nextRetry
		} else {
			event.Status = webhook.EventStatusFailed
			event.Error = delivery.Error
		}
	} else {
		event.Status = webhook.EventStatusSent
		now := time.Now()
		wh.LastTriggeredAt = &now

		// 更新Webhook的最后触发时间
		database.GetDB().Model(&webhook.WebhookModel{}).Where("webhook_id = ?", wh.WebhookID).Update("last_triggered_at", now)
	}

	// 更新事件状态
	var eventModel webhook.WebhookEventModel
	eventModel.FromWebhookEvent(event)
	database.GetDB().Model(&webhook.WebhookEventModel{}).Where("event_id = ?", event.EventID).Updates(map[string]interface{}{
		"status":          event.Status,
		"attempts":        event.Attempts,
		"last_attempt_at": event.LastAttemptAt,
		"next_retry_at":   event.NextRetryAt,
		"error":           event.Error,
	})

	s.recordDelivery(event, delivery)
}

// recordDelivery 记录投递历史
func (s *WebhookService) recordDelivery(event *webhook.WebhookEvent, delivery *webhook.WebhookDelivery) {
	var deliveryModel webhook.WebhookDeliveryModel
	deliveryModel.FromWebhookDelivery(delivery)
	if err := database.GetDB().Create(&deliveryModel).Error; err != nil {
		log.Printf("Failed to save delivery: %v", err)
	}
}

// GetDeliveries 获取事件投递历史
func (s *WebhookService) GetDeliveries(eventID string) ([]*webhook.WebhookDelivery, error) {
	var models []webhook.WebhookDeliveryModel
	if err := database.GetDB().Where("event_id = ?", eventID).Order("delivered_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get deliveries: %w", err)
	}

	deliveries := make([]*webhook.WebhookDelivery, 0, len(models))
	for _, model := range models {
		deliveries = append(deliveries, model.ToWebhookDelivery())
	}

	return deliveries, nil
}

// TestWebhook 测试Webhook
func (s *WebhookService) TestWebhook(webhookID string, req *webhook.TestWebhookRequest) (*webhook.WebhookDelivery, error) {
	wh, err := s.GetWebhook(webhookID)
	if err != nil {
		return nil, fmt.Errorf("webhook not found: %s", webhookID)
	}

	// 创建测试事件
	var payloadBytes []byte
	if req.Payload != nil {
		payloadBytes = req.Payload
	} else {
		testPayload := map[string]interface{}{
			"action":    "test",
			"timestamp": time.Now().Format(time.RFC3339),
			"message":   "This is a test webhook delivery",
		}
		payloadBytes, _ = json.Marshal(testPayload)
	}

	event := &webhook.WebhookEvent{
		EventID:   uuid.New().String(),
		WebhookID: webhookID,
		EventType: req.EventType,
		Timestamp: time.Now(),
		Status:    webhook.EventStatusPending,
		Payload:   payloadBytes,
	}

	// 将测试事件也写入事件表，确保统计口径一致（避免 deliveredEvents > totalEvents / successRate > 100%）
	{
		var eventModel webhook.WebhookEventModel
		eventModel.FromWebhookEvent(event)
		// 最佳努力：写入失败不影响测试投递本身
		_ = database.GetDB().Create(&eventModel).Error
	}

	// 创建投递记录
	delivery := &webhook.WebhookDelivery{
		DeliveryID:  uuid.New().String(),
		EventID:     event.EventID,
		WebhookID:   webhookID,
		DeliveredAt: time.Now(),
	}

	// 构建请求
	httpRequest, err := delivery.BuildRequest(event, wh)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// 发送请求
	err = webhook.SendDelivery(httpRequest, delivery)

	// 保存投递记录
	s.recordDelivery(event, delivery)

	if err != nil {
		return delivery, err
	}

	return delivery, nil
}

// GetStatistics 获取Webhook统计
func (s *WebhookService) GetStatistics() *webhook.WebhookStatistics {
	stats := &webhook.WebhookStatistics{}

	// 统计Webhook总数
	database.GetDB().Model(&webhook.WebhookModel{}).Count(&stats.TotalWebhooks)

	// 统计活跃Webhook
	database.GetDB().Model(&webhook.WebhookModel{}).Where("is_active = ?", true).Count(&stats.ActiveWebhooks)

	// 统计触发总数（以投递记录为准），仅统计“当前仍存在且未被软删除的 Webhook”对应的投递记录，
	// 避免用户删除 Webhook 后顶部统计卡片仍长期包含历史数据。
	db := database.GetDB()
	subQuery := db.Model(&webhook.WebhookModel{}).Select("webhook_id")

	db.Model(&webhook.WebhookDeliveryModel{}).
		Where("webhook_id IN (?)", subQuery).
		Count(&stats.TotalEvents)

	// 统计成功投递的事件
	db.Model(&webhook.WebhookDeliveryModel{}).
		Where("webhook_id IN (?) AND response_status >= ? AND response_status < ?", subQuery, 200, 300).
		Count(&stats.DeliveredEvents)

	// 统计失败的事件
	db.Model(&webhook.WebhookDeliveryModel{}).
		Where("webhook_id IN (?) AND (response_status < ? OR response_status >= ?)", subQuery, 200, 300).
		Count(&stats.FailedEvents)

	// 计算成功率
	if stats.TotalEvents > 0 {
		stats.SuccessRate = float64(stats.DeliveredEvents) / float64(stats.TotalEvents) * 100
	}

	return stats
}

// GetDeliveriesByWebhookID 根据Webhook ID获取投递历史
func (s *WebhookService) GetDeliveriesByWebhookID(webhookID string) ([]*webhook.WebhookDelivery, error) {
	var models []webhook.WebhookDeliveryModel
	if err := database.GetDB().Where("webhook_id = ?", webhookID).Order("delivered_at DESC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("failed to get deliveries: %w", err)
	}

	deliveries := make([]*webhook.WebhookDelivery, 0, len(models))
	for _, model := range models {
		deliveries = append(deliveries, model.ToWebhookDelivery())
	}

	return deliveries, nil
}

// StartEventWorker 启动事件处理Worker
func (s *WebhookService) StartEventWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SendEvents(ctx)
		}
	}
}
