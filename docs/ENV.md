## 环境变量（强制：任何环境/任何平台/任何系统可自动配置）

本项目提供跨平台自动配置能力：**首次启动即可自动在项目根目录生成 `.env`（全局配置中心）**，确保在 Windows/Linux/macOS、Docker/Podman 等环境下都能一键启动。

- Windows：`scripts/auto-config.ps1`
- Linux/macOS：`scripts/auto-config.sh`
- Docker 单镜像模式：`docker compose -f docker-compose.single.yml up -d --build` 时，由容器入口脚本自动执行 `scripts/auto-config.sh`，在宿主机项目根目录生成 `.env`

### 跨平台启动方式矩阵

| 平台 / 场景 | 第一步：生成配置（全局 `.env`） | 第二步：启动后端服务 | 第三步：启动前端 Web（可选） | 备注 |
| ----------- | -------------------------------- | --------------------- | ---------------------------- | ---- |
| **Windows（本机开发，PowerShell）** | 在仓库根目录执行：`.\scripts\auto-config.ps1` | 在仓库根目录执行：`go run .\cmd\server\main.go` | 在 `web` 目录执行：`npm install`（首次）+ `npm run dev` | 依赖已安装：Go、Node 20+；推荐使用 PowerShell 7+；浏览器访问 `http://localhost:3000`（前端）或 `http://localhost:8080`（后端） |
| **Linux/macOS（本机开发）** | 在仓库根目录执行：`./scripts/auto-config.sh` | 在仓库根目录执行：`go run ./cmd/server/main.go` | 在 `web` 目录执行：`npm install`（首次）+ `npm run dev` | 依赖已安装：Go、Node 20+；浏览器访问 `http://localhost:3000` / `http://localhost:8080` |
| **Docker 单镜像模式（推荐：Windows/macOS/Linux + Docker Desktop/Podman）** | 无需手动生成：`docker compose -f docker-compose.single.yml up -d --build` 时，由容器自动在宿主机项目根目录生成 `.env` | 同上：`docker compose -f docker-compose.single.yml up -d --build`（启动内置 Postgres + Redis + 后端） | 内置前端已在镜像构建时打包，直接访问 `http://localhost:8080` 即可 | 适合离线/单机/快速体验；所有配置通过宿主机根级 `.env` 控制，容器入口脚本自动加载 |
| **Linux 服务器 / 生产环境（推荐 Compose 部署）** | 手动准备 `.env`（参考 `env.production.example` 或本文件示例），放在部署目录根 | 使用生产 `docker-compose.yml` 启动：`docker compose up -d` | 同 Docker 单镜像模式，由容器内前端负责 | 建议使用固定版本镜像（如 `ghcr.io/addss-hub/cyp-registry:v1.0.7`），并在 `.env` 中显式设置强随机 `JWT_SECRET` / `DB_PASSWORD` 等 |

说明（重要约定）：
- 若 `.env` 已存在，脚本不会覆盖（可重复执行）。
- 自动生成的 `DB_PASSWORD` / `REDIS_PASSWORD` / `JWT_SECRET` 均为 32+ 字节强随机值，**可直接用于生产环境**；
- 如需与既有基础设施对接，仍可以按规范手动调整敏感值（如 `JWT_SECRET`、`DB_PASSWORD` 等）。
- 单镜像模式补充（生产可用）：当未显式提供 `DB_PASSWORD` / `JWT_SECRET` 时，容器会在**首次启动**自动生成并持久化到数据卷（后续重启不会改变，且“已自动生成”的提示日志不会重复刷屏）。

### 配置优先级（后端）

- **环境变量（来自根级 `.env` 或容器环境） > `config.yaml`（静态默认）**
- 后端加载 `config.yaml` 后，会用环境变量覆盖（`src/pkg/config/applyEnvOverrides`）。
- 单镜像模式补充：如未挂载/不存在 `/app/config.yaml`，容器入口脚本会在启动时自动生成一份（基于当前环境变量），并且**生成提示日志仅首次显示一次**。

### 前端派生规则

- 根级 `.env` 是唯一源头。
- 脚本会派生生成 `web/.env.local`（仅包含 `VITE_*`），前端只读取 `VITE_*` 变量，避免把敏感信息暴露到浏览器。

创建 `.env` 参考下面内容（示例；真实生产请使用 `env.production.example` 作为模板）：

```dotenv
APP_NAME=CYP-Registry
APP_ENV=production

API_BASE_URL=http://localhost:8080
WEB_BASE_URL=http://localhost:3000

DB_PASSWORD=registry_secret
REDIS_PASSWORD=
JWT_SECRET=please-change-me-in-production
STORAGE_TYPE=local
STORAGE_LOCAL_ROOT_PATH=/data/storage

# MinIO（当 STORAGE_TYPE=minio 时需要；支持 MINIO_* 或 STORAGE_MINIO_* 两套命名）
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=registry

# CORS（逗号分隔；用于覆盖 config.yaml 的 security.cors.allowed_origins）
CORS_ALLOWED_ORIGINS=http://localhost:3000
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin

# 服务器关闭与清理配置
# CLEANUP_ON_SHUTDOWN: 控制服务器关闭时是否清理所有数据
#   1 = 清理所有数据（删除模式）- 会永久删除所有用户数据、项目数据、镜像文件、缓存数据
#   0 或不设置 = 保留数据（停止模式）- 仅关闭服务，保留所有数据
# ⚠️ 警告：设置为 1 时，关闭服务器会永久删除所有数据，此操作不可恢复！
# 生产环境强烈建议设置为 0 或不设置，避免误操作导致数据丢失
CLEANUP_ON_SHUTDOWN=0
```

说明：
- `JWT_SECRET`：生产环境务必替换为强随机值（至少 32+ 字符）。
- `DB_PASSWORD`：会同时用于 `postgres` 容器初始化与 `core` 连接数据库。
- `REDIS_PASSWORD`：可为空；如在单镜像模式下设置了该值，入口脚本会自动为内置 Redis 启用 `requirepass`，确保配置一致。
- `MINIO_*`：仅当 `STORAGE_TYPE=minio` 时需要（后端同时兼容 `MINIO_*` 与 `STORAGE_MINIO_*`）。
- `CORS_ALLOWED_ORIGINS`：用于快速联调/部署时覆盖 CORS 白名单（逗号分隔）。
- `CLEANUP_ON_SHUTDOWN`：控制服务器关闭时是否清理所有数据。设置为 `1` 时，关闭服务器会永久删除所有数据（包括用户数据、项目数据、镜像文件、缓存数据），此操作不可恢复。生产环境强烈建议设置为 `0` 或不设置，避免误操作导致数据丢失。适用于测试环境重置、开发环境清理等场景。详细说明请参考 `docs/SHUTDOWN_CLEANUP.md`。

启动命令（生产）：

```bash
docker compose -f docker-compose.single.yml up -d --build
```

