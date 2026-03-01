// Package controller 提供镜像导入相关的HTTP接口
package controller

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	imageimportdto "github.com/cyp-registry/registry/src/modules/imageimport/dto"
	imageimportservice "github.com/cyp-registry/registry/src/modules/imageimport/service"
	projectservice "github.com/cyp-registry/registry/src/modules/project/service"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// ImageImportController 镜像导入控制器
// 路由前缀：/api/v1/projects/:id/images/import
type ImageImportController struct {
	svc        *imageimportservice.Service
	projectSvc projectservice.Service
}

// NewImageImportController 创建控制器
func NewImageImportController(
	svc *imageimportservice.Service,
	projectSvc projectservice.Service,
) *ImageImportController {
	return &ImageImportController{
		svc:        svc,
		projectSvc: projectSvc,
	}
}

// ImportImage 创建导入任务
// POST /api/v1/projects/:id/images/import
func (c *ImageImportController) ImportImage(ctx *gin.Context) {
	projectID := ctx.Param("id")
	if projectID == "" {
		response.ParamError(ctx, "项目ID不能为空")
		return
	}

	var req imageimportdto.ImportImageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "请求参数不合法")
		return
	}
	if req.SourceURL == "" {
		response.ParamError(ctx, "source_url 不能为空")
		return
	}

	// 获取当前用户ID（用于审计与追踪）
	var userID *uuid.UUID
	if v, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if id, ok := v.(uuid.UUID); ok && id != uuid.Nil {
			userID = &id
		}
	}

	// 确保项目存在：如果不存在则自动创建
	project, err := c.projectSvc.GetProject(ctx.Request.Context(), projectID)
	if err != nil {
		// 项目不存在时尝试自动创建项目
		// 规则：优先从 target_image 提取项目名（有斜杠时取第一段），否则使用 projectID 作为项目名
		var ownerID string
		if userID != nil {
			ownerID = userID.String()
		}

		// 只有在能识别到 ownerID 时才尝试自动创建项目
		if ownerID == "" {
			response.NotFound(ctx, "项目不存在")
			return
		}

		projectName := projectID
		if strings.TrimSpace(req.TargetImage) != "" {
			ti := strings.TrimSpace(req.TargetImage)
			if idx := strings.Index(ti, "/"); idx > 0 {
				projectName = ti[:idx]
			}
		}

		created, createErr := c.projectSvc.CreateProject(
			ctx.Request.Context(),
			projectName,
			"Auto created from image import",
			ownerID,
			false, // 默认私有
			0,     // 使用默认配额
		)
		if createErr != nil {
			response.NotFound(ctx, "project not found and failed to create: "+createErr.Error())
			return
		}
		project = created
		// 更新 projectID 为新创建项目的 ID，以便后续任务绑定正确的项目
		projectID = project.ID
	}

	task, err := c.svc.ImportImageFromURL(
		ctx.Request.Context(),
		projectID,
		project.Name,
		userID,
		&req,
	)
	if err != nil {
		response.InternalServerError(ctx, err.Error())
		return
	}

	resp := imageimportdto.FromModel(task)
	response.Success(ctx, resp)
}

// GetTask 获取任务详情
// GET /api/v1/projects/:id/images/import/:task_id
func (c *ImageImportController) GetTask(ctx *gin.Context) {
	projectID := ctx.Param("id")
	taskID := ctx.Param("task_id")
	if projectID == "" || taskID == "" {
		response.ParamError(ctx, "项目ID或任务ID不能为空")
		return
	}

	task, err := c.svc.GetTask(ctx.Request.Context(), projectID, taskID)
	if err != nil {
		response.NotFound(ctx, "任务不存在")
		return
	}

	resp := imageimportdto.FromModel(task)
	response.Success(ctx, resp)
}

// ListTasks 列出项目的导入任务
// GET /api/v1/projects/:id/images/import
func (c *ImageImportController) ListTasks(ctx *gin.Context) {
	projectID := ctx.Param("id")
	if projectID == "" {
		response.ParamError(ctx, "项目ID不能为空")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("page_size", "20"))

	tasks, total, err := c.svc.ListTasks(ctx.Request.Context(), projectID, page, pageSize)
	if err != nil {
		response.InternalServerError(ctx, "获取任务列表失败")
		return
	}

	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}

	data := imageimportdto.ImportTaskListResponse{
		Tasks:     imageimportdto.FromModelSlice(tasks),
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
	}

	response.Success(ctx, data)
}
