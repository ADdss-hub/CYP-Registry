# CYP-Registry

<div align="center">

![Version](https://img.shields.io/badge/version-1.0.8-blue.svg)
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
- **多种认证方式**：账号密码、Personal Access Token (PAT)、JWT Token
- **基于 RBAC 的细粒度权限控制**：支持角色和权限的灵活配置
- **项目级别权限**：支持项目级别的公开/私有设置
- **Token 管理**：JWT Token 自动刷新机制，PAT 支持自定义有效期
- **用户管理**：用户注册、登录、个人信息管理、头像上传
- **通知设置**：用户可自定义通知偏好设置

### 📦 镜像仓库管理
- **Docker Registry API v2 兼容**：严格遵循 OCI Distribution Specification
- **镜像操作**：镜像推送、拉取、删除、标签管理
- **镜像导入功能**：支持从 Docker Hub、GHCR、Quay.io 等公共仓库拉取镜像到私有仓库
  - 异步导入任务，支持任务状态查询和进度跟踪
  - 支持私有仓库认证（用户名/密码）
- **存储管理**：
  - 支持本地文件系统存储
  - 支持 MinIO 对象存储
  - 存储配额管理和使用量统计
- **自动项目创建**：推送镜像时自动创建项目（如果不存在）

### 🔔 Webhook 集成
- **多种事件类型**：镜像推送、拉取、删除等事件
- **自定义配置**：支持自定义 Webhook URL 和 HMAC 签名密钥
- **实时通知**：异步发送事件通知，支持重试机制
- **发送记录**：记录 Webhook 发送历史，便于排查问题

### 🛡️ 安全与审计
- **审计日志**：记录所有关键操作（用户操作、镜像操作等）
- **日志清理**：支持自动清理过期审计日志
- **安全配置**：
  - 速率限制（Rate Limiting）
  - 暴力破解防护（Brute Force Protection）
  - CORS 配置
  - 安全响应头
- **服务器关闭清理**：支持配置服务器关闭时是否清理所有数据（适用于测试环境）

### 🎨 Web 管理界面
- **现代化前端**：Vue 3 + TypeScript + Vite
- **响应式设计**：支持桌面端和移动端访问
- **主题切换**：支持深色/浅色主题切换
- **实时数据**：实时数据展示和操作反馈
- **完整功能**：项目管理、镜像管理、Webhook 管理、用户设置等

### 📊 监控与管理
- **健康检查**：内置健康检查端点（`/health`）
- **API 文档**：集成 Swagger UI，完整的 API 文档
- **统计信息**：项目统计、存储使用统计
- **管理员功能**：审计日志查询、用户管理

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
- ✅ AMD64/x86_64（默认，提供预构建镜像）
- ✅ ARM64（完全支持，提供预构建镜像，推荐用于ARM设备）
- ✅ ARMv7（支持，需自行构建）

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

#### 使用 Docker Desktop / 图形界面导入（可选）

如果你习惯通过 **Docker Desktop**（或其他支持 Compose 的图形化工具）来管理容器，可以直接导入本仓库的 `docker-compose.single.yml` 文件：

1. 打开 Docker Desktop，在左侧导航中选择 `Compose`（或类似入口）。
2. 点击右上角「新建项目」，在弹出的对话框中填写：
   - **项目名称**：例如 `cyp-registry`；
   - **路径**：选择本项目在宿主机上的目录（能访问到 `docker-compose.single.yml`）；
   - **来源**：选择「使用/上传 docker-compose.yml」，并选中 `docker-compose.single.yml`。
3. 如需调整端口、数据卷路径或环境变量，可以：
   - 在导入前直接编辑 `docker-compose.single.yml`；或
   - 在 Docker Desktop 提供的 YAML 编辑器中按需修改（例如更改 `8080:8080` 为其他宿主机端口）。
4. 确认无误后点击「确认/创建」，Docker Desktop 会在后台执行等价的：
   - `docker compose -f docker-compose.single.yml up -d --build`
5. 后续即可在 Docker Desktop 中通过图形界面查看容器状态、日志以及健康检查结果。

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
  ghcr.io/ADdss-hub/CYP-Registry:v1.1.0
```

### 方式二：使用预构建镜像

#### 从 GitHub Container Registry (GHCR) 拉取

```bash
# 拉取指定版本（推荐生产环境）
docker pull ghcr.io/ADdss-hub/CYP-Registry:v1.1.0

# 运行容器（单镜像模式）
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-registry-pg-data:/var/lib/postgresql/data \
  -v cyp-registry-redis-data:/data/redis \
  -v cyp-registry-storage:/data/storage \
  -v cyp-registry-logs:/app/logs \
  ghcr.io/ADdss-hub/CYP-Registry:v1.1.0
```

#### 从 Docker Hub 拉取（如果已同步）

```bash
# 拉取指定版本
docker pull ghcr.io/ADdss-hub/CYP-Registry:v1.1.0

# 运行容器
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-registry-pg-data:/var/lib/postgresql/data \
  -v cyp-registry-redis-data:/data/redis \
  -v cyp-registry-storage:/data/storage \
  -v cyp-registry-logs:/app/logs \
  ghcr.io/ADdss-hub/CYP-Registry:v1.1.0
```

**镜像版本说明：**
- `v1.1.0`：当前标准版本号（语义化版本，推荐使用）
- `latest`：main分支最新版本（仅GHCR自动构建）
- **注意**：镜像仓库使用语义化版本号标签，推荐使用类似 `v1.1.0` 这种纯语义化版本标签拉取镜像。main分支会自动构建并推送 `latest` 标签。

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
    image: ghcr.io/addss-hub/cyp-registry:v1.1.0
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
docker pull ghcr.io/addss-hub/cyp-registry:v1.0.7

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
  ghcr.io/ADdss-hub/CYP-Registry:v1.1.0

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
#   ghcr.io/ADdss-hub/CYP-Registry:v1.1.0
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

### 核心文档
- [系统平台环境架构完整文档](./docs/系统平台环境架构完整文档.md) - **全面深度化的系统平台、环境、架构、兼容、配置、权限、清理等完整说明**
- [系统平台环境架构快速参考](./docs/系统平台环境架构完整文档-补充.md) - 快速参考和常用命令
- [环境变量配置](./docs/ENV.md) - 完整的配置说明
- [API 文档](./docs/api/API.md) - RESTful API 接口文档

### 功能文档
- [权限系统完整文档](./docs/权限系统完整文档.md) - 权限系统详细说明
- [镜像导入功能完成报告](./docs/镜像导入功能完成报告.md) - 镜像导入功能说明
- [Docker操作日志检查报告](./docs/Docker操作日志检查报告.md) - Docker操作日志说明
- [日志清理机制说明](./docs/日志清理机制说明.md) - 日志清理机制说明
- [服务器关闭清理说明](./docs/SHUTDOWN_CLEANUP.md) - 服务器关闭时的数据清理机制
- [PAT 使用示例](./docs/PAT_使用示例.md) - Personal Access Token 使用指南
- [PAT 权限范围规范](./docs/PAT_SCOPES_规范.md) - PAT 权限范围说明


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
- 若你未显式提供 `DB_PASSWORD` / `JWT_SECRET`，单镜像容器会在首次启动时自动生成强随机值并持久化到数据卷（后续重启不会改变，也不会重复打印"已自动生成"的提示日志）。
- 需要查看当前自动生成的值时，可在容器内读取：
  - `cat /var/lib/postgresql/data/.cyp_registry_db_password`
  - `cat /var/lib/postgresql/data/.cyp_registry_jwt_secret`

**服务器关闭清理配置：**
- `CLEANUP_ON_SHUTDOWN`：控制服务器关闭时是否清理所有数据
  - `1`：清理所有数据（删除模式）- 会永久删除所有用户数据、项目数据、镜像文件、缓存数据
  - `0` 或不设置：保留数据（停止模式）- 仅关闭服务，保留所有数据
  - ⚠️ **警告**：设置为 `1` 时，关闭服务器会永久删除所有数据，此操作不可恢复！
  - 生产环境强烈建议设置为 `0` 或不设置，避免误操作导致数据丢失
  - 详细说明请参考 [SHUTDOWN_CLEANUP.md](./docs/SHUTDOWN_CLEANUP.md)

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
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/refresh` - 刷新 Token
- `POST /api/v1/auth/logout` - 退出登录
- `GET /api/v1/auth/default-admin-once` - 获取默认管理员提示（首次启动）

#### 用户管理
- `GET /api/v1/users/me` - 获取当前用户信息
- `PUT /api/v1/users/me` - 更新当前用户信息
- `PUT /api/v1/users/me/password` - 修改密码
- `POST /api/v1/users/me/avatar` - 上传头像
- `GET /api/v1/users/me/token-info` - 获取当前 Token 信息
- `GET /api/v1/users/me/notification-settings` - 获取通知设置
- `PUT /api/v1/users/me/notification-settings` - 更新通知设置
- `POST /api/v1/users/me/pat` - 创建 Personal Access Token
- `GET /api/v1/users/me/pat` - 列出所有 PAT
- `DELETE /api/v1/users/me/pat/:id` - 撤销 PAT

#### 项目管理
- `GET /api/v1/projects` - 获取项目列表
- `POST /api/v1/projects` - 创建项目
- `GET /api/v1/projects/statistics` - 获取项目统计信息
- `GET /api/v1/projects/:id` - 获取项目详情
- `PUT /api/v1/projects/:id` - 更新项目
- `PATCH /api/v1/projects/:id` - 更新项目（兼容）
- `DELETE /api/v1/projects/:id` - 删除项目
- `PUT /api/v1/projects/:id/quota` - 更新存储配额
- `GET /api/v1/projects/:id/storage` - 获取存储使用情况

#### 镜像管理
- `POST /api/v1/projects/:id/images/import` - 从 URL 导入镜像
- `GET /api/v1/projects/:id/images/import` - 获取导入任务列表
- `GET /api/v1/projects/:id/images/import/:task_id` - 获取导入任务详情

#### Webhook 管理
- `GET /api/v1/webhooks` - 获取 Webhook 列表
- `POST /api/v1/webhooks` - 创建 Webhook
- `GET /api/v1/webhooks/:id` - 获取 Webhook 详情
- `PUT /api/v1/webhooks/:id` - 更新 Webhook
- `DELETE /api/v1/webhooks/:id` - 删除 Webhook
- `POST /api/v1/webhooks/:id/test` - 测试 Webhook
- `GET /api/v1/webhooks/:id/deliveries` - 获取 Webhook 发送记录

#### 管理员功能
- `GET /api/v1/admin/logs` - 获取审计日志（需要管理员权限）

#### Docker Registry API v2
- `GET /v2/` - API 版本检查
- `GET /v2/:name/tags/list` - 列出镜像标签
- `GET /v2/:name/manifests/:ref` - 获取镜像清单
- `PUT /v2/:name/manifests/:ref` - 推送镜像清单
- `GET /v2/:name/blobs/:digest` - 拉取镜像层
- `POST /v2/:name/blobs/uploads/` - 开始上传 Blob
- `PATCH /v2/:name/blobs/uploads/:uuid` - 上传 Blob 块
- `PUT /v2/:name/blobs/uploads/:uuid` - 完成 Blob 上传
- `DELETE /v2/:name/manifests/:ref` - 删除镜像

**镜像导入功能说明：**

通过 Web 界面或 API 可以从公共镜像仓库（如 Docker Hub、GHCR）拉取镜像到私有仓库：

**Web 界面操作：**
1. 进入项目 → 镜像管理页面
2. 点击 "+ 添加镜像" 或 "导入镜像" 按钮
3. 填写镜像信息：
   - **镜像**（必填）：输入镜像名称或完整 URL
     - 示例：`nginx:latest`、`ghcr.io/addss-hub/cyp-registry:v1.1.0`
     - 支持 Docker Hub、GHCR、Quay.io 等公共仓库
   - **用户**（选填）：私有仓库的用户名（如果需要认证）
   - **密码**（选填）：私有仓库的密码或访问令牌
4. 点击 "确认" 开始导入镜像
5. 可以在任务列表中查看导入进度和状态

**API 调用示例：**
```bash
# 创建导入任务
curl -X POST http://localhost:8080/api/v1/projects/{project_id}/images/import \
  -H "Authorization: Bearer <your-access-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "image": "nginx:latest",
    "username": "optional_username",
    "password": "optional_password"
  }'

# 查询导入任务列表
curl -X GET http://localhost:8080/api/v1/projects/{project_id}/images/import \
  -H "Authorization: Bearer <your-access-token>"

# 查询特定任务详情
curl -X GET http://localhost:8080/api/v1/projects/{project_id}/images/import/{task_id} \
  -H "Authorization: Bearer <your-access-token>"
```

**支持的镜像源：**
- Docker Hub：`docker.io/library/nginx:latest` 或 `nginx:latest`
- GitHub Container Registry：`ghcr.io/owner/repo:tag`
- Quay.io：`quay.io/namespace/repo:tag`
- 其他符合 OCI Distribution Specification 的仓库

**功能特点：**
- ✅ 异步导入，不阻塞其他操作
- ✅ 支持任务状态查询和进度跟踪
- ✅ 支持私有仓库认证
- ✅ 自动创建项目（如果推送镜像时项目不存在）

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
docker tag cyp-registry:single ghcr.io/ADdss-hub/CYP-Registry:v1.1.0

# 推送镜像
docker push ghcr.io/ADdss-hub/CYP-Registry:v1.1.0
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
- **构建工具**: Vite 5
- **状态管理**: Pinia
- **路由**: Vue Router 4
- **HTTP 客户端**: Axios
- **UI 组件**: 自定义组件库
- **工具库**: VueUse、Day.js、Lodash
- **国际化**: Vue I18n
- **测试**: Cypress

### 基础设施
- **容器化**: Docker + Docker Compose
- **存储**: 本地文件系统 / MinIO 对象存储
- **监控**: Prometheus + Grafana（可选）
- **日志**: JSON 格式日志，支持文件输出和轮转
- **健康检查**: 内置健康检查端点

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
