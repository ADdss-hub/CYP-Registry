// Package controller 提供管理员相关HTTP处理
// 遵循《全平台通用开发任务设计规范文档》第6章API规范
package controller

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/modules/admin/dto"
	"github.com/cyp-registry/registry/src/modules/admin/service"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// AdminController 管理员控制器
type AdminController struct {
	svc *service.Service
}

// NewAdminController 创建管理员控制器
func NewAdminController(svc *service.Service) *AdminController {
	return &AdminController{svc: svc}
}

// ListAuditLogs 获取审计日志列表
// @Summary 获取审计日志列表
// @Description 获取系统审计日志，支持分页、筛选和搜索（需要管理员权限）
// @Tags admin
// @Produce json
// @Param page query int false "页码，从1开始" default(1)
// @Param page_size query int false "每页数量，默认20，最大100" default(20)
// @Param user_id query string false "用户ID筛选"
// @Param action query string false "操作类型筛选"
// @Param resource query string false "资源类型筛选"
// @Param start_time query string false "开始时间（RFC3339格式）"
// @Param end_time query string false "结束时间（RFC3339格式）"
// @Param keyword query string false "关键词搜索（搜索操作详情）"
// @Success 20000 {object} response.Response{data=dto.AuditLogListResponse}
// @Failure 30003 {object} response.Response
// @Security Bearer
// @Router /api/v1/admin/logs [get]
func (c *AdminController) ListAuditLogs(ctx *gin.Context) {
	// 解析分页参数
	page := 1
	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := ctx.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			if ps > 100 {
				pageSize = 100
			} else {
				pageSize = ps
			}
		}
	}

	// 解析筛选参数
	var userID *uuid.UUID
	if userIDStr := ctx.Query("user_id"); userIDStr != "" {
		if id, err := uuid.Parse(userIDStr); err == nil {
			userID = &id
		}
	}

	action := ctx.Query("action")
	resource := ctx.Query("resource")

	var startTime, endTime *time.Time
	if startTimeStr := ctx.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := ctx.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	keyword := ctx.Query("keyword")

	// 调用服务层获取日志列表
	logs, total, err := c.svc.ListAuditLogs(ctx.Request.Context(), page, pageSize, userID, action, resource, startTime, endTime, keyword)
	if err != nil {
		response.InternalServerError(ctx, "获取审计日志失败")
		return
	}

	response.Success(ctx, dto.AuditLogListResponse{
		Logs:      logs,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: (int(total) + pageSize - 1) / pageSize,
	})
}

// GetSystemConfig 获取系统配置
// @Summary 获取系统配置
// @Description 获取系统配置信息，包括HTTPS、CORS、速率限制等（需要管理员权限）
// @Tags admin
// @Produce json
// @Success 20000 {object} response.Response{data=dto.SystemConfigResponse}
// @Failure 30003 {object} response.Response
// @Security Bearer
// @Router /api/v1/admin/config [get]
func (c *AdminController) GetSystemConfig(ctx *gin.Context) {
	config, err := c.svc.GetSystemConfig()
	if err != nil {
		response.InternalServerError(ctx, "获取系统配置失败: "+err.Error())
		return
	}

	response.Success(ctx, config)
}

// UpdateSystemConfig 更新系统配置
// @Summary 更新系统配置
// @Description 更新系统配置，包括CORS、速率限制等（需要管理员权限）
// @Tags admin
// @Accept json
// @Produce json
// @Param config body dto.UpdateSystemConfigRequest true "系统配置"
// @Success 20000 {object} response.Response
// @Failure 10001 {object} response.Response
// @Failure 30003 {object} response.Response
// @Security Bearer
// @Router /api/v1/admin/config [put]
func (c *AdminController) UpdateSystemConfig(ctx *gin.Context) {
	var req dto.UpdateSystemConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "参数错误: "+err.Error())
		return
	}

	if err := c.svc.UpdateSystemConfig(&req); err != nil {
		response.InternalServerError(ctx, "更新系统配置失败: "+err.Error())
		return
	}

	response.Success(ctx, nil)
}
