// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/cyp-registry/registry/src/pkg/response"
)

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

		// 记录 panic 错误（JSON 格式）
		errMsg := fmt.Sprintf("%v", recovered)
		loc, _ := time.LoadLocation("Asia/Shanghai")
		now := time.Now().In(loc)
		log.Printf(`%s {"timestamp":"%s","level":"panic","trace_id":"%s","method":"%s","path":"%s","error":"%s","stack":"%s"}`,
			now.Format("2006/01/02 15:04:05"),
			now.Format(time.RFC3339),
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
