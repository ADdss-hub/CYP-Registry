// Package controller Webhook控制器
// 提供Webhook管理的HTTP API接口
package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	"github.com/cyp-registry/registry/src/modules/webhook"
	"github.com/cyp-registry/registry/src/modules/webhook/service"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/response"
	"github.com/gin-gonic/gin"
)

// WebhookController Webhook控制器
type WebhookController struct {
	webhookService *service.WebhookService
	authMw         *middleware.AuthMiddleware
}

// NewWebhookController 创建新的Webhook控制器
func NewWebhookController(webhookService *service.WebhookService, authMw *middleware.AuthMiddleware) *WebhookController {
	return &WebhookController{
		webhookService: webhookService,
		authMw:         authMw,
	}
}

// RegisterRoutes 注册路由
func (c *WebhookController) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1/webhooks")
	// 所有 Webhook 管理接口都需要登录（使用统一的 Auth 中间件）
	if c.authMw != nil {
		api.Use(c.authMw.Auth())
	}
	{
		api.POST("", c.CreateWebhook)
		api.GET("", c.ListWebhooks)
		api.GET("/statistics", c.GetStatistics)
		api.GET("/:webhookId", c.GetWebhook)
		api.PUT("/:webhookId", c.UpdateWebhook)
		api.DELETE("/:webhookId", c.DeleteWebhook)
		api.POST("/:webhookId/test", c.TestWebhook)
		api.GET("/:webhookId/deliveries", c.GetDeliveries)
	}

	// Registry 实时事件 SSE 流：用于前端订阅 push/delete 完成事件
	// 使用 OptionalAuth：前端 EventSource 无需显式携带 Authorization 头即可建立连接，
	// 若请求中包含有效 Token，则仍可在服务端获取到用户信息（当前实现未强依赖用户身份）。
	stream := r.Group("/api/v1/stream")
	if c.authMw != nil {
		stream.Use(c.authMw.OptionalAuth())
	}
	stream.GET("/registry", c.RegistryEventStream)
}

// RegistryEventStream SSE 事件流：推送 Registry push/delete 事件给前端
// GET /api/v1/stream/registry?projectId=xxx&repository=xxx
func (c *WebhookController) RegistryEventStream(ctx *gin.Context) {
	// 为保证前端最小使用成本，不强制必须传 projectId / repository，
	// 但前端可以根据当前项目自行过滤。

	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	ctx.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := ctx.Writer.(interface{ Flush() })
	if !ok {
		ctx.Status(500)
		return
	}

	events, cancel := webhook.SubscribeRegistryEvents()
	defer cancel()

	// 发送一个初始事件，帮助前端确认连接成功
	fmt.Fprintf(ctx.Writer, "event: ping\ndata: {\"time\": %q}\n\n", time.Now().Format(time.RFC3339))
	flusher.Flush()

	notify := ctx.Request.Context().Done()

	for {
		select {
		case <-notify:
			return
		case ev, ok := <-events:
			if !ok {
				return
			}

			payload, err := json.Marshal(ev)
			if err != nil {
				continue
			}

			fmt.Fprintf(ctx.Writer, "event: registry\ndata: %s\n\n", payload)
			flusher.Flush()
		}
	}
}

// CreateWebhook 创建Webhook
// @Summary 创建新的Webhook
// @Description 为项目创建一个新的Webhook配置
// @Tags webhooks
// @Accept json
// @Produce json
// @Param request body webhook.CreateWebhookRequest true "Webhook配置"
// @Success 201 {object} response.Response{data=*webhook.Webhook}
// @Failure 400 {object} response.Response
// @Router /api/v1/webhooks [post]
func (c *WebhookController) CreateWebhook(ctx *gin.Context) {
	var req webhook.CreateWebhookRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "Invalid request body: "+err.Error())
		return
	}

	// 输入验证
	if req.Name == "" {
		response.ParamError(ctx, "Webhook名称不能为空")
		return
	}
	if req.URL == "" {
		response.ParamError(ctx, "Webhook URL不能为空")
		return
	}

	// 从认证中间件获取用户ID
	userIDVal, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.InternalServerError(ctx, "用户标识异常")
		return
	}

	webhookObj, err := c.webhookService.CreateWebhook(&req, userID.String())
	if err != nil {
		log.Printf("[ERROR] 创建Webhook失败: %v, 用户ID: %s, Webhook名称: %s", err, userID, req.Name)
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "创建Webhook失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(ctx, "Webhook created successfully", webhookObj)
}

// ListWebhooks 列出Webhooks
// @Summary 列出项目的Webhook
// @Description 获取指定项目的所有Webhook配置
// @Tags webhooks
// @Produce json
// @Param projectId query string true "项目ID"
// @Success 200 {object} response.Response{data=[]*webhook.Webhook}
// @Router /api/v1/webhooks [get]
func (c *WebhookController) ListWebhooks(ctx *gin.Context) {
	projectID := ctx.Query("projectId")
	if projectID == "" {
		response.ParamError(ctx, "projectId is required")
		return
	}

	webhooks, err := c.webhookService.ListWebhooks(projectID)
	if err != nil {
		response.InternalServerError(ctx, "Failed to list webhooks: "+err.Error())
		return
	}

	response.Success(ctx, webhooks)
}

// GetWebhook 获取Webhook
// @Summary 获取Webhook详情
// @Description 获取指定Webhook的详细信息
// @Tags webhooks
// @Produce json
// @Param webhookId path string true "Webhook ID"
// @Success 200 {object} response.Response{data=*webhook.Webhook}
// @Failure 404 {object} response.Response
// @Router /api/v1/webhooks/{webhookId} [get]
func (c *WebhookController) GetWebhook(ctx *gin.Context) {
	webhookID := ctx.Param("webhookId")

	webhookObj, err := c.webhookService.GetWebhook(webhookID)
	if err != nil {
		response.NotFound(ctx, "Webhook 不存在: "+err.Error())
		return
	}

	response.Success(ctx, webhookObj)
}

// UpdateWebhook 更新Webhook
// @Summary 更新Webhook
// @Description 更新指定Webhook的配置
// @Tags webhooks
// @Accept json
// @Produce json
// @Param webhookId path string true "Webhook ID"
// @Param request body webhook.UpdateWebhookRequest true "更新内容"
// @Success 200 {object} response.Response{data=*webhook.Webhook}
// @Failure 400 {object} response.Response
// @Router /api/v1/webhooks/{webhookId} [put]
func (c *WebhookController) UpdateWebhook(ctx *gin.Context) {
	webhookID := ctx.Param("webhookId")

	var req webhook.UpdateWebhookRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "Invalid request body: "+err.Error())
		return
	}

	webhookObj, err := c.webhookService.UpdateWebhook(webhookID, &req)
	if err != nil {
		response.ParamError(ctx, "Failed to update webhook: "+err.Error())
		return
	}

	response.SuccessWithMessage(ctx, "Webhook updated successfully", webhookObj)
}

// DeleteWebhook 删除Webhook
// @Summary 删除Webhook
// @Description 删除指定的Webhook配置
// @Tags webhooks
// @Produce json
// @Param webhookId path string true "Webhook ID"
// @Success 204 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/webhooks/{webhookId} [delete]
func (c *WebhookController) DeleteWebhook(ctx *gin.Context) {
	webhookID := ctx.Param("webhookId")

	if err := c.webhookService.DeleteWebhook(webhookID); err != nil {
		response.NotFound(ctx, "Webhook 不存在: "+err.Error())
		return
	}

	response.Success(ctx, nil)
}

// TestWebhook 测试Webhook
// @Summary 测试Webhook
// @Description 向指定Webhook发送测试请求
// @Tags webhooks
// @Accept json
// @Produce json
// @Param webhookId path string true "Webhook ID"
// @Param request body webhook.TestWebhookRequest true "测试请求"
// @Success 200 {object} response.Response{data=*webhook.WebhookDelivery}
// @Failure 400 {object} response.Response
// @Router /api/v1/webhooks/{webhookId}/test [post]
func (c *WebhookController) TestWebhook(ctx *gin.Context) {
	webhookID := ctx.Param("webhookId")

	var req webhook.TestWebhookRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "Invalid request body: "+err.Error())
		return
	}

	// 设置默认事件类型
	if req.EventType == "" {
		req.EventType = webhook.EventTypePush
	}

	delivery, err := c.webhookService.TestWebhook(webhookID, &req)
	if err != nil {
		response.ParamError(ctx, "测试 Webhook 失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(ctx, "Test delivery completed", delivery)
}

// GetDeliveries 获取投递历史
// @Summary 获取投递历史
// @Description 获取指定Webhook的投递历史记录
// @Tags webhooks
// @Produce json
// @Param webhookId path string true "Webhook ID"
// @Param eventId query string false "事件ID"
// @Success 200 {object} response.Response{data=[]*webhook.WebhookDelivery}
// @Router /api/v1/webhooks/{webhookId}/deliveries [get]
func (c *WebhookController) GetDeliveries(ctx *gin.Context) {
	webhookID := ctx.Param("webhookId")
	eventID := ctx.Query("eventId")

	if eventID != "" {
		// 获取特定事件的投递记录
		deliveries, err := c.webhookService.GetDeliveries(eventID)
		if err != nil {
			response.NotFound(ctx, "事件不存在: "+err.Error())
			return
		}
		response.Success(ctx, deliveries)
	} else {
		// 获取该Webhook的所有投递记录
		deliveries, err := c.webhookService.GetDeliveriesByWebhookID(webhookID)
		if err != nil {
			response.NotFound(ctx, "Webhook 不存在: "+err.Error())
			return
		}
		response.Success(ctx, deliveries)
	}
}

// GetStatistics 获取统计信息
// @Summary 获取Webhook统计
// @Description 获取Webhook的整体统计信息
// @Tags webhooks
// @Produce json
// @Success 200 {object} response.Response{data=*webhook.WebhookStatistics}
// @Router /api/v1/webhooks/statistics [get]
func (c *WebhookController) GetStatistics(ctx *gin.Context) {
	stats := c.webhookService.GetStatistics()
	response.Success(ctx, stats)
}

// ParseInt64 安全解析int64
func ParseInt64(s string, defaultValue int64) int64 {
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return defaultValue
	}
	return val
}

// parseJSON 解析JSON
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
