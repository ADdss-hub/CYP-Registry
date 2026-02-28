// Package registry_controller Registry API控制器
// 实现Docker Registry HTTP API V2的RESTful接口
// cspell:ignore ORAS
package registry_controller

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/cyp-registry/registry/src/pkg/config"
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

func (c *RegistryController) checkProjectPermission(ctx *gin.Context, project, permission string) bool {
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
			return true
		default:
			return false
		}
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		return false
	}

	// 2) 加载项目信息（使用领域 Project 模型，OwnerID 为 string）
	if c.projectSvc == nil {
		// 没有项目服务时，保守起见只允许 pull
		return permission == "pull"
	}

	p, err := c.projectSvc.GetProjectByName(ctx.Request.Context(), projectSlug)
	if err != nil || p == nil {
		// 在开发/单机环境中，如果仓库已经存在但项目元数据尚未初始化，
		// 为避免阻塞镜像推送，这里对已认证用户放宽为允许访问。
		// 生产环境应确保项目元数据与仓库保持一致。
		return true
	}

	// 3) pull 权限：公开项目所有登录用户都可 pull；私有项目仅项目成员（后续可接入更细 RBAC）
	if permission == "pull" {
		if p.IsPublic {
			return true
		}
		// 私有项目：至少要求是该项目的 owner
		return p.OwnerID == userID.String()
	}

	// 4) push / delete 等写操作：仅项目所有者可以操作
	if permission == "push" || permission == "delete" {
		// 项目所有者始终允许
		if p.OwnerID == userID.String() {
			return true
		}

		return false
	}

	// 未知权限：默认拒绝
	return false
}

// apiVersionCheck API版本检查
// GET /v2/
func (c *RegistryController) APIVersionCheck(ctx *gin.Context) {
	ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
	ctx.Status(http.StatusOK)
}

// apiVersionCheckUnauthenticated 未认证的API版本检查
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

// catalog 获取仓库列表
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

// listTags 列出项目的所有标签
// GET /v2/<name>/tags/list
func (c *RegistryController) ListTags(ctx *gin.Context) {
	project := getProjectParam(ctx)

	// 检查读取权限
	if !c.checkProjectPermission(ctx, project, "pull") {
		response.Fail(ctx, 30001, "permission denied")
		return
	}

	// 解析分页参数
	n, last := parsePaginationParams(ctx, 100, 0)

	// 获取标签列表
	tags, err := c.registry.ListTags(ctx.Request.Context(), project)
	if err != nil {
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

	response.Success(ctx, result)
}

// getManifest 获取Manifest
// GET/HEAD /v2/<name>/manifests/<reference>
func (c *RegistryController) GetManifest(ctx *gin.Context) {
	project := getProjectParam(ctx)
	reference := getRepoParam(ctx, "reference")

	// 检查读取权限
	if !c.checkProjectPermission(ctx, project, "pull") {
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 获取原始 Manifest 数据（避免重新序列化导致 digest 不匹配）
	manifestData, digest, err := c.registry.GetManifestRaw(ctx.Request.Context(), project, reference)
	if err != nil {
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

	// 返回原始 Manifest 内容
	ctx.Data(http.StatusOK, mediaType, manifestData)
}

// putManifest 上传Manifest
// PUT /v2/<name>/manifests/<reference>
func (c *RegistryController) PutManifest(ctx *gin.Context) {
	repoName := getProjectParam(ctx)
	reference := getRepoParam(ctx, "reference")

	// 检查写入权限
	if !c.checkProjectPermission(ctx, repoName, "push") {
		// Docker Registry API 规范：权限拒绝返回 401
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
			response.Fail(ctx, 20002, "immutable tag cannot be overwritten; please use a new version tag")
			return
		}
		response.Fail(ctx, 50001, "failed to store manifest")
		return
	}

	// 确保对应的 Project 在项目系统中可见（用于 Dashboard 展示）
	// 只有在注入了 projectSvc 且当前请求已完成认证时才尝试自动创建/更新项目统计信息
	if c.projectSvc != nil {
		var proj interface{}

		// 1) 优先从 Gin 上下文中获取用户ID（由 Auth/OptionalAuth 中间件设置）
		var ownerID string
		var ownerUUID uuid.UUID
		if userIDVal, exists := ctx.Get(middleware.ContextKeyUserID); exists {
			if userUUID, ok := userIDVal.(uuid.UUID); ok {
				ownerUUID = userUUID
				ownerID = userUUID.String()
			}
		}

		// 2) 如果中间件没有设置用户（例如某些 /v2 流程未经过统一认证），
		//    尝试直接从 Authorization 头中解析 JWT / PAT，并通过 userSvc 反查用户ID
		if ownerID == "" && c.userSvc != nil {
			authHeader := ctx.GetHeader("Authorization")

			// 2.1 Bearer Token 形式
			if strings.HasPrefix(authHeader, "Bearer ") {
				raw := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

				// Bearer pat_v1_xxx：直接按 PAT 处理
				if strings.HasPrefix(raw, "pat_v1_") {
					if patModel, err := c.userSvc.ValidatePAT(ctx, raw); err == nil && patModel != nil {
						ownerUUID = patModel.UserID
						ownerID = patModel.UserID.String()
					}
				} else {
					// 标准 JWT：通过 JWT 服务解析出 UserID
					if claims, err := c.userSvc.ValidateAccessToken(raw); err == nil && claims != nil {
						ownerUUID = claims.UserID
						ownerID = claims.UserID.String()
					}
				}
			}

			// 2.2 Basic Auth 形式：docker login / docker push 使用 Basic username:PAT
			//     这里仅关心 password 是否为 pat_v1_ 开头
			if ownerID == "" && strings.HasPrefix(authHeader, "Basic ") {
				encoded := strings.TrimSpace(strings.TrimPrefix(authHeader, "Basic "))
				if decodedBytes, err := base64.StdEncoding.DecodeString(encoded); err == nil {
					parts := strings.SplitN(string(decodedBytes), ":", 2)
					if len(parts) == 2 {
						password := parts[1]
						if strings.HasPrefix(password, "pat_v1_") {
							if patModel, err := c.userSvc.ValidatePAT(ctx, password); err == nil && patModel != nil {
								ownerUUID = patModel.UserID
								ownerID = patModel.UserID.String()
							}
						}
					}
				}
			}
		}

		// 3) 若拿到了 ownerID，则在需要时自动创建项目并同步统计。
		//    若拿不到 ownerID，则尽量仅同步已有项目的统计信息（不自动创建）。
		if ownerID != "" {
			// 将 Registry 仓库名作为“逻辑项目名”（与前端约定保持一致）
			// 例：test/hello-world -> test/hello-world
			// 前端在项目详情页会直接使用 project.name 作为仓库名来调用 /v2/<name>/tags/list
			logicalProjectName := repoName

			// 如果项目不存在，则以 push 用户作为 owner 自动创建
			p, getErr := c.projectSvc.GetProjectByName(ctx.Request.Context(), logicalProjectName)
			if getErr != nil {
				p, _ = c.projectSvc.CreateProject(
					ctx.Request.Context(),
					logicalProjectName,
					fmt.Sprintf("Auto created from image push (%s)", reference),
					ownerID,
					false,
					0,
				)
			}
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

					// 4) 触发 Webhook Push 事件（用于外部系统或审计日志）
					//    最佳努力：Webhook 失败不影响推送本身
					if c.whSvc != nil {
						// 尝试获取本次推送镜像的大小（仅当前 tag）
						var imageSize int64
						if tagData, err := c.registry.GetTag(ctx.Request.Context(), repoName, reference); err == nil && tagData != nil {
							imageSize = tagData.Size
						}

						// 推送事件中需要用户信息；优先使用中间件写入的用户名
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

						// 触发 Push Webhook（忽略错误，记录由 WebhookService 负责）
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
			if p, err := c.projectSvc.GetProjectByName(ctx.Request.Context(), repoName); err == nil && p != nil {
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
				}
			}
		}
	}

	// 设置响应头
	ctx.Header("Docker-Content-Digest", digest)
	ctx.Status(http.StatusCreated)
}

// deleteManifest 删除Manifest
// DELETE /v2/<name>/manifests/<reference>
func (c *RegistryController) DeleteManifest(ctx *gin.Context) {
	project := getProjectParam(ctx)
	reference := getRepoParam(ctx, "reference")

	// 检查删除权限
	if !c.checkProjectPermission(ctx, project, "delete") {
		response.Fail(ctx, 30001, "permission denied")
		return
	}

	// 删除Manifest
	err := c.registry.DeleteManifest(ctx.Request.Context(), project, reference)
	if err != nil {
		if err == registry.ErrManifestNotFound {
			response.Fail(ctx, 20001, "manifest not found")
			return
		}
		response.Fail(ctx, 50001, "failed to delete manifest")
		return
	}

	// 删除成功后，尝试同步项目的镜像数量与存储用量统计，并触发删除 Webhook（最佳努力）
	if c.projectSvc != nil {
		if p, pErr := c.projectSvc.GetProjectByName(ctx.Request.Context(), project); pErr == nil && p != nil {
			// 重新统计当前仓库下的所有 tag 及大小
			if tags, tErr := c.registry.ListTags(ctx.Request.Context(), project); tErr == nil {
				updates := map[string]interface{}{
					"image_count": len(tags),
				}

				var totalSize int64
				for _, tag := range tags {
					if tagData, gErr := c.registry.GetTag(ctx.Request.Context(), project, tag); gErr == nil && tagData != nil {
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
						project,
						reference,
						"", // digest 在删除接口中为可选，这里暂不强依赖
						userID,
						username,
					)
				}
			}
		}
	}

	ctx.Status(http.StatusAccepted)
}

// checkBlob 检查Blob是否存在
// HEAD /v2/<name>/blobs/<digest>
func (c *RegistryController) CheckBlob(ctx *gin.Context) {
	project := getProjectParam(ctx)
	digest := getRepoParam(ctx, "digest")

	// 检查读取权限
	if !c.checkProjectPermission(ctx, project, "pull") {
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	exists, err := c.registry.CheckBlob(ctx.Request.Context(), project, digest)
	if err != nil {
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
	ctx.Status(http.StatusOK)
}

// getBlob 获取Blob
// GET /v2/<name>/blobs/<digest>
func (c *RegistryController) GetBlob(ctx *gin.Context) {
	project := getProjectParam(ctx)
	digest := getRepoParam(ctx, "digest")

	// 检查读取权限
	if !c.checkProjectPermission(ctx, project, "pull") {
		// Registry API 规范：无权限时返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	// 获取Blob
	reader, size, err := c.registry.GetBlob(ctx.Request.Context(), project, digest)
	if err != nil {
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

	// 返回数据
	ctx.DataFromReader(http.StatusOK, size, "", reader, nil)
}

// initiateBlobUpload 初始化Blob上传
// POST /v2/<name>/blobs/uploads/
// 支持三种模式：
// 1. 跨仓库挂载：mount=<digest>&from=<source>
// 2. Monolithic upload：digest=<digest> + 请求体
// 3. 分片上传初始化：无特殊参数
func (c *RegistryController) InitiateBlobUpload(ctx *gin.Context) {
	project := getProjectParam(ctx)

	// 检查写入权限
	if !c.checkProjectPermission(ctx, project, "push") {
		// Docker Registry API 规范：权限拒绝返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.Header("WWW-Authenticate", fmt.Sprintf(`Bearer realm="http://%s/v2/auth"`, ctx.Request.Host))
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
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
			response.Fail(ctx, 50001, "failed to initiate upload")
			return
		}

		// 上传数据
		_, err = c.registry.UploadBlobChunk(ctx.Request.Context(), project, info.UUID, 0, bytes.NewReader(body), int64(len(body)))
		if err != nil {
			response.Fail(ctx, 50001, "failed to upload blob")
			return
		}

		// 完成上传
		err = c.registry.CompleteBlobUpload(ctx.Request.Context(), project, info.UUID, digest, int64(len(body)))
		if err != nil {
			response.Fail(ctx, 50001, "failed to complete upload: "+err.Error())
			return
		}

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
		response.Fail(ctx, 50001, "failed to initiate upload")
		return
	}

	// 返回上传URL（相对路径）
	locationPath := fmt.Sprintf("/v2/%s/blobs/uploads/%s", project, info.UUID)
	ctx.Header("Location", locationPath)
	ctx.Header("Docker-Upload-UUID", info.UUID)
	ctx.Header("Range", "0-0")
	ctx.Status(http.StatusAccepted)
}

// uploadBlobChunk 上传Blob分片
// PATCH /v2/<name>/blobs/uploads/<uuid>
func (c *RegistryController) UploadBlobChunk(ctx *gin.Context) {
	project := getProjectParam(ctx)
	uploadID := getRepoParam(ctx, "uuid")

	// 检查写入权限
	if !c.checkProjectPermission(ctx, project, "push") {
		// Docker Registry API 规范：权限拒绝返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.Header("WWW-Authenticate", fmt.Sprintf(`Bearer realm="http://%s/v2/auth"`, ctx.Request.Host))
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
		response.Fail(ctx, 50001, "failed to upload chunk")
		return
	}

	// 返回结果（相对路径）
	locationPath := fmt.Sprintf("/v2/%s/blobs/uploads/%s", project, uploadID)
	ctx.Header("Location", locationPath)
	ctx.Header("Docker-Upload-UUID", uploadID)
	ctx.Header("Range", fmt.Sprintf("0-%d", newOffset-1))
	ctx.Status(http.StatusAccepted)
}

// completeBlobUpload 完成Blob上传
// PUT /v2/<name>/blobs/uploads/<uuid>?digest=<digest>
func (c *RegistryController) CompleteBlobUpload(ctx *gin.Context) {
	project := getProjectParam(ctx)
	uploadID := getRepoParam(ctx, "uuid")
	digest := ctx.Query("digest")

	// 检查写入权限
	if !c.checkProjectPermission(ctx, project, "push") {
		// Docker Registry API 规范：权限拒绝返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.Header("WWW-Authenticate", fmt.Sprintf(`Bearer realm="http://%s/v2/auth"`, ctx.Request.Host))
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
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// 返回结果（相对路径）
	locationPath := fmt.Sprintf("/v2/%s/blobs/%s", project, digest)
	ctx.Header("Location", locationPath)
	ctx.Header("Docker-Content-Digest", digest)
	ctx.Status(http.StatusCreated)
}

// cancelBlobUpload 取消Blob上传
// DELETE /v2/<name>/blobs/uploads/<uuid>
func (c *RegistryController) CancelBlobUpload(ctx *gin.Context) {
	project := getProjectParam(ctx)
	uploadID := getRepoParam(ctx, "uuid")

	// 检查权限
	if !c.checkProjectPermission(ctx, project, "push") {
		// Docker Registry API 规范：权限拒绝返回 401
		ctx.Header("Docker-Distribution-Api-Version", "registry/2.0")
		ctx.Header("WWW-Authenticate", fmt.Sprintf(`Bearer realm="http://%s/v2/auth"`, ctx.Request.Host))
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	err := c.registry.CancelBlobUpload(ctx.Request.Context(), project, uploadID)
	if err != nil {
		response.Fail(ctx, 50001, "failed to cancel upload")
		return
	}

	ctx.Status(http.StatusNoContent)
}

// deleteBlob 删除Blob
// DELETE /v2/<name>/blobs/<digest>
func (c *RegistryController) DeleteBlob(ctx *gin.Context) {
	project := getProjectParam(ctx)
	digest := getRepoParam(ctx, "digest")

	// 检查删除权限
	if !c.checkProjectPermission(ctx, project, "delete") {
		response.Fail(ctx, 30001, "permission denied")
		return
	}

	err := c.registry.DeleteBlob(ctx.Request.Context(), project, digest)
	if err != nil {
		response.Fail(ctx, 50001, "failed to delete blob")
		return
	}

	ctx.Status(http.StatusAccepted)
}

// getBlobUploadStatus 获取上传状态
// GET /v2/<name>/blobs/uploads/<uuid>
func (c *RegistryController) GetBlobUploadStatus(ctx *gin.Context) {
	project := getProjectParam(ctx)
	uploadID := getRepoParam(ctx, "uuid")

	info, err := c.registry.GetBlobUploadStatus(ctx.Request.Context(), project, uploadID)
	if err != nil {
		response.Fail(ctx, 20001, "upload not found")
		return
	}

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
	if !c.checkProjectPermission(ctx, project, "pull") {
		response.Fail(ctx, 30001, "permission denied")
		return
	}

	referrers, err := c.registry.ManifestReferrers(ctx.Request.Context(), project, digest)
	if err != nil {
		response.Fail(ctx, 50001, "failed to get referrers")
		return
	}

	// 返回referrers
	result := gin.H{
		"referrers": referrers,
	}

	response.Success(ctx, result)
}
