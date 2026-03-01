// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/cyp-registry/registry/src/pkg/config"
)

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
