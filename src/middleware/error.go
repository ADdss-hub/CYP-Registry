// Package middleware 提供全局错误处理中间件
// 遵循《全平台通用开发任务设计规范文档》第6.4节错误处理规范
package middleware

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/response"
	"github.com/gin-gonic/gin"
)

// ErrorHandlerMiddleware 全局错误处理中间件
type ErrorHandlerMiddleware struct{}

// NewErrorHandlerMiddleware 创建全局错误处理中间件
func NewErrorHandlerMiddleware() *ErrorHandlerMiddleware {
	return &ErrorHandlerMiddleware{}
}

// ErrorHandler 全局错误处理中间件处理函数
// 捕获所有panic和错误，统一记录日志并返回错误响应
func (e *ErrorHandlerMiddleware) ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取TraceID
		traceID, _ := c.Get(ContextKeyTraceID)
		traceIDStr := "unknown"
		if id, ok := traceID.(string); ok {
			traceIDStr = id
		}

		// 处理请求
		c.Next()

		// 检查HTTP状态码，记录4xx和5xx错误
		status := c.Writer.Status()
		if status >= 400 {
			// 获取错误信息
			errMsg := fmt.Sprintf("HTTP %d", status)
			if len(c.Errors) > 0 {
				errMsg = c.Errors.String()
			}

			// 记录错误日志
			log.Printf("[ERROR] [%s] %s %s - Status: %d, Error: %s",
				traceIDStr,
				c.Request.Method,
				c.Request.URL.Path,
				status,
				errMsg,
			)

			// JSON格式日志
			log.Printf(`{"timestamp":"%s","level":"error","trace_id":"%s","method":"%s","path":"%s","status":%d,"error":"%s"}`,
				time.Now().Format(time.RFC3339),
				traceIDStr,
				c.Request.Method,
				c.Request.URL.Path,
				status,
				strings.ReplaceAll(errMsg, `"`, `\"`),
			)
		}

		// 检查是否有错误
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				// 获取错误堆栈
				stack := make([]byte, 4096)
				length := runtime.Stack(stack, false)
				stackStr := string(stack[:length])

				// 记录错误日志
				errMsg := err.Error()
				log.Printf("[ERROR] [%s] %s %s - Error: %s\nStack:\n%s",
					traceIDStr,
					c.Request.Method,
					c.Request.URL.Path,
					errMsg,
					stackStr,
				)

				// JSON格式日志
				log.Printf(`{"timestamp":"%s","level":"error","trace_id":"%s","method":"%s","path":"%s","error":"%s","stack":"%s"}`,
					time.Now().Format(time.RFC3339),
					traceIDStr,
					c.Request.Method,
					c.Request.URL.Path,
					strings.ReplaceAll(errMsg, `"`, `\"`),
					strings.ReplaceAll(strings.ReplaceAll(stackStr, "\n", "\\n"), `"`, `\"`),
				)

				// 如果是CodeError，使用其错误码和消息
				if codeErr, ok := errors.As(err.Err); ok {
					response.Fail(c, codeErr.Code, codeErr.Message)
					return
				}

				// 默认返回内部服务器错误
				if !c.Writer.Written() {
					response.InternalServerError(c, "服务器内部错误，请稍后重试")
				}
			}
		}
	}
}
