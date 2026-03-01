// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/cyp-registry/registry/src/pkg/response"
)

// HasScope 检查PAT令牌是否拥有指定的scope
// 如果使用的是JWT token（非PAT），返回true（JWT token继承用户所有权限）
// 如果使用的是PAT token，检查scopes是否包含指定的scope
// 返回值：hasPermission bool, errorCode int, errorMessage string
func HasScope(ctx *gin.Context, requiredScope string) (bool, int, string) {
	tokenTypeVal, exists := ctx.Get(ContextKeyTokenType)
	if !exists {
		// 没有token类型信息，可能是JWT token，默认允许
		return true, 0, ""
	}

	tokenType, ok := tokenTypeVal.(string)
	if !ok || tokenType != "pat" {
		// 不是PAT token，是JWT token，继承用户所有权限
		return true, 0, ""
	}

	// 是PAT token，检查scopes
	scopesVal, scopesExists := ctx.Get(ContextKeyPATScopes)
	if !scopesExists {
		// scopes不存在，返回相应错误码
		return false, response.CodePATMissingScopes, "PAT令牌缺少权限信息"
	}

	scopes, ok := scopesVal.([]string)
	if !ok {
		return false, response.CodePATInvalidScopes, "PAT令牌权限信息格式错误"
	}

	// 检查是否包含所需的scope
	// 支持通配符：* 表示所有权限，admin:* 表示所有管理员权限
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == requiredScope || scope == "*" {
			return true, 0, ""
		}
		// admin scope 可以访问所有功能
		if scope == "admin" || scope == "admin:*" {
			return true, 0, ""
		}
		// write scope 包含 read 权限
		if requiredScope == "read" && (scope == "write" || scope == "delete") {
			return true, 0, ""
		}
		// delete scope 包含 write 和 read 权限
		if (requiredScope == "read" || requiredScope == "write") && scope == "delete" {
			return true, 0, ""
		}
	}

	// 根据requiredScope返回相应的错误码
	switch requiredScope {
	case "read":
		return false, response.CodePATMissingReadScope, "PAT令牌缺少读取权限，请在创建令牌时选择'读取'权限"
	case "write":
		return false, response.CodePATMissingWriteScope, "PAT令牌缺少写入权限，请在创建令牌时选择'写入'权限"
	case "delete":
		return false, response.CodePATMissingDeleteScope, "PAT令牌缺少删除权限，请在创建令牌时选择'删除'权限"
	case "admin":
		return false, response.CodePATMissingAdminScope, "PAT令牌缺少管理员权限，请在创建令牌时选择'管理'权限"
	default:
		return false, response.CodeInsufficientPermission, "PAT令牌权限不足"
	}
}
