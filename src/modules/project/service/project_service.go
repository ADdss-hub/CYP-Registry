// Package project 项目管理模块
// 提供项目（镜像仓库）的CRUD操作和配额管理
package project

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cyp-registry/registry/src/modules/storage"
	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/cyp-registry/registry/src/pkg/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrProjectNotFound 项目不存在
var ErrProjectNotFound = errors.New("project: project not found")

// ErrProjectExists 项目已存在
var ErrProjectExists = errors.New("project: project already exists")

// ErrQuotaExceeded 配额超限
var ErrQuotaExceeded = errors.New("project: storage quota exceeded")

// ErrInvalidQuota 无效的配额值
var ErrInvalidQuota = errors.New("project: invalid quota value")

// DefaultQuota 默认存储配额（10GB）
const DefaultQuota int64 = 10 * 1024 * 1024 * 1024

// Project 项目实体（数据库模型）
type Project struct {
	ID           string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	Name         string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"name"`
	Description  string         `gorm:"type:text" json:"description"`
	OwnerID      string         `gorm:"type:varchar(36);index;not null" json:"owner_id"`
	IsPublic     bool           `gorm:"default:false" json:"is_public"`
	StorageUsed  int64          `gorm:"default:0" json:"storage_used"`
	StorageQuota int64          `gorm:"default:10737418240" json:"storage_quota"` // 默认10GB
	ImageCount   int            `gorm:"default:0" json:"image_count"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// TableName 指定表名
func (Project) TableName() string {
	return "registry_projects"
}

// ProjectMember 项目成员
type ProjectMember struct {
	ID        string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	ProjectID string         `gorm:"type:varchar(36);index;not null" json:"project_id"`
	UserID    string         `gorm:"type:varchar(36);index;not null" json:"user_id"`
	RoleID    string         `gorm:"type:varchar(36);not null" json:"role_id"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// TableName 指定表名
func (ProjectMember) TableName() string {
	return "registry_project_members"
}

// Service 项目服务接口
type Service interface {
	// CRUD操作
	CreateProject(ctx context.Context, name, description, ownerID string, isPublic bool, storageQuota int64) (*Project, error)
	GetProject(ctx context.Context, projectID string) (*Project, error)
	GetProjectByName(ctx context.Context, name string) (*Project, error)
	UpdateProject(ctx context.Context, projectID string, updates map[string]interface{}) error
	DeleteProject(ctx context.Context, projectID string) error

	// 列表查询
	ListProjects(ctx context.Context, userID string, page, pageSize int) ([]Project, int64, error)
	ListUserProjects(ctx context.Context, userID string, page, pageSize int) ([]Project, int64, error)

	// 配额管理
	UpdateQuota(ctx context.Context, projectID string, quota int64) error
	CheckQuota(ctx context.Context, projectID string, additionalSize int64) (bool, error)
	UpdateStorageUsage(ctx context.Context, projectID string, delta int64) error

	// 访问控制（仅基于公开性与项目所有者，无团队/成员角色）
	CanAccess(ctx context.Context, userID, projectID string, action string) (bool, error)
	IsOwner(ctx context.Context, userID, projectID string) (bool, error)

	// 统计
	GetStatistics(ctx context.Context, userID string) (int64, int64, int64, error)
}

// projectService 项目服务实现
type projectService struct {
	db      *gorm.DB
	storage storage.Storage
	cfg     *config.Config
}

// NewService 创建项目服务
func NewService(db *gorm.DB, store storage.Storage, cfg *config.Config) Service {
	return &projectService{
		db:      db,
		storage: store,
		cfg:     cfg,
	}
}

// CreateProject 创建项目
func (s *projectService) CreateProject(ctx context.Context, name, description, ownerID string, isPublic bool, storageQuota int64) (*Project, error) {
	// 验证输入参数
	if name == "" {
		return nil, fmt.Errorf("项目名不能为空")
	}
	if ownerID == "" {
		return nil, fmt.Errorf("所有者ID不能为空")
	}

	// 规范化项目名（去除首尾空格）
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("项目名不能为空")
	}

	// 创建项目
	project := &Project{
		ID:           uuid.New().String(),
		Name:         name,
		Description:  description,
		OwnerID:      ownerID,
		StorageQuota: DefaultQuota,
		IsPublic:     isPublic,
		StorageUsed:  0,
		ImageCount:   0,
	}

	// 请求中显式传入 storageQuota 且合法时优先使用（<=0 则忽略，继续用默认/配置）
	if storageQuota > 0 {
		project.StorageQuota = storageQuota
	}

	// 设置默认配额
	if s.cfg != nil {
		defaultQuota := s.cfg.GetInt64("project.default_quota")
		// 只有当未显式指定 storageQuota 时才应用默认配置
		if defaultQuota > 0 && storageQuota <= 0 {
			project.StorageQuota = defaultQuota
		}
	}

	// 使用事务创建项目，确保数据一致性（在事务内部检查是否存在，避免竞态条件）
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 在事务内部检查项目名是否已存在（排除软删除的项目）
		var count int64
		if err := tx.Model(&Project{}).Where("name = ? AND deleted_at IS NULL", name).Count(&count).Error; err != nil {
			return fmt.Errorf("检查项目是否存在失败: %w", err)
		}
		if count > 0 {
			return ErrProjectExists
		}

		// 创建项目
		if err := tx.Create(project).Error; err != nil {
			// 检查是否是唯一约束冲突（项目名重复）
			if strings.Contains(err.Error(), "duplicate key") ||
				strings.Contains(err.Error(), "UNIQUE constraint") ||
				strings.Contains(err.Error(), "violates unique constraint") ||
				strings.Contains(err.Error(), "Duplicate entry") {
				return ErrProjectExists
			}
			return fmt.Errorf("创建项目失败: %w", err)
		}

		return nil
	})

	if err != nil {
		if err == ErrProjectExists {
			return nil, ErrProjectExists
		}
		return nil, err
	}

	// 重新查询创建的项目，确保获取完整数据（包括数据库自动填充的字段）
	var createdProject Project
	if err := s.db.Where("id = ? AND deleted_at IS NULL", project.ID).First(&createdProject).Error; err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"project","operation":"create","project_id":"%s","project_name":"%s","owner_id":"%s","error":"failed to query created project: %v"}`, time.Now().Format(time.RFC3339), project.ID, name, ownerID, err)
		return nil, fmt.Errorf("查询创建的项目失败: %w", err)
	}

	log.Printf(`{"timestamp":"%s","level":"info","module":"project","operation":"create","project_id":"%s","project_name":"%s","owner_id":"%s","is_public":%t,"storage_quota":%d}`, time.Now().Format(time.RFC3339), createdProject.ID, name, ownerID, isPublic, createdProject.StorageQuota)
	return &createdProject, nil
}

// GetProject 获取项目
func (s *projectService) GetProject(ctx context.Context, projectID string) (*Project, error) {
	var project Project
	err := s.db.Where("id = ? AND deleted_at IS NULL", projectID).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return &project, nil
}

// GetProjectByName 通过名称获取项目
func (s *projectService) GetProjectByName(ctx context.Context, name string) (*Project, error) {
	var project Project
	err := s.db.Where("name = ? AND deleted_at IS NULL", name).First(&project).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, err
	}
	return &project, nil
}

// UpdateProject 更新项目
func (s *projectService) UpdateProject(ctx context.Context, projectID string, updates map[string]interface{}) error {
	result := s.db.Model(&Project{}).Where("id = ? AND deleted_at IS NULL", projectID).Updates(updates)
	if result.Error != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"project","operation":"update","project_id":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), projectID, result.Error)
		return result.Error
	}
	if result.RowsAffected == 0 {
		log.Printf(`{"timestamp":"%s","level":"warn","module":"project","operation":"update","project_id":"%s","error":"project not found"}`, time.Now().Format(time.RFC3339), projectID)
		return ErrProjectNotFound
	}
	log.Printf(`{"timestamp":"%s","level":"info","module":"project","operation":"update","project_id":"%s","updates":%v}`, time.Now().Format(time.RFC3339), projectID, updates)
	return nil
}

// DeleteProject 删除项目
func (s *projectService) DeleteProject(ctx context.Context, projectID string) error {
	// 获取项目信息
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	// 删除项目数据（软删除）
	if err := s.db.Delete(project).Error; err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"project","operation":"delete","project_id":"%s","project_name":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), projectID, project.Name, err)
		return err
	}

	// 清理存储数据
	_ = project.Name
	// 注意：这里只是标记删除，实际存储数据可能需要异步清理

	log.Printf(`{"timestamp":"%s","level":"info","module":"project","operation":"delete","project_id":"%s","project_name":"%s"}`, time.Now().Format(time.RFC3339), projectID, project.Name)
	return nil
}

// ListProjects 列出所有项目（管理员）
func (s *projectService) ListProjects(ctx context.Context, userID string, page, pageSize int) ([]Project, int64, error) {
	var projects []Project
	var total int64

	offset := (page - 1) * pageSize

	// 查询总数
	if err := s.db.Model(&Project{}).Where("deleted_at IS NULL").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := s.db.Where("deleted_at IS NULL").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

// ListUserProjects 列出用户有权限访问的项目
func (s *projectService) ListUserProjects(ctx context.Context, userID string, page, pageSize int) ([]Project, int64, error) {
	var projects []Project
	var total int64

	offset := (page - 1) * pageSize

	// 查询用户作为所有者或成员的项目
	subQuery := s.db.Model(&ProjectMember{}).
		Select("project_id").
		Where("user_id = ? AND deleted_at IS NULL", userID)

	query := s.db.Model(&Project{}).
		Where("(owner_id = ? OR id IN (?)) AND deleted_at IS NULL", userID, subQuery)

	// 查询总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 查询列表
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

// UpdateQuota 更新存储配额
func (s *projectService) UpdateQuota(ctx context.Context, projectID string, quota int64) error {
	if quota < 0 {
		return ErrInvalidQuota
	}

	// 检查新配额是否小于已使用空间
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	if quota < project.StorageUsed {
		log.Printf(`{"timestamp":"%s","level":"warn","module":"project","operation":"update_quota","project_id":"%s","old_quota":%d,"new_quota":%d,"storage_used":%d,"error":"quota exceeded"}`, time.Now().Format(time.RFC3339), projectID, project.StorageQuota, quota, project.StorageUsed)
		return ErrQuotaExceeded
	}

	err = s.UpdateProject(ctx, projectID, map[string]interface{}{
		"storage_quota": quota,
	})
	if err == nil {
		log.Printf(`{"timestamp":"%s","level":"info","module":"project","operation":"update_quota","project_id":"%s","old_quota":%d,"new_quota":%d,"storage_used":%d}`, time.Now().Format(time.RFC3339), projectID, project.StorageQuota, quota, project.StorageUsed)
	}
	return err
}

// CheckQuota 检查配额是否充足
func (s *projectService) CheckQuota(ctx context.Context, projectID string, additionalSize int64) (bool, error) {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return false, err
	}

	return project.StorageUsed+additionalSize <= project.StorageQuota, nil
}

// UpdateStorageUsage 更新存储使用量
func (s *projectService) UpdateStorageUsage(ctx context.Context, projectID string, delta int64) error {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	newUsed := project.StorageUsed + delta
	if newUsed < 0 {
		newUsed = 0
	}

	// 检查是否超配额
	if newUsed > project.StorageQuota {
		log.Printf(`{"timestamp":"%s","level":"warn","module":"project","operation":"update_storage_usage","project_id":"%s","old_used":%d,"new_used":%d,"quota":%d,"delta":%d,"error":"quota exceeded"}`, time.Now().Format(time.RFC3339), projectID, project.StorageUsed, newUsed, project.StorageQuota, delta)
		return ErrQuotaExceeded
	}

	// 记录显著的存储使用量变化（超过100MB）
	if delta > 100*1024*1024 || delta < -100*1024*1024 {
		log.Printf(`{"timestamp":"%s","level":"info","module":"project","operation":"update_storage_usage","project_id":"%s","old_used":%d,"new_used":%d,"delta":%d}`, time.Now().Format(time.RFC3339), projectID, project.StorageUsed, newUsed, delta)
	}

	return s.UpdateProject(ctx, projectID, map[string]interface{}{
		"storage_used": newUsed,
	})
}

// CanAccess 检查用户是否有权限访问项目
func (s *projectService) CanAccess(ctx context.Context, userID, projectID string, action string) (bool, error) {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return false, err
	}

	// 公开项目允许pull操作
	if project.IsPublic && action == "pull" {
		return true, nil
	}

	// 未认证用户只能访问公开项目的pull
	if userID == "" {
		return false, nil
	}

	// 所有者有所有权限
	if project.OwnerID == userID {
		return true, nil
	}
	return false, nil
}

// IsOwner 检查是否是项目所有者
func (s *projectService) IsOwner(ctx context.Context, userID, projectID string) (bool, error) {
	project, err := s.GetProject(ctx, projectID)
	if err != nil {
		return false, err
	}
	return project.OwnerID == userID, nil
}

// GetStatistics 获取用户可访问项目的统计信息
func (s *projectService) GetStatistics(ctx context.Context, userID string) (totalProjects int64, totalImages int64, totalStorage int64, err error) {
	// 查询用户可访问的项目（所有者或成员）
	subQuery := s.db.Model(&ProjectMember{}).
		Select("project_id").
		Where("user_id = ? AND deleted_at IS NULL", userID)

	baseQuery := s.db.Model(&Project{}).
		Where("(owner_id = ? OR id IN (?)) AND deleted_at IS NULL", userID, subQuery)

	// 统计项目总数
	if err := baseQuery.Count(&totalProjects).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	// 统计镜像总数和存储总量
	var result struct {
		TotalImages  int64 `gorm:"column:total_images"`
		TotalStorage int64 `gorm:"column:total_storage"`
	}
	
	// 使用新的查询实例，避免 Count 操作影响后续查询
	statsQuery := s.db.Model(&Project{}).
		Where("(owner_id = ? OR id IN (?)) AND deleted_at IS NULL", userID, subQuery)
	
	if err := statsQuery.Select("COALESCE(SUM(image_count), 0) as total_images, COALESCE(SUM(storage_used), 0) as total_storage").
		Scan(&result).Error; err != nil {
		return 0, 0, 0, fmt.Errorf("failed to calculate statistics: %w", err)
	}

	return totalProjects, result.TotalImages, result.TotalStorage, nil
}

// EnsureDefaultQuota 确保默认配额常量已使用
var _ = response.ErrNotFound
var _ = database.DB
