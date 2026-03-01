// Package service 实现镜像导入的核心业务逻辑
package service

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	imageimportdto "github.com/cyp-registry/registry/src/modules/imageimport/dto"
	"github.com/cyp-registry/registry/src/modules/imageimport/models"
	"github.com/cyp-registry/registry/src/pkg/database"
)

// Service 镜像导入服务
type Service struct {
	db                *gorm.DB
	localRegistryHost string
}

// NewService 创建镜像导入服务
// localRegistryHost 示例：localhost:8080
func NewService(localRegistryHost string) *Service {
	return &Service{
		db:                database.GetDB(),
		localRegistryHost: localRegistryHost,
	}
}

// ImportImageFromURL 创建导入任务并异步执行
// projectID: 项目ID（UUID字符串）
// projectName: 项目名称，用于构建本地仓库路径，例如 my-project
// userID: 当前用户ID，可为空
func (s *Service) ImportImageFromURL(
	ctx context.Context,
	projectID string,
	projectName string,
	userID *uuid.UUID,
	req *imageimportdto.ImportImageRequest,
) (*models.ImportTask, error) {
	if req == nil || strings.TrimSpace(req.SourceURL) == "" {
		return nil, fmt.Errorf("source_url 不能为空")
	}

	source := strings.TrimSpace(req.SourceURL)
	normalized := normalizeImageURL(source)

	targetImage := strings.TrimSpace(req.TargetImage)
	if targetImage == "" {
		targetImage = inferTargetImageFromSource(normalized)
	}
	if targetImage == "" {
		return nil, fmt.Errorf("无法从 source_url 推断目标镜像名称，请显式指定 target_image")
	}

	targetTag := strings.TrimSpace(req.TargetTag)
	if targetTag == "" {
		targetTag = inferTagFromSource(normalized)
	}
	if targetTag == "" {
		targetTag = "latest"
	}

	uid := ""
	if userID != nil && *userID != uuid.Nil {
		uid = userID.String()
	}

	task := models.NewImportTask(projectID, uid, source, normalized, targetImage, targetTag)
	if req.Auth != nil {
		task.AuthUsername = strings.TrimSpace(req.Auth.Username)
		task.AuthPassword = req.Auth.Password
	}

	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建导入任务失败: %w", err)
	}

	// 异步执行导入，不阻塞 API 响应
	go s.executeImport(context.Background(), task.ID, projectName)

	return task, nil
}

// GetTask 获取单个任务
func (s *Service) GetTask(ctx context.Context, projectID, taskID string) (*models.ImportTask, error) {
	var task models.ImportTask
	if err := s.db.WithContext(ctx).
		Where("id = ? AND project_id = ?", taskID, projectID).
		First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在")
		}
		return nil, err
	}
	return &task, nil
}

// ListTasks 列出项目的导入任务
func (s *Service) ListTasks(ctx context.Context, projectID string, page, pageSize int) ([]models.ImportTask, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	if err := s.db.WithContext(ctx).
		Model(&models.ImportTask{}).
		Where("project_id = ?", projectID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tasks []models.ImportTask
	if err := s.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// executeImport 实际执行导入任务
func (s *Service) executeImport(ctx context.Context, taskID string, projectName string) {
	var task models.ImportTask
	if err := s.db.WithContext(ctx).First(&task, "id = ?", taskID).Error; err != nil {
		return
	}

	updateStatus := func(status models.ImportTaskStatus, progress int, msg string) {
		now := time.Now()
		updates := map[string]interface{}{
			"status":     string(status),
			"progress":   progress,
			"message":    msg,
			"updated_at": now,
		}
		if status == models.TaskStatusSuccess || status == models.TaskStatusFailed {
			updates["completed_at"] = now
		}
		_ = s.db.WithContext(ctx).Model(&models.ImportTask{}).
			Where("id = ?", task.ID).
			Updates(updates).Error
	}

	fail := func(err error, msg string) {
		now := time.Now()
		_ = s.db.WithContext(ctx).Model(&models.ImportTask{}).
			Where("id = ?", task.ID).
			Updates(map[string]interface{}{
				"status":       string(models.TaskStatusFailed),
				"progress":     100,
				"message":      msg,
				"error":        err.Error(),
				"completed_at": now,
				"updated_at":   now,
			}).Error
	}

	updateStatus(models.TaskStatusRunning, 10, "开始导入镜像...")

	// 解析 registry host
	registryHost := extractRegistryHost(task.NormalizedSource)

	// 如提供认证信息，先登录源仓库
	if task.AuthUsername != "" && task.AuthPassword != "" && registryHost != "" {
		updateStatus(models.TaskStatusRunning, 20, "正在登录源镜像仓库...")
		loginCmd := exec.CommandContext(ctx, "docker", "login", registryHost, "-u", task.AuthUsername, "--password-stdin")
		loginCmd.Stdin = strings.NewReader(task.AuthPassword)
		if output, err := loginCmd.CombinedOutput(); err != nil {
			fail(fmt.Errorf("docker login 失败: %v, output=%s", err, strings.TrimSpace(string(output))), "登录源镜像仓库失败")
			return
		}
	}

	// 拉取镜像
	updateStatus(models.TaskStatusRunning, 30, "正在拉取源镜像...")
	if output, err := exec.CommandContext(ctx, "docker", "pull", task.NormalizedSource).CombinedOutput(); err != nil {
		fail(fmt.Errorf("docker pull 失败: %v, output=%s", err, strings.TrimSpace(string(output))), "拉取源镜像失败")
		return
	}

	// 重新标记到本地仓库
	updateStatus(models.TaskStatusRunning, 60, "正在重新标记镜像...")
	localRef := fmt.Sprintf("%s/%s/%s:%s", s.localRegistryHost, projectName, task.TargetImage, task.TargetTag)
	if output, err := exec.CommandContext(ctx, "docker", "tag", task.NormalizedSource, localRef).CombinedOutput(); err != nil {
		fail(fmt.Errorf("docker tag 失败: %v, output=%s", err, strings.TrimSpace(string(output))), "重新标记镜像失败")
		return
	}

	// 推送到本地仓库
	updateStatus(models.TaskStatusRunning, 80, "正在推送镜像到本地仓库...")
	if output, err := exec.CommandContext(ctx, "docker", "push", localRef).CombinedOutput(); err != nil {
		fail(fmt.Errorf("docker push 失败: %v, output=%s", err, strings.TrimSpace(string(output))), "推送镜像到本地仓库失败")
		return
	}

	// 简单清理本地临时镜像（最佳努力）
	_, _ = exec.CommandContext(ctx, "docker", "rmi", localRef).CombinedOutput()

	updateStatus(models.TaskStatusSuccess, 100, "镜像导入完成")
}

// normalizeImageURL 根据文档规则规范化镜像URL
// 例如：
//
//	nginx:latest           -> docker.io/library/nginx:latest
//	user/nginx:1.20        -> docker.io/user/nginx:1.20
//	ghcr.io/owner/repo:tag -> ghcr.io/owner/repo:tag
func normalizeImageURL(source string) string {
	s := strings.TrimSpace(source)
	if s == "" {
		return s
	}

	// 已包含显式 registry 前缀（包含 . 或 :），直接返回
	if idx := strings.IndexRune(s, '/'); idx > 0 {
		hostPart := s[:idx]
		if strings.Contains(hostPart, ".") || strings.Contains(hostPart, ":") {
			return s
		}
	}

	// 不包含显式 registry：视为 docker.io
	repo := s
	tag := "latest"
	if idx := strings.LastIndex(repo, ":"); idx > -1 && idx > strings.LastIndex(repo, "/") {
		tag = repo[idx+1:]
		repo = repo[:idx]
	}

	// 若只有一个段，自动补全为 library/<name>
	if !strings.Contains(repo, "/") {
		repo = "library/" + repo
	}

	return "docker.io/" + repo + ":" + tag
}

// inferTargetImageFromSource 从规范化后的URL推断目标镜像名
func inferTargetImageFromSource(normalized string) string {
	s := normalized
	// 去掉 registry host
	if idx := strings.IndexRune(s, '/'); idx > 0 {
		s = s[idx+1:]
	}
	// 去掉 tag
	if idx := strings.LastIndex(s, ":"); idx > -1 && idx > strings.LastIndex(s, "/") {
		s = s[:idx]
	}
	parts := strings.Split(s, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// inferTagFromSource 从规范化后的URL推断 tag
func inferTagFromSource(normalized string) string {
	if idx := strings.LastIndex(normalized, ":"); idx > -1 && idx > strings.LastIndex(normalized, "/") {
		return normalized[idx+1:]
	}
	return ""
}

// extractRegistryHost 从规范化URL中提取 registry host
// 例如 docker.io/library/nginx:latest -> docker.io
func extractRegistryHost(normalized string) string {
	if normalized == "" {
		return ""
	}
	parts := strings.SplitN(normalized, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}
