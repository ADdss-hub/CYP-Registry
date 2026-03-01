// Package models 定义镜像导入任务的数据模型
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ImportTaskStatus 导入任务状态
type ImportTaskStatus string

const (
	// TaskStatusPending 等待执行
	TaskStatusPending ImportTaskStatus = "pending"
	// TaskStatusRunning 正在执行
	TaskStatusRunning ImportTaskStatus = "running"
	// TaskStatusSuccess 执行成功
	TaskStatusSuccess ImportTaskStatus = "success"
	// TaskStatusFailed 执行失败
	TaskStatusFailed ImportTaskStatus = "failed"
)

// ImportTask 镜像导入任务模型
// 表名: image_import_tasks
type ImportTask struct {
	ID string `gorm:"type:varchar(36);primaryKey" json:"id"`

	ProjectID string `gorm:"type:varchar(36);index;not null;comment:项目ID" json:"project_id"`
	UserID    string `gorm:"type:varchar(36);index;comment:用户ID" json:"user_id"`

	SourceURL        string `gorm:"type:varchar(512);not null;comment:原始镜像URL" json:"source_url"`
	NormalizedSource string `gorm:"type:varchar(512);not null;comment:规范化后的镜像URL" json:"normalized_source"`

	TargetImage string `gorm:"type:varchar(256);not null;comment:目标镜像名称" json:"target_image"`
	TargetTag   string `gorm:"type:varchar(128);not null;comment:目标镜像标签" json:"target_tag"`

	Status   string `gorm:"type:varchar(32);index;not null;comment:任务状态" json:"status"`
	Progress int    `gorm:"type:int;not null;default:0;comment:进度百分比" json:"progress"`
	Message  string `gorm:"type:text;comment:状态消息" json:"message"`
	Error    string `gorm:"type:text;comment:错误信息" json:"error"`

	AuthUsername string `gorm:"type:varchar(128);comment:源仓库用户名(可选)" json:"-"`
	AuthPassword string `gorm:"type:varchar(256);comment:源仓库密码或Token(可选)" json:"-"`

	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	CompletedAt *time.Time     `json:"completed_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName 表名
func (ImportTask) TableName() string {
	return "image_import_tasks"
}

// NewImportTask 创建新的导入任务实体
func NewImportTask(projectID, userID, sourceURL, normalizedSource, targetImage, targetTag string) *ImportTask {
	return &ImportTask{
		ID:              uuid.New().String(),
		ProjectID:       projectID,
		UserID:          userID,
		SourceURL:       sourceURL,
		NormalizedSource: normalizedSource,
		TargetImage:     targetImage,
		TargetTag:       targetTag,
		Status:          string(TaskStatusPending),
		Progress:        0,
		Message:         "任务已创建，等待执行",
	}
}
