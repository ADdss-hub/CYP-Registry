// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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
