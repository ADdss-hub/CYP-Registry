// Package controller 提供用户资料相关 HTTP 处理
package controller

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	"github.com/cyp-registry/registry/src/modules/user/dto"
	"github.com/cyp-registry/registry/src/pkg/errors"
	"github.com/cyp-registry/registry/src/pkg/response"
)

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
