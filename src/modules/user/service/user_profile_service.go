// Package service 提供用户认证相关业务逻辑
// 遵循《全平台通用用户认证设计规范》
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/pkg/cache"
	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/models"
)

// NotificationSettings 用户通知设置（服务内部结构）
type NotificationSettings struct {
	EmailEnabled         bool   `json:"email_enabled"`
	ScanCompleted        bool   `json:"scan_completed"`
	SecurityAlerts       bool   `json:"security_alerts"`
	WebhookNotifications bool   `json:"webhook_notifications"`
	Digest               string `json:"digest"`
	NotificationEmail    string `json:"notification_email"`
}

// defaultNotificationSettings 默认的通知设置
func defaultNotificationSettings() *NotificationSettings {
	return &NotificationSettings{
		EmailEnabled:         true,
		ScanCompleted:        true,
		SecurityAlerts:       true,
		WebhookNotifications: true,
		Digest:               "realtime",
		NotificationEmail:    "",
	}
}

// GetUserByID 根据ID获取用户
func (s *Service) GetUserByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if database.DB == nil {
		return nil, errors.ErrDatabaseError
	}

	var user models.User
	result := database.DB.Where("id = ?", userID).
		Where("deleted_at IS NULL").
		First(&user)

	if result.Error != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *Service) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	if database.DB == nil {
		return nil, errors.ErrDatabaseError
	}

	var user models.User
	result := database.DB.Where("username = ?", username).
		Where("deleted_at IS NULL").
		First(&user)

	if result.Error != nil {
		return nil, ErrUserNotFound
	}

	return &user, nil
}

// UpdateUser 更新用户信息
func (s *Service) UpdateUser(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) error {
	if database.DB == nil {
		return errors.ErrDatabaseError
	}

	// GORM 的 Updates 方法可以接受 map，键名可以是结构体字段名（如 "Avatar"）或数据库字段名（如 "avatar"）
	// 为了确保兼容性，我们将所有键名统一转换为结构体字段名（首字母大写）
	// GORM 会自动将结构体字段名转换为数据库字段名（snake_case）
	dbUpdates := make(map[string]interface{})
	for k, v := range updates {
		// 将小写字段名转换为首字母大写的结构体字段名
		// 例如：avatar -> Avatar, nickname -> Nickname
		structFieldName := strings.ToUpper(k[:1]) + k[1:]
		dbUpdates[structFieldName] = v
	}

	result := database.DB.Model(&models.User{}).
		Where("id = ? AND deleted_at IS NULL", userID).
		Updates(dbUpdates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// DeleteUser 软删除用户
func (s *Service) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	if database.DB == nil {
		return errors.ErrDatabaseError
	}

	result := database.DB.Model(&models.User{}).
		Where("id = ?", userID).
		Update("deleted_at", time.Now())

	if result.Error != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"delete_user","user_id":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), userID.String(), result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		log.Printf(`{"timestamp":"%s","level":"warn","module":"user","operation":"delete_user","user_id":"%s","error":"user not found"}`, time.Now().Format(time.RFC3339), userID.String())
		return ErrUserNotFound
	}

	log.Printf(`{"timestamp":"%s","level":"info","module":"user","operation":"delete_user","user_id":"%s"}`, time.Now().Format(time.RFC3339), userID.String())
	return nil
}

// ListUsers 列出用户（管理员）
func (s *Service) ListUsers(ctx context.Context, page, pageSize int, keyword string) ([]models.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var users []models.User
	var total int64

	q := database.DB.Model(&models.User{}).Where("deleted_at IS NULL")
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("username ILIKE ? OR email ILIKE ? OR nickname ILIKE ?", like, like, like)
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetNotificationSettings 获取用户通知设置（从缓存中读取，不存在时返回默认值）
func (s *Service) GetNotificationSettings(ctx context.Context, userID uuid.UUID) (*NotificationSettings, error) {
	// 如果缓存未初始化，直接返回默认值，避免影响主流程
	if cache.Cache == nil {
		return defaultNotificationSettings(), nil
	}

	key := fmt.Sprintf("user:notification:%s", userID.String())

	exists, err := cache.Exists(ctx, key)
	if err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"get_notification_settings","user_id":"%s","error":"failed to check cache: %v"}`, time.Now().Format(time.RFC3339), userID.String(), err)
		return nil, err
	}
	if !exists {
		return defaultNotificationSettings(), nil
	}

	raw, err := cache.Get(ctx, key)
	if err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"get_notification_settings","user_id":"%s","error":"failed to get from cache: %v"}`, time.Now().Format(time.RFC3339), userID.String(), err)
		return nil, err
	}

	var settings NotificationSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		// 解析失败时返回默认值，避免前端卡死
		log.Printf(`{"timestamp":"%s","level":"warn","module":"user","operation":"get_notification_settings","user_id":"%s","error":"failed to unmarshal settings: %v"}`, time.Now().Format(time.RFC3339), userID.String(), err)
		return defaultNotificationSettings(), nil
	}

	// 兜底处理 Digest 字段
	if settings.Digest == "" {
		settings.Digest = "realtime"
	}

	return &settings, nil
}

// UpdateNotificationSettings 更新用户通知设置（写入缓存）
func (s *Service) UpdateNotificationSettings(ctx context.Context, userID uuid.UUID, settings *NotificationSettings) error {
	// 缓存未初始化时直接返回成功，避免影响主流程
	if cache.Cache == nil {
		log.Printf(`{"timestamp":"%s","level":"warn","module":"user","operation":"update_notification_settings","user_id":"%s","error":"cache not initialized"}`, time.Now().Format(time.RFC3339), userID.String())
		return nil
	}

	// 兜底处理 Digest
	if settings.Digest == "" {
		settings.Digest = "realtime"
	}

	data, err := json.Marshal(settings)
	if err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"update_notification_settings","user_id":"%s","error":"failed to marshal settings: %v"}`, time.Now().Format(time.RFC3339), userID.String(), err)
		return err
	}

	key := fmt.Sprintf("user:notification:%s", userID.String())
	// 不设置过期时间（0 表示不过期），由业务显式管理
	err = cache.Set(ctx, key, string(data), 0)
	if err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"update_notification_settings","user_id":"%s","error":"failed to save to cache: %v"}`, time.Now().Format(time.RFC3339), userID.String(), err)
		return err
	}

	log.Printf(`{"timestamp":"%s","level":"info","module":"user","operation":"update_notification_settings","user_id":"%s","email_enabled":%t,"scan_completed":%t,"security_alerts":%t,"webhook_notifications":%t,"digest":"%s","notification_email":"%s"}`,
		time.Now().Format(time.RFC3339),
		userID.String(),
		settings.EmailEnabled,
		settings.ScanCompleted,
		settings.SecurityAlerts,
		settings.WebhookNotifications,
		settings.Digest,
		maskEmail(settings.NotificationEmail))
	return nil
}
