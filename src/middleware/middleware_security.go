// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

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
