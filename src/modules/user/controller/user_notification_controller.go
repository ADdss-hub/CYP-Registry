// Package controller 提供用户通知设置相关 HTTP 处理
package controller

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	"github.com/cyp-registry/registry/src/modules/user/dto"
	"github.com/cyp-registry/registry/src/modules/user/service"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// GetNotificationSettings 获取当前用户的通知设置
// @Summary 获取当前用户通知设置
// @Description 获取当前登录用户的通知偏好（邮件通知、扫描完成通知、安全告警、Webhook通知等）
// @Tags user
// @Produce json
// @Success 20000 {object} response.Response{data=dto.NotificationSettings}
// @Failure 30001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/notification-settings [get]
func (c *UserController) GetNotificationSettings(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	settings, err := c.svc.GetNotificationSettings(ctx, userID.(uuid.UUID))
	if err != nil {
		response.InternalServerError(ctx, "获取通知设置失败")
		return
	}

	resp := dto.NotificationSettings{
		EmailEnabled:         settings.EmailEnabled,
		ScanCompleted:        settings.ScanCompleted,
		SecurityAlerts:       settings.SecurityAlerts,
		WebhookNotifications: settings.WebhookNotifications,
		Digest:               settings.Digest,
		NotificationEmail:    settings.NotificationEmail,
	}

	response.Success(ctx, resp)
}

// UpdateNotificationSettings 更新当前用户的通知设置
// @Summary 更新当前用户通知设置
// @Description 更新当前登录用户的通知偏好（邮件通知、扫描完成通知、安全告警、Webhook通知等）
// @Tags user
// @Accept json
// @Produce json
// @Param request body dto.NotificationSettings true "通知设置"
// @Success 20000 {object} response.Response{data=dto.NotificationSettings}
// @Failure 10001 {object} response.Response
// @Failure 30001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/notification-settings [put]
func (c *UserController) UpdateNotificationSettings(ctx *gin.Context) {
	var req dto.NotificationSettings
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "参数校验失败")
		return
	}

	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	// 兜底通知频率
	if req.Digest == "" {
		req.Digest = "realtime"
	}

	settings := &service.NotificationSettings{
		EmailEnabled:         req.EmailEnabled,
		ScanCompleted:        req.ScanCompleted,
		SecurityAlerts:       req.SecurityAlerts,
		WebhookNotifications: req.WebhookNotifications,
		Digest:               req.Digest,
		NotificationEmail:    req.NotificationEmail,
	}

	if err := c.svc.UpdateNotificationSettings(ctx, userID.(uuid.UUID), settings); err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"update_notification_settings","user_id":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), userID.(uuid.UUID).String(), err)
		response.InternalServerError(ctx, "更新通知设置失败")
		return
	}

	response.Success(ctx, req)
}
