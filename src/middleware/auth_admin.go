// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/pkg/response"
)

// AdminRequired 需要管理员权限
func (m *AuthMiddleware) AdminRequired() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 获取用户ID
		userIDVal, exists := ctx.Get(ContextKeyUserID)
		if !exists {
			response.Unauthorized(ctx, "未登录")
			ctx.Abort()
			return
		}

		userID, ok := userIDVal.(uuid.UUID)
		if !ok {
			response.Unauthorized(ctx, "无效的用户ID")
			ctx.Abort()
			return
		}

		// 查询用户信息，检查是否为管理员
		user, err := m.svc.GetUserByID(ctx.Request.Context(), userID)
		if err != nil {
			response.Unauthorized(ctx, "用户不存在")
			ctx.Abort()
			return
		}

		// 检查用户状态
		if !user.IsActive {
			response.Forbidden(ctx, "账户已被禁用")
			ctx.Abort()
			return
		}

		// 检查是否为管理员
		if !user.IsAdmin {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"auth","operation":"admin_required_failed","user_id":"%s","ip":"%s","path":"%s"}`, time.Now().Format(time.RFC3339), userID.String(), ctx.ClientIP(), ctx.Request.URL.Path)
			response.Forbidden(ctx, "需要管理员权限")
			ctx.Abort()
			return
		}

		// 如果使用的是PAT令牌，需要检查scopes是否包含管理员权限
		// 按照界面选择：选择什么权限就是什么权限
		tokenTypeVal, exists := ctx.Get(ContextKeyTokenType)
		if exists {
			tokenType, ok := tokenTypeVal.(string)
			if ok && tokenType == "pat" {
				// 获取PAT的scopes
				scopesVal, scopesExists := ctx.Get(ContextKeyPATScopes)
				if !scopesExists {
					// scopes不存在，拒绝访问管理员功能
					log.Printf(`{"timestamp":"%s","level":"warn","module":"auth","operation":"admin_required_pat_no_scopes","user_id":"%s","ip":"%s","path":"%s"}`, time.Now().Format(time.RFC3339), userID.String(), ctx.ClientIP(), ctx.Request.URL.Path)
					response.PATMissingScopes(ctx)
					ctx.Abort()
					return
				}

				scopes, ok := scopesVal.([]string)
				if !ok {
					log.Printf(`{"timestamp":"%s","level":"warn","module":"auth","operation":"admin_required_pat_invalid_scopes","user_id":"%s","ip":"%s","path":"%s"}`, time.Now().Format(time.RFC3339), userID.String(), ctx.ClientIP(), ctx.Request.URL.Path)
					response.PATInvalidScopes(ctx)
					ctx.Abort()
					return
				}

				// 检查scopes是否包含管理员权限
				// 支持的管理员权限标识：admin、admin:*、*
				hasAdminScope := false
				for _, scope := range scopes {
					scope = strings.TrimSpace(scope)
					if scope == "admin" || scope == "admin:*" || scope == "*" {
						hasAdminScope = true
						break
					}
				}

				if !hasAdminScope {
					log.Printf(`{"timestamp":"%s","level":"warn","module":"auth","operation":"admin_required_pat_no_admin_scope","user_id":"%s","scopes":"%v","ip":"%s","path":"%s"}`, time.Now().Format(time.RFC3339), userID.String(), scopes, ctx.ClientIP(), ctx.Request.URL.Path)
					response.PATMissingAdminScope(ctx)
					ctx.Abort()
					return
				}
			}
		}

		// 将用户信息设置到上下文
		ctx.Set(ContextKeyUsername, user.Username)

		ctx.Next()
	}
}
