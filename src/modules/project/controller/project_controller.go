// Package project_controller 项目管理控制器
// 提供项目的RESTful API接口
package project_controller

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	"github.com/cyp-registry/registry/src/modules/project/dto"
	project "github.com/cyp-registry/registry/src/modules/project/service"
	user_service "github.com/cyp-registry/registry/src/modules/user/service"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// ProjectController 项目管理控制器
type ProjectController struct {
	svc     project.Service
	userSvc *user_service.Service
}

// NewProjectController 创建项目控制器
func NewProjectController(svc project.Service, userSvc *user_service.Service) *ProjectController {
	return &ProjectController{
		svc:     svc,
		userSvc: userSvc,
	}
}

// Create 创建项目
// POST /api/v1/projects
func (c *ProjectController) Create(ctx *gin.Context) {
	// 解析请求
	var req dto.CreateProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, err.Error())
		return
	}

	// 获取用户ID（从JWT token中），由认证中间件提前写入上下文
	userIDVal, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	userUUID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	// 创建项目
	p, err := c.svc.CreateProject(ctx.Request.Context(), req.Name, req.Description, userUUID.String(), req.IsPublic, req.StorageQuota)
	if err != nil {
		log.Printf("[ERROR] 创建项目失败: %v, 项目名: %s, 用户ID: %s", err, req.Name, userUUID.String())
		if err == project.ErrProjectExists {
			response.Conflict(ctx, fmt.Sprintf("项目 '%s' 已存在，请使用其他名称", req.Name))
			return
		}
		// 检查是否是自定义错误
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		// 数据库错误或其他错误
		response.InternalServerError(ctx, "failed to create project")
		return
	}

	// 返回结果
	response.Success(ctx, gin.H{
		"project": toProjectResponse(p),
	})
}

// Get 获取项目详情
// GET /api/v1/projects/:id
func (c *ProjectController) Get(ctx *gin.Context) {
	projectID := ctx.Param("id")

	// 获取用户ID（从JWT token中），由认证中间件提前写入上下文
	userIDVal, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	userUUID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}
	userID := userUUID.String()

	// 访问控制：与列表接口保持一致，只允许有权限的用户查看项目详情
	canAccess, err := c.svc.CanAccess(ctx.Request.Context(), userID, projectID, "pull")
	if err != nil {
		response.InternalServerError(ctx, "failed to check permission")
		return
	}
	if !canAccess {
		response.Forbidden(ctx, "permission denied")
		return
	}

	p, err := c.svc.GetProject(ctx.Request.Context(), projectID)
	if err != nil {
		if err == project.ErrProjectNotFound {
			response.NotFound(ctx, "project not found")
			return
		}
		response.InternalServerError(ctx, "failed to get project")
		return
	}

	response.Success(ctx, gin.H{
		"project": toProjectResponse(p),
	})
}

// List 列出项目
// GET /api/v1/projects
func (c *ProjectController) List(ctx *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取用户ID（从JWT token中），由认证中间件提前写入上下文
	userIDVal, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	userUUID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	userID := userUUID.String()

	// 默认仅返回当前用户可访问的项目；若为管理员，则返回所有项目
	var (
		projects []project.Project
		total    int64
		err      error
	)

	if c.userSvc != nil {
		if user, uErr := c.userSvc.GetUserByID(ctx.Request.Context(), userUUID); uErr == nil && user != nil && user.IsAdmin {
			projects, total, err = c.svc.ListProjects(ctx.Request.Context(), userID, page, pageSize)
		} else {
			projects, total, err = c.svc.ListUserProjects(ctx.Request.Context(), userID, page, pageSize)
		}
	} else {
		projects, total, err = c.svc.ListUserProjects(ctx.Request.Context(), userID, page, pageSize)
	}
	if err != nil {
		response.InternalServerError(ctx, "failed to list projects")
		return
	}

	// 转换为响应
	projectList := make([]dto.ProjectResponse, len(projects))
	for i, p := range projects {
		projectList[i] = toProjectResponse(&p)
	}

	response.Success(ctx, gin.H{
		"projects":  projectList,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Update 更新项目
// PUT /api/v1/projects/:id
func (c *ProjectController) Update(ctx *gin.Context) {
	projectID := ctx.Param("id")

	// 解析请求
	var req dto.UpdateProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, err.Error())
		return
	}

	// 获取用户ID（从JWT token中），由认证中间件提前写入上下文
	userIDVal, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	userUUID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	userID := userUUID.String()

	// 验证是否是所有者
	isOwner, err := c.svc.IsOwner(ctx.Request.Context(), userID, projectID)
	if err != nil {
		response.InternalServerError(ctx, "failed to check ownership")
		return
	}
	if !isOwner {
		response.Forbidden(ctx, "only owner can update project")
		return
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}
	if req.StorageQuota != nil {
		updates["storage_quota"] = *req.StorageQuota
	}

	if len(updates) == 0 {
		response.ParamError(ctx, "no fields to update")
		return
	}

	if err := c.svc.UpdateProject(ctx.Request.Context(), projectID, updates); err != nil {
		if err == project.ErrProjectNotFound {
			response.NotFound(ctx, "project not found")
			return
		}
		response.InternalServerError(ctx, "failed to update project")
		return
	}

	response.Success(ctx, gin.H{
		"message": "project updated successfully",
	})
}

// Delete 删除项目
// DELETE /api/v1/projects/:id
func (c *ProjectController) Delete(ctx *gin.Context) {
	projectID := ctx.Param("id")

	// 获取用户ID（从JWT token中），由认证中间件提前写入上下文
	userIDVal, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	userUUID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	userID := userUUID.String()

	// 验证是否是所有者
	isOwner, err := c.svc.IsOwner(ctx.Request.Context(), userID, projectID)
	if err != nil {
		response.InternalServerError(ctx, "failed to check ownership")
		return
	}
	if !isOwner {
		response.Forbidden(ctx, "only owner can delete project")
		return
	}

	if err := c.svc.DeleteProject(ctx.Request.Context(), projectID); err != nil {
		if err == project.ErrProjectNotFound {
			response.NotFound(ctx, "project not found")
			return
		}
		response.InternalServerError(ctx, "failed to delete project")
		return
	}

	response.Success(ctx, gin.H{
		"message": "project deleted successfully",
	})
}

// UpdateQuota 更新存储配额
// PUT /api/v1/projects/:id/quota
func (c *ProjectController) UpdateQuota(ctx *gin.Context) {
	projectID := ctx.Param("id")

	// 解析请求
	var req struct {
		Quota int64 `json:"quota" binding:"required,min=0"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, err.Error())
		return
	}

	// 获取用户ID（从JWT token中），由认证中间件提前写入上下文
	userIDVal, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	userUUID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.Unauthorized(ctx, "user not authenticated")
		return
	}

	userID := userUUID.String()

	// 验证是否是所有者
	isOwner, err := c.svc.IsOwner(ctx.Request.Context(), userID, projectID)
	if err != nil {
		response.InternalServerError(ctx, "failed to check ownership")
		return
	}
	if !isOwner {
		response.Forbidden(ctx, "only owner can update quota")
		return
	}

	// 获取当前项目信息
	proj, err := c.svc.GetProject(ctx.Request.Context(), projectID)
	if err != nil {
		response.NotFound(ctx, "project not found")
		return
	}

	// 更新配额
	if err := c.svc.UpdateQuota(ctx.Request.Context(), projectID, req.Quota); err != nil {
		if err == project.ErrQuotaExceeded {
			response.ParamError(ctx, "quota cannot be less than used storage")
			return
		}
		if err == project.ErrInvalidQuota {
			response.ParamError(ctx, "invalid quota value")
			return
		}
		response.InternalServerError(ctx, "failed to update quota")
		return
	}

	response.Success(ctx, gin.H{
		"old_quota":    proj.StorageQuota,
		"new_quota":    req.Quota,
		"storage_used": proj.StorageUsed,
		"storage_left": req.Quota - proj.StorageUsed,
	})
}

// GetStorageUsage 获取存储使用量
// GET /api/v1/projects/:id/storage
func (c *ProjectController) GetStorageUsage(ctx *gin.Context) {
	projectID := ctx.Param("id")

	p, err := c.svc.GetProject(ctx.Request.Context(), projectID)
	if err != nil {
		if err == project.ErrProjectNotFound {
			response.NotFound(ctx, "project not found")
			return
		}
		response.InternalServerError(ctx, "failed to get project")
		return
	}

	usagePercent := float64(0)
	if p.StorageQuota > 0 {
		usagePercent = float64(p.StorageUsed) / float64(p.StorageQuota) * 100
	}

	response.Success(ctx, gin.H{
		"project_id":    p.ID,
		"project_name":  p.Name,
		"storage_used":  p.StorageUsed,
		"storage_quota": p.StorageQuota,
		"usage_percent": usagePercent,
	})
}

// toProjectResponse 转换为项目响应
func toProjectResponse(p *project.Project) dto.ProjectResponse {
	return dto.ProjectResponse{
		ID:           p.ID,
		Name:         p.Name,
		Description:  p.Description,
		OwnerID:      p.OwnerID,
		IsPublic:     p.IsPublic,
		StorageUsed:  p.StorageUsed,
		StorageQuota: p.StorageQuota,
		ImageCount:   p.ImageCount,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}
