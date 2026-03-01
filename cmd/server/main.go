// Package main 应用入口
// 遵循《全平台通用开发任务设计规范文档》
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	ginswagger "github.com/cyp-registry/registry/internal/stub/gin-swagger"
	"github.com/gin-gonic/gin"

	"github.com/cyp-registry/registry/src/docs"
	"github.com/cyp-registry/registry/src/middleware"
	admin_controller "github.com/cyp-registry/registry/src/modules/admin/controller"
	admin_service "github.com/cyp-registry/registry/src/modules/admin/service"
	imageimport_module "github.com/cyp-registry/registry/src/modules/imageimport"
	imageimport_controller "github.com/cyp-registry/registry/src/modules/imageimport/controller"
	imageimport_service "github.com/cyp-registry/registry/src/modules/imageimport/service"
	project_controller "github.com/cyp-registry/registry/src/modules/project/controller"
	project_service "github.com/cyp-registry/registry/src/modules/project/service"
	"github.com/cyp-registry/registry/src/modules/rbac"
	"github.com/cyp-registry/registry/src/modules/registry"
	registry_controller "github.com/cyp-registry/registry/src/modules/registry/controller"
	"github.com/cyp-registry/registry/src/modules/storage/factory"
	"github.com/cyp-registry/registry/src/modules/user/controller"
	"github.com/cyp-registry/registry/src/modules/user/service"
	webhook_module "github.com/cyp-registry/registry/src/modules/webhook"
	webhook_controller "github.com/cyp-registry/registry/src/modules/webhook/controller"
	webhook_service "github.com/cyp-registry/registry/src/modules/webhook/service"
	"github.com/cyp-registry/registry/src/pkg/cache"
	"github.com/cyp-registry/registry/src/pkg/config"
	"github.com/cyp-registry/registry/src/pkg/database"
	appversion "github.com/cyp-registry/registry/src/pkg/version"
)

func main() {
	// 1. 强制使用 release 模式
	os.Setenv("GIN_MODE", "release")
	gin.SetMode(gin.ReleaseMode)

	// 1.1 禁用 log 包的默认时间戳前缀（因为我们使用 JSON 格式，自己控制时间戳）
	// 这样可以避免出现 "026/03/01" 这样的日期格式问题
	log.SetFlags(0)

	// 2. 尝试从当前工作目录的 .env 加载默认环境变量（仅补齐未显式设置的键）
	//    这样在本地直接运行二进制/`go run` 时，也能复用全局配置中心 .env。
	loadDotEnvDefaults(".env")

	// 3. 加载配置
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 4. 初始化数据库（带重试，避免 DB 启动慢导致服务直接退出）
	// 默认重试 60 次 * 1s；可通过环境变量覆盖：
	// - DB_INIT_RETRIES（次数）
	// - DB_INIT_INTERVAL_MS（间隔毫秒）
	dbRetries := 60
	if v := os.Getenv("DB_INIT_RETRIES"); v != "" {
		if n, e := strconv.Atoi(strings.TrimSpace(v)); e == nil && n > 0 {
			dbRetries = n
		}
	}
	dbInterval := 1000 * time.Millisecond
	if v := os.Getenv("DB_INIT_INTERVAL_MS"); v != "" {
		if n, e := strconv.Atoi(strings.TrimSpace(v)); e == nil && n > 0 {
			dbInterval = time.Duration(n) * time.Millisecond
		}
	}
	var lastDBErr error
	for i := 0; i < dbRetries; i++ {
		if err := database.Init(&cfg.Database); err == nil {
			lastDBErr = nil
			break
		} else {
			lastDBErr = err
			time.Sleep(dbInterval)
		}
	}
	if lastDBErr != nil {
		log.Fatalf("初始化数据库失败: %v", lastDBErr)
	}
	// 注意：数据库关闭将在优雅关闭流程中显式调用，不使用defer

	// 4. 初始化Redis
	if err := cache.Init(&cfg.Redis); err != nil {
		log.Printf("警告: 初始化Redis失败: %v，将使用内存缓存", err)
	}
	// 注意：缓存关闭将在优雅关闭流程中显式调用，不使用defer

	// 初始化缓存前缀
	cache.InitConfig(cfg.Redis.KeyPrefix)

	// 5. 初始化服务
	userSvc := service.NewService(&cfg.Auth.JWT, &cfg.Auth.PAT, cfg.Auth.BcryptCost)
	authMw := middleware.NewAuthMiddleware(userSvc)

	// 5.1 初始化存储（local/minio）
	store, err := factory.NewStorage(cfg)
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}

	// 5.2 初始化项目/Registry/Webhook服务
	projectSvc := project_service.NewService(database.GetDB(), store, cfg)
	regSvc := registry.NewRegistry(store)
	whSvc := webhook_service.NewWebhookService(&webhook_service.ServiceConfig{
		WorkerCount: 5,
		// 发送超时时间适当放宽，避免外部系统轻微抖动导致大量失败
		SendTimeout: 30 * time.Second,
	})

	// 5.3 初始化数据库表（Webhook）
	if err := webhook_module.InitWebhookDatabase(); err != nil {
		log.Printf("警告: 初始化Webhook数据库表失败: %v", err)
	}

	// 5.4 初始化数据库表（镜像导入）
	if err := imageimport_module.InitDatabase(); err != nil {
		log.Printf("警告: 初始化镜像导入数据库表失败: %v", err)
	}

	// 6. 初始化RBAC
	rbacSvc := rbac.NewService()
	if err := rbacSvc.InitDefaultRoles(context.TODO()); err != nil {
		log.Printf("警告: 初始化默认角色失败: %v", err)
	}
	if err := rbacSvc.InitDefaultPermissions(context.TODO()); err != nil {
		log.Printf("警告: 初始化默认权限失败: %v", err)
	}
	if err := rbacSvc.InitDefaultRolePermissions(context.TODO()); err != nil {
		log.Printf("警告: 初始化默认角色权限失败: %v", err)
	}

	// 7. 创建Gin引擎
	r := gin.New()

	// 显式配置 Trusted Proxies，避免默认“信任所有代理”带来的安全风险与警告
	// 单机/普通部署场景下不信任任何反向代理；如需自定义可通过代码/后续配置扩展
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("配置 Gin TrustedProxies 失败: %v", err)
	}

	// 8. 添加全局中间件
	// 注意：中间件顺序很重要
	// 1. Recovery中间件必须在最前面，用于捕获panic
	r.Use(middleware.NewRecoveryMiddleware().Recovery())
	// 2. RequestID中间件，生成请求追踪ID
	r.Use(middleware.NewRequestIDMiddleware().RequestID())
	// 3. 日志中间件，记录所有请求
	r.Use(middleware.NewLoggerMiddleware(&cfg.Logging).Logger())
	// 4. 全局错误处理中间件，捕获并记录所有错误
	r.Use(middleware.NewErrorHandlerMiddleware().ErrorHandler())
	// 5. CORS和安全头中间件
	r.Use(middleware.NewCORSMiddleware(&cfg.Security.CORS).CORS())
	r.Use(middleware.NewSecurityHeadersMiddleware().SecurityHeaders())

	// 9. 创建控制器
	userCtrl := controller.NewUserController(userSvc)
	projectCtrl := project_controller.NewProjectController(projectSvc, userSvc)
	regCtrl := registry_controller.NewRegistryController(regSvc, rbacSvc, authMw, projectSvc, userSvc, whSvc)
	whCtrl := webhook_controller.NewWebhookController(whSvc, authMw)
	adminSvc := admin_service.NewService()
	adminCtrl := admin_controller.NewAdminController(adminSvc)

	// 创建镜像导入服务（使用配置中的Host和Port构建本地仓库地址）
	localRegistryHost := cfg.App.Host
	if localRegistryHost == "0.0.0.0" {
		localRegistryHost = "localhost"
	}
	localRegistryHost = fmt.Sprintf("%s:%d", localRegistryHost, cfg.App.Port)
	imageImportSvc := imageimport_service.NewService(localRegistryHost)
	imageImportCtrl := imageimport_controller.NewImageImportController(imageImportSvc, projectSvc)

	// 10. 配置路由
	// 健康检查 - 必须在最前面
	healthHandler := func(c *gin.Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": cfg.App.Name,
			"version": appversion.GetVersion(),
		})
	}
	// 兼容前端通过 /api 前缀访问健康检查接口（如 /api/health）
	r.GET("/health", healthHandler)
	r.GET("/api/health", healthHandler)

	// 静态上传资源（头像等），统一挂载到 /uploads 前缀
	// 目录结构：<UPLOADS_DIR>/avatars/<userID>.<ext>
	// 使用环境变量或默认路径，确保使用绝对路径
	// 容器环境下优先使用 /tmp/uploads 或环境变量指定的目录
	uploadsDir := os.Getenv("UPLOADS_DIR")
	if uploadsDir == "" {
		// 容器环境下优先使用 /tmp 目录（通常有写入权限）
		if _, err := os.Stat("/tmp"); err == nil {
			uploadsDir = "/tmp/uploads"
		} else {
			// 否则尝试使用当前工作目录
			wd, err := os.Getwd()
			if err != nil {
				// 如果获取工作目录失败，尝试使用可执行文件所在目录
				execPath, execErr := os.Executable()
				if execErr != nil {
					log.Printf("警告: 无法确定上传目录路径，头像上传功能可能不可用")
				} else {
					uploadsDir = filepath.Join(filepath.Dir(execPath), "uploads")
				}
			} else {
				uploadsDir = filepath.Join(wd, "uploads")
			}
		}
	}

	if uploadsDir != "" {
		// 转换为绝对路径
		absUploadsDir, err := filepath.Abs(uploadsDir)
		if err != nil {
			log.Printf("警告: 解析上传目录绝对路径失败: %v，头像上传功能可能不可用", err)
		} else {
			avatarsDir := filepath.Join(absUploadsDir, "avatars")
			// 使用 0755 权限创建目录（所有者可读写执行，组和其他用户可读执行）
			// 如果目录已存在，MkdirAll 不会报错，但权限可能不正确
			if err := os.MkdirAll(avatarsDir, 0755); err != nil {
				log.Printf("警告: 创建上传目录失败: %v，头像上传功能可能不可用。请确保目录 %s 有写入权限或设置 UPLOADS_DIR 环境变量指向可写目录", err, absUploadsDir)
			} else {
				// 确保目录权限正确（即使目录已存在）
				if err := os.Chmod(avatarsDir, 0755); err != nil {
					log.Printf("警告: 设置上传目录权限失败: %v，目录: %s", err, avatarsDir)
				}
				log.Printf("已创建/验证上传目录: %s", avatarsDir)
				r.Static("/uploads", absUploadsDir)
			}
		}
	}

	// Swagger API文档配置
	docs.SwaggerInfo.Title = "CYP-Registry 容器镜像仓库管理 API"
	// 使用说明 - 将显示在 Swagger UI 页面顶部
	// 注意：Swagger UI 2.0 支持 Markdown 格式，但代码块需要使用正确的格式
	docs.SwaggerInfo.Description = "CYP-Registry 容器镜像仓库管理系统 RESTful 接口文档\n\n" +
		"## 使用说明\n\n" +
		"### 1. 认证方式\n" +
		"所有API接口（除登录和注册外）都需要在请求头中携带认证令牌：\n" +
		"- **Header名称**: Authorization\n" +
		"- **Header值**: Bearer {your_token}\n" +
		"- **获取令牌**: 通过 POST /api/v1/auth/login 接口登录后获取 access_token\n\n" +
		"### 2. 请求格式\n" +
		"- **Content-Type**: application/json\n" +
		"- **请求体**: JSON格式\n" +
		"- **分页参数**: page（页码，从1开始）、page_size（每页数量，默认20）\n\n" +
		"### 3. 响应格式\n" +
		"所有接口统一返回格式：\n" +
		"```\n" +
		"{\n" +
		"  \"code\": 20000,\n" +
		"  \"message\": \"success\",\n" +
		"  \"data\": {},\n" +
		"  \"timestamp\": 1234567890,\n" +
		"  \"trace_id\": \"xxx\"\n" +
		"}\n" +
		"```\n" +
		"**说明**：code=20000表示成功，其他为错误码\n\n" +
		"### 4. 错误码说明\n" +
		"- **20000**: 成功\n" +
		"- **20001**: 资源不存在\n" +
		"- **20002**: 资源已存在（冲突）\n" +
		"- **10001**: 参数错误\n" +
		"- **30001**: 未授权（需要登录）\n" +
		"- **30003**: 禁止访问（权限不足）\n" +
		"- **50001**: 服务器内部错误\n\n" +
		"### 5. 常用接口\n" +
		"- **用户登录**: POST /api/v1/auth/login\n" +
		"- **获取当前用户**: GET /api/v1/users/me\n" +
		"- **项目列表**: GET /api/v1/projects\n" +
		"- **创建项目**: POST /api/v1/projects\n" +
		"- **上传镜像**: POST /api/v1/repositories/{project}/images\n\n" +
		"### 6. 测试接口\n" +
		"1. 点击右上角 **\"Authorize\"** 按钮\n" +
		"2. 输入 access_token（从登录接口获取）\n" +
		"3. 点击 **\"Authorize\"** 确认\n" +
		"4. 现在可以测试需要认证的接口了\n\n" +
		"### 7. 注意事项\n" +
		"- **令牌有效期**: 默认7天，可在系统设置中创建自定义有效期的令牌\n" +
		"- **令牌权限**: 与创建令牌的用户权限相同\n" +
		"- **安全建议**: 不要在代码中硬编码令牌，使用环境变量或配置文件\n" +
		"- **生产环境**: 建议使用HTTPS协议访问API"
	// 如果 Host 是 0.0.0.0，使用 localhost 作为 Swagger Host（0.0.0.0 不能用于访问）
	swaggerHost := cfg.App.Host
	if swaggerHost == "0.0.0.0" {
		swaggerHost = "localhost"
	}
	docs.SwaggerInfo.Host = swaggerHost + ":" + fmt.Sprintf("%d", cfg.App.Port)
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
	docs.SwaggerInfo.BasePath = "/api/v1"
	r.GET("/swagger/*any", ginswagger.WrapHandler())

	// Web 静态资源（单镜像模式：/app/webdist）
	// 先配置静态资源路由，再配置根路径
	// 注意：静态资源路由必须在所有动态路由之前配置，确保静态文件优先匹配
	if st, err := os.Stat("./webdist"); err == nil && st.IsDir() {
		// 配置静态资源路由（必须在根路径之前）
		// 支持常见的静态资源路径：assets, static, js, css, images 等
		// 这些路径会优先匹配，不会影响 API 路由（/api/*, /v2/*, /swagger/* 等）
		r.Static("/assets", filepath.Join("webdist", "assets"))
		r.Static("/static", filepath.Join("webdist", "static"))
		r.Static("/js", filepath.Join("webdist", "js"))
		r.Static("/css", filepath.Join("webdist", "css"))
		r.Static("/images", filepath.Join("webdist", "images"))
		r.Static("/img", filepath.Join("webdist", "img"))
		// 单个静态文件
		r.StaticFile("/favicon.ico", filepath.Join("webdist", "favicon.ico"))
		r.StaticFile("/robots.txt", filepath.Join("webdist", "robots.txt"))
		// 注意：不要使用 r.StaticFS("/", ...)，这会匹配所有路径包括 API 路由
		// 如果前端构建后还有其他静态资源路径，需要在这里添加对应的 Static 配置
	}

	// 根路径配置（必须在所有其他路由之后，但在 NoRoute 之前）
	// 检查是否存在 webdist 目录
	hasWebdist := false
	if st, err := os.Stat("./webdist"); err == nil && st.IsDir() {
		hasWebdist = true
	}

	// 默认主页 HTML（当没有 webdist 时使用）
	defaultHomePageHTML := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>CYP-Registry - 容器镜像仓库管理系统</title>
  <style>
    * {
      margin: 0;
      padding: 0;
      box-sizing: border-box;
    }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      color: #333;
    }
    .container {
      background: white;
      border-radius: 12px;
      box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
      padding: 40px;
      max-width: 600px;
      width: 90%;
      text-align: center;
    }
    h1 {
      color: #667eea;
      margin-bottom: 10px;
      font-size: 2.5em;
    }
    .subtitle {
      color: #666;
      margin-bottom: 30px;
      font-size: 1.1em;
    }
    .links {
      display: flex;
      flex-direction: column;
      gap: 15px;
      margin-top: 30px;
    }
    a {
      display: inline-block;
      padding: 15px 30px;
      background: #667eea;
      color: white;
      text-decoration: none;
      border-radius: 8px;
      font-weight: 500;
      transition: all 0.3s ease;
      font-size: 1.1em;
    }
    a:hover {
      background: #5568d3;
      transform: translateY(-2px);
      box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
    }
    .status {
      margin-top: 30px;
      padding: 15px;
      background: #f0f4ff;
      border-radius: 8px;
      color: #667eea;
    }
    .status-item {
      margin: 8px 0;
      font-size: 0.95em;
    }
  </style>
</head>
<body>
  <div class="container">
    <h1>CYP-Registry</h1>
    <p class="subtitle">容器镜像仓库管理系统</p>
    
    <div class="links">
      <a href="/swagger/index.html">📚 API 文档 (Swagger)</a>
      <a href="/health">💚 健康检查</a>
    </div>
    
    <div class="status">
      <div class="status-item">✅ 服务运行中</div>
      <div class="status-item">🔗 访问 API 文档以查看所有可用接口</div>
    </div>
  </div>
</body>
</html>`

	// 配置根路径
	r.GET("/", func(c *gin.Context) {
		if hasWebdist {
			// 如果存在 webdist 目录，尝试返回前端应用
			indexPath := filepath.Join("webdist", "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
				c.File(indexPath)
				return
			}
		}
		// 否则显示默认主页
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.String(http.StatusOK, defaultHomePageHTML)
	})

	// API v1路由组
	v1 := r.Group("/api/v1")
	{
		// 认证路由（无需认证）
		auth := v1.Group("/auth")
		{
			auth.POST("/login", userCtrl.Login)
			auth.POST("/refresh", userCtrl.RefreshToken)
			auth.POST("/logout", authMw.Auth(), userCtrl.Logout)
			// 默认管理员首次提示接口（无鉴权，仅在进程启动后短时间内有效，且仅返回一次）
			auth.GET("/default-admin-once", userCtrl.GetDefaultAdminOnce)
		}

		// 用户路由（需要认证）
		users := v1.Group("/users")
		users.Use(authMw.Auth())
		{
			users.GET("/me", userCtrl.GetCurrentUser)
			users.GET("/me/token-info", userCtrl.GetCurrentTokenInfo)
			users.PUT("/me", userCtrl.UpdateCurrentUser)
			users.PUT("/me/password", userCtrl.ChangePassword)
			users.POST("/me/avatar", userCtrl.UploadAvatar)
			users.GET("/me/notification-settings", userCtrl.GetNotificationSettings)
			users.PUT("/me/notification-settings", userCtrl.UpdateNotificationSettings)
			users.POST("/me/pat", userCtrl.CreatePAT)
			users.GET("/me/pat", userCtrl.ListPAT)
			users.DELETE("/me/pat/:id", userCtrl.RevokePAT)

			// 管理员用户管理（前端兼容）
			users.GET("", authMw.AdminRequired(), userCtrl.ListUsers)
			users.GET("/:id", authMw.AdminRequired(), userCtrl.GetUser)
			users.PATCH("/:id", authMw.AdminRequired(), userCtrl.UpdateUser)
			users.DELETE("/:id", authMw.AdminRequired(), userCtrl.DeleteUser)
		}

		// 项目路由（需要认证）
		projects := v1.Group("/projects")
		projects.Use(authMw.Auth())
		{
			projects.POST("", projectCtrl.Create)
			projects.GET("", projectCtrl.List)
			projects.GET("/statistics", projectCtrl.GetStatistics)
			projects.GET("/:id", projectCtrl.Get)
			// 前端使用 PATCH，这里兼容 PUT/PATCH
			projects.PUT("/:id", projectCtrl.Update)
			projects.PATCH("/:id", projectCtrl.Update)
			projects.DELETE("/:id", projectCtrl.Delete)
			projects.PUT("/:id/quota", projectCtrl.UpdateQuota)
			projects.GET("/:id/storage", projectCtrl.GetStorageUsage)

			// 镜像导入路由
			projects.POST("/:id/images/import", imageImportCtrl.ImportImage)
			projects.GET("/:id/images/import", imageImportCtrl.ListTasks)
			projects.GET("/:id/images/import/:task_id", imageImportCtrl.GetTask)

			// 团队/成员功能已下线，这些路由保留占位但不再提供实际能力
			projects.POST("/:id/members", func(c *gin.Context) {
				c.JSON(410, gin.H{
					"code":    410,
					"message": "项目成员/团队功能已取消，请使用访问令牌与项目所有者权限进行管理",
				})
			})
			projects.GET("/:id/members", func(c *gin.Context) {
				c.JSON(410, gin.H{
					"code":    410,
					"message": "项目成员/团队功能已取消，不再提供成员列表",
				})
			})
			projects.DELETE("/:id/members/:user_id", func(c *gin.Context) {
				c.JSON(410, gin.H{
					"code":    410,
					"message": "项目成员/团队功能已取消，不再支持移除成员",
				})
			})
		}

		// 管理员路由（需要管理员权限）
		admin := v1.Group("/admin")
		admin.Use(authMw.Auth())
		admin.Use(authMw.AdminRequired())
		{
			admin.GET("/logs", adminCtrl.ListAuditLogs)
			admin.GET("/config", adminCtrl.GetSystemConfig)
			admin.PUT("/config", adminCtrl.UpdateSystemConfig)
		}
	}

	// Webhook API（controller 内部已使用 /api/v1/webhooks）
	whCtrl.RegisterRoutes(r)

	// Registry V2 API（实现 Docker Registry HTTP API V2）
	regCtrl.RegisterRoutes(r, userSvc)

	// NoRoute 必须在所有路由之后，用于前端 SPA 路由回退
	// 注意：如果前面的路由（/health, /swagger/*, /, /api/*, /v2/*）已经匹配，NoRoute 不会被调用
	if st, err := os.Stat("./webdist"); err == nil && st.IsDir() {
		r.NoRoute(func(c *gin.Context) {
			// 对于未匹配的路径，返回前端 index.html（SPA 路由）
			indexPath := filepath.Join("webdist", "index.html")
			if _, err := os.Stat(indexPath); err == nil {
				c.File(indexPath)
				return
			}
			c.JSON(404, gin.H{"error": "Not Found"})
		})
	} else {
		// 如果没有 webdist，为未匹配的路由返回 404
		r.NoRoute(func(c *gin.Context) {
			c.JSON(404, gin.H{"error": "Not Found"})
		})
	}

	// 11. 启动服务器
	// 为避免在容器/不同平台下因 Host 配置错误导致端口对外不可达，这里统一绑定 0.0.0.0
	addr := net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", cfg.App.Port))

	// 优化启动信息显示
	displayHost := cfg.App.Host
	if displayHost == "" || displayHost == "0.0.0.0" {
		displayHost = "localhost"
	}

	// 使用 http.Server 以便更好地控制启动过程
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 在 goroutine 中启动服务器
	serverErr := make(chan error, 1)
	serverStarted := make(chan bool, 1)

	go func() {
		// 启动服务器
		serverStarted <- true
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// 启动日志清理定时任务
	go startAuditLogCleanupTask()

	// 等待服务器开始启动
	<-serverStarted
	time.Sleep(300 * time.Millisecond) // 给服务器一点时间真正开始监听

	// 等待服务器真正开始监听（通过尝试连接来验证）
	maxRetries := 30
	serverReady := false
	for i := 0; i < maxRetries; i++ {
		// 检查是否有启动错误
		select {
		case err := <-serverErr:
			log.Fatalf("启动服务器失败: %v", err)
		default:
		}

		// 尝试连接服务器来验证是否已启动
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			serverReady = true
			// 服务器已启动，显示启动信息
			log.Printf("")
			log.Printf("╔════════════════════════════════════════════════════════════╗")
			log.Printf("║                 服务启动成功                                ║")
			log.Printf("╠════════════════════════════════════════════════════════════╣")
			log.Printf("║  应用名称: %-45s ║", cfg.App.Name)
			log.Printf("║  监听地址: %-45s ║", addr)
			if cfg.App.Host == "0.0.0.0" {
				log.Printf("║  本地访问: http://localhost:%-38d ║", cfg.App.Port)
				log.Printf("║  外部访问: http://<容器IP>:%-38d ║", cfg.App.Port)
			} else {
				log.Printf("║  访问地址: http://%-42s ║", fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port))
			}
			log.Printf("╠════════════════════════════════════════════════════════════╣")
			log.Printf("║  快速链接:                                                ║")
			log.Printf("║    • 主页:     http://%-42s ║", fmt.Sprintf("%s:%d/", displayHost, cfg.App.Port))
			log.Printf("║    • API文档:  http://%-42s ║", fmt.Sprintf("%s:%d/swagger/index.html", displayHost, cfg.App.Port))
			log.Printf("║    • 健康检查: http://%-42s ║", fmt.Sprintf("%s:%d/health", displayHost, cfg.App.Port))
			log.Printf("╚════════════════════════════════════════════════════════════╝")
			log.Printf("")
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	// 如果服务器在重试后仍未就绪，记录警告但继续运行
	if !serverReady {
		log.Printf("⚠️  警告: 无法验证服务器是否已启动，但将继续运行...")
		log.Printf("   请检查服务器日志确认服务是否正常运行")
		log.Printf("   如果服务未启动，请检查端口 %d 是否被占用", cfg.App.Port)
	}

	// 检查是否需要清理数据（通过环境变量与全局配置中心 .env 控制）
	// CLEANUP_ON_SHUTDOWN 由全局配置中心（根级 .env 文件）统一控制，并可被容器 environment 显式覆盖：
	// - CLEANUP_ON_SHUTDOWN=1 表示关闭时清理所有数据（删除操作）
	// - CLEANUP_ON_SHUTDOWN=0 或未设置表示关闭时仅停止服务，保留数据（停止操作）
	// 配置优先级：容器/进程环境变量 > 根级 .env 文件 > 默认值（保留数据，安全优先）
	// 建议做法：
	//   - 生产环境：在配置中心显式下发 CLEANUP_ON_SHUTDOWN=0，确保永远只停止、不删除数据
	//   - 开发/测试环境：按需设置为 1（需要每次停机清空数据时）
	cleanupEnv, shouldCleanup, cleanupSource, cleanupConflict, cleanupDotEnvVal := detectCleanupConfig()

	// 启动时自动检测并显示当前配置状态（从全局配置中心读取）
	log.Printf("")
	log.Printf("╔════════════════════════════════════════════════════════════╗")
	log.Printf("║  服务器关闭模式配置检测（全局配置中心）                    ║")
	log.Printf("╠════════════════════════════════════════════════════════════╣")
	if cleanupEnv == "" {
		log.Printf("║  配置来源: 环境变量/全局配置中心 (.env) - 未设置           ║")
		log.Printf("║  环境变量: CLEANUP_ON_SHUTDOWN (未设置)                    ║")
		log.Printf("║  关闭模式: 停止模式（保留数据）                            ║")
		log.Printf("║  说明: 当前未显式配置，将在关闭时仅停止服务并保留所有数据  ║")
		log.Printf("║  提示: 如需在关闭时清理所有数据，请在根级 .env 中设置为 1  ║")
	} else if shouldCleanup {
		if cleanupSource == "env" || cleanupSource == "env+.env" {
			log.Printf("║  配置来源: 容器/进程环境变量                               ║")
			if cleanupDotEnvVal != "" {
				if cleanupConflict {
					log.Printf("║  .env 中的值: CLEANUP_ON_SHUTDOWN=%s (已被环境变量覆盖)    ║", cleanupDotEnvVal)
					log.Printf("║  ⚠️  提示: 环境变量与 .env 中配置不一致，已优先采用环境变量 ║")
				} else {
					log.Printf("║  .env 中的值: CLEANUP_ON_SHUTDOWN=%s                       ║", cleanupDotEnvVal)
				}
			}
		} else {
			log.Printf("║  配置来源: 全局配置中心 (.env)                            ║")
		}
		log.Printf("║  生效值:   CLEANUP_ON_SHUTDOWN=%s                          ║", cleanupEnv)
		log.Printf("║  关闭模式: 删除模式（清理所有数据）                        ║")
		log.Printf("║  ⚠️  警告: 关闭服务器时将永久删除所有数据！                ║")
		log.Printf("║  ⚠️  警告: 包括用户数据、项目数据、镜像文件、缓存数据      ║")
		log.Printf("║  ⚠️  警告: 此操作不可恢复！                                ║")
		log.Printf("║  提示: 如需禁用清理模式，请在根级 .env 中设置为 0         ║")
	} else {
		if cleanupSource == "env" || cleanupSource == "env+.env" {
			log.Printf("║  配置来源: 容器/进程环境变量                               ║")
			if cleanupDotEnvVal != "" {
				if cleanupConflict {
					log.Printf("║  .env 中的值: CLEANUP_ON_SHUTDOWN=%s (已被环境变量覆盖)    ║", cleanupDotEnvVal)
					log.Printf("║  ⚠️  提示: 环境变量与 .env 中配置不一致，已优先采用环境变量 ║")
				} else {
					log.Printf("║  .env 中的值: CLEANUP_ON_SHUTDOWN=%s                       ║", cleanupDotEnvVal)
				}
			}
		} else {
			log.Printf("║  配置来源: 全局配置中心 (.env)                            ║")
		}
		log.Printf("║  生效值:   CLEANUP_ON_SHUTDOWN=%s                          ║", cleanupEnv)
		log.Printf("║  关闭模式: 停止模式（保留数据）                            ║")
		log.Printf("║  说明: 已显式关闭清理模式，关闭服务器时将保留所有数据     ║")
		log.Printf("║  提示: 如需启用清理模式，请删除该配置或设置为非 0 值      ║")
	}
	log.Printf("╚════════════════════════════════════════════════════════════╝")
	log.Printf("")

	// 等待中断信号
	// SIGINT/SIGTERM: 正常停止（保留数据，除非设置了CLEANUP_ON_SHUTDOWN=1）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// 关闭时再次确认当前模式
	log.Printf("")
	log.Printf("╔════════════════════════════════════════════════════════════╗")
	if shouldCleanup {
		log.Printf("║  ⚠️  收到停止信号 - 删除模式已启用                            ║")
		log.Printf("║  ⚠️  正在关闭服务器并清理所有数据...                        ║")
		log.Printf("║  ⚠️  警告: 所有数据将被永久删除，此操作不可恢复！            ║")
	} else {
		log.Printf("║  收到停止信号 - 停止模式（保留数据）                        ║")
		log.Printf("║  正在关闭服务器（保留所有数据）...                          ║")
	}
	log.Printf("╚════════════════════════════════════════════════════════════╝")
	log.Printf("")

	// 优雅关闭流程：
	// 1. 先停止接受新请求，等待正在处理的请求完成
	// 2. 关闭HTTP服务器
	// 3. 如果需要清理：清理数据库、文件、缓存
	// 4. 关闭数据库和缓存连接

	// 第一步：优雅关闭HTTP服务器（等待正在处理的请求完成）
	log.Println("正在关闭HTTP服务器...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("警告: HTTP服务器关闭超时或失败: %v", err)
		// 如果优雅关闭失败，强制关闭
		if err := srv.Close(); err != nil {
			log.Printf("错误: 强制关闭HTTP服务器失败: %v", err)
		}
	} else {
		log.Println("HTTP服务器已关闭")
	}

	// 第三步：如果需要清理，执行数据清理
	if shouldCleanup {
		log.Println("===========================================")
		log.Println("开始清理所有数据...")
		log.Println("===========================================")

		// 3.1 清理数据库数据
		log.Println("正在清理数据库数据...")
		if err := cleanupDatabase(); err != nil {
			log.Printf("警告: 清理数据库数据失败: %v", err)
		} else {
			log.Println("数据库数据已清理")
		}

		// 3.2 清理文件存储
		log.Println("正在清理文件存储...")
		if err := cleanupStorage(store); err != nil {
			log.Printf("警告: 清理文件存储失败: %v", err)
		} else {
			log.Println("文件存储已清理")
		}

		// 3.3 清理上传文件（头像等）
		log.Println("正在清理上传文件...")
		if err := cleanupUploads(uploadsDir); err != nil {
			log.Printf("警告: 清理上传文件失败: %v", err)
		} else {
			log.Println("上传文件已清理")
		}

		// 3.4 清理缓存数据
		log.Println("正在清理缓存数据...")
		if err := cleanupCache(); err != nil {
			log.Printf("警告: 清理缓存数据失败: %v", err)
		} else {
			log.Println("缓存数据已清理")
		}

		log.Println("===========================================")
		log.Println("数据清理完成")
		log.Println("===========================================")
	}

	// 第四步：关闭缓存连接
	log.Println("正在关闭缓存连接...")
	if err := cache.Close(); err != nil {
		log.Printf("警告: 关闭缓存连接失败: %v", err)
	} else {
		log.Println("缓存连接已关闭")
	}

	// 第五步：关闭数据库连接（最后关闭，确保所有数据库操作都已完成）
	log.Println("正在关闭数据库连接...")
	if err := database.Close(); err != nil {
		log.Printf("警告: 关闭数据库连接失败: %v", err)
	} else {
		log.Println("数据库连接已关闭")
	}

	if shouldCleanup {
		log.Println("服务器已完全关闭，所有数据已清理")
	} else {
		log.Println("服务器已完全关闭，数据已保留")
	}
}
