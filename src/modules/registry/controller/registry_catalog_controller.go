// Package registry_controller 提供 Registry API 中与版本检查、catalog、标签列表相关的 HTTP 处理
package registry_controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	"github.com/cyp-registry/registry/src/pkg/audit"
	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// APIVersionCheck API版本检查
// GET /v2/
func (c *RegistryController) APIVersionCheck(ctx *gin.Context) {
	ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
	ctx.Status(http.StatusOK)
}

// APIVersionCheckUnauthenticated 未认证的API版本检查
func (c *RegistryController) APIVersionCheckUnauthenticated(ctx *gin.Context) {
	ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
	// 需要认证时返回401，并根据全局配置中心生成 Bearer Token 端点地址
	cfg := config.Get()
	hostPort := ctx.Request.Host
	if cfg != nil && cfg.App.Host != "" && cfg.App.Port != 0 {
		hostPort = fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
	}
	ctx.Header("WWW-Authenticate", fmt.Sprintf(`Bearer realm="http://%s/v2/auth"`, hostPort))
	ctx.Status(http.StatusUnauthorized)
}

// Catalog 获取仓库列表
// GET /v2/_catalog
func (c *RegistryController) Catalog(ctx *gin.Context) {
	// 解析分页参数
	n, last := parsePaginationParams(ctx, 100, 1000)

	// 获取仓库列表
	repos, err := c.registry.Catalog(ctx.Request.Context(), n, last)
	if err != nil {
		response.Fail(ctx, 50001, "failed to get catalog")
		return
	}

	// 返回结果
	result := gin.H{
		"repositories": repos,
	}
	if len(repos) == n {
		result["next"] = repos[len(repos)-1]
	}

	response.Success(ctx, result)
}

// ListTags 列出项目的所有标签
// GET /v2/<name>/tags/list
func (c *RegistryController) ListTags(ctx *gin.Context) {
	project := getProjectParam(ctx)

	// 检查读取权限
	hasPermission, errorCode, errorMessage := c.checkProjectPermission(ctx, project, "pull")
	if !hasPermission {
		if errorCode != 0 {
			response.Fail(ctx, errorCode, errorMessage)
		} else {
			response.Fail(ctx, response.CodeInsufficientPermission, "权限不足")
		}
		return
	}

	// 解析分页参数
	n, last := parsePaginationParams(ctx, 100, 0)

	// 获取标签列表
	tags, err := c.registry.ListTags(ctx.Request.Context(), project)
	if err != nil {
		// 记录失败日志
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.RecordError(ctx.Request.Context(), "list_tags", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
			"repository": project,
		})
		response.Fail(ctx, 50001, "failed to list tags")
		return
	}

	// 应用分页
	var paginatedTags []string
	var next string
	if len(tags) > n {
		offset := 0
		if last != "" {
			for i, tag := range tags {
				if tag == last {
					offset = i + 1
					break
				}
			}
		}
		if offset < len(tags) {
			end := offset + n
			if end > len(tags) {
				end = len(tags)
			}
			paginatedTags = tags[offset:end]
			next = paginatedTags[len(paginatedTags)-1]
		}
	} else {
		paginatedTags = tags
	}

	// 为前端提供每个 tag 的摘要、精确大小以及最近一次推送时间/用户，
	// 修复镜像列表中始终显示 "0 B · 未知时间 · 未知用户" 的问题。
	// 为保持向后兼容，仍然保留原有 "tags": []string 结构，并在 data 中新增可选字段：
	// - tag_sizes:       map[tag]size
	// - tag_digests:     map[tag]digest
	// - tag_push_times:  map[tag]RFC3339Time
	// - tag_pushed_by:   map[tag]username
	tagSizes := make(map[string]int64, len(paginatedTags))
	tagDigests := make(map[string]string, len(paginatedTags))
	tagPushTimes := make(map[string]string, len(paginatedTags))
	tagPushedBy := make(map[string]string, len(paginatedTags))
	for _, t := range paginatedTags {
		if t == "" {
			continue
		}
		if tagData, err := c.registry.GetTag(ctx.Request.Context(), project, t); err == nil && tagData != nil {
			// Size 语义：镜像所有层大小之和（字节），在 registry.Manifest.Put/PuManifestRaw 中已保证。
			if tagData.Size > 0 {
				tagSizes[t] = tagData.Size
			}
			if tagData.Digest != "" {
				tagDigests[t] = tagData.Digest
			}
		}

		// 若已注入 WebhookService，则尝试从 webhook_events 中补充最近一次 push 元信息
		if c.whSvc != nil {
			if ts, username, err := c.whSvc.GetLastPushMeta(project, t); err == nil && ts != nil {
				tagPushTimes[t] = ts.UTC().Format(time.RFC3339)
				if username != "" {
					tagPushedBy[t] = username
				}
			}
		}
	}

	result := gin.H{
		"name":           project,
		"tags":           paginatedTags,
		"tag_sizes":      tagSizes,
		"tag_digests":    tagDigests,
		"tag_push_times": tagPushTimes,
		"tag_pushed_by":  tagPushedBy,
	}
	if next != "" {
		result["next"] = next
	}

	// 记录成功日志
	var userID *uuid.UUID
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = &userUUID
		}
	}
	audit.Record(ctx.Request.Context(), "list_tags", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
		"repository": project,
		"tag_count":  len(paginatedTags),
	})

	response.Success(ctx, result)
}
