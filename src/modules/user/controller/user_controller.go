// Package controller 提供用户认证相关HTTP处理
// 遵循《全平台通用开发任务设计规范文档》第6章API规范
package controller

import (
	"github.com/cyp-registry/registry/src/modules/user/service"
)

// UserController 用户控制器
type UserController struct {
	svc *service.Service
}

// NewUserController 创建用户控制器
func NewUserController(svc *service.Service) *UserController {
	return &UserController{svc: svc}
}
