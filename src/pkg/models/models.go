// Package models 定义数据模型
// 遵循《全平台通用数据库个人管理规范》第11章
// 表名使用 snake_case，复数形式
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel 基类模型，包含所有表必备字段
// 遵循规范：id, created_at, updated_at, deleted_at (软删除)
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"` // 软删除字段
}

// BeforeCreate 创建前自动生成UUID
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// SoftDelete 软删除
func (b *BaseModel) SoftDelete(tx *gorm.DB) error {
	b.DeletedAt = gorm.DeletedAt{
		Time:  time.Now(),
		Valid: true,
	}
	return tx.Save(b).Error
}

// Restore 恢复软删除的记录
func (b *BaseModel) Restore(tx *gorm.DB) error {
	b.DeletedAt = gorm.DeletedAt{
		Time:  time.Time{},
		Valid: false,
	}
	return tx.Save(b).Error
}

// User 用户模型
// 表名: registry_users
type User struct {
	BaseModel
	Username string `gorm:"type:varchar(64);uniqueIndex;not null;comment:用户名" json:"username"`
	Email    string `gorm:"type:varchar(255);uniqueIndex;not null;comment:邮箱" json:"email"`
	Password string `gorm:"type:varchar(255);not null;comment:密码哈希" json:"-"` // 不返回密码
	Nickname string `gorm:"type:varchar(128);comment:昵称" json:"nickname"`
	Avatar   string `gorm:"type:varchar(512);comment:头像URL" json:"avatar"`
	Bio      string `gorm:"type:text;comment:个人简介" json:"bio"`
	IsActive bool   `gorm:"default:true;comment:是否激活" json:"is_active"`
	IsAdmin  bool   `gorm:"default:false;comment:是否管理员" json:"is_admin"`
	// FirstLogin 是否为首次登录用户，用于强制提示修改默认密码
	FirstLogin  bool      `gorm:"default:false;comment:是否首次登录用户" json:"first_login"`
	LastLoginAt time.Time `gorm:"comment:最后登录时间" json:"last_login_at"`
	LastLoginIP string    `gorm:"type:varchar(45);comment:最后登录IP" json:"last_login_ip"`
	LoginCount  int       `gorm:"default:0;comment:登录次数" json:"login_count"`
}

// TableName 指定表名
func (User) TableName() string {
	return "registry_users"
}

// Project 项目模型
// 表名: registry_projects
type Project struct {
	BaseModel
	Name         string    `gorm:"type:varchar(128);uniqueIndex;not null;comment:项目名称" json:"name"`
	Description  string    `gorm:"type:text;comment:项目描述" json:"description"`
	OwnerID      uuid.UUID `gorm:"type:uuid;not null;index;comment:所有者ID" json:"owner_id"`
	IsPublic     bool      `gorm:"default:false;comment:是否公开" json:"is_public"`
	StorageUsed  int64     `gorm:"default:0;comment:已使用存储(字节)" json:"storage_used"`
	StorageQuota int64     `gorm:"default:10737418240;comment:存储配额(字节)" json:"storage_quota"` // 默认10GB
	ImageCount   int       `gorm:"default:0;comment:镜像数量" json:"image_count"`
}

// TableName 指定表名
func (Project) TableName() string {
	return "registry_projects"
}

// Role 角色模型
// 表名: registry_roles
type Role struct {
	BaseModel
	Name        string `gorm:"type:varchar(64);uniqueIndex;not null;comment:角色名称" json:"name"`
	DisplayName string `gorm:"type:varchar(128);comment:显示名称" json:"display_name"`
	Description string `gorm:"type:text;comment:角色描述" json:"description"`
	IsSystem    bool   `gorm:"default:false;comment:是否系统角色" json:"is_system"`
}

// TableName 指定表名
func (Role) TableName() string {
	return "registry_roles"
}

// Permission 权限模型
// 表名: registry_permissions
type Permission struct {
	BaseModel
	Code        string `gorm:"type:varchar(128);uniqueIndex;not null;comment:权限代码" json:"code"`
	Name        string `gorm:"type:varchar(128);not null;comment:权限名称" json:"name"`
	Description string `gorm:"type:text;comment:权限描述" json:"description"`
	Resource    string `gorm:"type:varchar(64);index;comment:资源类型" json:"resource"`
	Action      string `gorm:"type:varchar(32);comment:操作类型" json:"action"`
}

// TableName 指定表名
func (Permission) TableName() string {
	return "registry_permissions"
}

// ProjectMember 项目成员关联模型
// 表名: registry_project_members
type ProjectMember struct {
	BaseModel
	ProjectID uuid.UUID `gorm:"type:uuid;not null;index;comment:项目ID" json:"project_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index;comment:用户ID" json:"user_id"`
	RoleID    uuid.UUID `gorm:"type:uuid;not null;index;comment:角色ID" json:"role_id"`
}

// TableName 指定表名
func (ProjectMember) TableName() string {
	return "registry_project_members"
}

// RolePermission 角色-权限关联模型
// 表名: registry_role_permissions
type RolePermission struct {
	BaseModel
	RoleID       uuid.UUID `gorm:"type:uuid;not null;index;comment:角色ID" json:"role_id"`
	PermissionID uuid.UUID `gorm:"type:uuid;not null;index;comment:权限ID" json:"permission_id"`
}

// TableName 指定表名
func (RolePermission) TableName() string {
	return "registry_role_permissions"
}

// RefreshToken Refresh Token模型
// 表名: registry_refresh_tokens
type RefreshToken struct {
	BaseModel
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index;comment:用户ID" json:"user_id"`
	Token     string     `gorm:"type:varchar(256);uniqueIndex;not null;comment:Token" json:"-"`
	ExpiresAt time.Time  `gorm:"not null;comment:过期时间" json:"expires_at"`
	RevokedAt *time.Time `gorm:"comment:撤销时间" json:"revoked_at"`
	UserAgent string     `gorm:"type:varchar(512);comment:用户代理" json:"user_agent"`
	IP        string     `gorm:"type:varchar(45);comment:IP地址" json:"ip"`
}

// TableName 指定表名
func (RefreshToken) TableName() string {
	return "registry_refresh_tokens"
}

// PersonalAccessToken Personal Access Token模型
// 表名: registry_pat_tokens
type PersonalAccessToken struct {
	BaseModel
	// 同一用户下 PAT 名称需要唯一：通过联合唯一索引 (user_id, name) 保证
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index;uniqueIndex:idx_pat_user_name;comment:用户ID" json:"user_id"`
	TokenHash  string     `gorm:"type:varchar(256);not null;uniqueIndex;comment:Token哈希" json:"-"`
	Name       string     `gorm:"type:varchar(128);not null;uniqueIndex:idx_pat_user_name;comment:Token名称" json:"name"`
	Scopes     string     `gorm:"type:text;comment:权限范围(JSON数组)" json:"scopes"`
	ExpiresAt  time.Time  `gorm:"not null;index;comment:过期时间" json:"expires_at"`
	LastUsedAt *time.Time `gorm:"comment:最后使用时间" json:"last_used_at"`
	RevokedAt  *time.Time `gorm:"index;comment:撤销时间" json:"revoked_at"`
}

// TableName 指定表名
func (PersonalAccessToken) TableName() string {
	return "registry_pat_tokens"
}

// AuditLog 审计日志模型
// 表名: registry_audit_logs
type AuditLog struct {
	BaseModel
	UserID     *uuid.UUID `gorm:"type:uuid;index;comment:用户ID(可选)" json:"user_id"`
	Action     string     `gorm:"type:varchar(64);not null;index;comment:操作类型" json:"action"`
	Resource   string     `gorm:"type:varchar(128);index;comment:资源类型" json:"resource"`
	ResourceID *uuid.UUID `gorm:"type:uuid;index;comment:资源ID" json:"resource_id"`
	IP         string     `gorm:"type:varchar(45);comment:IP地址" json:"ip"`
	UserAgent  string     `gorm:"type:varchar(512);comment:用户代理" json:"user_agent"`
	Details    string     `gorm:"type:text;comment:操作详情(JSON)" json:"details"`
	Status     string     `gorm:"type:varchar(32);comment:状态" json:"status"`
}

// TableName 指定表名
func (AuditLog) TableName() string {
	return "registry_audit_logs"
}

// Image 镜像模型
// 表名: registry_images
type Image struct {
	BaseModel
	ProjectID   uuid.UUID `gorm:"type:uuid;not null;index;comment:项目ID" json:"project_id"`
	Name        string    `gorm:"type:varchar(256);not null;comment:镜像名称" json:"name"`
	Description string    `gorm:"type:text;comment:镜像描述" json:"description"`
	TagsCount   int       `gorm:"default:0;comment:标签数量" json:"tags_count"`
}

// TableName 指定表名
func (Image) TableName() string {
	return "registry_images"
}

// ImageTag 镜像标签模型
// 表名: registry_image_tags
type ImageTag struct {
	BaseModel
	ImageID    uuid.UUID `gorm:"type:uuid;not null;index;comment:镜像ID" json:"image_id"`
	Name       string    `gorm:"type:varchar(128);not null;comment:标签名称" json:"name"`
	Digest     string    `gorm:"type:varchar(256);uniqueIndex;comment:摘要" json:"digest"`
	Manifest   string    `gorm:"type:text;comment:Manifest内容" json:"manifest"`
	Size       int64     `gorm:"default:0;comment:大小(字节)" json:"size"`
	LastPullAt time.Time `gorm:"comment:最后拉取时间" json:"last_pull_at"`
	PullCount  int       `gorm:"default:0;comment:拉取次数" json:"pull_count"`
}

// TableName 指定表名
func (ImageTag) TableName() string {
	return "registry_image_tags"
}

// SecurityEvent 安全事件模型
// 表名: registry_security_events
type SecurityEvent struct {
	BaseModel
	EventType  string     `gorm:"type:varchar(64);not null;index;comment:事件类型" json:"event_type"`
	Severity   string     `gorm:"type:varchar(32);not null;comment:严重程度" json:"severity"`
	UserID     *uuid.UUID `gorm:"type:uuid;index;comment:用户ID" json:"user_id"`
	IP         string     `gorm:"type:varchar(45);comment:IP地址" json:"ip"`
	Details    string     `gorm:"type:text;comment:事件详情" json:"details"`
	ResolvedAt *time.Time `gorm:"comment:解决时间" json:"resolved_at"`
}

// TableName 指定表名
func (SecurityEvent) TableName() string {
	return "registry_security_events"
}
