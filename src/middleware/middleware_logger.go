// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"bytes"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/pkg/config"
)

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

		// 统一使用 JSON 格式日志（默认格式）
		// 如果配置了非 JSON 格式，也使用 JSON 格式以确保一致性
		level := "info"
		if status >= 400 {
			level = "error"
		}

		// 获取当前时间（使用 Asia/Shanghai 时区，UTC+8）
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Now().In(loc)
		// 构建 JSON 日志，前面加上可读的时间戳前缀（格式：2026/03/01 13:21:00）
		// timestamp 字段使用本地时间的 RFC3339 格式（带时区偏移）
		log.Printf(`%s {"timestamp":"%s","level":"%s","trace_id":"%s","method":"%s","path":"%s","status":%d,"latency_ms":%.2f,"ip":"%s","user_agent":"%s"}`,
			now.Format("2006/01/02 15:04:05"),
			now.Format(time.RFC3339),
			level,
			traceID,
			ctx.Request.Method,
			ctx.Request.URL.Path,
			status,
			float64(latency.Nanoseconds())/1e6,
			ctx.ClientIP(),
			ctx.Request.UserAgent(),
		)
	}
}
