// Package registry_controller 提供 Registry 中 Manifest 相关 HTTP 处理
package registry_controller

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	project "github.com/cyp-registry/registry/src/modules/project/service"
	"github.com/cyp-registry/registry/src/modules/registry"
	"github.com/cyp-registry/registry/src/pkg/audit"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// GetManifest 获取Manifest
// GET/HEAD /v2/<name>/manifests/<reference>
func (c *RegistryController) GetManifest(ctx *gin.Context) {
	project := getProjectParam(ctx)
	reference := getRepoParam(ctx, "reference")

	// 检查读取权限
	hasPermission, errorCode, errorMessage := c.checkProjectPermission(ctx, project, "pull")
	if !hasPermission {
		// Docker Registry API 规范：权限拒绝返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		// 如果是PAT权限错误，记录错误码（但Docker Registry API仍返回401）
		if errorCode != 0 {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"permission_denied","error_code":%d,"error_message":"%s","permission":"pull","project":"%s"}`, time.Now().Format(time.RFC3339), errorCode, errorMessage, project)
		}
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 获取原始 Manifest 数据（避免重新序列化导致 digest 不匹配）
	manifestData, digest, err := c.registry.GetManifestRaw(ctx.Request.Context(), project, reference)
	if err != nil {
		// 记录失败日志
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.RecordError(ctx.Request.Context(), "get_manifest", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
			"repository": project,
			"reference":  reference,
			"not_found":  err == registry.ErrManifestNotFound,
		})
		if err == registry.ErrManifestNotFound {
			// Docker Registry API 规范：manifest 不存在时返回 404
			ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
			ctx.AbortWithStatus(http.StatusNotFound)
			return
		}
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 解析 manifest 获取 MediaType
	var manifest registry.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	mediaType := manifest.MediaType
	if mediaType == "" {
		mediaType = registry.MediaTypeDocker2Manifest
	}

	// 设置响应头
	ctx.Header("Docker-Content-Digest", digest)
	ctx.Header("Content-Type", mediaType)
	ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")

	// 记录成功日志
	var userID *uuid.UUID
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = &userUUID
		}
	}
	audit.Record(ctx.Request.Context(), "get_manifest", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
		"repository": project,
		"reference":  reference,
		"digest":     digest,
	})

	// 返回原始 Manifest 内容
	ctx.Data(http.StatusOK, mediaType, manifestData)
}

// PutManifest 上传Manifest
// PUT /v2/<name>/manifests/<reference>
func (c *RegistryController) PutManifest(ctx *gin.Context) {
	repoName := getProjectParam(ctx)
	reference := getRepoParam(ctx, "reference")

	// 检查写入权限
	hasPermission, errorCode, errorMessage := c.checkProjectPermission(ctx, repoName, "push")
	if !hasPermission {
		// Docker Registry API 规范：权限拒绝返回 401
		if errorCode != 0 {
			response.Fail(ctx, errorCode, errorMessage)
		} else {
			response.Fail(ctx, response.CodeInsufficientPermission, "权限不足")
		}
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.Header("WWW-Authenticate", fmt.Sprintf(`Bearer realm="http://%s/v2/auth"`, ctx.Request.Host))
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 读取请求体（保留原始字节用于 digest 计算）
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		response.Fail(ctx, 10001, "failed to read request body")
		return
	}

	// 确定内容类型
	contentType := ctx.GetHeader("Content-Type")
	if contentType == "" {
		contentType = registry.MediaTypeDocker2Manifest
	}

	// 验证 Manifest 格式（但不使用解析后的数据）
	var manifest registry.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		response.Fail(ctx, 10001, "invalid manifest format")
		return
	}

	// 上传Manifest（使用原始字节，避免重新序列化导致的 digest 不匹配）
	digest, err := c.registry.PutManifestRaw(ctx.Request.Context(), repoName, reference, body, contentType)
	if err != nil {
		// 不可覆盖的历史版本标签：返回明确的业务错误码
		if err == registry.ErrImmutableTag {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"push_manifest","repository":"%s","reference":"%s","error":"immutable tag"}`, time.Now().Format(time.RFC3339), repoName, reference)
			response.Fail(ctx, 20002, "immutable tag cannot be overwritten; please use a new version tag")
			return
		}
		log.Printf(`{"timestamp":"%s","level":"error","module":"registry","operation":"push_manifest","repository":"%s","reference":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), repoName, reference, err)
		response.Fail(ctx, 50001, "failed to store manifest")
		return
	}

	// 触发项目统计和 Webhook 更新逻辑（与原实现保持一致）
	c.afterManifestPushed(ctx, repoName, reference, digest)

	// 设置响应头
	ctx.Header("Docker-Content-Digest", digest)
	ctx.Status(http.StatusCreated)
}

// DeleteManifest 删除Manifest
// DELETE /v2/<name>/manifests/<reference>
func (c *RegistryController) DeleteManifest(ctx *gin.Context) {
	project := getProjectParam(ctx)
	reference := getRepoParam(ctx, "reference")

	// 检查删除权限
	hasPermission, errorCode, errorMessage := c.checkProjectPermission(ctx, project, "delete")
	if !hasPermission {
		if errorCode != 0 {
			response.Fail(ctx, errorCode, errorMessage)
		} else {
			response.Fail(ctx, response.CodeInsufficientPermission, "权限不足")
		}
		return
	}

	// 删除Manifest
	err := c.registry.DeleteManifest(ctx.Request.Context(), project, reference)
	if err != nil {
		if err == registry.ErrManifestNotFound {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"delete_manifest","repository":"%s","reference":"%s","error":"manifest not found"}`, time.Now().Format(time.RFC3339), project, reference)
			response.Fail(ctx, 20001, "manifest not found")
			return
		}
		log.Printf(`{"timestamp":"%s","level":"error","module":"registry","operation":"delete_manifest","repository":"%s","reference":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), project, reference, err)
		response.Fail(ctx, 50001, "failed to delete manifest")
		return
	}

	// 获取用户信息用于日志
	var userID, username string
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = userUUID.String()
			if usernameVal, uOk := ctx.Get(middleware.ContextKeyUsername); uOk {
				if name, ok := usernameVal.(string); ok {
					username = name
				}
			}
		}
	}

	log.Printf(`{"timestamp":"%s","level":"info","module":"registry","operation":"delete_manifest","repository":"%s","reference":"%s","user_id":"%s","username":"%s","ip":"%s"}`, time.Now().Format(time.RFC3339), project, reference, userID, username, ctx.ClientIP())

	// 删除成功后，尝试同步项目的镜像数量与存储用量统计，并触发删除 Webhook（最佳努力）
	c.afterManifestDeleted(ctx, project, reference)

	ctx.Status(http.StatusAccepted)
}

// afterManifestPushed 在 Manifest 推送成功后更新项目统计并触发 Webhook（从原 controller 中提炼）
func (c *RegistryController) afterManifestPushed(ctx *gin.Context, repoName, reference, digest string) {
	// 确保对应的 Project 在项目系统中可见（用于 Dashboard 展示）
	// 只有在注入了 projectSvc 且当前请求已完成认证时才尝试自动创建/更新项目统计信息
	if c.projectSvc != nil {
		var proj interface{}

		// 获取ownerID（用于自动创建项目）
		ownerID, ownerUUID := c.getOwnerIDFromContext(ctx)

		// 若拿到了 ownerID，则在需要时自动创建项目并同步统计。
		// 若拿不到 ownerID，则尽量仅同步已有项目的统计信息（不自动创建）。
		if ownerID != "" {
			// 从仓库名中提取项目名（第一段）
			// 例：test-project/test-image-1 -> test-project
			projectSlug := repoName
			if idx := strings.Index(repoName, "/"); idx > 0 {
				projectSlug = repoName[:idx]
			}

			// 确保项目存在（如果不存在则自动创建）
			p := c.ensureProjectExists(ctx, projectSlug, ownerID, fmt.Sprintf("Auto created from image push (%s)", reference))
			proj = p

			// 同步项目的镜像数量与存储用量统计（最佳努力，不阻断推送）
			if p, ok := proj.(*project.Project); ok && p != nil {
				if tags, err := c.registry.ListTags(ctx.Request.Context(), repoName); err == nil {
					updates := map[string]interface{}{
						"image_count": len(tags),
					}

					var totalSize int64
					for _, tag := range tags {
						if tagData, err := c.registry.GetTag(ctx.Request.Context(), repoName, tag); err == nil {
							totalSize += tagData.Size
						}
					}
					updates["storage_used"] = totalSize

					_ = c.projectSvc.UpdateProject(ctx.Request.Context(), p.ID, updates)

					// 获取用户信息用于日志和Webhook
					var username string
					if usernameVal, exists := ctx.Get(middleware.ContextKeyUsername); exists {
						if name, ok := usernameVal.(string); ok {
							username = name
						}
					}
					// 如果用户名为空且有 userSvc，则尝试从用户服务补全
					if username == "" && c.userSvc != nil && ownerUUID != uuid.Nil {
						if u, err := c.userSvc.GetUserByID(ctx.Request.Context(), ownerUUID); err == nil && u != nil {
							username = u.Username
						}
					}

					// 尝试获取本次推送镜像的大小（仅当前 tag）
					var imageSize int64
					if tagData, err := c.registry.GetTag(ctx.Request.Context(), repoName, reference); err == nil && tagData != nil {
						imageSize = tagData.Size
					}

					// 记录推送成功日志
					log.Printf(`{"timestamp":"%s","level":"info","module":"registry","operation":"push_manifest","repository":"%s","reference":"%s","digest":"%s","size":%d,"user_id":"%s","username":"%s","ip":"%s","project_id":"%s"}`, time.Now().Format(time.RFC3339), repoName, reference, digest, imageSize, ownerID, username, ctx.ClientIP(), p.ID)

					// 触发 Webhook Push 事件（最佳努力）
					if c.whSvc != nil {
						_ = c.whSvc.PushPushEvent(
							p.ID,
							repoName,
							reference,
							digest,
							imageSize,
							ownerID,
							username,
						)
					}
				}
			}
		} else {
			// 无法识别推送用户时，尽量仅同步已有项目的统计信息，避免 Dashboard 长期显示为 0。
			projectSlug := repoName
			if idx := strings.Index(repoName, "/"); idx > 0 {
				projectSlug = repoName[:idx]
			}
			if p, err := c.projectSvc.GetProjectByName(ctx.Request.Context(), projectSlug); err == nil && p != nil {
				if tags, err := c.registry.ListTags(ctx.Request.Context(), repoName); err == nil {
					updates := map[string]interface{}{
						"image_count": len(tags),
					}

					var totalSize int64
					for _, tag := range tags {
						if tagData, err := c.registry.GetTag(ctx.Request.Context(), repoName, tag); err == nil {
							totalSize += tagData.Size
						}
					}
					updates["storage_used"] = totalSize

					_ = c.projectSvc.UpdateProject(ctx.Request.Context(), p.ID, updates)

					// 记录推送成功日志（即使无法识别用户）
					var imageSize int64
					if tagData, err := c.registry.GetTag(ctx.Request.Context(), repoName, reference); err == nil && tagData != nil {
						imageSize = tagData.Size
					}
					log.Printf(`{"timestamp":"%s","level":"info","module":"registry","operation":"push_manifest","repository":"%s","reference":"%s","digest":"%s","size":%d,"ip":"%s","project_id":"%s","user_id":"","username":""}`, time.Now().Format(time.RFC3339), repoName, reference, digest, imageSize, ctx.ClientIP(), p.ID)
				}
			} else {
				// 项目不存在且无法识别用户，仍然记录推送成功日志（基本信息）
				var imageSize int64
				if tagData, err := c.registry.GetTag(ctx.Request.Context(), repoName, reference); err == nil && tagData != nil {
					imageSize = tagData.Size
				}
				log.Printf(`{"timestamp":"%s","level":"info","module":"registry","operation":"push_manifest","repository":"%s","reference":"%s","digest":"%s","size":%d,"ip":"%s","project_id":"","user_id":"","username":""}`, time.Now().Format(time.RFC3339), repoName, reference, digest, imageSize, ctx.ClientIP())
			}
		}
	} else {
		// 没有projectSvc时，仍然记录推送成功日志（基本信息）
		var imageSize int64
		if tagData, err := c.registry.GetTag(ctx.Request.Context(), repoName, reference); err == nil && tagData != nil {
			imageSize = tagData.Size
		}
		// 尝试获取用户信息
		var userID, username string
		ownerID, ownerUUID := c.getOwnerIDFromContext(ctx)
		if ownerID != "" {
			userID = ownerID
			if usernameVal, exists := ctx.Get(middleware.ContextKeyUsername); exists {
				if name, ok := usernameVal.(string); ok {
					username = name
				}
			}
			if username == "" && c.userSvc != nil && ownerUUID != uuid.Nil {
				if u, err := c.userSvc.GetUserByID(ctx.Request.Context(), ownerUUID); err == nil && u != nil {
					username = u.Username
				}
			}
		}
		log.Printf(`{"timestamp":"%s","level":"info","module":"registry","operation":"push_manifest","repository":"%s","reference":"%s","digest":"%s","size":%d,"user_id":"%s","username":"%s","ip":"%s","project_id":""}`, time.Now().Format(time.RFC3339), repoName, reference, digest, imageSize, userID, username, ctx.ClientIP())
	}
}

// afterManifestDeleted 在 Manifest 删除成功后更新项目统计并触发删除 Webhook（从原 controller 中提炼）
func (c *RegistryController) afterManifestDeleted(ctx *gin.Context, projectName, reference string) {
	if c.projectSvc != nil {
		projectSlug := projectName
		if idx := strings.Index(projectName, "/"); idx > 0 {
			projectSlug = projectName[:idx]
		}
		if p, pErr := c.projectSvc.GetProjectByName(ctx.Request.Context(), projectSlug); pErr == nil && p != nil {
			// 重新统计当前仓库下的所有 tag 及大小
			if tags, tErr := c.registry.ListTags(ctx.Request.Context(), projectName); tErr == nil {
				updates := map[string]interface{}{
					"image_count": len(tags),
				}

				var totalSize int64
				for _, tag := range tags {
					if tagData, gErr := c.registry.GetTag(ctx.Request.Context(), projectName, tag); gErr == nil && tagData != nil {
						totalSize += tagData.Size
					}
				}
				updates["storage_used"] = totalSize

				_ = c.projectSvc.UpdateProject(ctx.Request.Context(), p.ID, updates)

				// 触发镜像删除 Webhook（忽略错误，记录由 WebhookService 负责）
				if c.whSvc != nil {
					// 删除事件中同样需要用户信息
					var userID, username string

					if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
						if userUUID, ok := userIDVal.(uuid.UUID); ok {
							userID = userUUID.String()
							if usernameVal, uOk := ctx.Get(middleware.ContextKeyUsername); uOk {
								if name, ok := usernameVal.(string); ok {
									username = name
								}
							}
						}
					}

					// 若仍然缺少用户名且有 userSvc，可以尝试补全
					if username == "" && c.userSvc != nil && userID != "" {
						if uUUID, err := uuid.Parse(userID); err == nil {
							if u, err := c.userSvc.GetUserByID(ctx.Request.Context(), uUUID); err == nil && u != nil {
								username = u.Username
							}
						}
					}

					_ = c.whSvc.PushDeleteEvent(
						p.ID,
						projectName,
						reference,
						"", // digest 在删除接口中为可选，这里暂不强依赖
						userID,
						username,
					)
				}
			}
		}
	}
}
