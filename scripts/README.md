# 环境检测和配置脚本

本目录包含遵循《全平台通用容器开发设计规范》的环境自动检测和配置脚本。

## 脚本说明

### 0. 全局配置中心与自动初始化（`.env` + 前端配置）

- 本项目将**仓库根目录下的 `.env` 视为全局配置中心唯一源头**，统一管理：
  - 应用基础配置（`APP_NAME`、`APP_HOST`、`APP_PORT`、`APP_ENV`）；
  - 后端 API 与前端 Web 入口（`API_BASE_URL`、`WEB_BASE_URL`）；
  - 数据库/Redis/存储敏感配置（如 `DB_PASSWORD`、`REDIS_PASSWORD`、`MINIO_*` 等）；
  - JWT 密钥与 CORS、Grafana 等通用配置。
- 前端项目不会直接手写环境变量，而是从全局 `.env` 派生生成 `web/.env.local`（仅包含 `VITE_*` 变量），确保**前后端、容器、数据库等配置完全由同一中心统一管理**。

**自动初始化行为（配置中心配置完成后的自动派生）**：

- 当执行 `scripts/auto-config.sh`（Linux/macOS）或 `scripts/auto-config.ps1`（Windows）且根目录不存在 `.env` 时：
  1. 自动生成根级 `.env`，完成**全局配置中心初始化**；
  2. 自动生成 `web/.env.local`，从根级 `.env` 中读取 `APP_NAME`、`APP_ENV`、`API_BASE_URL` 生成前端所需的 `VITE_*` 变量；
  3. 之后前端/后端/容器均应从上述文件读取配置，而不是分散硬编码。

### 1. detect-docker-env.sh
**Linux/macOS 环境检测脚本**

自动检测宿主机Docker环境，包括：
- 操作系统版本
- 硬件资源（CPU、内存）
- 容器环境类型
- Docker/Podman版本和状态
- 网络配置
- 存储路径

**使用方法：**
```bash
./scripts/detect-docker-env.sh
```

**输出：**
- 控制台输出检测结果摘要
- JSON格式检测报告：`/tmp/docker_env_detect_report.json`

### 2. detect-docker-env.ps1
**Windows 环境检测脚本**

功能同 `detect-docker-env.sh`，适用于Windows PowerShell环境。

**使用方法：**
```powershell
.\scripts\detect-docker-env.ps1
```

**输出：**
- 控制台输出检测结果摘要
- JSON格式检测报告：`%TEMP%\docker_env_detect_report.json`

### 3. pre-start-check.sh
**生产环境启动前自动检查脚本**

遵循规范2.2节要求，在容器启动前执行全量检查：
- 宿主机与容器网络连通性
- 数据库服务可用性
- 依赖服务（Redis、MinIO）运行状态
- 配置文件完整性与权限
- 存储目录可读写性
- 镜像版本一致性
- 资源配额检查

**自动修复功能：**
- 自动创建缺失的存储目录
- 自动修复配置文件权限

**使用方法：**
```bash
# 手动执行
./scripts/pre-start-check.sh

# 在容器启动时自动执行（已集成到Dockerfile）
```

**环境变量：**
- `DB_HOST`: 数据库主机（默认：postgres）
- `DB_PORT`: 数据库端口（默认：5432）
- `REDIS_HOST`: Redis主机（默认：redis）
- `REDIS_PORT`: Redis端口（默认：6379）
- `STORAGE_TYPE`: 存储类型（local/minio）
- `STORAGE_LOCAL_ROOT_PATH`: 本地存储路径
- `CONFIG_FILE`: 配置文件路径（默认：/app/config.yaml）

### 4. detect-container-env.sh
**容器内环境自动检测脚本**

在容器启动时自动检测：
- 容器引擎类型（Docker/Podman）
- 容器网络配置
- 环境变量配置
- 存储配置
- 依赖服务连通性

**使用方法：**
```bash
# 在容器内手动执行
/app/scripts/detect-container-env.sh

# 容器启动时自动执行（已集成到Dockerfile）
```

## 集成说明

### Dockerfile集成

启动脚本 `docker-entrypoint.sh` 已自动集成：
1. 容器启动时自动执行 `detect-container-env.sh`
2. 生产环境（`APP_ENV=production`）自动执行 `pre-start-check.sh`
3. 检查通过后启动应用

### docker-compose.single.yml（单镜像）集成

环境变量已配置，启动时会自动：
1. 执行环境检测
2. 执行启动前检查
3. 启动应用服务

## 使用示例

### 开发环境

```bash
# 1. 检测本地Docker环境
./scripts/detect-docker-env.sh

# 2. 启动服务（自动执行检测）
docker-compose up -d
```

### 生产环境

```bash
# 1. 检测生产环境
./scripts/detect-docker-env.sh

# 2. 手动执行启动前检查
./scripts/pre-start-check.sh

# 3. 启动服务（自动执行检测和检查）
APP_ENV=production docker-compose up -d
```

### Windows环境

```powershell
# 1. 检测Docker环境
.\scripts\detect-docker-env.ps1

# 2. 启动服务
docker-compose up -d
```

## 故障排查

### 检查失败

如果启动前检查失败：
1. 查看错误信息，定位失败项
2. 脚本会自动尝试修复部分问题
3. 手动修复后重新执行检查

### 常见问题

**Q: 数据库连接失败**
```bash
# 检查数据库服务是否运行
docker-compose ps postgres

# 检查数据库端口
docker-compose exec core nc -z postgres 5432
```

**Q: 存储目录权限不足**
```bash
# 检查目录权限
ls -la /data/storage

# 修复权限
chmod 755 /data/storage
```

**Q: 配置文件不存在**
```bash
# 检查配置文件
ls -la /app/config.yaml

# 从宿主机挂载
docker-compose up -d
```

## 规范遵循

所有脚本严格遵循《全平台通用容器开发设计规范》：
- **2.1节**：环境变量配置规范
- **2.2节**：生产环境启动前自动检查
- **3.3节**：跨平台环境自动配置

## 相关文档

- [全平台通用容器开发设计规范](../../规范文件/全平台通用容器开发设计规范.md)
- [部署文档](../../deploy/DEPLOYMENT.md)
- [运维手册](../../deploy/OPERATIONS.md)


