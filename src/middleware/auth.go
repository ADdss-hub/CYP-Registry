// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/cyp-registry/registry/src/modules/auth/jwt"
	"github.com/cyp-registry/registry/src/modules/user/service"
	"github.com/cyp-registry/registry/src/pkg/models"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// ContextKey 上下文键名
const (
	ContextKeyUserID    = "user_id"
	ContextKeyUsername  = "username"
	ContextKeyTraceID   = "trace_id"
	ContextKeyTokenType = "token_type"
	ContextKeyPATScopes = "pat_scopes"
	ContextKeyPATID     = "pat_id"
)

// AuthMiddleware 认证中间件
type AuthMiddleware struct {
	svc *service.Service
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(svc *service.Service) *AuthMiddleware {
	return &AuthMiddleware{svc: svc}
}

// Auth 认证中间件处理函数
// 支持JWT Bearer Token和PAT两种认证方式
func (m *AuthMiddleware) Auth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 从Header获取Token
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(ctx, "缺少认证信息")
			ctx.Abort()
			return
		}

		var claims *jwt.TokenClaims
		var err error

		var patModel *models.PersonalAccessToken
		// 判断认证类型
		if len(authHeader) >= 7 && authHeader[:7] == "Bearer " {
			// Bearer Token：优先支持 JWT；同时兼容 Bearer pat_v1_xxx（令牌免账号密码场景）
			raw := authHeader[7:]
			if strings.HasPrefix(raw, "pat_v1_") {
				patModel, patErr := m.svc.ValidatePAT(ctx, raw)
				if patErr == nil && patModel != nil {
					claims = &jwt.TokenClaims{
						UserID:    patModel.UserID,
						Username:  "",
						TokenType: "pat",
					}
					err = nil
				} else {
					err = patErr
				}
			} else {
				// JWT Bearer Token
				claims, err = m.svc.ValidateAccessToken(raw)
			}
		} else if len(authHeader) >= 7 && strings.HasPrefix(authHeader, "pat_v1_") {
			// Personal Access Token (格式: pat_v1_<token>)
			patModel, err = m.svc.ValidatePAT(ctx, authHeader)
			if err == nil && patModel != nil {
				// 验证成功，设置用户信息
				claims = &jwt.TokenClaims{
					UserID:    patModel.UserID,
					Username:  "", // PAT不包含用户名
					TokenType: "pat",
				}
			}
		} else {
			err = fmt.Errorf("不支持的认证类型")
		}

		if err != nil || claims == nil {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"auth","operation":"auth_failed","ip":"%s","user_agent":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), ctx.ClientIP(), ctx.Request.UserAgent(), err)
			response.Unauthorized(ctx, "认证失败")
			ctx.Abort()
			return
		}

		// 设置用户信息到上下文
		ctx.Set(ContextKeyUserID, claims.UserID)
		ctx.Set(ContextKeyUsername, claims.Username)
		ctx.Set(ContextKeyTokenType, claims.TokenType)

		// 如果是PAT令牌，解析并存储scopes信息
		if patModel != nil {
			var scopes []string
			if patModel.Scopes != "" {
				if parseErr := json.Unmarshal([]byte(patModel.Scopes), &scopes); parseErr != nil {
					// 如果解析失败，尝试作为单个字符串处理（兼容旧数据）
					if patModel.Scopes != "" && patModel.Scopes != "[]" {
						scopes = []string{patModel.Scopes}
					}
				}
			}
			ctx.Set(ContextKeyPATScopes, scopes)
			ctx.Set(ContextKeyPATID, patModel.ID)
		}

		ctx.Next()
	}
}
