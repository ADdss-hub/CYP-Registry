// Package errors 提供统一的错误处理和错误码定义
// 遵循《全平台通用开发任务设计规范文档》第6.4节错误码规范
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// 错误码规范：
// 10001-19999: 系统级错误(参数校验失败等)
// 20001-29999: 业务级错误(用户不存在等)
// 30001-39999: 权限错误(认证/授权失败)
// 50001-59999: 系统错误(服务器内部错误)

// 自定义错误类型
type CodeError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *CodeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *CodeError) Unwrap() error {
	return e.Err
}

// NewCodeError 创建自定义错误
func NewCodeError(code int, message string) *CodeError {
	return &CodeError{
		Code:    code,
		Message: message,
		Err:     nil,
	}
}

// NewCodeErrorWithErr 创建带原始错误的自定义错误
func NewCodeErrorWithErr(code int, message string, err error) *CodeError {
	return &CodeError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Wrap 包装错误
func (e *CodeError) Wrap(err error) *CodeError {
	return NewCodeErrorWithErr(e.Code, e.Message, err)
}

// 系统级错误码 (10001-19999)
var (
	ErrParamInvalid      = NewCodeError(10001, "参数无效")
	ErrParamMissing      = NewCodeError(10002, "参数缺失")
	ErrParamTypeMismatch = NewCodeError(10003, "参数类型不匹配")
	ErrValidationFailed  = NewCodeError(10004, "参数校验失败")
	ErrTooManyRequests   = NewCodeError(10005, "请求过于频繁")
	ErrRateLimitExceeded = NewCodeError(10006, "超过速率限制")
)

// 业务级错误码 (20001-29999)
var (
	ErrUserNotFound       = NewCodeError(20001, "用户不存在")
	ErrUserAlreadyExists  = NewCodeError(20002, "用户已存在")
	ErrProjectNotFound    = NewCodeError(20003, "项目不存在")
	ErrProjectExists      = NewCodeError(20004, "项目已存在")
	ErrImageNotFound      = NewCodeError(20005, "镜像不存在")
	ErrImageExists        = NewCodeError(20006, "镜像已存在")
	ErrTagNotFound        = NewCodeError(20007, "标签不存在")
	ErrBlobNotFound       = NewCodeError(20008, "Blob不存在")
	ErrDuplicateName      = NewCodeError(20009, "名称重复")
	ErrStorageExceeded    = NewCodeError(20010, "存储空间不足")
	ErrInvalidOperation   = NewCodeError(20011, "无效操作")
	ErrResourceLocked     = NewCodeError(20012, "资源被锁定")
)

// 权限错误码 (30001-39999)
var (
	ErrUnauthorized          = NewCodeError(30001, "未授权访问")
	ErrTokenExpired          = NewCodeError(30002, "Token已过期")
	ErrTokenInvalid          = NewCodeError(30003, "Token无效")
	ErrInsufficientPermission = NewCodeError(30004, "权限不足")
	ErrAccessDenied          = NewCodeError(30005, "访问被拒绝")
	ErrAccountLocked         = NewCodeError(30006, "账户已锁定")
	ErrAccountDisabled       = NewCodeError(30007, "账户已禁用")
	ErrPasswordExpired       = NewCodeError(30008, "密码已过期")
	ErrPasswordIncorrect     = NewCodeError(30009, "密码错误")
	ErrBruteForceDetected    = NewCodeError(30010, "检测到暴力破解行为")
	ErrIPBlocked             = NewCodeError(30011, "IP地址被封禁")
	ErrRefreshTokenInvalid   = NewCodeError(30012, "Refresh Token无效")
	ErrPATInvalid            = NewCodeError(30013, "Personal Access Token无效")
)

// 系统错误码 (50001-59999)
var (
	ErrInternalServer       = NewCodeError(50001, "服务器内部错误")
	ErrDatabaseError        = NewCodeError(50002, "数据库错误")
	ErrCacheError           = NewCodeError(50003, "缓存错误")
	ErrStorageError         = NewCodeError(50004, "存储错误")
	ErrThirdPartyService    = NewCodeError(50005, "第三方服务错误")
	ErrScanTimeout          = NewCodeError(50006, "扫描超时")
	ErrScanFailed           = NewCodeError(50007, "扫描失败")
	ErrWebhookFailed        = NewCodeError(50008, "Webhook调用失败")
	ErrConfigurationError   = NewCodeError(50009, "配置错误")
)

// HTTP状态码映射
func (e *CodeError) HTTPStatus() int {
	switch {
	case e.Code >= 10001 && e.Code <= 19999:
		return http.StatusBadRequest
	case e.Code >= 20001 && e.Code <= 29999:
		if e.Code == 20010 {
			return http.StatusForbidden
		}
		return http.StatusConflict
	case e.Code >= 30001 && e.Code <= 39999:
		if e.Code == 30006 || e.Code == 30011 {
			return http.StatusForbidden
		}
		return http.StatusUnauthorized
	case e.Code >= 50001 && e.Code <= 59999:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Is 判断错误类型
func Is(err, target error) bool {
	var codeErr *CodeError
	if errors.As(err, &codeErr) {
		var targetErr *CodeError
		if errors.As(target, &targetErr) {
			return codeErr.Code == targetErr.Code
		}
	}
	return errors.Is(err, target)
}

// As 转换为CodeError
func As(err error) (*CodeError, bool) {
	var codeErr *CodeError
	if errors.As(err, &codeErr) {
		return codeErr, true
	}
	return nil, false
}

