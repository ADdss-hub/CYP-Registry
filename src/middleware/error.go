// Package middleware 提供全局错误处理中间件
// 遵循《全平台通用开发任务设计规范文档》第6.4节错误处理规范
package middleware

import (
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
		// 处理请求
		c.Next()

		// 注意：错误日志已在 Logger 中间件中记录，这里不再重复记录
		// 只处理错误响应，不记录日志（避免重复）

		// 检查是否有错误（仅处理错误响应，不记录日志）
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
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
