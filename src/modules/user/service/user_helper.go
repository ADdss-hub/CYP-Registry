// Package service 提供用户认证相关业务逻辑
// 遵循《全平台通用用户认证设计规范》
package service

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"regexp"
	"strings"

	"github.com/cyp-registry/registry/src/pkg/config"
)

// maskEmail 脱敏邮箱地址，只显示前3个字符和@符号后的域名
func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***"
	}
	username := parts[0]
	domain := parts[1]
	if len(username) <= 3 {
		return "***@" + domain
	}
	return username[:3] + "***@" + domain
}

// hashToken 计算token哈希
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// defaultAdminUsernameFromAppName 将项目名称（APP_NAME / config.app.name）转换为合法用户名：
// - 允许字母数字及 _ . -，且必须以字母/数字开头
// - 长度 3-64
// 若无法得到合法值，则回退到 "cyp-registry"。
func defaultAdminUsernameFromAppName() string {
	// 优先使用全局配置（已应用环境变量覆盖）
	appName := ""
	if c := config.Get(); c != nil {
		appName = strings.TrimSpace(c.App.Name)
	}
	if appName == "" {
		appName = strings.TrimSpace(os.Getenv("APP_NAME"))
	}
	if appName == "" {
		appName = "CYP-Registry"
	}

	// 与单镜像入口脚本保持一致的规则：
	// 1. 统一转为小写
	// 2. 非 [a-z0-9] 字符全部替换为 '-'
	// 3. 连续的 '-' 折叠为一个
	// 4. 去掉首尾 '-'
	lower := strings.ToLower(appName)
	var b strings.Builder
	b.Grow(len(lower))
	prevDash := false
	for _, r := range lower {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			r = '-'
		}

		if r == '-' {
			if prevDash {
				continue
			}
			prevDash = true
		} else {
			prevDash = false
		}
		b.WriteRune(r)
	}
	username := strings.Trim(b.String(), "-")
	// 必须以字母/数字开头：否则加一个前缀
	if username == "" || !regexp.MustCompile(`^[a-z0-9]`).MatchString(username) {
		username = "cyp-" + username
		username = strings.Trim(username, "-")
	}
	// 限制长度 3-64
	if len(username) > 64 {
		username = username[:64]
		username = strings.Trim(username, "-")
	}
	if len(username) < 3 {
		username = "cyp-registry"
	}

	// 最终校验（与注册规则保持一致）
	usernameRe := regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{2,63}$`)
	if !usernameRe.MatchString(username) {
		return "cyp-registry"
	}
	return username
}
