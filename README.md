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
- **从 URL 添加镜像**：支持从 Docker Hub、GHCR 等公共仓库拉取镜像到私有仓库
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

- Docker 20.10+ 或 Podman 4.0+
- Docker Compose 2.0+（可选，单镜像模式可直接使用 `docker run`）
- 4GB+ 可用内存
- 10GB+ 可用磁盘空间

### 支持的环境和平台

**操作系统：**
- ✅ Linux（Ubuntu、CentOS、Debian、Alpine、RHEL、SUSE 等）
  - ✅ Ubuntu 18.04+ / Debian 10+（标准 GNU 工具集）
  - ✅ CentOS 7+ / RHEL 7+（SELinux 兼容，容器内通常不需要特殊配置）
  - ✅ Alpine Linux 3.15+（BusyBox 工具集，已优化兼容性）
  - ✅ SUSE Linux Enterprise Server / openSUSE（标准 Linux 工具集）
- ✅ macOS（Docker Desktop for Mac）
- ✅ Windows（Docker Desktop for Windows、WSL2）
- ✅ NAS 系统（群晖 Synology、QNAP、威联通等）

**文件系统支持：**
- ✅ ext4（Linux 标准文件系统）
- ✅ xfs（RHEL/CentOS 常用）
- ✅ btrfs（SUSE/openSUSE 常用）
- ✅ zfs（高级 NAS 系统）
- ✅ overlay2（Docker 默认存储驱动）
- ✅ tmpfs（/run、/tmp 等临时文件系统）

**架构支持：**
- ✅ AMD64/x86_64（默认）
- ⚠️ ARM64（需自行构建，见下方说明）
- ⚠️ ARMv7（需自行构建）

**容器运行时：**
- ✅ Docker（推荐）
- ✅ Podman（兼容 Docker CLI）
- ✅ containerd（通过 Docker/containerd）

**部署方式：**
- ✅ Docker Compose
- ✅ Docker 直接运行
- ✅ Kubernetes（需自行编写 YAML，见下方说明）
- ✅ 云平台（AWS ECS、Azure Container Instances、GCP Cloud Run 等）

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

**单镜像配置说明（重要）：**
- 默认**不需要**提供 `config.yaml`：容器会在启动时自动生成 `/app/config.yaml`（基于当前环境变量），并且**生成提示日志仅首次显示一次**。
- 如需固定配置（推荐生产）：在宿主机准备 `./config.yaml`，并在 `docker-compose.single.yml` 中启用对应的 volume 挂载（只读）。

**访问服务：**
- Web 界面：http://localhost:8080
- Registry API：http://localhost:8080/v2/
- API 文档：http://localhost:8080/docs

**使用 Podman（替代 Docker）：**
```bash
# Podman 兼容 Docker CLI，只需将 docker 替换为 podman
podman compose -f docker-compose.single.yml up -d --build

# 或直接运行
podman run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-registry-pg-data:/var/lib/postgresql/data \
  -v cyp-registry-redis-data:/data/redis \
  -v cyp-registry-storage:/data/storage \
  -v cyp-registry-logs:/app/logs \
  ghcr.io/addss-hub/cyp-registry:v1.0.3
```

### 方式二：使用预构建镜像

#### 从 GitHub Container Registry (GHCR) 拉取

```bash
# 拉取指定版本（推荐生产环境）
docker pull ghcr.io/addss-hub/cyp-registry:v1.0.3

# 或拉取带日期的版本号
docker pull ghcr.io/addss-hub/cyp-registry:v1.0.3-2026-02-28

# 运行容器（单镜像模式）
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-registry-pg-data:/var/lib/postgresql/data \
  -v cyp-registry-redis-data:/data/redis \
  -v cyp-registry-storage:/data/storage \
  -v cyp-registry-logs:/app/logs \
  ghcr.io/addss-hub/cyp-registry:v1.0.3
```

#### 从 Docker Hub 拉取（如果已同步）

```bash
# 拉取指定版本
docker pull addss-hub/cyp-registry:v1.0.3

# 运行容器
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-registry-pg-data:/var/lib/postgresql/data \
  -v cyp-registry-redis-data:/data/redis \
  -v cyp-registry-storage:/data/storage \
  -v cyp-registry-logs:/app/logs \
  addss-hub/cyp-registry:v1.0.3
```

**镜像版本说明：**
- `v1.0.3`：标准版本号（语义化版本，推荐使用）
- `v1.0.3-2026-02-28`：带日期的版本号（便于识别发布日期）
- **注意**：镜像仓库使用语义化版本号标签，不提供 `latest` 标签。请使用具体的版本号标签拉取镜像。

#### 在其他环境部署（生产环境推荐）

**使用 Docker Compose 部署（推荐）：**

1. **创建部署目录和配置文件：**
```bash
mkdir -p /opt/cyp-registry
cd /opt/cyp-registry

# 创建 docker-compose.yml
cat > docker-compose.yml << 'EOF'
version: '3.8'

services:
  cyp-registry:
    image: ghcr.io/addss-hub/cyp-registry:v1.0.3
    container_name: cyp-registry
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      # 应用配置
      - APP_ENV=production
      - APP_HOST=0.0.0.0
      - APP_PORT=8080
      
      # 数据库配置（内置 PostgreSQL）
      - DB_HOST=127.0.0.1
      - DB_PORT=5432
      - DB_USER=registry
      - DB_NAME=registry_db
      - DB_PASSWORD=${DB_PASSWORD:-}  # 建议设置强密码
      
      # Redis 配置（内置 Redis）
      - REDIS_HOST=127.0.0.1
      - REDIS_PORT=6379
      - REDIS_PASSWORD=${REDIS_PASSWORD:-}  # 建议设置密码
      
      # JWT 密钥（必须设置）
      - JWT_SECRET=${JWT_SECRET:-}  # 必须设置强随机值
      
      # 存储配置
      - STORAGE_TYPE=local
      - STORAGE_LOCAL_ROOT_PATH=/data/storage
    volumes:
      # 数据持久化
      - pg_data:/var/lib/postgresql/data
      - redis_data:/data/redis
      - storage_data:/data/storage
      - logs_data:/app/logs
      # 可选：挂载自定义配置文件
      # - ./config.yaml:/app/config.yaml:ro
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3

volumes:
  pg_data:
  redis_data:
  storage_data:
  logs_data:
EOF

# 创建 .env 文件（包含敏感信息）
cat > .env << 'EOF'
# 数据库密码（建议使用强随机值）
DB_PASSWORD=your_strong_db_password_here

# Redis 密码（可选，建议设置）
REDIS_PASSWORD=your_redis_password_here

# JWT 密钥（必须设置，建议使用强随机值）
JWT_SECRET=your_jwt_secret_here
EOF

# 设置 .env 文件权限
chmod 600 .env
```

2. **启动服务：**
```bash
# 拉取最新镜像
docker compose pull

# 启动服务
docker compose up -d

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f
```

**使用 Docker 命令直接部署：**

```bash
# 拉取镜像
docker pull ghcr.io/addss-hub/cyp-registry:v1.0.3

# 创建数据目录（Linux/macOS）
mkdir -p /data/cyp-registry/{pg-data,redis-data,storage,logs}
chmod -R 755 /data/cyp-registry

# Windows/NAS 环境：建议使用 Docker 命名卷（自动管理权限）
# docker volume create cyp-registry-pg-data
# docker volume create cyp-registry-redis-data
# docker volume create cyp-registry-storage
# docker volume create cyp-registry-logs

# 运行容器
docker run -d \
  --name cyp-registry \
  --restart unless-stopped \
  -p 8080:8080 \
  -e APP_ENV=production \
  -e DB_PASSWORD=your_strong_db_password \
  -e REDIS_PASSWORD=your_redis_password \
  -e JWT_SECRET=your_jwt_secret \
  -v /data/cyp-registry/pg-data:/var/lib/postgresql/data \
  -v /data/cyp-registry/redis-data:/data/redis \
  -v /data/cyp-registry/storage:/data/storage \
  -v /data/cyp-registry/logs:/app/logs \
  ghcr.io/addss-hub/cyp-registry:v1.0.3

# Windows/NAS 环境使用命名卷的示例：
# docker run -d \
#   --name cyp-registry \
#   --restart unless-stopped \
#   -p 8080:8080 \
#   -e APP_ENV=production \
#   -e DB_PASSWORD=your_strong_db_password \
#   -e REDIS_PASSWORD=your_redis_password \
#   -e JWT_SECRET=your_jwt_secret \
#   -v cyp-registry-pg-data:/var/lib/postgresql/data \
#   -v cyp-registry-redis-data:/data/redis \
#   -v cyp-registry-storage:/data/storage \
#   -v cyp-registry-logs:/app/logs \
#   ghcr.io/addss-hub/cyp-registry:v1.0.3
```

**生产环境注意事项：**

1. **安全配置：**
   - ✅ 必须设置强密码的 `DB_PASSWORD` 和 `JWT_SECRET`
   - ✅ 建议设置 `REDIS_PASSWORD`
   - ✅ 使用 HTTPS（通过反向代理，如 Nginx）
   - ✅ 定期更新镜像到最新稳定版本

2. **数据持久化：**
   - ✅ 使用命名卷或绑定挂载确保数据持久化
   - ✅ 定期备份 PostgreSQL 数据目录
   - ✅ 监控磁盘空间使用情况
   - ✅ **NAS/Windows 环境**：建议使用 Docker 命名卷而非绑定挂载，避免权限问题

3. **网络配置：**
   - ✅ 生产环境建议使用反向代理（Nginx/Caddy）
   - ✅ 配置防火墙规则，仅开放必要端口
   - ✅ 如需外部访问，配置域名和 SSL 证书

4. **监控和维护：**
   - ✅ 配置健康检查（已内置）
   - ✅ 设置日志轮转
   - ✅ 监控容器资源使用情况

5. **NAS/Windows Docker 环境特殊说明：**
   - ✅ 系统会自动检测挂载点并在需要时使用子目录（`/var/lib/postgresql/data/pgdata`）
   - ✅ 所有权限设置都有重试机制，兼容不同的权限模型
   - ✅ 日志文件会自动创建并设置正确的权限
   - ✅ 健康检查使用 `wget`，兼容 Alpine/BusyBox 环境

**访问服务：**
- Web 界面：http://your-server-ip:8080
- Registry API：http://your-server-ip:8080/v2/
- API 文档：http://your-server-ip:8080/docs

**首次登录：**
1. 访问 Web 界面
2. 注册管理员账号
3. 创建项目并开始使用

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

**生产环境补充（自动设置密码 & 仅提示一次）：**
- 若你未显式提供 `DB_PASSWORD` / `JWT_SECRET`，单镜像容器会在首次启动时自动生成强随机值并持久化到数据卷（后续重启不会改变，也不会重复打印“已自动生成”的提示日志）。
- 需要查看当前自动生成的值时，可在容器内读取：
  - `cat /var/lib/postgresql/data/.cyp_registry_db_password`
  - `cat /var/lib/postgresql/data/.cyp_registry_jwt_secret`

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
- `POST /api/v1/projects/:id/images/add-from-url` - 从 URL 添加镜像
- `DELETE /api/v1/projects/:id/images/:name` - 删除镜像

**从 URL 添加镜像功能说明：**

通过 Web 界面或 API 可以从公共镜像仓库（如 Docker Hub、GHCR）拉取镜像到私有仓库：

**Web 界面操作：**
1. 进入项目 → 镜像管理页面
2. 点击 "+ 添加镜像" 按钮
3. 选择 "从 URL 添加"
4. 填写镜像信息：
   - **镜像**（必填）：输入镜像名称或完整 URL
     - 示例：`nginx:latest`、`ghcr.io/addss-hub/cyp-registry:v1.0.3`（注意：本仓库使用版本号标签，不使用 latest）
     - 支持 Docker Hub、GHCR、Quay.io 等公共仓库
   - **用户**（选填）：私有仓库的用户名（如果需要认证）
   - **密码**（选填）：私有仓库的密码或访问令牌
5. 点击 "确认" 开始拉取镜像

**API 调用示例：**
```bash
curl -X POST http://localhost:8080/api/v1/projects/{project_id}/images/add-from-url \
  -H "Authorization: Bearer <your-access-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "image": "nginx:latest",
    "username": "optional_username",
    "password": "optional_password"
  }'
```

**支持的镜像源：**
- Docker Hub：`docker.io/library/nginx:latest` 或 `nginx:latest`
- GitHub Container Registry：`ghcr.io/owner/repo:tag`
- Quay.io：`quay.io/namespace/repo:tag`
- 其他符合 OCI Distribution Specification 的仓库

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

# 标记镜像（使用版本号标签）
docker tag cyp-registry:single addss-hub/cyp-registry:v1.0.3
docker tag cyp-registry:single addss-hub/cyp-registry:v1.0.3-2026-02-28

# 推送镜像
docker push addss-hub/cyp-registry:v1.0.3
docker push addss-hub/cyp-registry:v1.0.3-2026-02-28
```

## 🧪 测试

```bash
# 运行后端测试
cd src && go test ./...

# 运行前端测试
cd web && npm run test

# 运行 E2E 测试（Cypress）
cd web && npm run test
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
