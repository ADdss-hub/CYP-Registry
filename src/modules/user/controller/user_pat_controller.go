// Package controller 提供用户 PAT 相关 HTTP 处理
package controller

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	"github.com/cyp-registry/registry/src/modules/user/dto"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// CreatePAT 创建Personal Access Token
// @Summary 创建PAT
// @Description 创建Personal Access Token
// @Tags pat
// @Accept json
// @Produce json
// @Param request body dto.CreatePATRequest true "PAT信息"
// @Success 20000 {object} response.Response{data=dto.CreatePATResponse}
// @Failure 10001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/pat [post]
func (c *UserController) CreatePAT(ctx *gin.Context) {
	var req dto.CreatePATRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "参数校验失败")
		return
	}

	// 输入验证
	if req.Name == "" {
		response.ParamError(ctx, "PAT名称不能为空")
		return
	}
	if len(req.Name) > 100 {
		response.ParamError(ctx, "PAT名称长度不能超过100个字符")
		return
	}

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

	result, err := c.svc.CreatePAT(ctx, userID, req.Name, req.Scopes, req.ExpireIn)
	if err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"user","operation":"create_pat","user_id":"%s","pat_name":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), userID.String(), req.Name, err)
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "创建PAT失败")
		return
	}

	response.Success(ctx, dto.CreatePATResponse{
		ID:        result.ID,
		Name:      result.Name,
		Scopes:    result.Scopes,
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
		Token:     result.Token,
		TokenType: result.TokenType,
	})
}

// ListPAT 列出所有PAT
// @Summary 列出PAT
// @Description 列出当前用户的所有Personal Access Token
// @Tags pat
// @Produce json
// @Success 20000 {object} response.Response{data=[]dto.PATResponse}
// @Security Bearer
// @Router /api/v1/users/me/pat [get]
func (c *UserController) ListPAT(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	tokens, err := c.svc.ListPAT(ctx, userID.(uuid.UUID))
	if err != nil {
		response.InternalServerError(ctx, "获取PAT列表失败")
		return
	}

	response.Success(ctx, tokens)
}

// RevokePAT 删除PAT（兼容历史命名，语义为删除）
// @Summary 删除PAT
// @Description 删除指定的Personal Access Token（幂等：即使ID无效或不存在也视为删除成功）
// @Tags pat
// @Produce json
// @Param id path string true "PAT ID"
// @Success 20000 {object} response.Response
// @Failure 10001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/pat/{id} [delete]
func (c *UserController) RevokePAT(ctx *gin.Context) {
	// 直接从路径中读取ID，避免对 URI 参数做过于严格的 UUID 绑定校验
	idStr := ctx.Param("id")

	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	// 尝试解析为UUID；如果解析失败，按“已删除”处理，实现幂等删除
	tokenID, err := uuid.Parse(idStr)
	if err != nil {
		// ID 本身无效，对调用方而言等同于“已经不存在”的资源
		response.SuccessWithMessage(ctx, "PAT已删除", nil)
		return
	}

	if err := c.svc.RevokePAT(ctx, userID.(uuid.UUID), tokenID); err != nil {
		// 后端删除失败（例如记录不存在），也按幂等删除处理
		response.SuccessWithMessage(ctx, "PAT已删除", nil)
		return
	}

	response.SuccessWithMessage(ctx, "PAT已删除", nil)
}
