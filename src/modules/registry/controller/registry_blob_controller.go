// Package registry_controller 提供 Registry 中 Blob 相关 HTTP 处理
package registry_controller

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	"github.com/cyp-registry/registry/src/modules/registry"
	"github.com/cyp-registry/registry/src/pkg/audit"
	"github.com/cyp-registry/registry/src/pkg/response"
)

// CheckBlob 检查Blob是否存在
// HEAD /v2/<name>/blobs/<digest>
func (c *RegistryController) CheckBlob(ctx *gin.Context) {
	project := getProjectParam(ctx)
	digest := getRepoParam(ctx, "digest")

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

	exists, err := c.registry.CheckBlob(ctx.Request.Context(), project, digest)
	if err != nil {
		// 记录失败日志
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.RecordError(ctx.Request.Context(), "check_blob", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
			"repository": project,
			"digest":     digest,
		})
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if !exists {
		// Docker Registry API 规范：blob 不存在时返回 404
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	// 获取Blob大小
	size, err := c.registry.GetBlobSize(ctx.Request.Context(), project, digest)
	if err != nil {
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ctx.Header("Content-Length", strconv.FormatInt(size, 10))
	ctx.Header("Docker-Content-Digest", digest)
	ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")

	// 记录成功日志
	var userID *uuid.UUID
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = &userUUID
		}
	}
	audit.Record(ctx.Request.Context(), "check_blob", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
		"repository": project,
		"digest":     digest,
		"size":       size,
	})

	ctx.Status(http.StatusOK)
}

// GetBlob 获取Blob
// GET /v2/<name>/blobs/<digest>
func (c *RegistryController) GetBlob(ctx *gin.Context) {
	project := getProjectParam(ctx)
	digest := getRepoParam(ctx, "digest")

	// 检查读取权限
	hasPermission, errorCode, errorMessage := c.checkProjectPermission(ctx, project, "pull")
	if !hasPermission {
		// Registry API 规范：无权限时返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		// 如果是PAT权限错误，记录错误码（但Docker Registry API仍返回401）
		if errorCode != 0 {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"permission_denied","error_code":%d,"error_message":"%s","permission":"pull","project":"%s"}`, time.Now().Format(time.RFC3339), errorCode, errorMessage, project)
		}
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 获取Blob
	reader, size, err := c.registry.GetBlob(ctx.Request.Context(), project, digest)
	if err != nil {
		// 记录失败日志
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.RecordError(ctx.Request.Context(), "get_blob", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
			"repository": project,
			"digest":     digest,
			"not_found":  err == registry.ErrBlobNotFound,
		})
		// Blob 不存在：返回 404；其他错误：500
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		if err == registry.ErrBlobNotFound {
			ctx.AbortWithStatus(http.StatusNotFound)
		} else {
			ctx.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	// 设置响应头
	ctx.Header("Content-Length", strconv.FormatInt(size, 10))
	ctx.Header("Docker-Content-Digest", digest)
	ctx.Header("Accept-Ranges", "bytes")
	ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")

	// 记录成功日志
	var userID *uuid.UUID
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = &userUUID
		}
	}
	audit.Record(ctx.Request.Context(), "get_blob", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
		"repository": project,
		"digest":     digest,
		"size":       size,
	})

	// 返回数据
	ctx.DataFromReader(http.StatusOK, size, "", reader, nil)
}

// InitiateBlobUpload 初始化Blob上传
// POST /v2/<name>/blobs/uploads/
// 支持三种模式：
// 1. 跨仓库挂载：mount=<digest>&from=<source>
// 2. Monolithic upload：digest=<digest> + 请求体
// 3. 分片上传初始化：无特殊参数
func (c *RegistryController) InitiateBlobUpload(ctx *gin.Context) {
	project := getProjectParam(ctx)

	// 检查写入权限
	hasPermission, errorCode, errorMessage := c.checkProjectPermission(ctx, project, "push")
	if !hasPermission {
		// Docker Registry API 规范：权限拒绝返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.Header("WWW-Authenticate", fmt.Sprintf(`Bearer realm="http://%s/v2/auth"`, ctx.Request.Host))
		// 如果是PAT权限错误，记录错误码（但Docker Registry API仍返回401）
		if errorCode != 0 {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"permission_denied","error_code":%d,"error_message":"%s","permission":"push","project":"%s"}`, time.Now().Format(time.RFC3339), errorCode, errorMessage, project)
		}
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 确保项目存在（推送时必须创建项目，以便界面显示）
	if c.projectSvc != nil {
		ownerID, _ := c.getOwnerIDFromContext(ctx)
		if ownerID != "" {
			// 从仓库名中提取项目名（第一段）
			projectSlug := project
			if idx := strings.Index(project, "/"); idx > 0 {
				projectSlug = project[:idx]
			}
			// 确保项目存在（如果不存在则自动创建）
			_ = c.ensureProjectExists(ctx, projectSlug, ownerID, "Auto created from image push")
		}
	}

	// 检查是否指定mount参数（跨仓库挂载）
	mount := ctx.Query("mount")
	from := ctx.Query("from")

	if mount != "" && from != "" {
		// 跨仓库挂载Blob
		err := c.registry.MountBlob(ctx.Request.Context(), project, from, mount)
		if err != nil {
			// 兼容 Docker 客户端：当 mount 失败（源 blob 不存在）时，必须回退到“普通上传初始化”，
			// 而不是返回自定义 JSON（会导致客户端拿不到 upload Location，进而出现 `https:?digest=...` 这类无 Host URL）。
			// 参考：OCI Distribution / Docker Registry 挂载失败应返回 202 并提供上传地址（或直接走普通上传流程）。
			if err != registry.ErrBlobNotFound {
				ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
				ctx.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			// ErrBlobNotFound: fallthrough -> continue to normal upload init below
		} else {
			// 获取挂载的Blob信息
			size, _ := c.registry.GetBlobSize(ctx.Request.Context(), from, mount)
			digest := mount

			// NOTE: Docker Registry API 允许 Location 使用相对路径。
			// 在部分 Docker/BuildKit 场景下，绝对 URL 可能被错误解析为 `https:?digest=...`（无 Host），导致 push 失败。
			// 因此这里统一返回相对路径，交由客户端按当前 registry origin 解析。
			locationPath := fmt.Sprintf("/v2/%s/blobs/%s", project, digest)
			ctx.Header("Location", locationPath)
			ctx.Header("Docker-Content-Digest", digest)
			ctx.Header("Content-Length", strconv.FormatInt(size, 10))
			ctx.Status(http.StatusCreated)
			return
		}
	}

	// 检查是否是 monolithic upload（单次上传）
	// Docker 客户端可能在 POST 请求中直接包含 digest 参数和请求体
	digest := ctx.Query("digest")
	if digest != "" && ctx.Request.ContentLength > 0 {
		// Monolithic upload：直接完成上传
		body, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			response.Fail(ctx, 10001, "failed to read request body")
			return
		}

		// 初始化上传
		info, err := c.registry.InitiateBlobUpload(ctx.Request.Context(), project)
		if err != nil {
			// 记录失败日志
			var userID *uuid.UUID
			if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
				if userUUID, ok := userIDVal.(uuid.UUID); ok {
					userID = &userUUID
				}
			}
			audit.RecordError(ctx.Request.Context(), "initiate_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
				"repository": project,
				"digest":     digest,
				"mode":       "monolithic",
			})
			response.Fail(ctx, 50001, "failed to initiate upload")
			return
		}

		// 上传数据
		_, err = c.registry.UploadBlobChunk(ctx.Request.Context(), project, info.UUID, 0, bytes.NewReader(body), int64(len(body)))
		if err != nil {
			// 记录失败日志
			var userID *uuid.UUID
			if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
				if userUUID, ok := userIDVal.(uuid.UUID); ok {
					userID = &userUUID
				}
			}
			audit.RecordError(ctx.Request.Context(), "upload_blob_chunk", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
				"repository": project,
				"upload_id":  info.UUID,
				"mode":       "monolithic",
			})
			response.Fail(ctx, 50001, "failed to upload blob")
			return
		}

		// 完成上传
		err = c.registry.CompleteBlobUpload(ctx.Request.Context(), project, info.UUID, digest, int64(len(body)))
		if err != nil {
			// 记录失败日志
			var userID *uuid.UUID
			if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
				if userUUID, ok := userIDVal.(uuid.UUID); ok {
					userID = &userUUID
				}
			}
			audit.RecordError(ctx.Request.Context(), "complete_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
				"repository": project,
				"upload_id":  info.UUID,
				"digest":     digest,
				"mode":       "monolithic",
			})
			response.Fail(ctx, 50001, "failed to complete upload: "+err.Error())
			return
		}

		// 记录成功日志（monolithic模式）
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.Record(ctx.Request.Context(), "complete_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
			"repository": project,
			"upload_id":  info.UUID,
			"digest":     digest,
			"size":       len(body),
			"mode":       "monolithic",
		})

		// 返回结果
		locationPath := fmt.Sprintf("/v2/%s/blobs/%s", project, digest)
		ctx.Header("Location", locationPath)
		ctx.Header("Docker-Content-Digest", digest)
		ctx.Status(http.StatusCreated)
		return
	}

	// 初始化新上传（分片上传模式）
	info, err := c.registry.InitiateBlobUpload(ctx.Request.Context(), project)
	if err != nil {
		// 记录失败日志
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.RecordError(ctx.Request.Context(), "initiate_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
			"repository": project,
			"mount":      mount,
			"from":       from,
		})
		response.Fail(ctx, 50001, "failed to initiate upload")
		return
	}

	// 记录成功日志
	var userID *uuid.UUID
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = &userUUID
		}
	}
	audit.Record(ctx.Request.Context(), "initiate_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
		"repository": project,
		"upload_id":  info.UUID,
		"mount":      mount,
		"from":       from,
	})

	// 返回上传URL（相对路径）
	locationPath := fmt.Sprintf("/v2/%s/blobs/uploads/%s", project, info.UUID)
	ctx.Header("Location", locationPath)
	ctx.Header("Docker-Upload-UUID", info.UUID)
	ctx.Header("Range", "0-0")
	ctx.Status(http.StatusAccepted)
}

// UploadBlobChunk 上传Blob分片
// PATCH /v2/<name>/blobs/uploads/<uuid>
func (c *RegistryController) UploadBlobChunk(ctx *gin.Context) {
	project := getProjectParam(ctx)
	uploadID := getRepoParam(ctx, "uuid")

	// 检查写入权限
	hasPermission, errorCode, errorMessage := c.checkProjectPermission(ctx, project, "push")
	if !hasPermission {
		// Docker Registry API 规范：权限拒绝返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.Header("WWW-Authenticate", fmt.Sprintf(`Bearer realm="http://%s/v2/auth"`, ctx.Request.Host))
		// 如果是PAT权限错误，记录错误码（但Docker Registry API仍返回401）
		if errorCode != 0 {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"permission_denied","error_code":%d,"error_message":"%s","permission":"push","project":"%s"}`, time.Now().Format(time.RFC3339), errorCode, errorMessage, project)
		}
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 解析请求头
	contentRange := ctx.GetHeader("Content-Range")
	var offset int64 = 0
	if contentRange != "" {
		start, _, total, err := registry.ParseContentRange(contentRange)
		if err == nil {
			offset = start
			ctx.Request.ContentLength = total - start
		}
	}

	// 读取请求体
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		response.Fail(ctx, 10001, "failed to read request body")
		return
	}

	size := int64(len(body))

	// 上传分片
	newOffset, err := c.registry.UploadBlobChunk(ctx.Request.Context(), project, uploadID, offset, bytes.NewReader(body), size)
	if err != nil {
		// 记录失败日志
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.RecordError(ctx.Request.Context(), "upload_blob_chunk", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
			"repository": project,
			"upload_id":  uploadID,
			"offset":     offset,
			"size":       size,
		})
		response.Fail(ctx, 50001, "failed to upload chunk")
		return
	}

	// 记录成功日志（分片上传操作很频繁，但为了完整性仍然记录）
	var userID *uuid.UUID
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = &userUUID
		}
	}
	audit.Record(ctx.Request.Context(), "upload_blob_chunk", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
		"repository": project,
		"upload_id":  uploadID,
		"offset":     offset,
		"new_offset": newOffset,
		"size":       size,
	})

	// 返回结果（相对路径）
	locationPath := fmt.Sprintf("/v2/%s/blobs/uploads/%s", project, uploadID)
	ctx.Header("Location", locationPath)
	ctx.Header("Docker-Upload-UUID", uploadID)
	ctx.Header("Range", fmt.Sprintf("0-%d", newOffset-1))
	ctx.Status(http.StatusAccepted)
}

// CompleteBlobUpload 完成Blob上传
// PUT /v2/<name>/blobs/uploads/<uuid>?digest=<digest>
func (c *RegistryController) CompleteBlobUpload(ctx *gin.Context) {
	project := getProjectParam(ctx)
	uploadID := getRepoParam(ctx, "uuid")
	digest := ctx.Query("digest")

	// 检查写入权限
	hasPermission, errorCode, errorMessage := c.checkProjectPermission(ctx, project, "push")
	if !hasPermission {
		// Docker Registry API 规范：权限拒绝返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.Header("WWW-Authenticate", fmt.Sprintf(`Bearer realm="http://%s/v2/auth"`, ctx.Request.Host))
		// 如果是PAT权限错误，记录错误码（但Docker Registry API仍返回401）
		if errorCode != 0 {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"permission_denied","error_code":%d,"error_message":"%s","permission":"push","project":"%s"}`, time.Now().Format(time.RFC3339), errorCode, errorMessage, project)
		}
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	if digest == "" {
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// 读取请求体（如果有）
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Docker/BuildKit 可能使用“单次 PUT 完成上传”的模式：
	// POST /blobs/uploads/ -> PUT /blobs/uploads/<uuid>?digest=... (带请求体)
	// 此时必须先把本次 PUT 的 body 追加到 upload 临时对象中，否则 CompleteBlobUpload 会对空文件计算 digest 导致失败，
	// 且如果错误被包装成 200，会造成客户端误判“已推送”但实际 blob 缺失（进而 Trivy 拉取 404）。
	if len(body) > 0 {
		// 尽量从当前上传状态获取 offset，实现幂等追加
		var offset int64 = 0
		if info, stErr := c.registry.GetBlobUploadStatus(ctx.Request.Context(), project, uploadID); stErr == nil && info != nil {
			offset = info.Size
		}
		if _, upErr := c.registry.UploadBlobChunk(
			ctx.Request.Context(),
			project,
			uploadID,
			offset,
			bytes.NewReader(body),
			int64(len(body)),
		); upErr != nil {
			ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	}

	// 完成上传
	// size 传 0：由存储层按实际内容校验 digest，并避免客户端在不同上传模式下导致 size mismatch
	err = c.registry.CompleteBlobUpload(ctx.Request.Context(), project, uploadID, digest, 0)
	if err != nil {
		// 记录失败日志
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.RecordError(ctx.Request.Context(), "complete_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
			"repository": project,
			"upload_id":  uploadID,
			"digest":     digest,
		})
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 记录成功日志
	var userID *uuid.UUID
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = &userUUID
		}
	}
	audit.Record(ctx.Request.Context(), "complete_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
		"repository": project,
		"upload_id":  uploadID,
		"digest":     digest,
	})

	// 返回结果（相对路径）
	locationPath := fmt.Sprintf("/v2/%s/blobs/%s", project, digest)
	ctx.Header("Location", locationPath)
	ctx.Header("Docker-Content-Digest", digest)
	ctx.Status(http.StatusCreated)
}
