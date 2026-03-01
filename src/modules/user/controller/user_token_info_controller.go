// Package controller 提供当前 Token 信息相关 HTTP 处理
package controller

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	"github.com/cyp-registry/registry/src/modules/user/dto"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// GetCurrentTokenInfo 获取当前Token信息（类型和权限）
// @Summary 获取当前Token信息
// @Description 获取当前使用的Token类型（JWT或PAT）以及权限信息
// @Tags user
// @Produce json
// @Success 20000 {object} response.Response{data=dto.TokenInfoResponse}
// @Failure 30001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/token-info [get]
func (c *UserController) GetCurrentTokenInfo(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	// 获取用户信息
	user, err := c.svc.GetUserByID(ctx, userID.(uuid.UUID))
	if err != nil {
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "获取用户信息失败")
		return
	}

	// 获取Token类型
	tokenTypeVal, tokenTypeExists := ctx.Get(middleware.ContextKeyTokenType)
	tokenType := "jwt" // 默认为JWT
	if tokenTypeExists {
		if tt, ok := tokenTypeVal.(string); ok {
			tokenType = tt
		}
	}

	// 构建响应
	resp := dto.TokenInfoResponse{
		TokenType: tokenType,
		User:      formatUserResponse(user),
		// JWT token默认拥有所有权限
		HasRead:   true,
		HasWrite:  true,
		HasDelete: true,
		HasAdmin:  user.IsAdmin,
	}

	// 如果是PAT token，获取scopes和权限信息
	if tokenType == "pat" {
		// 获取PAT ID
		patIDVal, patIDExists := ctx.Get(middleware.ContextKeyPATID)
		if patIDExists {
			if pid, ok := patIDVal.(uuid.UUID); ok {
				resp.PATID = &pid
			}
		}

		// 获取scopes
		scopesVal, scopesExists := ctx.Get(middleware.ContextKeyPATScopes)
		if scopesExists {
			if scopes, ok := scopesVal.([]string); ok {
				resp.Scopes = scopes
				// 根据scopes计算权限
				resp.HasRead = false
				resp.HasWrite = false
				resp.HasDelete = false
				resp.HasAdmin = false

				for _, scope := range scopes {
					scope = strings.TrimSpace(scope)
					switch scope {
					case "read":
						resp.HasRead = true
					case "write":
						resp.HasWrite = true
						resp.HasRead = true // write包含read
					case "delete":
						resp.HasDelete = true
						resp.HasWrite = true // delete包含write
						resp.HasRead = true  // delete包含read
					case "admin", "admin:*", "*":
						resp.HasAdmin = true
						resp.HasDelete = true
						resp.HasWrite = true
						resp.HasRead = true
					}
				}
			}
		}
	}

	response.Success(ctx, resp)
}
