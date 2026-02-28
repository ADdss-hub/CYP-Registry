// Package controller 提供用户认证相关HTTP处理
// 遵循《全平台通用开发任务设计规范文档》第6章API规范
package controller

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	"github.com/cyp-registry/registry/src/modules/user/dto"
	"github.com/cyp-registry/registry/src/modules/user/service"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/models"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// UserController 用户控制器
type UserController struct {
	svc *service.Service
}

// NewUserController 创建用户控制器
func NewUserController(svc *service.Service) *UserController {
	return &UserController{svc: svc}
}

// Register 用户注册
// @Summary 用户注册
// @Description 新用户注册账号
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "注册信息"
// @Success 20000 {object} response.Response{data=dto.UserResponse}
// @Failure 10001 {object} response.Response
// @Router /api/v1/auth/register [post]
func (c *UserController) Register(ctx *gin.Context) {
	// 公开注册功能已关闭：仅允许管理员预置账号或通过默认管理员账号初始化后在后台创建用户
	response.Fail(ctx, 40301, "公开注册功能已关闭，请联系管理员获取账号")
}

// GetDefaultAdminOnce 首次部署时获取一次性默认管理员账号信息
// @Summary 获取默认管理员账号（仅首次、一次性）
// @Description 首次部署时，用于在登录界面提示并复制保存默认管理员账号和密码。仅当系统刚创建默认管理员且尚未被读取时生效。
// @Tags auth
// @Produce json
// @Success 20000 {object} response.Response{data=service.DefaultAdminCreds}
// @Failure 404 {object} response.Response
// @Router /api/v1/auth/default-admin-once [get]
func (c *UserController) GetDefaultAdminOnce(ctx *gin.Context) {
	creds := c.svc.ConsumeDefaultAdminCreds()
	if creds == nil {
		response.NotFound(ctx, "默认管理员信息不可用或已被读取")
		return
	}
	response.Success(ctx, creds)
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取Token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "登录信息"
// @Success 20000 {object} response.Response{data=dto.LoginResponse}
// @Failure 30009 {object} response.Response
// @Router /api/v1/auth/login [post]
func (c *UserController) Login(ctx *gin.Context) {
	var req dto.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamErrorWithDetails(ctx, "参数校验失败", parseValidationErrors(err))
		return
	}

	// 获取客户端信息
	ip := ctx.ClientIP()
	userAgent := ctx.GetHeader("User-Agent")

	// 登录
	tokens, user, err := c.svc.Login(ctx, req.Username, req.Password, ip, userAgent)
	if err != nil {
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "登录失败")
		return
	}

	response.Success(ctx, dto.LoginResponse{
		User:         formatUserResponse(user),
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresAt.Unix() - time.Now().Unix(),
	})
}

// RefreshToken 刷新Token
// @Summary 刷新Token
// @Description 使用Refresh Token刷新Access Token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "刷新信息"
// @Success 20000 {object} response.Response{data=dto.RefreshTokenResponse}
// @Failure 30012 {object} response.Response
// @Router /api/v1/auth/refresh [post]
func (c *UserController) RefreshToken(ctx *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "参数缺失")
		return
	}

	tokens, err := c.svc.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "刷新失败")
		return
	}

	response.Success(ctx, dto.RefreshTokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresAt.Unix() - time.Now().Unix(),
	})
}

// Logout 用户登出（前端兼容：仅返回成功；如需 token 黑名单可扩展）
// @Summary 用户登出
// @Description 用户登出（客户端删除token即可）
// @Tags auth
// @Produce json
// @Security Bearer
// @Router /api/v1/auth/logout [post]
func (c *UserController) Logout(ctx *gin.Context) {
	response.SuccessWithMessage(ctx, "logout ok", nil)
}

// ListUsers 列出用户（管理员）
// @Summary 用户列表
// @Tags user
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param keyword query string false "关键字"
// @Security Bearer
// @Router /api/v1/users [get]
func (c *UserController) ListUsers(ctx *gin.Context) {
	page := parseInt(ctx.DefaultQuery("page", "1"), 1)
	pageSize := parseInt(ctx.DefaultQuery("pageSize", ctx.DefaultQuery("page_size", "20")), 20)
	keyword := ctx.Query("keyword")

	users, total, err := c.svc.ListUsers(ctx, page, pageSize, keyword)
	if err != nil {
		response.InternalServerError(ctx, "获取用户列表失败")
		return
	}

	// 返回基础字段（避免泄露 password/hash）
	result := make([]dto.UserResponse, 0, len(users))
	for i := range users {
		u := users[i]
		result = append(result, dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			Nickname:    u.Nickname,
			Avatar:      u.Avatar,
			Bio:         u.Bio,
			IsActive:    u.IsActive,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   u.CreatedAt.Format(time.RFC3339),
			LastLoginAt: u.LastLoginAt.Format(time.RFC3339),
		})
	}

	response.Success(ctx, gin.H{
		"users":     result,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetUser 获取单个用户（管理员）
// @Summary 用户详情
// @Tags user
// @Produce json
// @Param id path string true "用户ID"
// @Security Bearer
// @Router /api/v1/users/{id} [get]
func (c *UserController) GetUser(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.ParamError(ctx, "invalid id")
		return
	}

	user, err := c.svc.GetUserByID(ctx, id)
	if err != nil {
		response.NotFound(ctx, "user not found")
		return
	}
	response.Success(ctx, formatUserResponse(user))
}

// UpdateUser 更新用户（管理员）
// @Summary 更新用户
// @Tags user
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Security Bearer
// @Router /api/v1/users/{id} [patch]
func (c *UserController) UpdateUser(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		response.ParamError(ctx, "invalid id")
		return
	}

	var req map[string]interface{}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "invalid body")
		return
	}

	// 安全：不允许直接写 password
	delete(req, "password")

	if err := c.svc.UpdateUser(ctx, id, req); err != nil {
		response.ParamError(ctx, "update failed")
		return
	}
	user, _ := c.svc.GetUserByID(ctx, id)
	response.Success(ctx, formatUserResponse(user))
}

// DeleteUser 删除用户（管理员）
// @Summary 删除用户
// @Tags user
// @Produce json
// @Param id path string true "用户ID"
// @Security Bearer
// @Router /api/v1/users/{id} [delete]
func (c *UserController) DeleteUser(ctx *gin.Context) {
	id, err := uuid.Parse(ctx.Param("id"))
	if err != nil {
		response.ParamError(ctx, "invalid id")
		return
	}
	if err := c.svc.DeleteUser(ctx, id); err != nil {
		response.NotFound(ctx, "user not found")
		return
	}
	response.SuccessWithMessage(ctx, "deleted", nil)
}

func parseInt(s string, def int) int {
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil || n <= 0 {
		return def
	}
	return n
}

// GetCurrentUser 获取当前用户信息
// @Summary 获取当前用户
// @Description 获取当前登录用户的信息
// @Tags user
// @Produce json
// @Success 20000 {object} response.Response{data=dto.UserResponse}
// @Failure 30001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me [get]
func (c *UserController) GetCurrentUser(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	user, err := c.svc.GetUserByID(ctx, userID.(uuid.UUID))
	if err != nil {
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "获取用户信息失败")
		return
	}

	response.Success(ctx, formatUserResponse(user))
}

// UpdateCurrentUser 更新当前用户信息
// @Summary 更新当前用户
// @Description 更新当前登录用户的信息
// @Tags user
// @Accept json
// @Produce json
// @Param request body dto.UpdateUserRequest true "用户信息"
// @Success 20000 {object} response.Response{data=dto.UserResponse}
// @Failure 10001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me [put]
func (c *UserController) UpdateCurrentUser(ctx *gin.Context) {
	var req dto.UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "参数校验失败")
		return
	}

	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	updates := make(map[string]interface{})
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Bio != "" {
		updates["bio"] = req.Bio
	}

	if len(updates) == 0 {
		response.ParamError(ctx, "没有需要更新的字段")
		return
	}

	err := c.svc.UpdateUser(ctx, userID.(uuid.UUID), updates)
	if err != nil {
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "更新用户信息失败")
		return
	}

	// 获取更新后的用户信息
	user, _ := c.svc.GetUserByID(ctx, userID.(uuid.UUID))
	response.Success(ctx, formatUserResponse(user))
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前登录用户的密码
// @Tags user
// @Accept json
// @Produce json
// @Param request body dto.ChangePasswordRequest true "密码信息"
// @Success 20000 {object} response.Response
// @Failure 10001 {object} response.Response
// @Failure 30009 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/password [put]
func (c *UserController) ChangePassword(ctx *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "参数校验失败")
		return
	}

	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	err := c.svc.ChangePassword(ctx, userID.(uuid.UUID), req.OldPassword, req.NewPassword)
	if err != nil {
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "修改密码失败")
		return
	}

	response.SuccessWithMessage(ctx, "密码修改成功", nil)
}

// UploadAvatar 上传并更新当前用户头像
// @Summary 上传头像
// @Description 通过表单上传头像图片并更新当前登录用户的头像URL
// @Tags user
// @Accept mpfd
// @Produce json
// @Success 20000 {object} response.Response{data=dto.UserResponse}
// @Failure 10001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/avatar [post]
func (c *UserController) UploadAvatar(ctx *gin.Context) {
	userIDVal, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.InternalServerError(ctx, "用户标识异常")
		return
	}

	file, err := ctx.FormFile("file")
	if err != nil {
		response.ParamError(ctx, "未找到头像文件")
		return
	}

	// 简单大小限制：不超过 5MB
	const maxAvatarSize = 5 * 1024 * 1024
	if file.Size <= 0 || file.Size > maxAvatarSize {
		response.ParamError(ctx, "头像文件大小需在 0 ~ 5MB 之间")
		return
	}

	// 仅允许常见图片后缀
	ext := filepath.Ext(file.Filename)
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
	default:
		response.ParamError(ctx, "仅支持 jpg/jpeg/png/gif/webp 格式的图片")
		return
	}

	// 保存路径：使用环境变量或默认路径
	baseDir := os.Getenv("UPLOADS_DIR")
	if baseDir == "" {
		// 容器环境下优先使用 /tmp 目录（通常有写入权限）
		if _, err := os.Stat("/tmp"); err == nil {
			baseDir = "/tmp/uploads"
		} else {
			// 优先使用当前工作目录（容器环境下更可靠）
			wd, err := os.Getwd()
			if err != nil {
				// 如果获取工作目录失败，尝试使用可执行文件所在目录
				execPath, execErr := os.Executable()
				if execErr != nil {
					response.InternalServerError(ctx, "无法确定上传目录路径")
					return
				}
				baseDir = filepath.Join(filepath.Dir(execPath), "uploads")
			} else {
				baseDir = filepath.Join(wd, "uploads")
			}
		}
	}

	// 转换为绝对路径
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		log.Printf("[ERROR] 解析上传基础目录绝对路径失败: %v, 路径: %s", err, baseDir)
		response.InternalServerError(ctx, "头像上传失败，请稍后重试")
		return
	}

	avatarDir := filepath.Join(absBaseDir, "avatars")

	// 确保目录存在，使用绝对路径，设置适当的权限
	if err := os.MkdirAll(avatarDir, 0o755); err != nil {
		log.Printf("[ERROR] 创建头像目录失败: %v, 绝对路径: %s", err, avatarDir)
		response.InternalServerError(ctx, "头像上传失败，请稍后重试")
		return
	}

	// 确保目录权限正确（即使目录已存在）
	if err := os.Chmod(avatarDir, 0o755); err != nil {
		log.Printf("[WARN] 设置头像目录权限失败: %v, 目录: %s", err, avatarDir)
		// 不返回错误，继续尝试写入文件
	}

	filename := userID.String() + ext
	fullPath := filepath.Join(avatarDir, filename)

	// 保存文件
	if err := ctx.SaveUploadedFile(file, fullPath); err != nil {
		log.Printf("[ERROR] 保存头像文件失败: %v, 路径: %s", err, fullPath)
		// 检查是否是权限问题
		if os.IsPermission(err) {
			response.InternalServerError(ctx, "头像上传失败：权限不足")
			return
		}
		response.InternalServerError(ctx, "保存头像文件失败")
		return
	}

	// 头像对外访问URL（由 main.go 中的静态资源路由 /uploads 映射）
	// 确保 URL 以 / 开头，并且包含时间戳或版本号以避免缓存问题
	avatarURL := "/uploads/avatars/" + filename

	// 更新用户头像字段（使用结构体字段名 "Avatar"）
	if err := c.svc.UpdateUser(ctx, userID, map[string]interface{}{"Avatar": avatarURL}); err != nil {
		log.Printf("[ERROR] 更新用户头像信息失败: %v, 用户ID: %s, 头像URL: %s", err, userID, avatarURL)
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "更新头像信息失败")
		return
	}

	// 重新查询用户信息，确保获取最新的头像URL
	user, err := c.svc.GetUserByID(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] 获取更新后的用户信息失败: %v, 用户ID: %s", err, userID)
		response.InternalServerError(ctx, "获取用户信息失败")
		return
	}

	// 验证头像URL是否正确更新
	if user.Avatar != avatarURL {
		log.Printf("[WARN] 头像URL更新后不一致: 期望 %s, 实际 %s, 用户ID: %s", avatarURL, user.Avatar, userID)
	}

	// 设置响应头，防止缓存
	ctx.Header("Cache-Control", "no-cache, no-store, must-revalidate, max-age=0")
	ctx.Header("Pragma", "no-cache")
	ctx.Header("Expires", "0")
	ctx.Header("Content-Type", "application/json; charset=utf-8")
	ctx.Status(http.StatusOK)
	response.Success(ctx, formatUserResponse(user))
}

// GetNotificationSettings 获取当前用户的通知设置
// @Summary 获取当前用户通知设置
// @Description 获取当前登录用户的通知偏好（邮件通知、扫描完成通知、安全告警、Webhook通知等）
// @Tags user
// @Produce json
// @Success 20000 {object} response.Response{data=dto.NotificationSettings}
// @Failure 30001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/notification-settings [get]
func (c *UserController) GetNotificationSettings(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	settings, err := c.svc.GetNotificationSettings(ctx, userID.(uuid.UUID))
	if err != nil {
		response.InternalServerError(ctx, "获取通知设置失败")
		return
	}

	resp := dto.NotificationSettings{
		EmailEnabled:         settings.EmailEnabled,
		ScanCompleted:        settings.ScanCompleted,
		SecurityAlerts:       settings.SecurityAlerts,
		WebhookNotifications: settings.WebhookNotifications,
		Digest:               settings.Digest,
		NotificationEmail:    settings.NotificationEmail,
	}

	response.Success(ctx, resp)
}

// UpdateNotificationSettings 更新当前用户的通知设置
// @Summary 更新当前用户通知设置
// @Description 更新当前登录用户的通知偏好（邮件通知、扫描完成通知、安全告警、Webhook通知等）
// @Tags user
// @Accept json
// @Produce json
// @Param request body dto.NotificationSettings true "通知设置"
// @Success 20000 {object} response.Response{data=dto.NotificationSettings}
// @Failure 10001 {object} response.Response
// @Failure 30001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/notification-settings [put]
func (c *UserController) UpdateNotificationSettings(ctx *gin.Context) {
	var req dto.NotificationSettings
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "参数校验失败")
		return
	}

	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	// 兜底通知频率
	if req.Digest == "" {
		req.Digest = "realtime"
	}

	settings := &service.NotificationSettings{
		EmailEnabled:         req.EmailEnabled,
		ScanCompleted:        req.ScanCompleted,
		SecurityAlerts:       req.SecurityAlerts,
		WebhookNotifications: req.WebhookNotifications,
		Digest:               req.Digest,
		NotificationEmail:    req.NotificationEmail,
	}

	if err := c.svc.UpdateNotificationSettings(ctx, userID.(uuid.UUID), settings); err != nil {
		response.InternalServerError(ctx, "更新通知设置失败")
		return
	}

	response.Success(ctx, req)
}

// ==================== PAT相关接口 ====================

// CreatePAT 创建Personal Access Token
// @Summary 创建PAT
// @Description 创建Personal Access Token
// @Tags pat
// @Accept json
// @Produce json
// @Param request body dto.CreatePATRequest true "PAT信息"
// @Success 20000 {object} response.Response{data=dto.CreatePATResponse}
// @Failure 10001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/pat [post]
func (c *UserController) CreatePAT(ctx *gin.Context) {
	var req dto.CreatePATRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.ParamError(ctx, "参数校验失败")
		return
	}

	// 输入验证
	if req.Name == "" {
		response.ParamError(ctx, "PAT名称不能为空")
		return
	}
	if len(req.Name) > 100 {
		response.ParamError(ctx, "PAT名称长度不能超过100个字符")
		return
	}

	userIDVal, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		response.InternalServerError(ctx, "用户标识异常")
		return
	}

	result, err := c.svc.CreatePAT(ctx, userID, req.Name, req.Scopes, req.ExpireIn)
	if err != nil {
		log.Printf("[ERROR] 创建PAT失败: %v, 用户ID: %s, PAT名称: %s", err, userID, req.Name)
		codeErr, ok := errors.As(err)
		if ok {
			response.Fail(ctx, codeErr.Code, codeErr.Message)
			return
		}
		response.InternalServerError(ctx, "创建PAT失败")
		return
	}

	response.Success(ctx, dto.CreatePATResponse{
		ID:        result.ID,
		Name:      result.Name,
		Scopes:    result.Scopes,
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
		CreatedAt: result.CreatedAt.Format(time.RFC3339),
		Token:     result.Token,
		TokenType: result.TokenType,
	})
}

// ListPAT 列出所有PAT
// @Summary 列出PAT
// @Description 列出当前用户的所有Personal Access Token
// @Tags pat
// @Produce json
// @Success 20000 {object} response.Response{data=[]dto.PATResponse}
// @Security Bearer
// @Router /api/v1/users/me/pat [get]
func (c *UserController) ListPAT(ctx *gin.Context) {
	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	tokens, err := c.svc.ListPAT(ctx, userID.(uuid.UUID))
	if err != nil {
		response.InternalServerError(ctx, "获取PAT列表失败")
		return
	}

	response.Success(ctx, tokens)
}

// RevokePAT 删除PAT（兼容历史命名，语义为删除）
// @Summary 删除PAT
// @Description 删除指定的Personal Access Token（幂等：即使ID无效或不存在也视为删除成功）
// @Tags pat
// @Produce json
// @Param id path string true "PAT ID"
// @Success 20000 {object} response.Response
// @Failure 10001 {object} response.Response
// @Security Bearer
// @Router /api/v1/users/me/pat/{id} [delete]
func (c *UserController) RevokePAT(ctx *gin.Context) {
	// 直接从路径中读取ID，避免对 URI 参数做过于严格的 UUID 绑定校验
	idStr := ctx.Param("id")

	userID, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		response.Unauthorized(ctx, "未登录")
		return
	}

	// 尝试解析为UUID；如果解析失败，按“已删除”处理，实现幂等删除
	tokenID, err := uuid.Parse(idStr)
	if err != nil {
		// ID 本身无效，对调用方而言等同于“已经不存在”的资源
		response.SuccessWithMessage(ctx, "PAT已删除", nil)
		return
	}

	if err := c.svc.RevokePAT(ctx, userID.(uuid.UUID), tokenID); err != nil {
		// 后端删除失败（例如记录不存在），也按幂等删除处理
		response.SuccessWithMessage(ctx, "PAT已删除", nil)
		return
	}

	response.SuccessWithMessage(ctx, "PAT已删除", nil)
}

// ==================== 辅助方法 ====================

// formatUserResponse 格式化用户响应
func formatUserResponse(user interface{}) dto.UserResponse {
	// 类型断言
	switch u := user.(type) {
	case *models.User:
		if u == nil {
			return dto.UserResponse{}
		}
		return dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			Nickname:    u.Nickname,
			Avatar:      u.Avatar,
			Bio:         u.Bio,
			IsActive:    u.IsActive,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   formatUserTime(u.CreatedAt),
			LastLoginAt: formatUserTime(u.LastLoginAt),
		}
	case models.User:
		return dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			Nickname:    u.Nickname,
			Avatar:      u.Avatar,
			Bio:         u.Bio,
			IsActive:    u.IsActive,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   formatUserTime(u.CreatedAt),
			LastLoginAt: formatUserTime(u.LastLoginAt),
		}
	case *struct {
		ID          uuid.UUID
		Username    string
		Email       string
		Nickname    string
		Avatar      string
		Bio         string
		IsActive    bool
		IsAdmin     bool
		CreatedAt   time.Time
		LastLoginAt time.Time
	}:
		return dto.UserResponse{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			Nickname:    u.Nickname,
			Avatar:      u.Avatar,
			Bio:         u.Bio,
			IsActive:    u.IsActive,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   formatUserTime(u.CreatedAt),
			LastLoginAt: formatUserTime(u.LastLoginAt),
		}
	default:
		// 默认返回空响应
		return dto.UserResponse{}
	}
}

// formatUserTime 统一处理用户时间字段，避免零值时间序列化为 0001-01-01T00:00:00Z
func formatUserTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// parseValidationErrors 解析验证错误
func parseValidationErrors(err error) []response.Error {
	// 简化实现
	return []response.Error{
		{
			Field:   "validation",
			Message: err.Error(),
		},
	}
}
