// Package audit 提供审计日志记录与清理能力
// 对应数据模型见 src/pkg/models/models.go 中的 AuditLog
// 清理策略与说明见 docs/日志清理机制说明.md
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/cyp-registry/registry/src/pkg/models"
)

// Record 记录一条成功的审计日志
// action: 操作类型，例如 "list_tags" / "get_manifest"
// resource: 资源类型，例如 "image"
// resourceID: 资源ID，可为空
// userID: 用户ID，可为空（匿名/未认证请求）
// ip: 客户端IP
// userAgent: User-Agent
// details: 额外的详情信息（将序列化为JSON字符串）
func Record(
	ctx context.Context,
	action string,
	resource string,
	resourceID *uuid.UUID,
	userID *uuid.UUID,
	ip string,
	userAgent string,
	details map[string]interface{},
) {
	_ = recordInternal(ctx, action, resource, resourceID, userID, ip, userAgent, details, "success", nil)
}

// RecordError 记录一条失败的审计日志
// err 会被安全地写入 details.error 字段中，便于排查问题
func RecordError(
	ctx context.Context,
	action string,
	resource string,
	resourceID *uuid.UUID,
	userID *uuid.UUID,
	ip string,
	userAgent string,
	err error,
	details map[string]interface{},
) {
	_ = recordInternal(ctx, action, resource, resourceID, userID, ip, userAgent, details, "error", err)
}

func recordInternal(
	ctx context.Context,
	action string,
	resource string,
	resourceID *uuid.UUID,
	userID *uuid.UUID,
	ip string,
	userAgent string,
	details map[string]interface{},
	status string,
	err error,
) error {
	if database.DB == nil {
		// 数据库未初始化时直接跳过，避免影响主流程
		return fmt.Errorf("database not initialized")
	}

	if details == nil {
		details = make(map[string]interface{})
	}
	if err != nil {
		// 将错误信息附加到 details 中，但不覆盖已有 error 字段
		if _, exists := details["error"]; !exists {
			details["error"] = err.Error()
		}
	}

	detailsBytes, marshalErr := json.Marshal(details)
	if marshalErr != nil {
		// 即使详情序列化失败，也不要阻断主流程；仅记录最小信息
		detailsBytes = []byte(`{"marshal_error":"` + marshalErr.Error() + `"}`)
	}

	var uid *uuid.UUID
	if userID != nil && *userID != uuid.Nil {
		tmp := *userID
		uid = &tmp
	}

	var rid *uuid.UUID
	if resourceID != nil && *resourceID != uuid.Nil {
		tmp := *resourceID
		rid = &tmp
	}

	logEntry := &models.AuditLog{
		UserID:     uid,
		Action:     action,
		Resource:   resource,
		ResourceID: rid,
		IP:         ip,
		UserAgent:  userAgent,
		Details:    string(detailsBytes),
		Status:     status,
	}

	return database.DB.WithContext(ctx).Create(logEntry).Error
}

// GetOldLogCount 返回超过指定保留天数的审计日志数量
// retentionDays <= 0 时按 15 天处理
func GetOldLogCount(retentionDays int) (int64, error) {
	if database.DB == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	if retentionDays <= 0 {
		retentionDays = 15
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	var count int64
	if err := database.DB.Model(&models.AuditLog{}).
		Where("created_at < ?", cutoff).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetLogCount 返回当前审计日志总数
func GetLogCount() (int64, error) {
	if database.DB == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	var count int64
	if err := database.DB.Model(&models.AuditLog{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CleanupOldLogs 清理超过指定保留天数的审计日志
// 默认保留 15 天，执行逻辑：
//
//	DELETE FROM registry_audit_logs WHERE created_at < NOW() - INTERVAL '{retentionDays} days';
//
// 注意：数据库中还提供了 cleanup_old_audit_logs() 函数用于 90 天归档+删除；
// 这里的 Go 实现用于应用侧定制化保留策略（例如 15/30 天），不负责归档。
func CleanupOldLogs(retentionDays int) error {
	if database.DB == nil {
		return fmt.Errorf("database not initialized")
	}
	if retentionDays <= 0 {
		retentionDays = 15
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	return database.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("created_at < ?", cutoff).
			Delete(&models.AuditLog{}).Error; err != nil {
			return fmt.Errorf("failed to cleanup old audit logs: %w", err)
		}
		return nil
	})
}
