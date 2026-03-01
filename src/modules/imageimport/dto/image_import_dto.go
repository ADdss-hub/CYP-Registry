// Package dto 定义镜像导入相关的请求与响应结构体
package dto

import (
	"time"

	"github.com/cyp-registry/registry/src/modules/imageimport/models"
)

// ImportImageRequest 导入镜像请求体
// 对应前端 web/src/services/imageImport.ts 中的 ImportImageRequest 接口
type ImportImageRequest struct {
	SourceURL   string       `json:"source_url"`             // 源镜像URL，例如 nginx:latest / ghcr.io/owner/repo:tag
	TargetImage string       `json:"target_image,omitempty"` // 目标镜像名称（可选）
	TargetTag   string       `json:"target_tag,omitempty"`   // 目标标签（可选）
	Auth        *AuthRequest `json:"auth,omitempty"`         // 认证信息（私有仓库时可选）
}

// AuthRequest 源仓库认证信息
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ImportTaskResponse 前端使用的任务响应结构
// 对应 web/src/services/imageImport.ts 中的 ImportTask 接口
type ImportTaskResponse struct {
	TaskID      string     `json:"task_id"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress"`
	Message     string     `json:"message"`
	SourceURL   string     `json:"source_url"`
	TargetImage string     `json:"target_image"`
	TargetTag   string     `json:"target_tag"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ImportTaskListResponse 任务列表响应结构
// 对应 web/src/services/imageImport.ts 中的 ImportTaskListResponse 接口
type ImportTaskListResponse struct {
	Tasks     []ImportTaskResponse `json:"tasks"`
	Total     int64                `json:"total"`
	Page      int                  `json:"page"`
	PageSize  int                  `json:"page_size"`
	TotalPage int                  `json:"total_page"`
}

// FromModel 将模型转换为响应结构
func FromModel(task *models.ImportTask) ImportTaskResponse {
	if task == nil {
		return ImportTaskResponse{}
	}
	return ImportTaskResponse{
		TaskID:      task.ID,
		Status:      task.Status,
		Progress:    task.Progress,
		Message:     task.Message,
		SourceURL:   task.SourceURL,
		TargetImage: task.TargetImage,
		TargetTag:   task.TargetTag,
		Error:       task.Error,
		CreatedAt:   task.CreatedAt,
		CompletedAt: task.CompletedAt,
	}
}

// FromModelSlice 批量转换
func FromModelSlice(tasks []models.ImportTask) []ImportTaskResponse {
	result := make([]ImportTaskResponse, len(tasks))
	for i := range tasks {
		result[i] = FromModel(&tasks[i])
	}
	return result
}
