# CYP-Registry

<div align="center">

![Version](https://img.shields.io/badge/version-1.0.3-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Go Version](https://img.shields.io/badge/go-1.24-blue.svg)
![Docker](https://img.shields.io/badge/docker-ready-blue.svg)

**科技赋能，规范引领** —— 安全可控的私有镜像仓库解决方案

[功能特性](#功能特性) • [快速开始](#快速开始) • [文档](#文档) • [API](#api) • [贡献](#贡献)

</div>

---

## 📖 项目简介

CYP-Registry 是一款面向个人开发者和中小型团队的中文私有容器镜像仓库管理系统，严格遵循 **OCI Distribution Specification**，提供完整的容器镜像管理、权限控制等功能。

### 核心优势

- ✅ **零兼容性问题**：严格遵循 OCI Distribution Specification，与 Docker、Podman 等客户端完全兼容
- ✅ **零意外中断**：高可用架构设计，支持自动故障恢复
- ✅ **零回归缺陷**：完整的自动化测试流程
- ✅ **单镜像部署**：All-in-One 模式，内置 PostgreSQL + Redis，一键启动
- ✅ **中文友好**：完整的中文界面和文档支持

## ✨ 功能特性

### 🔐 用户认证与权限管理
- 多种认证方式：账号密码、Personal Access Token (PAT)、Robot Account
- 基于 RBAC 的细粒度权限控制
- 支持项目级别的公开/私有设置
- JWT Token 自动刷新机制

### 📦 镜像仓库管理
- 支持 Docker Registry API v2
- 镜像推送、拉取、删除操作
- 镜像标签管理
- 存储配额管理
- 支持本地存储和 MinIO 对象存储

### 🔔 Webhook 集成
- 支持多种事件类型（镜像推送、拉取等）
- 自定义 Webhook URL 和密钥
- 实时事件通知

### 🎨 Web 管理界面
- 现代化 Vue3 + TypeScript 前端
- 响应式设计，支持移动端
- 深色/浅色主题切换
- 实时数据展示和操作

## 🚀 快速开始

### 前置要求

- Docker 20.10+ 
- Docker Compose 2.0+
- 4GB+ 可用内存
- 10GB+ 可用磁盘空间

### 方式一：单镜像模式（推荐）

单镜像模式适合**离线/单机/开发环境**，一个容器内置 PostgreSQL + Redis + 应用服务，无需额外依赖。

```bash
# 克隆项目
git clone https://github.com/ADdss-hub/CYP-Registry.git
cd CYP-Registry

# 构建并启动（首次启动会自动生成 .env 配置文件）
docker compose -f docker-compose.single.yml up -d --build

# 查看服务状态
docker compose -f docker-compose.single.yml ps

# 查看日志
docker compose -f docker-compose.single.yml logs -f
```

**访问服务：**
- Web 界面：http://localhost:8080
- Registry API：http://localhost:8080/v2/
- API 文档：http://localhost:8080/docs

### 方式二：使用预构建镜像

```bash
# 拉取镜像（待发布到 Docker Hub）
docker pull addss-hub/cyp-registry:latest

# 运行容器
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-registry-data:/data \
  addss-hub/cyp-registry:latest
```

### 方式三：从源码构建

```bash
# 克隆项目
git clone https://github.com/ADdss-hub/CYP-Registry.git
cd CYP-Registry

# 构建后端
cd src && go build -o bin/registry-server ./cmd/server

# 构建前端
cd web && npm install && npm run build

# 启动服务
./bin/registry-server
```

## 📚 文档

- [快速开始指南](./deploy/QUICK_START.md) - 详细的安装和使用教程
- [部署文档](./deploy/DEPLOYMENT.md) - 生产环境部署指南
- [运维手册](./deploy/OPERATIONS.md) - 日常运维操作
- [环境变量配置](./docs/ENV.md) - 完整的配置说明
- [API 文档](./docs/api/API.md) - RESTful API 接口文档

## 🔧 配置说明

### 环境变量

项目支持通过 `.env` 文件或环境变量进行配置。首次启动会自动生成 `.env` 文件。

**关键配置项：**

```env
# 应用配置
APP_NAME=CYP-Registry
APP_ENV=production
APP_HOST=0.0.0.0
APP_PORT=8080

# 数据库配置
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=registry
DB_NAME=registry_db
DB_PASSWORD=<自动生成>

# Redis 配置
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT 配置
JWT_SECRET=<自动生成>

# 存储配置
STORAGE_TYPE=local  # 或 minio
STORAGE_LOCAL_ROOT_PATH=/data/storage
```

完整配置说明请参考 [环境变量文档](./docs/ENV.md)。

### Docker Registry 配置

在 Docker 客户端配置 insecure registry（开发环境）：

**Linux/macOS:**
```json
// /etc/docker/daemon.json
{
  "insecure-registries": ["localhost:8080"]
}
```

**Windows (Docker Desktop):**
在 Settings → Docker Engine 中添加：
```json
{
  "insecure-registries": ["localhost:8080"]
}
```

重启 Docker 服务后即可使用。

## 🔌 API

### 认证

所有 API 请求需要在 Header 中包含 Access Token：

```bash
curl -X GET http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer <your-access-token>"
```

### 常用 API 端点

#### 认证
- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/refresh` - 刷新 Token

#### 项目管理
- `GET /api/v1/projects` - 获取项目列表
- `POST /api/v1/projects` - 创建项目
- `GET /api/v1/projects/:id` - 获取项目详情
- `PUT /api/v1/projects/:id` - 更新项目
- `DELETE /api/v1/projects/:id` - 删除项目

#### 镜像管理
- `GET /api/v1/projects/:id/images` - 获取镜像列表
- `DELETE /api/v1/projects/:id/images/:name` - 删除镜像

完整的 API 文档请访问：http://localhost:8080/docs

## 🐳 构建 Docker 镜像

### 构建单镜像版本

```bash
# 构建镜像
docker build -f Dockerfile.single -t cyp-registry:single .

# 运行容器
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-registry-data:/data \
  cyp-registry:single
```

### 推送到 Docker Hub

```bash
# 登录 Docker Hub
docker login

# 标记镜像
docker tag cyp-registry:single addss-hub/cyp-registry:latest
docker tag cyp-registry:single addss-hub/cyp-registry:v1.0.3

# 推送镜像
docker push addss-hub/cyp-registry:latest
docker push addss-hub/cyp-registry:v1.0.3
```

## 🧪 测试

```bash
# 运行后端测试
cd src && go test ./...

# 运行前端测试
cd web && npm run test

# 运行 E2E 测试
cd web && npm run test:e2e
```

## 📊 技术栈

### 后端
- **语言**: Go 1.24
- **框架**: Gin
- **数据库**: PostgreSQL 15
- **缓存**: Redis
- **ORM**: GORM
- **认证**: JWT

### 前端
- **框架**: Vue 3 + TypeScript
- **构建工具**: Vite
- **UI 组件**: 自定义组件库
- **测试**: Cypress

### 基础设施
- **容器化**: Docker + Docker Compose
- **存储**: 本地文件系统 / MinIO
- **监控**: Prometheus + Grafana

## 🤝 贡献

欢迎贡献代码！请遵循以下步骤：

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📝 许可证

本项目采用 MIT 许可证。详情请参阅 [LICENSE](LICENSE) 文件。

## 👤 作者

**CYP**

- 邮箱: nasDSSCYP@outlook.com
- GitHub: [@ADdss-hub](https://github.com/ADdss-hub)

## 🙏 致谢

- [Docker Registry](https://github.com/distribution/distribution) - OCI Distribution Specification 参考实现
- [Gin](https://github.com/gin-gonic/gin) - Go Web 框架
- [Vue.js](https://vuejs.org/) - 渐进式 JavaScript 框架

## 📞 获取帮助

- 📧 邮件: nasDSSCYP@outlook.com
- 🐛 问题反馈: [GitHub Issues](https://github.com/ADdss-hub/CYP-Registry/issues)
- 📖 文档: [项目文档](./docs/)

---

<div align="center">

**如果这个项目对你有帮助，请给个 ⭐ Star！**

Made with ❤️ by CYP

</div>
