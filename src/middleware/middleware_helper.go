// Package middleware 提供Gin中间件
// 包含认证、日志、限流等功能
package middleware

// 辅助函数
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
