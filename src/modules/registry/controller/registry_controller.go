// Package registry_controller Registry API控制器
// 实现Docker Registry HTTP API V2的RESTful接口
// cspell:ignore ORAS
package registry_controller

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cyp-registry/registry/src/middleware"
	project "github.com/cyp-registry/registry/src/modules/project/service"
	"github.com/cyp-registry/registry/src/modules/rbac"
	"github.com/cyp-registry/registry/src/modules/registry"
	user_service "github.com/cyp-registry/registry/src/modules/user/service"
	webhook_service "github.com/cyp-registry/registry/src/modules/webhook/service"
	"github.com/cyp-registry/registry/src/pkg/audit"
	"github.com/cyp-registry/registry/src/pkg/response"
	"github.com/gin-gonic/gin"
)

// RegistryController Registry API控制器
type RegistryController struct {
	registry       *registry.Registry
	rbacSvc        *rbac.Service
	authMiddleware *middleware.AuthMiddleware
	projectSvc     project.Service
	userSvc        *user_service.Service
	whSvc          *webhook_service.WebhookService
}

// TokenEndpoint Docker Registry Bearer Token 端点（最小可用实现）
// - 支持 Basic Auth（username/password）换取 JWT AccessToken
// - 支持 Basic Auth（username/PAT）使用个人令牌认证
// - 返回 registry 兼容字段：token/access_token/expires_in/issued_at
func (c *RegistryController) TokenEndpoint(userSvc *user_service.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		username, password, ok := ctx.Request.BasicAuth()
		if !ok {
			// Docker Registry Token API 规范：缺少认证返回 401
			ctx.Header("WWW-Authenticate", `Basic realm="registry"`)
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// 检查password是否是PAT（以pat_v1_开头）
		var accessToken string
		var expiresIn int64

		if strings.HasPrefix(password, "pat_v1_") {
			// 使用PAT认证（自动工具场景）
			patModel, err := userSvc.ValidatePAT(ctx, password)
			if err != nil || patModel == nil {
				ctx.Header("WWW-Authenticate", `Basic realm="registry"`)
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			// 获取PAT关联用户（用于生成 registry 侧可用的 JWT）
			user, err := userSvc.GetUserByID(ctx, patModel.UserID)
			if err != nil {
				ctx.Header("WWW-Authenticate", `Basic realm="registry"`)
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			// 说明：Docker CLI / 自动化工具通常必须提供一个 username，
			// 这里不再强制 username 与真实用户名一致，实现"只用令牌，不用账号密码"。
			_ = username

			// 为了兼容Docker Registry，生成一个短期JWT token
			// 使用JWT服务生成token对
			jwtSvc := userSvc.GetJWTService()
			if jwtSvc == nil {
				ctx.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			tokenPair, err := jwtSvc.GenerateTokenPair(patModel.UserID, user.Username)
			if err != nil {
				ctx.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			accessToken = tokenPair.AccessToken
			expiresIn = int64(time.Until(tokenPair.ExpiresAt).Seconds())
		} else {
			// 使用用户名密码认证（Docker CLI登录场景）
			tokens, _, err := userSvc.Login(ctx, username, password, ctx.ClientIP(), ctx.GetHeader("User-Agent"))
			if err != nil || tokens == nil {
				ctx.Header("WWW-Authenticate", `Basic realm="registry"`)
				ctx.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			accessToken = tokens.AccessToken
			expiresIn = int64(time.Until(tokens.ExpiresAt).Seconds())
		}

		if expiresIn < 0 {
			expiresIn = 0
		}

		// Docker Registry Token 响应格式（直接返回，不包装在 data 中）
		// 参考: https://docs.docker.com/registry/spec/auth/token/
		issuedAt := time.Now()
		ctx.JSON(http.StatusOK, gin.H{
			"token":        accessToken,
			"access_token": accessToken,
			"expires_in":   expiresIn,
			"issued_at":    issuedAt.Format(time.RFC3339),
		})
	}
}

// parseRepoPath 从 /v2/* 路径中解析仓库名和子路径
// 例如: "pat-test/small/manifests/latest" -> project="pat-test/small", subPath="manifests/latest"
// 支持多段仓库名（OCI Distribution Spec 允许 name 包含 /）
func parseRepoPath(path string) (project, subPath string, ok bool) {
	path = strings.TrimPrefix(path, "/")
	parts := strings.SplitN(path, "/manifests/", 2)
	if len(parts) == 2 {
		return parts[0], "manifests/" + parts[1], true
	}
	parts = strings.SplitN(path, "/blobs/", 2)
	if len(parts) == 2 {
		return parts[0], "blobs/" + parts[1], true
	}
	parts = strings.SplitN(path, "/tags/", 2)
	if len(parts) == 2 {
		return parts[0], "tags/" + parts[1], true
	}
	return "", "", false
}

// getProjectParam 从上下文获取解析后的 project（支持多段仓库名）
func getProjectParam(ctx *gin.Context) string {
	if v, ok := ctx.Get("_project"); ok {
		return v.(string)
	}
	return ctx.Param("project")
}

// getRepoParam 从上下文获取解析后的参数，若无则从 Param 读取
func getRepoParam(ctx *gin.Context, key string) string {
	if v, ok := ctx.Get("_" + key); ok {
		return v.(string)
	}
	return ctx.Param(key)
}

// dispatchRepoRequest 解析 /v2/* 路径并分发到对应 handler
func (c *RegistryController) dispatchRepoRequest(ctx *gin.Context) {
	rawPath := ctx.Param("path")
	if rawPath == "" || rawPath == "/" {
		response.Fail(ctx, 10001, "invalid repository path")
		return
	}
	path := strings.TrimPrefix(rawPath, "/")

	project, subPath, ok := parseRepoPath("/" + path)
	if !ok || project == "" {
		response.Fail(ctx, 10001, "invalid repository path")
		return
	}
	ctx.Set("_project", project)

	// 根据 subPath 和 method 分发
	// subPath 格式: manifests/<ref>, blobs/<digest>, tags/list, blobs/uploads/, blobs/uploads/<uuid>
	switch {
	case strings.HasPrefix(subPath, "manifests/"):
		ref := strings.TrimPrefix(subPath, "manifests/")
		if strings.HasSuffix(ref, "/referrers") {
			ref = strings.TrimSuffix(ref, "/referrers")
			ctx.Set("_reference", ref)
			c.GetReferrers(ctx)
		} else {
			ctx.Set("_reference", ref)
			switch ctx.Request.Method {
			case http.MethodGet, http.MethodHead:
				c.GetManifest(ctx)
			case http.MethodPut:
				c.PutManifest(ctx)
			case http.MethodDelete:
				c.DeleteManifest(ctx)
			default:
				ctx.Status(http.StatusMethodNotAllowed)
			}
		}
	case strings.HasPrefix(subPath, "blobs/"):
		rest := strings.TrimSuffix(strings.TrimPrefix(subPath, "blobs/"), "/")
		if rest == "uploads" {
			// POST /v2/<name>/blobs/uploads/
			c.InitiateBlobUpload(ctx)
		} else if strings.HasPrefix(rest, "uploads/") {
			uuidPart := strings.TrimPrefix(rest, "uploads/")
			ctx.Set("_uuid", uuidPart)
			switch ctx.Request.Method {
			case http.MethodPatch:
				c.UploadBlobChunk(ctx)
			case http.MethodPut:
				c.CompleteBlobUpload(ctx)
			case http.MethodDelete:
				c.CancelBlobUpload(ctx)
			case http.MethodGet:
				c.GetBlobUploadStatus(ctx)
			default:
				ctx.Status(http.StatusMethodNotAllowed)
			}
		} else {
			// blobs/<digest>，digest 可能含 sha256:xxx 格式
			ctx.Set("_digest", rest)
			switch ctx.Request.Method {
			case http.MethodHead:
				c.CheckBlob(ctx)
			case http.MethodGet:
				c.GetBlob(ctx)
			case http.MethodDelete:
				c.DeleteBlob(ctx)
			default:
				ctx.Status(http.StatusMethodNotAllowed)
			}
		}
	case strings.HasPrefix(subPath, "tags/"):
		if strings.TrimPrefix(subPath, "tags/") == "list" {
			c.ListTags(ctx)
		} else {
			response.Fail(ctx, 10001, "invalid tags path")
		}
	default:
		response.Fail(ctx, 10001, "invalid repository path")
	}
}

// RegisterRoutes 注册 Registry V2 路由（最小可用实现）
// 使用OptionalAuth中间件，支持Bearer Token和PAT认证
// 支持多段仓库名（如 pat-test/small），符合 OCI Distribution Spec
// 注意：Gin 不允许 catch-all (*path) 与静态路径共存，故仅用 /*path 统一处理（含 /v2/auth）
func (c *RegistryController) RegisterRoutes(r *gin.Engine, userSvc *user_service.Service) {
	c.userSvc = userSvc
	v2 := r.Group("/v2")
	if c.authMiddleware != nil {
		v2.Use(c.authMiddleware.OptionalAuth())
	}
	{
		v2.Any("/*path", c.dispatchV2Path)
	}
}

// dispatchV2Path 统一处理所有 /v2/* 请求：auth、API版本检查、catalog、仓库操作
func (c *RegistryController) dispatchV2Path(ctx *gin.Context) {
	rawPath := ctx.Param("path")
	path := strings.TrimPrefix(rawPath, "/")

	switch {
	case path == "auth" || strings.HasPrefix(path, "auth/"):
		c.TokenEndpoint(c.userSvc)(ctx)
	case path == "" || path == "/":
		c.APIVersionCheck(ctx)
	case path == "_catalog" || strings.HasPrefix(path, "_catalog/"):
		c.Catalog(ctx)
	default:
		c.dispatchRepoRequest(ctx)
	}
}

// NewRegistryController 创建Registry API控制器
func NewRegistryController(
	reg *registry.Registry,
	rbacSvc *rbac.Service,
	authMw *middleware.AuthMiddleware,
	projectSvc project.Service,
	userSvc *user_service.Service,
	whSvc *webhook_service.WebhookService,
) *RegistryController {
	return &RegistryController{
		registry:       reg,
		rbacSvc:        rbacSvc,
		authMiddleware: authMw,
		projectSvc:     projectSvc,
		userSvc:        userSvc,
		whSvc:          whSvc,
	}
}

// parsePaginationParams 解析分页参数
// 返回: n (每页数量), last (最后一项标识)
func parsePaginationParams(ctx *gin.Context, defaultN, maxN int) (n int, last string) {
	nStr := ctx.Query("n")
	last = ctx.Query("last")

	n = defaultN
	if nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
			if maxN > 0 && parsed > maxN {
				n = maxN
			} else {
				n = parsed
			}
		}
	}
	return n, last
}

// checkProjectPermission 检查项目权限
// 返回值：hasPermission bool, errorCode int, errorMessage string
func (c *RegistryController) checkProjectPermission(ctx *gin.Context, project, permission string) (bool, int, string) {
	// NOTE:
	// Docker/OCI 仓库名允许多段路径，例如：project/image 或 project/sub/image
	// 但当前领域模型中的 Project.Name 仅使用第一段（如 "project"）作为项目标识。
	// 为了让基于项目的权限校验与仓库名兼容，这里统一将 project 解析为：
	// - projectSlug: 第一段，用于查询和权限校验（领域 Project）
	// - fullRepo: 原始字符串，用于 registry 存储和 Catalog 等操作
	projectSlug := project
	if idx := strings.Index(project, "/"); idx > 0 {
		projectSlug = project[:idx]
	}

	// 1) 解析当前用户
	userIDVal, exists := ctx.Get(middleware.ContextKeyUserID)
	if !exists {
		// 单机/开发模式：为简化本地调试，未认证用户也允许基础操作
		// 生产环境应启用完整认证链路（TokenEndpoint + AuthMiddleware）并收紧此处逻辑。
		switch permission {
		case "pull", "push", "delete":
			return true, 0, ""
		default:
			return false, response.CodeInsufficientPermission, "权限不足"
		}
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return false, response.CodeUnauthorized, "无效的用户ID"
	}

	// 2) 先检查PAT权限（如果使用PAT token）
	//    必须在项目存在性检查之前，确保权限检查的严格性
	tokenTypeVal, tokenTypeExists := ctx.Get(middleware.ContextKeyTokenType)
	if tokenTypeExists {
		if tokenType, ok := tokenTypeVal.(string); ok && tokenType == "pat" {
			// PAT令牌需要检查scopes
			var hasPermission bool
			var errorCode int
			var errorMessage string
			switch permission {
			case "pull":
				// pull需要read权限
				hasPermission, errorCode, errorMessage = middleware.HasScope(ctx, "read")
			case "push":
				// push需要write权限
				hasPermission, errorCode, errorMessage = middleware.HasScope(ctx, "write")
			case "delete":
				// delete需要delete权限
				hasPermission, errorCode, errorMessage = middleware.HasScope(ctx, "delete")
			}
			if !hasPermission {
				return false, errorCode, errorMessage
			}
		}
	}

	// 3) 加载项目信息（使用领域 Project 模型，OwnerID 为 string）
	if c.projectSvc == nil {
		// 没有项目服务时，保守起见只允许 pull
		if permission == "pull" {
			return true, 0, ""
		}
		return false, response.CodeInsufficientPermission, "权限不足"
	}

	p, err := c.projectSvc.GetProjectByName(ctx.Request.Context(), projectSlug)
	if err != nil || p == nil {
		// 项目不存在
		if permission == "push" {
			// push操作：如果用户已通过PAT权限检查（有write权限）或使用JWT token，允许自动创建项目
			// 注意：PAT权限检查已在上面完成，这里只允许有write权限的用户自动创建项目
			return true, 0, ""
		} else if permission == "pull" {
			// pull操作：项目不存在，无法拉取
			return false, response.CodeNotFound, "项目不存在"
		}
		// 其他操作：项目不存在，拒绝
		return false, response.CodeNotFound, "项目不存在"
	}

	// 4) pull 权限：公开项目所有登录用户都可 pull；私有项目仅项目成员（后续可接入更细 RBAC）
	if permission == "pull" {
		if p.IsPublic {
			return true, 0, ""
		}
		// 私有项目：至少要求是该项目的 owner
		if p.OwnerID == userID.String() {
			return true, 0, ""
		}
		return false, response.CodeInsufficientPermission, "权限不足：仅项目所有者可以访问私有项目"
	}

	// 5) push / delete 等写操作：仅项目所有者可以操作
	if permission == "push" || permission == "delete" {
		// 项目所有者始终允许
		if p.OwnerID == userID.String() {
			return true, 0, ""
		}

		return false, response.CodeInsufficientPermission, "权限不足：仅项目所有者可以执行此操作"
	}

	// 未知权限：默认拒绝
	return false, response.CodeInsufficientPermission, "未知的权限类型"
}

// deleteManifest DeleteManifest 的实现已在 registry_manifest_controller.go 中

// InitiateBlobUpload 已在 registry_blob_controller.go 中实现

// UploadBlobChunk 已在 registry_blob_controller.go 中实现

// CompleteBlobUpload 已在 registry_blob_controller.go 中实现

// cancelBlobUpload 取消Blob上传
// DELETE /v2/<name>/blobs/uploads/<uuid>
func (c *RegistryController) CancelBlobUpload(ctx *gin.Context) {
	project := getProjectParam(ctx)
	uploadID := getRepoParam(ctx, "uuid")

	// 检查权限
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

	err := c.registry.CancelBlobUpload(ctx.Request.Context(), project, uploadID)
	if err != nil {
		// 记录失败日志
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.RecordError(ctx.Request.Context(), "cancel_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
			"repository": project,
			"upload_id":  uploadID,
		})
		response.Fail(ctx, 50001, "failed to cancel upload")
		return
	}

	// 记录成功日志
	var userID *uuid.UUID
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = &userUUID
		}
	}
	audit.Record(ctx.Request.Context(), "cancel_blob_upload", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
		"repository": project,
		"upload_id":  uploadID,
	})

	ctx.Status(http.StatusNoContent)
}

// deleteBlob 删除Blob
// DELETE /v2/<name>/blobs/<digest>
func (c *RegistryController) DeleteBlob(ctx *gin.Context) {
	project := getProjectParam(ctx)
	digest := getRepoParam(ctx, "digest")

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

	err := c.registry.DeleteBlob(ctx.Request.Context(), project, digest)
	if err != nil {
		log.Printf(`{"timestamp":"%s","level":"error","module":"registry","operation":"delete_blob","repository":"%s","digest":"%s","error":"%v"}`, time.Now().Format(time.RFC3339), project, digest, err)
		response.Fail(ctx, 50001, "failed to delete blob")
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

	log.Printf(`{"timestamp":"%s","level":"info","module":"registry","operation":"delete_blob","repository":"%s","digest":"%s","user_id":"%s","username":"%s","ip":"%s"}`, time.Now().Format(time.RFC3339), project, digest, userID, username, ctx.ClientIP())

	// 删除成功后，尝试同步项目的存储用量统计（最佳努力）
	if c.projectSvc != nil {
		projectSlug := project
		if idx := strings.Index(project, "/"); idx > 0 {
			projectSlug = project[:idx]
		}
		if p, pErr := c.projectSvc.GetProjectByName(ctx.Request.Context(), projectSlug); pErr == nil && p != nil {
			// 重新统计当前仓库下的所有 tag 及大小
			if tags, tErr := c.registry.ListTags(ctx.Request.Context(), project); tErr == nil {
				var totalSize int64
				for _, tag := range tags {
					if tagData, gErr := c.registry.GetTag(ctx.Request.Context(), project, tag); gErr == nil && tagData != nil {
						totalSize += tagData.Size
					}
				}
				updates := map[string]interface{}{
					"storage_used": totalSize,
				}
				_ = c.projectSvc.UpdateProject(ctx.Request.Context(), p.ID, updates)
			}
		}
	}

	ctx.Status(http.StatusAccepted)
}

// getBlobUploadStatus 获取上传状态
// GET /v2/<name>/blobs/uploads/<uuid>
func (c *RegistryController) GetBlobUploadStatus(ctx *gin.Context) {
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

	info, err := c.registry.GetBlobUploadStatus(ctx.Request.Context(), project, uploadID)
	if err != nil {
		// 记录失败日志
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.RecordError(ctx.Request.Context(), "get_blob_upload_status", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
			"repository": project,
			"upload_id":  uploadID,
		})
		response.Fail(ctx, 20001, "upload not found")
		return
	}

	// 记录成功日志
	var userID *uuid.UUID
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = &userUUID
		}
	}
	audit.Record(ctx.Request.Context(), "get_blob_upload_status", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
		"repository": project,
		"upload_id":  uploadID,
		"size":       info.Size,
	})

	ctx.Header("Docker-Upload-UUID", uploadID)
	ctx.Header("Range", fmt.Sprintf("0-%d", info.Size-1))
	ctx.Status(http.StatusNoContent)
}

// getReferrers 获取引用列表
// GET /v2/<name>/manifests/<reference>/referrers
func (c *RegistryController) GetReferrers(ctx *gin.Context) {
	project := getProjectParam(ctx)
	// 这里的 reference 实际上应该是 manifest 的 digest（符合 ORAS Referrers API 设计）
	digest := getRepoParam(ctx, "reference")

	// 检查读取权限
	hasPermission, errorCode, errorMessage := c.checkProjectPermission(ctx, project, "pull")
	if !hasPermission {
		// Docker Registry API 规范：权限拒绝返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		// 如果是PAT权限错误，记录错误码（但Docker Registry API仍返回401）
		if errorCode != 0 {
			log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"permission_denied","error_code":%d,"error_message":"%s","permission":"pull","project":"%s"}`, time.Now().Format(time.RFC3339), errorCode, errorMessage, project)
		}
		if errorCode != 0 {
			response.Fail(ctx, errorCode, errorMessage)
		} else {
			response.Fail(ctx, response.CodeInsufficientPermission, "权限不足")
		}
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	referrers, err := c.registry.ManifestReferrers(ctx.Request.Context(), project, digest)
	if err != nil {
		// 记录失败日志
		var userID *uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				userID = &userUUID
			}
		}
		audit.RecordError(ctx.Request.Context(), "get_referrers", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), err, map[string]interface{}{
			"repository": project,
			"digest":     digest,
		})
		response.Fail(ctx, 50001, "failed to get referrers")
		return
	}

	// 记录成功日志
	var userID *uuid.UUID
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			userID = &userUUID
		}
	}
	audit.Record(ctx.Request.Context(), "get_referrers", "image", nil, userID, ctx.ClientIP(), ctx.Request.UserAgent(), map[string]interface{}{
		"repository": project,
		"digest":     digest,
		"count":      len(referrers),
	})

	// 返回referrers
	result := gin.H{
		"referrers": referrers,
	}

	response.Success(ctx, result)
}

// getOwnerIDFromContext 从上下文中获取ownerID（用于自动创建项目）
// 优先从Gin上下文获取，如果不存在则从Authorization头解析
func (c *RegistryController) getOwnerIDFromContext(ctx *gin.Context) (ownerID string, ownerUUID uuid.UUID) {
	// 1) 优先从 Gin 上下文中获取用户ID（由 Auth/OptionalAuth 中间件设置）
	if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
		if userUUID, ok := userIDVal.(uuid.UUID); ok {
			return userUUID.String(), userUUID
		}
	}

	// 2) 如果中间件没有设置用户（例如某些 /v2 流程未经过统一认证），
	//    尝试直接从 Authorization 头中解析 JWT / PAT，并通过 userSvc 反查用户ID
	if c.userSvc != nil {
		authHeader := ctx.GetHeader("Authorization")

		// 2.1 Bearer Token 形式
		if strings.HasPrefix(authHeader, "Bearer ") {
			raw := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

			// Bearer pat_v1_xxx：直接按 PAT 处理
			if strings.HasPrefix(raw, "pat_v1_") {
				if patModel, err := c.userSvc.ValidatePAT(ctx, raw); err == nil && patModel != nil {
					return patModel.UserID.String(), patModel.UserID
				}
			} else {
				// 标准 JWT：通过 JWT 服务解析出 UserID
				if claims, err := c.userSvc.ValidateAccessToken(raw); err == nil && claims != nil {
					return claims.UserID.String(), claims.UserID
				}
			}
		}

		// 2.2 Basic Auth 形式：docker login / docker push 使用 Basic username:PAT
		//     这里仅关心 password 是否为 pat_v1_ 开头
		if username, password, ok := ctx.Request.BasicAuth(); ok {
			// 检查 password 是否是 PAT
			if strings.HasPrefix(password, "pat_v1_") {
				if patModel, err := c.userSvc.ValidatePAT(ctx, password); err == nil && patModel != nil {
					return patModel.UserID.String(), patModel.UserID
				}
			} else {
				// 尝试用户名密码认证
				if tokens, _, loginErr := c.userSvc.Login(ctx, username, password, ctx.ClientIP(), ctx.GetHeader("User-Agent")); loginErr == nil && tokens != nil {
					if claims, err := c.userSvc.ValidateAccessToken(tokens.AccessToken); err == nil && claims != nil {
						return claims.UserID.String(), claims.UserID
					}
				}
			}
		} else if strings.HasPrefix(authHeader, "Basic ") {
			// 备用方法：手动解析 Basic Auth
			encoded := strings.TrimSpace(strings.TrimPrefix(authHeader, "Basic "))
			if decodedBytes, err := base64.StdEncoding.DecodeString(encoded); err == nil {
				parts := strings.SplitN(string(decodedBytes), ":", 2)
				if len(parts) == 2 {
					password := parts[1]
					if strings.HasPrefix(password, "pat_v1_") {
						if patModel, err := c.userSvc.ValidatePAT(ctx, password); err == nil && patModel != nil {
							return patModel.UserID.String(), patModel.UserID
						}
					}
				}
			}
		}
	}

	return "", uuid.Nil
}

// ensureProjectExists 确保项目存在，如果不存在则自动创建
// 返回项目对象（如果创建失败或无法创建，返回nil）
func (c *RegistryController) ensureProjectExists(ctx *gin.Context, projectSlug, ownerID, description string) *project.Project {
	if c.projectSvc == nil || ownerID == "" {
		return nil
	}

	// 检查项目是否存在
	p, err := c.projectSvc.GetProjectByName(ctx.Request.Context(), projectSlug)
	if err == nil && p != nil {
		// 项目已存在，直接返回
		return p
	}

	// 项目不存在，自动创建
	created, createErr := c.projectSvc.CreateProject(
		ctx.Request.Context(),
		projectSlug,
		description,
		ownerID,
		false, // 默认私有项目
		0,     // 使用默认配额
	)
	if createErr != nil {
		// 创建失败（可能是并发创建导致已存在），尝试再次获取
		if p, err := c.projectSvc.GetProjectByName(ctx.Request.Context(), projectSlug); err == nil && p != nil {
			return p
		}
		log.Printf(`{"timestamp":"%s","level":"warn","module":"registry","operation":"ensure_project","project":"%s","owner_id":"%s","error":"failed to create project: %v"}`, time.Now().Format(time.RFC3339), projectSlug, ownerID, createErr)
		return nil
	}

	log.Printf(`{"timestamp":"%s","level":"info","module":"registry","operation":"auto_create_project","project":"%s","project_id":"%s","owner_id":"%s"}`, time.Now().Format(time.RFC3339), projectSlug, created.ID, ownerID)
	return created
}
