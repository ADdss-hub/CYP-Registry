// Package controller 提供用户认证相关HTTP处理
// 遵循《全平台通用开发任务设计规范文档》第6章API规范
package controller

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/cyp-registry/registry/src/modules/user/dto"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// Register 用户注册
// @Summary 用户注册
// @Description 新用户注册账号
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "注册信息"
// @Success 20000 {object} response.Response{data=dto.UserResponse}
// @Failure 10001 {object} response.Response
// @Router /api/v1/auth/register [post]
func (c *UserController) Register(ctx *gin.Context) {
	// 公开注册功能已关闭：仅允许管理员预置账号或通过默认管理员账号初始化后在后台创建用户
	response.Fail(ctx, 40301, "公开注册功能已关闭，请联系管理员获取账号")
}

// GetDefaultAdminOnce 首次部署时获取一次性默认管理员账号信息
// @Summary 获取默认管理员账号（仅首次、一次性）
// @Description 首次部署时，用于在登录界面提示并复制保存默认管理员账号和密码。仅当系统刚创建默认管理员且尚未被读取时生效。
// @Tags auth
// @Produce json
// @Success 20000 {object} response.Response{data=service.DefaultAdminCreds}
// @Failure 404 {object} response.Response
// @Router /api/v1/auth/default-admin-once [get]
func (c *UserController) GetDefaultAdminOnce(ctx *gin.Context) {
	creds := c.svc.ConsumeDefaultAdminCreds()
	if creds == nil {
		response.NotFound(ctx, "默认管理员信息不可用或已被读取")
		return
	}
	response.Success(ctx, creds)
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取Token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "登录信息"
// @Success 20000 {object} response.Response{data=dto.LoginResponse}
// @Failure 30009 {object} response.Response
// @Router /api/v1/auth/login [post]
func (c *UserController) Login(ctx *gin.Context) {
	var req dto.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamErrorWithDetails(ctx, "参数校验失败", parseValidationErrors(err))
		return
	}

	// 获取客户端信息
	ip := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")

	// 登录
	tokens, user, err := c.svc.Login(ctx, req.Username, req.Password, ip, userAgent)
	if err != nil {
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "登录失败")
		return
	}

	response.Success(ctx, dto.LoginResponse{
		User:         formatUserResponse(user),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresAt.Unix() - time.Now().Unix(),
	})
}

// RefreshToken 刷新Token
// @Summary 刷新Token
// @Description 使用Refresh Token刷新Access Token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "刷新信息"
// @Success 20000 {object} response.Response{data=dto.RefreshTokenResponse}
// @Failure 30012 {object} response.Response
// @Router /api/v1/auth/refresh [post]
func (c *UserController) RefreshToken(ctx *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "参数缺失")
		return
	}

	tokens, err := c.svc.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "刷新失败")
		return
	}

	response.Success(ctx, dto.RefreshTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresAt.Unix() - time.Now().Unix(),
	})
}

// Logout 用户登出（前端兼容：仅返回成功；如需 token 黑名单可扩展）
// @Summary 用户登出
// @Description 用户登出（客户端删除token即可）
// @Tags auth
// @Produce json
// @Security Bearer
// @Router /api/v1/auth/logout [post]
func (c *UserController) Logout(ctx *gin.Context) {
	response.SuccessWithMessage(ctx, "logout ok", nil)
}
