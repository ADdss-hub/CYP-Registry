// Package dto 定义数据传输对象
// 用于API请求和响应的数据格式定义
package dto

import "github.com/google/uuid"

// ==================== 用户相关DTO ====================

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	// username 规则：允许字母数字 + _ . -，且必须以字母/数字开头
	// 具体格式校验在 service 层做更友好的错误提示（避免 validator 标签表达受限）
	Username string `json:"username" binding:"required,min=3,max=64"`
	Email    string `json:"email" binding:"required,email"`
	// bcrypt 输入最大有效长度为 72，超出部分会被截断；这里直接限制到 72
	Password string `json:"password" binding:"required,min=8,max=72"`
	Nickname string `json:"nickname" binding:"max=128"`
}

// LoginRequest 用户登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"` // 秒
}

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse 刷新Token响应
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// UserResponse 用户信息响应
type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Nickname    string    `json:"nickname"`
	Avatar      string    `json:"avatar"`
	Bio         string    `json:"bio"`
	IsActive    bool      `json:"is_active"`
	IsAdmin     bool      `json:"is_admin"`
	CreatedAt   string    `json:"created_at"`
	LastLoginAt string    `json:"last_login_at"`
}

// NotificationSettings 用户通知设置
// 用于控制是否接收不同类型的通知，以及汇总频率
type NotificationSettings struct {
	EmailEnabled         bool   `json:"email_enabled"`         // 是否开启邮件通知
	ScanCompleted        bool   `json:"scan_completed"`        // 是否接收扫描完成通知
	SecurityAlerts       bool   `json:"security_alerts"`       // 是否接收安全告警
	WebhookNotifications bool   `json:"webhook_notifications"` // 是否接收 Webhook 通知
	Digest               string `json:"digest"`                // 通知频率: realtime/daily/weekly
	NotificationEmail    string `json:"notification_email"`    // 通知接收邮箱（为空则可由后端回退到账号邮箱）
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Nickname string `json:"nickname" binding:"max=128"`
	// Avatar 为可选字段：当未提供或为空字符串时不做 URL 校验，避免无头像时更新资料出现“参数校验失败”
	Avatar string `json:"avatar" binding:"omitempty,url,max=512"`
	Bio    string `json:"bio" binding:"max=500"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=20"`
}

// ==================== PAT相关DTO ====================

// CreatePATRequest 创建PAT请求
type CreatePATRequest struct {
	Name   string   `json:"name" binding:"required,max=128"`
	Scopes []string `json:"scopes" binding:"required,min=1"`
	// ExpireIn 过期时间（秒）
	//   - >0: 使用该值作为过期秒数
	//   - =0: 使用配置中的默认过期时间
	//   - <0: 表示永不过期
	ExpireIn int64 `json:"expire_in"`
}

// PATResponse PAT响应
type PATResponse struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Scopes     []string  `json:"scopes"`
	ExpiresAt  string    `json:"expires_at"`
	CreatedAt  string    `json:"created_at"`
	LastUsedAt *string   `json:"last_used_at"`
}

// CreatePATResponse 创建PAT响应（包含完整token）
type CreatePATResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt string    `json:"expires_at"`
	CreatedAt string    `json:"created_at"`
	Token     string    `json:"token"` // 只返回一次
	TokenType string    `json:"token_type"`
}

// TokenInfoResponse 当前Token信息响应
type TokenInfoResponse struct {
	TokenType string       `json:"token_type"` // "jwt" 或 "pat"
	User      UserResponse `json:"user"`        // 用户信息
	PATID     *uuid.UUID   `json:"pat_id,omitempty"` // PAT ID（仅PAT token时返回）
	Scopes    []string     `json:"scopes,omitempty"` // PAT权限范围（仅PAT token时返回）
	HasRead   bool         `json:"has_read"`   // 是否有读取权限
	HasWrite  bool         `json:"has_write"`  // 是否有写入权限
	HasDelete bool         `json:"has_delete"` // 是否有删除权限
	HasAdmin  bool         `json:"has_admin"`  // 是否有管理员权限
}

// ==================== 项目相关DTO ====================

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	Name        string `json:"name" binding:"required,min=3,max=128,alphanum"`
	Description string `json:"description" binding:"max=500"`
	IsPublic    bool   `json:"is_public"`
}

// UpdateProjectRequest 更新项目请求
type UpdateProjectRequest struct {
	Description  string `json:"description" binding:"max=500"`
	IsPublic     *bool  `json:"is_public"`
	StorageQuota int64  `json:"storage_quota"` // 字节
}

// ProjectResponse 项目响应
type ProjectResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	OwnerID      uuid.UUID `json:"owner_id"`
	IsPublic     bool      `json:"is_public"`
	StorageUsed  int64     `json:"storage_used"`
	StorageQuota int64     `json:"storage_quota"`
	ImageCount   int       `json:"image_count"`
	CreatedAt    string    `json:"created_at"`
}

// ==================== 成员相关DTO ====================

// AddMemberRequest 添加成员请求
type AddMemberRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	RoleID uuid.UUID `json:"role_id" binding:"required"`
}

// MemberResponse 成员响应
type MemberResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Nickname string    `json:"nickname"`
	RoleID   uuid.UUID `json:"role_id"`
	RoleName string    `json:"role_name"`
	AddedAt  string    `json:"added_at"`
}

// ==================== 通用DTO ====================

// IDRequest ID请求
type IDRequest struct {
	ID uuid.UUID `json:"id" uri:"id" binding:"required"`
}

// PageRequest 分页请求
type PageRequest struct {
	Page     int `json:"page" binding:"min=1"`
	PageSize int `json:"page_size" binding:"min=1,max=100"`
}

// DeleteResponse 删除响应
type DeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
