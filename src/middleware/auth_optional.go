// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/cyp-registry/registry/src/modules/auth/jwt"
	"github.com/cyp-registry/registry/src/pkg/models"
)

// OptionalAuth 可选认证中间件
// 如果提供了Token则验证，否则跳过
// 支持Bearer Token、PAT和Basic Auth三种认证方式
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.Next()
			return
		}

		var claims *jwt.TokenClaims
		var err error
		var patModel *models.PersonalAccessToken

		// 判断认证类型
		if len(authHeader) >= 7 && authHeader[:7] == "Bearer " {
			// Bearer Token：兼容 JWT 与 Bearer pat_v1_xxx 两种形式
			raw := authHeader[7:]
			if strings.HasPrefix(raw, "pat_v1_") {
				// 直接使用 PAT 作为 Bearer Token（免 JWT 中转）
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
				// 标准 JWT Bearer Token
				claims, err = m.svc.ValidateAccessToken(raw)
			}
		} else if len(authHeader) >= 6 && authHeader[:6] == "Basic " {
			// Basic Auth：Docker 客户端使用此方式
			// 格式: Basic base64(username:password)
			// password 可以是 PAT (pat_v1_xxx) 或用户密码
			encoded := authHeader[6:]
			decodedBytes, decodeErr := base64.StdEncoding.DecodeString(encoded)
			if decodeErr == nil {
				parts := strings.SplitN(string(decodedBytes), ":", 2)
				if len(parts) == 2 {
					password := parts[1]
					// 检查 password 是否是 PAT
					if strings.HasPrefix(password, "pat_v1_") {
						patModel, patErr := m.svc.ValidatePAT(ctx, password)
						if patErr == nil && patModel != nil {
							claims = &jwt.TokenClaims{
								UserID:    patModel.UserID,
								Username:  parts[0],
								TokenType: "pat",
							}
							err = nil
						} else {
							err = patErr
						}
					} else {
						// 尝试用户名密码认证
						tokens, _, loginErr := m.svc.Login(ctx, parts[0], password, ctx.ClientIP(), ctx.Request.UserAgent())
						if loginErr == nil && tokens != nil {
							// 解析 JWT 获取用户信息
							claims, err = m.svc.ValidateAccessToken(tokens.AccessToken)
						} else {
							err = loginErr
						}
					}
				}
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
		}

		if err == nil && claims != nil {
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
		}

		ctx.Next()
	}
}
