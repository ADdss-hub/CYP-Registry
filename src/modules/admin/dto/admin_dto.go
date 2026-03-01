// Package dto 提供管理员相关数据传输对象
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/pkg/models"
)

// AuditLogResponse 审计日志响应
type AuditLogResponse struct {
	ID         uuid.UUID  `json:"id"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	Action     string     `json:"action"`
	Resource   string     `json:"resource"`
	ResourceID *uuid.UUID `json:"resource_id,omitempty"`
	IP         string     `json:"ip"`
	UserAgent  string     `json:"user_agent"`
	Details    string     `json:"details"`
	Status     string     `json:"status"`
	CreatedAt  string     `json:"created_at"`
}

// AuditLogListResponse 审计日志列表响应
type AuditLogListResponse struct {
	Logs      []AuditLogResponse `json:"logs"`
	Total     int64              `json:"total"`
	Page      int                `json:"page"`
	PageSize  int                `json:"page_size"`
	TotalPage int                `json:"total_page"`
}

// ToAuditLogResponse 将模型转换为响应DTO
func ToAuditLogResponse(log *models.AuditLog) AuditLogResponse {
	resp := AuditLogResponse{
		ID:        log.ID,
		Action:    log.Action,
		Resource:  log.Resource,
		IP:        log.IP,
		UserAgent: log.UserAgent,
		Details:   log.Details,
		Status:    log.Status,
		CreatedAt: log.CreatedAt.Format(time.RFC3339),
	}

	if log.UserID != nil {
		resp.UserID = log.UserID
	}
	if log.ResourceID != nil {
		resp.ResourceID = log.ResourceID
	}

	return resp
}

// SystemConfigResponse 系统配置响应
type SystemConfigResponse struct {
	HTTPS     HTTPSConfig             `json:"https"`
	CORS      CORSConfigResponse      `json:"cors"`
	RateLimit RateLimitConfigResponse `json:"rate_limit"`
}

// HTTPSConfig HTTPS配置
type HTTPSConfig struct {
	Enabled               bool     `json:"enabled"`
	SSLCertificatePath    string   `json:"ssl_certificate_path"`
	SSLCertificateKeyPath string   `json:"ssl_certificate_key_path"`
	SSLProtocols          []string `json:"ssl_protocols"`
	HTTPRedirect          bool     `json:"http_redirect"`
}

// CORSConfigResponse CORS配置响应
type CORSConfigResponse struct {
	AllowedOrigins []string `json:"allowed_origins"`
	AllowedMethods []string `json:"allowed_methods"`
	AllowedHeaders []string `json:"allowed_headers"`
}

// RateLimitConfigResponse 速率限制配置响应
type RateLimitConfigResponse struct {
	Enabled           bool `json:"enabled"`
	RequestsPerSecond int  `json:"requests_per_second"`
	Burst             int  `json:"burst"`
}

// UpdateSystemConfigRequest 更新系统配置请求
type UpdateSystemConfigRequest struct {
	CORS      *CORSConfigResponse      `json:"cors,omitempty"`
	RateLimit *RateLimitConfigResponse `json:"rate_limit,omitempty"`
}
