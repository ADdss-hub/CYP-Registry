package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cyp-registry/registry/src/modules/webhook"
	"github.com/cyp-registry/registry/src/pkg/database"
)

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
