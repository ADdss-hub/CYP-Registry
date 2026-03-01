// Package response 提供统一API响应格式
// 遵循《全平台通用开发任务设计规范文档》第6.3节响应格式规范
package response

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrNotFound 资源不存在错误
var ErrNotFound = errors.New("resource not found")

// 其他错误变量（供controller使用）
var (
	ErrInternalError = errors.New("internal server error")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrInvalidParams = errors.New("invalid parameters")
)

// 错误码常量
const (
	CodeSuccess                = 20000 // 成功
	CodeNotFound               = 20001 // 资源不存在
	CodeResourceExists         = 20002 // 资源已存在（冲突）
	CodeInvalidParams          = 10001 // 参数错误
	CodeUnauthorized           = 30001 // 未授权
	CodeForbidden              = 30003 // 禁止访问
	CodeInsufficientPermission = 30004 // 权限不足
	CodeInternalError          = 50001 // 内部服务器错误
	// PAT权限相关错误码
	CodePATMissingReadScope   = 30014 // PAT缺少读取权限
	CodePATMissingWriteScope  = 30015 // PAT缺少写入权限
	CodePATMissingDeleteScope = 30016 // PAT缺少删除权限
	CodePATMissingAdminScope  = 30017 // PAT缺少管理员权限
	CodePATMissingScopes      = 30018 // PAT缺少权限信息
	CodePATInvalidScopes      = 30019 // PAT权限信息格式错误
)

// Response 统一API响应结构
// 遵循规范：{code, message, data, timestamp, trace_id}
type Response struct {
	Code      int         `json:"code"`      // 错误码：20000成功, 30001参数错误等
	Message   string      `json:"message"`   // 响应消息
	Data      interface{} `json:"data"`      // 响应数据
	Timestamp int64       `json:"timestamp"` // 时间戳(ISO 8601)
	TraceID   string      `json:"trace_id"`  // 链路追踪ID
}

// PageData 分页数据结构
type PageData struct {
	List      interface{} `json:"list"`
	Total     int64       `json:"total"`
	Page      int         `json:"page"`
	PageSize  int         `json:"page_size"`
	TotalPage int         `json:"total_page"`
}

// Error 错误信息结构
type Error struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// NewResponse 创建统一响应
func NewResponse() *Response {
	return &Response{
		Timestamp: time.Now().Unix(),
		TraceID:   generateTraceID(),
	}
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      20000,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now().Unix(),
		TraceID:   getTraceID(c),
	})
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      20000,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		TraceID:   getTraceID(c),
	})
}

// SuccessWithPage 分页成功响应
func SuccessWithPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	totalPage := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPage++
	}

	pageData := PageData{
		List:      list,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
		TotalPage: totalPage,
	}

	c.JSON(http.StatusOK, Response{
		Code:      20000,
		Message:   "success",
		Data:      pageData,
		Timestamp: time.Now().Unix(),
		TraceID:   getTraceID(c),
	})
}

// Fail 失败响应（业务错误）
func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:      code,
		Message:   message,
		Data:      nil,
		Timestamp: time.Now().Unix(),
		TraceID:   getTraceID(c),
	})
}

// FailWithErrors 验证失败响应
func FailWithErrors(c *gin.Context, code int, message string, errors []Error) {
	c.JSON(http.StatusOK, Response{
		Code:      code,
		Message:   message,
		Data:      errors,
		Timestamp: time.Now().Unix(),
		TraceID:   getTraceID(c),
	})
}

// FailWithData 带数据的失败响应
func FailWithData(c *gin.Context, code int, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		TraceID:   getTraceID(c),
	})
}

// ParamError 参数错误响应（10001-19999）
func ParamError(c *gin.Context, message string) {
	Fail(c, 10001, message)
}

// ParamErrorWithDetails 参数错误响应（带详细信息）
func ParamErrorWithDetails(c *gin.Context, message string, errors []Error) {
	FailWithErrors(c, 10001, message, errors)
}

// Unauthorized 未授权响应（30001-39999）
func Unauthorized(c *gin.Context, message string) {
	Fail(c, 30001, message)
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, message string) {
	Fail(c, 30003, message)
}

// PermissionDenied 权限不足响应（通用）
func PermissionDenied(c *gin.Context, message string) {
	Fail(c, 30004, message)
}

// PATMissingReadScope PAT缺少读取权限响应
func PATMissingReadScope(c *gin.Context) {
	Fail(c, 30014, "PAT令牌缺少读取权限，请在创建令牌时选择'读取'权限")
}

// PATMissingWriteScope PAT缺少写入权限响应
func PATMissingWriteScope(c *gin.Context) {
	Fail(c, 30015, "PAT令牌缺少写入权限，请在创建令牌时选择'写入'权限")
}

// PATMissingDeleteScope PAT缺少删除权限响应
func PATMissingDeleteScope(c *gin.Context) {
	Fail(c, 30016, "PAT令牌缺少删除权限，请在创建令牌时选择'删除'权限")
}

// PATMissingAdminScope PAT缺少管理员权限响应
func PATMissingAdminScope(c *gin.Context) {
	Fail(c, 30017, "PAT令牌缺少管理员权限，请在创建令牌时选择'管理'权限")
}

// PATMissingScopes PAT缺少权限信息响应
func PATMissingScopes(c *gin.Context) {
	Fail(c, 30018, "PAT令牌缺少权限信息")
}

// PATInvalidScopes PAT权限信息格式错误响应
func PATInvalidScopes(c *gin.Context) {
	Fail(c, 30019, "PAT令牌权限信息格式错误")
}

// NotFound 资源不存在响应（20001-29999）
func NotFound(c *gin.Context, message string) {
	Fail(c, 20001, message)
}

// Conflict 资源冲突响应
func Conflict(c *gin.Context, message string) {
	Fail(c, 20002, message)
}

// InternalServerError 服务器内部错误响应（50001-59999）
func InternalServerError(c *gin.Context, message string) {
	Fail(c, 50001, message)
}

// ServiceUnavailable 服务不可用响应
func ServiceUnavailable(c *gin.Context, message string) {
	Fail(c, 50002, message)
}

// TooManyRequests 请求过于频繁响应
func TooManyRequests(c *gin.Context, message string) {
	Fail(c, 10002, message)
}

// 生成TraceID
func generateTraceID() string {
	return uuid.New().String()[:8]
}

// 从上下文获取TraceID
func getTraceID(c *gin.Context) string {
	if traceID, exists := c.Get("trace_id"); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return generateTraceID()
}

// SetTraceID 设置TraceID到上下文
func SetTraceID(c *gin.Context) string {
	traceID := generateTraceID()
	c.Set("trace_id", traceID)
	return traceID
}
