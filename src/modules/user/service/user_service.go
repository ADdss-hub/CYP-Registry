// Package service 聚合用户领域各子模块（账号、资料、安全、Token 等）的公共入口。
// 这里只保留 Service 结构体与构造函数，以及对外暴露的通用错误与数据结构，
// 具体业务逻辑拆分在 user_account_service.go / user_profile_service.go /
// user_security_service.go / user_token_service.go 中实现。
package service

import (
	"time"

	"github.com/cyp-registry/registry/src/modules/auth/jwt"
	"github.com/cyp-registry/registry/src/modules/auth/pat"
	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/cyp-registry/registry/src/pkg/errors"
)

// Config 用户服务配置
type Config struct {
	BcryptCost int
}

// Service 用户服务聚合根
type Service struct {
	cfg    *Config
	jwtSvc *jwt.Service
	patSvc *pat.Service
}

// DefaultAdminCreds 默认管理员凭据（仅在进程内短暂保存，用于前端首屏提示一次）
// 结构定义在这里，具体生成与消费逻辑在 user_account_service.go 中实现。
type DefaultAdminCreds struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
}

// 统一对外导出用户领域常用错误，便于 controller 层与其他模块使用。
var (
	ErrUserNotFound       = errors.ErrUserNotFound
	ErrUserAlreadyExists  = errors.ErrUserAlreadyExists
	ErrPasswordIncorrect  = errors.ErrPasswordIncorrect
	ErrAccountLocked      = errors.ErrAccountLocked
	ErrBruteForceDetected = errors.ErrBruteForceDetected
)

// NewService 创建用户服务
// jwtCfg: JWT 配置
// patCfg: PAT 配置
// bcryptCost: 密码哈希复杂度
func NewService(jwtCfg *config.JWTConfig, patCfg *config.PATConfig, bcryptCost int) *Service {
	svc := &Service{
		cfg: &Config{
			BcryptCost: bcryptCost,
		},
		jwtSvc: jwt.NewService(jwtCfg),
		patSvc: pat.NewService(patCfg),
	}

	// 保持与原实现一致：在服务初始化时自动尝试创建默认管理员账号（仅在用户表为空时执行一次）
	go func() {
		_ = svc.ensureDefaultAdmin()
	}()

	return svc
}
