package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cyp-registry/registry/src/modules/webhook"
	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/google/uuid"
)

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

