// Package service 提供管理员相关业务逻辑
package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/modules/admin/dto"
	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/cyp-registry/registry/src/pkg/models"
)

// Service 管理员服务
type Service struct{}

// NewService 创建管理员服务
func NewService() *Service {
	return &Service{}
}

// ListAuditLogs 获取审计日志列表
func (s *Service) ListAuditLogs(
	ctx context.Context,
	page, pageSize int,
	userID *uuid.UUID,
	action, resource string,
	startTime, endTime *time.Time,
	keyword string,
) ([]dto.AuditLogResponse, int64, error) {
	if database.DB == nil {
		return nil, 0, fmt.Errorf("数据库未初始化")
	}

	// 构建查询
	query := database.DB.Model(&models.AuditLog{})

	// 应用筛选条件
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}
	if keyword != "" {
		// 关键词搜索：在details字段中搜索
		keyword = "%" + strings.TrimSpace(keyword) + "%"
		query = query.Where("details ILIKE ?", keyword)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取日志总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	var logs []models.AuditLog
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("获取日志列表失败: %w", err)
	}

	// 转换为响应DTO
	responses := make([]dto.AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = dto.ToAuditLogResponse(&log)
	}

	return responses, total, nil
}

// GetSystemConfig 获取系统配置
func (s *Service) GetSystemConfig() (*dto.SystemConfigResponse, error) {
	// 从全局配置获取
	cfg := config.Get()
	if cfg == nil {
		return nil, fmt.Errorf("配置未初始化")
	}

	// 检查HTTPS是否启用（通过检查请求是否使用TLS，这里我们通过环境变量或配置判断）
	// 注意：HTTPS实际是在Nginx层面配置的，这里只是展示配置信息
	httpsEnabled := false
	sslCertPath := ""
	sslKeyPath := ""
	sslProtocols := []string{"TLSv1.2", "TLSv1.3"}
	httpRedirect := true

	// 从环境变量读取HTTPS相关配置（如果存在）
	if os.Getenv("HTTPS_ENABLED") == "true" || os.Getenv("HTTPS_ENABLED") == "1" {
		httpsEnabled = true
	}
	if path := os.Getenv("SSL_CERTIFICATE_PATH"); path != "" {
		sslCertPath = path
	}
	if path := os.Getenv("SSL_CERTIFICATE_KEY_PATH"); path != "" {
		sslKeyPath = path
	}

	return &dto.SystemConfigResponse{
		HTTPS: dto.HTTPSConfig{
			Enabled:               httpsEnabled,
			SSLCertificatePath:    sslCertPath,
			SSLCertificateKeyPath: sslKeyPath,
			SSLProtocols:          sslProtocols,
			HTTPRedirect:          httpRedirect,
		},
		CORS: dto.CORSConfigResponse{
			AllowedOrigins: cfg.Security.CORS.AllowedOrigins,
			AllowedMethods: cfg.Security.CORS.AllowedMethods,
			AllowedHeaders: cfg.Security.CORS.AllowedHeaders,
		},
		RateLimit: dto.RateLimitConfigResponse{
			Enabled:           cfg.Security.RateLimit.Enabled,
			RequestsPerSecond: cfg.Security.RateLimit.RequestsPerSecond,
			Burst:             cfg.Security.RateLimit.Burst,
		},
	}, nil
}

// UpdateSystemConfig 更新系统配置
func (s *Service) UpdateSystemConfig(req *dto.UpdateSystemConfigRequest) error {
	cfg := config.Get()
	if cfg == nil {
		return fmt.Errorf("配置未初始化")
	}

	// 更新CORS配置
	if req.CORS != nil {
		cfg.Security.CORS.AllowedOrigins = req.CORS.AllowedOrigins
		if len(req.CORS.AllowedMethods) > 0 {
			cfg.Security.CORS.AllowedMethods = req.CORS.AllowedMethods
		}
		if len(req.CORS.AllowedHeaders) > 0 {
			cfg.Security.CORS.AllowedHeaders = req.CORS.AllowedHeaders
		}
	}

	// 更新速率限制配置
	if req.RateLimit != nil {
		cfg.Security.RateLimit.Enabled = req.RateLimit.Enabled
		cfg.Security.RateLimit.RequestsPerSecond = req.RateLimit.RequestsPerSecond
		cfg.Security.RateLimit.Burst = req.RateLimit.Burst
	}

	// 注意：这里只是更新内存中的配置，实际配置应该保存到配置文件或环境变量
	// 生产环境建议通过环境变量或配置文件管理，这里仅提供读取和临时更新功能
	// HTTPS配置需要在Nginx层面配置，这里不做实际修改

	return nil
}
