// Package controller 提供用户管理相关 HTTP 处理
// 遵循《全平台通用开发任务设计规范文档》第6章API规范
package controller

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/modules/user/dto"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// ListUsers 列出用户（管理员）
// @Summary 用户列表
// @Tags user
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param keyword query string false "关键字"
// @Security Bearer
// @Router /api/v1/users [get]
func (c *UserController) ListUsers(ctx *gin.Context) {
	page := parseInt(ctx.DefaultQuery("page", "1"), 1)
	pageSize := parseInt(ctx.DefaultQuery("pageSize", ctx.DefaultQuery("page_size", "20")), 20)
	keyword := ctx.Query("keyword")

	users, total, err := c.svc.ListUsers(ctx, page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(ctx, "获取用户列表失败")
		return
	}

	// 返回基础字段（避免泄露 password/hash）
	result := make([]dto.UserResponse, 0, len(users))
	for i := range users {
		u := users[i]
		result = append(result, dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			Nickname:    u.Nickname,
			Avatar:      u.Avatar,
			Bio:         u.Bio,
			IsActive:    u.IsActive,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   u.CreatedAt.Format(time.RFC3339),
			LastLoginAt: u.LastLoginAt.Format(time.RFC3339),
		})
	}

	response.Success(ctx, gin.H{
		"users":     result,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetUser 获取单个用户（管理员）
// @Summary 用户详情
// @Tags user
// @Produce json
// @Param id path string true "用户ID"
// @Security Bearer
// @Router /api/v1/users/{id} [get]
func (c *UserController) GetUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.ParamError(ctx, "invalid id")
		return
	}

	user, err := c.svc.GetUserByID(ctx, id)
	if err != nil {
		response.NotFound(ctx, "user not found")
		return
	}
	response.Success(ctx, formatUserResponse(user))
}

// UpdateUser 更新用户（管理员）
// @Summary 更新用户
// @Tags user
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Security Bearer
// @Router /api/v1/users/{id} [patch]
func (c *UserController) UpdateUser(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		response.ParamError(ctx, "invalid id")
		return
	}

	var req map[string]interface{}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "invalid body")
		return
	}

	// 安全：不允许直接写 password
	delete(req, "password")

	if err := c.svc.UpdateUser(ctx, id, req); err != nil {
		response.ParamError(ctx, "update failed")
		return
	}
	user, _ := c.svc.GetUserByID(ctx, id)
	response.Success(ctx, formatUserResponse(user))
}

// DeleteUser 删除用户（管理员）
// @Summary 删除用户
// @Tags user
// @Produce json
// @Param id path string true "用户ID"
// @Security Bearer
// @Router /api/v1/users/{id} [delete]
func (c *UserController) DeleteUser(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		response.ParamError(ctx, "invalid id")
		return
	}
	if err := c.svc.DeleteUser(ctx, id); err != nil {
		response.NotFound(ctx, "user not found")
		return
	}
	response.SuccessWithMessage(ctx, "deleted", nil)
}
