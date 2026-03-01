// Package controller 提供用户认证相关HTTP处理
// 遵循《全平台通用开发任务设计规范文档》第6章API规范
package controller

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/modules/user/dto"
	"github.com/cyp-registry/registry/src/pkg/models"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// formatUserResponse 格式化用户响应
func formatUserResponse(user interface{}) dto.UserResponse {
	// 类型断言
	switch u := user.(type) {
	case *models.User:
		if u == nil {
			return dto.UserResponse{}
		}
		return dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			Nickname:    u.Nickname,
			Avatar:      u.Avatar,
			Bio:         u.Bio,
			IsActive:    u.IsActive,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   formatUserTime(u.CreatedAt),
			LastLoginAt: formatUserTime(u.LastLoginAt),
		}
	case models.User:
		return dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			Nickname:    u.Nickname,
			Avatar:      u.Avatar,
			Bio:         u.Bio,
			IsActive:    u.IsActive,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   formatUserTime(u.CreatedAt),
			LastLoginAt: formatUserTime(u.LastLoginAt),
		}
	case *struct {
		ID          uuid.UUID
		Username    string
		Email       string
		Nickname    string
		Avatar      string
		Bio         string
		IsActive    bool
		IsAdmin     bool
		CreatedAt   time.Time
		LastLoginAt time.Time
	}:
		return dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			Nickname:    u.Nickname,
			Avatar:      u.Avatar,
			Bio:         u.Bio,
			IsActive:    u.IsActive,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   formatUserTime(u.CreatedAt),
			LastLoginAt: formatUserTime(u.LastLoginAt),
		}
	default:
		// 默认返回空响应
		return dto.UserResponse{}
	}
}

// formatUserTime 统一处理用户时间字段，避免零值时间序列化为 0001-01-01T00:00:00Z
func formatUserTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// parseValidationErrors 解析验证错误
func parseValidationErrors(err error) []response.Error {
	// 简化实现
	return []response.Error{
		{
			Field:   "validation",
			Message: err.Error(),
		},
	}
}

// parseInt 解析字符串为整数，失败时返回默认值
func parseInt(s string, def int) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n <= 0 {
		return def
	}
	return n
}
