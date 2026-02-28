// Package project_test 项目模块测试
package project_test

import (
	"testing"

	project "github.com/cyp-registry/registry/src/modules/project/service"
	"github.com/stretchr/testify/assert"
)

// TestProjectErrors 测试错误定义
func TestProjectErrors(t *testing.T) {
	assert.Error(t, project.ErrProjectNotFound)
	assert.Error(t, project.ErrProjectExists)
	assert.Error(t, project.ErrQuotaExceeded)
	assert.Error(t, project.ErrInvalidQuota)
}

// TestDefaultQuota 测试默认配额常量
func TestDefaultQuota(t *testing.T) {
	// 默认配额为10GB
	expected := int64(10 * 1024 * 1024 * 1024)
	assert.Equal(t, expected, project.DefaultQuota)
}

// TestProjectStructure 测试项目结构
func TestProjectStructure(t *testing.T) {
	p := &project.Project{
		ID:           "test-id",
		Name:         "test-project",
		Description:  "Test project description",
		OwnerID:      "owner-id",
		IsPublic:     false,
		StorageUsed:  1024,
		StorageQuota: 10 * 1024 * 1024 * 1024,
		ImageCount:   5,
	}

	assert.Equal(t, "test-id", p.ID)
	assert.Equal(t, "test-project", p.Name)
	assert.False(t, p.IsPublic)
	assert.Equal(t, int64(1024), p.StorageUsed)
	assert.Equal(t, 5, p.ImageCount)
}

// TestProjectMemberStructure 测试项目成员结构
func TestProjectMemberStructure(t *testing.T) {
	member := &project.ProjectMember{
		ID:        "member-id",
		ProjectID: "project-id",
		UserID:    "user-id",
		RoleID:    "developer",
	}

	assert.Equal(t, "member-id", member.ID)
	assert.Equal(t, "project-id", member.ProjectID)
	assert.Equal(t, "user-id", member.UserID)
	assert.Equal(t, "developer", member.RoleID)
}
