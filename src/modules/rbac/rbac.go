// Package rbac 提供基于角色的访问控制服务
// 遵循《全平台通用用户认证设计规范》RBAC规范
package rbac

import (
	"context"

	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/pkg/database"
	"github.com/cyp-registry/registry/src/pkg/models"
)

// Service RBAC服务
type Service struct{}

// NewService 创建RBAC服务
func NewService() *Service {
	return &Service{}
}

// InitDefaultRoles 初始化默认角色
func (s *Service) InitDefaultRoles(ctx context.Context) error {
	roles := []models.Role{
		{
			Name:        "owner",
			DisplayName: "项目所有者",
			Description: "项目所有者，拥有所有权限",
			IsSystem:    true,
		},
		{
			Name:        "maintainer",
			DisplayName: "维护者",
			Description: "项目维护者，可以管理镜像和标签",
			IsSystem:    true,
		},
		{
			Name:        "developer",
			DisplayName: "开发者",
			Description: "开发者，可以拉取和推送镜像",
			IsSystem:    true,
		},
		{
			Name:        "guest",
			DisplayName: "访客",
			Description: "访客，只能拉取公开镜像",
			IsSystem:    true,
		},
	}

	for _, role := range roles {
		var existing models.Role
		result := database.DB.Where("name = ?", role.Name).Limit(1).Find(&existing)
		if result.Error != nil {
			return result.Error
		}
		// 仅在不存在时创建，避免 GORM 输出 ErrRecordNotFound 日志
		if result.RowsAffected == 0 {
			if err := database.DB.Create(&role).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// InitDefaultPermissions 初始化默认权限
func (s *Service) InitDefaultPermissions(ctx context.Context) error {
	permissions := []models.Permission{
		// 项目权限
		{Code: "project:read", Name: "读取项目", Description: "查看项目信息", Resource: "project", Action: "read"},
		{Code: "project:write", Name: "编辑项目", Description: "编辑项目信息", Resource: "project", Action: "write"},
		{Code: "project:delete", Name: "删除项目", Description: "删除项目", Resource: "project", Action: "delete"},
		{Code: "project:manage_member", Name: "管理成员", Description: "添加/移除项目成员", Resource: "project", Action: "manage_member"},
		// 镜像权限
		{Code: "image:read", Name: "读取镜像", Description: "查看镜像列表和标签", Resource: "image", Action: "read"},
		{Code: "image:push", Name: "推送镜像", Description: "推送镜像", Resource: "image", Action: "push"},
		{Code: "image:pull", Name: "拉取镜像", Description: "拉取镜像", Resource: "image", Action: "pull"},
		{Code: "image:delete", Name: "删除镜像", Description: "删除镜像和标签", Resource: "image", Action: "delete"},
		// Tag权限
		{Code: "tag:read", Name: "读取标签", Description: "查看标签信息", Resource: "tag", Action: "read"},
		{Code: "tag:delete", Name: "删除标签", Description: "删除标签", Resource: "tag", Action: "delete"},
	}

	for _, perm := range permissions {
		var existing models.Permission
		result := database.DB.Where("code = ?", perm.Code).Limit(1).Find(&existing)
		if result.Error != nil {
			return result.Error
		}
		// 仅在不存在时创建，避免 GORM 输出 ErrRecordNotFound 日志
		if result.RowsAffected == 0 {
			if err := database.DB.Create(&perm).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// InitDefaultRolePermissions 初始化默认角色权限
func (s *Service) InitDefaultRolePermissions(ctx context.Context) error {
	// 角色权限映射
	rolePermissions := map[string][]string{
		"owner": {
			"project:read", "project:write", "project:delete", "project:manage_member",
			"image:read", "image:push", "image:pull", "image:delete",
			"tag:read", "tag:delete",
		},
		"maintainer": {
			"project:read", "project:write",
			"image:read", "image:push", "image:pull", "image:delete",
			"tag:read", "tag:delete",
		},
		"developer": {
			"project:read",
			"image:read", "image:push", "image:pull",
			"tag:read",
		},
		"guest": {
			"project:read",
			"image:read", "image:pull",
			"tag:read",
		},
	}

	for roleName, permCodes := range rolePermissions {
		// 获取角色
		var role models.Role
		if err := database.DB.Where("name = ?", roleName).Limit(1).Find(&role).Error; err != nil {
			continue
		}

		// 获取权限
		var permissions []models.Permission
		if err := database.DB.Where("code IN ?", permCodes).Find(&permissions).Error; err != nil {
			return err
		}

		// 为角色分配权限
		for _, perm := range permissions {
			var rp models.RolePermission
			result := database.DB.Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).Limit(1).Find(&rp)
			if result.Error != nil {
				return result.Error
			}
			// 不存在则创建，避免 ErrRecordNotFound 日志
			if result.RowsAffected == 0 {
				if err := database.DB.Create(&models.RolePermission{
					RoleID:       role.ID,
					PermissionID: perm.ID,
				}).Error; err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// AddMember 添加项目成员
func (s *Service) AddMember(ctx context.Context, projectID, userID, roleID uuid.UUID) error {
	// 检查是否已是成员
	var existing models.ProjectMember
	result := database.DB.Where("project_id = ? AND user_id = ?", projectID, userID).First(&existing)
	if result.Error == nil {
		// 已存在，更新角色
		return database.DB.Model(&existing).Update("role_id", roleID).Error
	}

	// 创建新成员
	member := &models.ProjectMember{
		ProjectID: projectID,
		UserID:    userID,
		RoleID:    roleID,
	}

	return database.DB.Create(member).Error
}

// RemoveMember 移除项目成员
func (s *Service) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return database.DB.Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&models.ProjectMember{}).Error
}

// GetMemberRole 获取用户在项目中的角色
func (s *Service) GetMemberRole(ctx context.Context, projectID, userID uuid.UUID) (*models.Role, error) {
	var member models.ProjectMember
	if err := database.DB.Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&member).Error; err != nil {
		return nil, err
	}

	var role models.Role
	if err := database.DB.Where("id = ?", member.RoleID).First(&role).Error; err != nil {
		return nil, err
	}

	return &role, nil
}

// HasPermission 检查用户是否拥有指定权限
func (s *Service) HasPermission(ctx context.Context, userID, projectID uuid.UUID, permissionCode string) (bool, error) {
	// 如果是公开项目，访客可能有权限
	var project models.Project
	if err := database.DB.Where("id = ?", projectID).First(&project).Error; err != nil {
		return false, err
	}

	// 检查用户是否是项目成员
	var member models.ProjectMember
	result := database.DB.Where("project_id = ? AND user_id = ?", projectID, userID).First(&member)
	if result.Error != nil {
		// 不是成员，如果是公开项目，检查是否有访客权限
		if project.IsPublic {
			return s.hasPublicPermission(permissionCode), nil
		}
		return false, nil
	}

	// 获取角色的权限
	return s.roleHasPermission(member.RoleID, permissionCode)
}

// hasPublicPermission 检查公开权限
func (s *Service) hasPublicPermission(permissionCode string) bool {
	publicPerms := map[string]bool{
		"project:read": true,
		"image:read":   true,
		"image:pull":   true,
		"tag:read":     true,
	}
	return publicPerms[permissionCode]
}

// roleHasPermission 检查角色是否拥有指定权限
func (s *Service) roleHasPermission(roleID uuid.UUID, permissionCode string) (bool, error) {
	var count int64
	database.DB.Model(&models.RolePermission{}).
		Joins("JOIN registry_permissions ON registry_role_permissions.permission_id = registry_permissions.id").
		Where("registry_role_permissions.role_id = ? AND registry_permissions.code = ?", roleID, permissionCode).
		Count(&count)

	return count > 0, nil
}

// GetUserRoles 获取用户的所有角色
func (s *Service) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error) {
	var members []models.ProjectMember
	if err := database.DB.Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, err
	}

	roleIDs := make([]uuid.UUID, len(members))
	for i, member := range members {
		roleIDs[i] = member.RoleID
	}

	var roles []models.Role
	if err := database.DB.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return nil, err
	}

	return roles, nil
}

// GetRolePermissions 获取角色的所有权限
func (s *Service) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]models.Permission, error) {
	var permissions []models.Permission
	err := database.DB.Model(&models.RolePermission{}).
		Joins("JOIN registry_permissions ON registry_role_permissions.permission_id = registry_permissions.id").
		Where("registry_role_permissions.role_id = ?", roleID).
		Scan(&permissions).Error

	return permissions, err
}

// IsProjectOwner 检查用户是否是项目所有者
func (s *Service) IsProjectOwner(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	var project models.Project
	if err := database.DB.Where("id = ?", projectID).First(&project).Error; err != nil {
		return false, err
	}
	return project.OwnerID == userID, nil
}

// GetProjectMembers 获取项目的所有成员
func (s *Service) GetProjectMembers(ctx context.Context, projectID uuid.UUID) ([]models.ProjectMember, error) {
	var members []models.ProjectMember
	err := database.DB.Where("project_id = ?", projectID).Find(&members).Error
	return members, err
}
