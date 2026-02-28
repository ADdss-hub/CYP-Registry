// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/modules/auth/jwt"
	"github.com/cyp-registry/registry/src/modules/user/service"
	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// ContextKey 上下文键名
const (
	ContextKeyUserID   = "user_id"
	ContextKeyUsername = "username"
	ContextKeyTraceID  = "trace_id"
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
			patModel, err := m.svc.ValidatePAT(ctx, authHeader)
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
			response.Unauthorized(ctx, "认证失败")
			ctx.Abort()
			return
		}

		// 设置用户信息到上下文
		ctx.Set(ContextKeyUserID, claims.UserID)
		ctx.Set(ContextKeyUsername, claims.Username)

		ctx.Next()
	}
}

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
			patModel, err := m.svc.ValidatePAT(ctx, authHeader)
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
		}

		ctx.Next()
	}
}

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
			response.Forbidden(ctx, "需要管理员权限")
			ctx.Abort()
			return
		}

		// 将用户信息设置到上下文
		ctx.Set(ContextKeyUsername, user.Username)

		ctx.Next()
	}
}

// LoggerMiddleware 日志中间件
type LoggerMiddleware struct {
	config *config.LoggingConfig
}

// NewLoggerMiddleware 创建日志中间件
func NewLoggerMiddleware(cfg *config.LoggingConfig) *LoggerMiddleware {
	return &LoggerMiddleware{config: cfg}
}

// Logger 日志中间件处理函数
func (l *LoggerMiddleware) Logger() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 生成TraceID
		traceID := uuid.New().String()[:8]
		ctx.Set(ContextKeyTraceID, traceID)

		// 记录请求开始时间
		start := time.Now()

		// 读取请求体（用于日志记录）
		var requestBody []byte
		if ctx.Request.Body != nil {
			requestBody, _ = io.ReadAll(ctx.Request.Body)
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 处理请求
		ctx.Next()

		// 计算耗时
		latency := time.Since(start)

		// 获取状态码
		status := ctx.Writer.Status()

		// 根据日志格式输出
		if l.config != nil && l.config.Format == "json" {
			// JSON格式日志
			log.Printf(`{"timestamp":"%s","level":"info","trace_id":"%s","method":"%s","path":"%s","status":%d,"latency_ms":%.2f,"ip":"%s","user_agent":"%s"}`,
				time.Now().Format(time.RFC3339),
				traceID,
				ctx.Request.Method,
				ctx.Request.URL.Path,
				status,
				float64(latency.Nanoseconds())/1e6,
				ctx.ClientIP(),
				ctx.Request.UserAgent(),
			)
		} else {
			// 控制台格式日志
			log.Printf("[%s] %s %s %d %v %s %s",
				traceID,
				ctx.Request.Method,
				ctx.Request.URL.Path,
				status,
				latency,
				ctx.ClientIP(),
				ctx.Request.UserAgent(),
			)
		}

		// 如果是错误状态码，记录错误日志
		if status >= 400 {
			if l.config != nil && l.config.Format == "json" {
				log.Printf(`{"timestamp":"%s","level":"error","trace_id":"%s","method":"%s","path":"%s","status":%d,"latency_ms":%.2f,"ip":"%s","error":"HTTP %d"}`,
					time.Now().Format(time.RFC3339),
					traceID,
					ctx.Request.Method,
					ctx.Request.URL.Path,
					status,
					float64(latency.Nanoseconds())/1e6,
					ctx.ClientIP(),
					status,
				)
			} else {
				log.Printf("[ERROR] [%s] %s %s %d %v %s",
					traceID,
					ctx.Request.Method,
					ctx.Request.URL.Path,
					status,
					latency,
					ctx.ClientIP(),
				)
			}
		}
	}
}

// RequestIDMiddleware 请求ID中间件
type RequestIDMiddleware struct{}

// NewRequestIDMiddleware 创建请求ID中间件
func NewRequestIDMiddleware() *RequestIDMiddleware {
	return &RequestIDMiddleware{}
}

// RequestID 请求ID中间件处理函数
func (r *RequestIDMiddleware) RequestID() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 从Header获取或生成请求ID
		requestID := ctx.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()[:8]
		}

		ctx.Set(ContextKeyTraceID, requestID)
		ctx.Header("X-Request-ID", requestID)

		ctx.Next()
	}
}

// CORSMiddleware CORS中间件
type CORSMiddleware struct {
	config *config.CORSConfig
}

// NewCORSMiddleware 创建CORS中间件
func NewCORSMiddleware(cfg *config.CORSConfig) *CORSMiddleware {
	return &CORSMiddleware{config: cfg}
}

// CORS CORS中间件处理函数
func (c *CORSMiddleware) CORS() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")

		// 检查Origin是否在允许列表中
		allowed := false
		for _, o := range c.config.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Access-Control-Allow-Methods", joinStrings(c.config.AllowedMethods, ","))
			ctx.Header("Access-Control-Allow-Headers", joinStrings(c.config.AllowedHeaders, ","))
			ctx.Header("Access-Control-Max-Age", "86400")
		}

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}

		ctx.Next()
	}
}

// SecurityHeadersMiddleware 安全头中间件
type SecurityHeadersMiddleware struct{}

// NewSecurityHeadersMiddleware 创建安全头中间件
func NewSecurityHeadersMiddleware() *SecurityHeadersMiddleware {
	return &SecurityHeadersMiddleware{}
}

// SecurityHeaders 安全头中间件处理函数
func (s *SecurityHeadersMiddleware) SecurityHeaders() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path

		// 防止XSS
		ctx.Header("X-Content-Type-Options", "nosniff")
		ctx.Header("X-XSS-Protection", "1; mode=block")

		// Swagger UI 需要允许内联脚本和样式，以及 iframe
		if strings.HasPrefix(path, "/swagger/") {
			// 为 Swagger UI 放宽 CSP 策略，允许从 unpkg.com 加载资源
			ctx.Header("Content-Security-Policy", "default-src 'self' https://unpkg.com 'unsafe-inline' 'unsafe-eval'; script-src 'self' https://unpkg.com 'unsafe-inline' 'unsafe-eval'; style-src 'self' https://unpkg.com 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data: https:; connect-src 'self';")
			ctx.Header("X-Frame-Options", "SAMEORIGIN")
		} else {
			// 其他路径使用严格策略，但允许内联样式以支持Vue运行时
			ctx.Header("X-Frame-Options", "DENY")
			ctx.Header("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; img-src 'self' data: https:; font-src 'self' data: https:; connect-src 'self'")
		}

		// 严格传输安全（仅 HTTPS）
		if ctx.Request.TLS != nil {
			ctx.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// 缓存控制（Swagger 静态资源可以缓存）
		if strings.HasPrefix(path, "/swagger/") && (strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") || strings.HasSuffix(path, ".gif")) {
			ctx.Header("Cache-Control", "public, max-age=3600")
		} else {
			ctx.Header("Cache-Control", "no-store, no-cache, must-revalidate")
			ctx.Header("Pragma", "no-cache")
		}

		ctx.Next()
	}
}

// RecoveryMiddleware 恢复中间件
type RecoveryMiddleware struct{}

// NewRecoveryMiddleware 创建恢复中间件
func NewRecoveryMiddleware() *RecoveryMiddleware {
	return &RecoveryMiddleware{}
}

// Recovery 恢复中间件处理函数
// 增强版Recovery中间件，确保错误信息输出到日志
func (r *RecoveryMiddleware) Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// 获取TraceID
		traceID, _ := c.Get(ContextKeyTraceID)
		traceIDStr := "unknown"
		if id, ok := traceID.(string); ok {
			traceIDStr = id
		}

		// 获取错误堆栈
		stack := make([]byte, 4096)
		length := runtime.Stack(stack, false)
		stackStr := string(stack[:length])

		// 记录详细错误信息到日志
		errMsg := fmt.Sprintf("%v", recovered)
		log.Printf("[PANIC] [%s] %s %s - Error: %s\nStack:\n%s",
			traceIDStr,
			c.Request.Method,
			c.Request.URL.Path,
			errMsg,
			stackStr,
		)

		// 也输出JSON格式日志（如果配置了）
		log.Printf(`{"timestamp":"%s","level":"panic","trace_id":"%s","method":"%s","path":"%s","error":"%s","stack":"%s"}`,
			time.Now().Format(time.RFC3339),
			traceIDStr,
			c.Request.Method,
			c.Request.URL.Path,
			strings.ReplaceAll(errMsg, `"`, `\"`),
			strings.ReplaceAll(strings.ReplaceAll(stackStr, "\n", "\\n"), `"`, `\"`),
		)

		// 返回错误响应
		response.InternalServerError(c, "服务器内部错误，请稍后重试")
		c.Abort()
	})
}

// 辅助函数
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
