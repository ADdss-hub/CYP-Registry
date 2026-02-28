// Package project DTO数据传输对象
// 定义项目管理的请求和响应结构
package dto

import "time"

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	Name         string `json:"name" binding:"required,min=2,max=255"`
	Description  string `json:"description" binding:"omitempty,max=2000"`
	IsPublic     bool   `json:"is_public"`
	StorageQuota int64  `json:"storage_quota"` // 单位：字节
}

// UpdateProjectRequest 更新项目请求
type UpdateProjectRequest struct {
	Name         *string `json:"name,omitempty" binding:"omitempty,min=2,max=255"`
	Description  *string `json:"description,omitempty" binding:"omitempty,max=2000"`
	IsPublic     *bool   `json:"is_public,omitempty"`
	StorageQuota *int64  `json:"storage_quota,omitempty"` // 单位：字节
}

// ProjectResponse 项目响应
type ProjectResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	OwnerID      string    `json:"owner_id"`
	IsPublic     bool      `json:"is_public"`
	StorageUsed  int64     `json:"storage_used"`
	StorageQuota int64     `json:"storage_quota"`
	ImageCount   int       `json:"image_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProjectListResponse 项目列表响应
type ProjectListResponse struct {
	Projects []ProjectResponse `json:"projects"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

// AddMemberRequest 添加成员请求
type AddMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
	RoleID string `json:"role_id" binding:"required"`
}

// UpdateMemberRoleRequest 更新成员角色请求
type UpdateMemberRoleRequest struct {
	RoleID string `json:"role_id" binding:"required"`
}

// MemberResponse 成员响应
type MemberResponse struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	UserID    string    `json:"user_id"`
	RoleID    string    `json:"role_id"`
	Username  string    `json:"username,omitempty"`
	Email     string    `json:"email,omitempty"`
	JoinedAt  time.Time `json:"joined_at"`
}

// MemberListResponse 成员列表响应
type MemberListResponse struct {
	Members []MemberResponse `json:"members"`
	Total   int64            `json:"total"`
}

// QuotaUpdateResponse 配额更新响应
type QuotaUpdateResponse struct {
	ProjectID    string `json:"project_id"`
	OldQuota     int64  `json:"old_quota"`
	NewQuota     int64  `json:"new_quota"`
	StorageUsed  int64  `json:"storage_used"`
	StorageLeft  int64  `json:"storage_left"`
}

// StorageUsageResponse 存储使用量响应
type StorageUsageResponse struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	StorageUsed int64  `json:"storage_used"`
	StorageQuota int64 `json:"storage_quota"`
	UsagePercent float64 `json:"usage_percent"`
}
