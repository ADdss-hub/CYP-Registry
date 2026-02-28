# 快速开始指南

本指南将帮助您快速上手 CYP-Registry 容器镜像仓库平台。

## 前置要求

在开始之前，请确保您已满足以下要求：

- Docker 20.10 或更高版本
- Docker Compose 2.0 或更高版本
- 有效的域名（可选，用于生产环境）

## 步骤 1: 安装与启动

### 方式一：单镜像模式（生产/离线/单机/开发统一推荐）

适用场景：**离线/单机/开发环境**，要求只运行一个容器（镜像内置 PostgreSQL + Redis + 应用），避免 `docker compose` 自动拉取 `postgres/redis/minio/prometheus...` 等多个镜像。

```bash
# 克隆项目
git clone https://github.com/cyp-registry/registry.git
cd registry

# 直接构建并启动单镜像容器
# 说明：
# - 如宿主机项目根目录不存在 .env，容器入口脚本会自动在宿主机生成 .env（含强随机 DB_PASSWORD/JWT_SECRET 等）
# - 如已存在 .env，则不会覆盖现有配置
# - 默认无需提供 config.yaml：容器启动时会自动生成 /app/config.yaml（提示日志仅首次显示一次）
# - 如需固定配置（推荐生产）：在宿主机准备 ./config.yaml，并在 docker-compose.single.yml 中启用对应 volume 挂载（只读）
docker compose -f docker-compose.single.yml up -d --build

# 查看状态
docker compose -f docker-compose.single.yml ps
```

可选：如需在**不启动容器**的情况下手动生成/更新 `.env`（例如本地开发、预设部分变量），可以使用项目自带的自动配置脚本：

```bash
# Windows（PowerShell）
scripts\auto-config.ps1

# Linux/macOS
bash scripts/auto-config.sh
```

说明：
- **数据持久化**：PostgreSQL/Redis/本地存储均使用命名卷（首次启动会自动初始化数据库并执行 `init-scripts/01-schema.sql`）。
- **默认关闭扫描器**：单镜像模式默认 `SCANNER_ENABLED=false`（避免额外依赖）。

### 方式二：从源码编译

```bash
# 克隆项目
git clone https://github.com/cyp-registry/registry.git
cd registry

# 编译后端
cd src && go build -o bin/registry ./cmd/server

# 编译前端
cd web && npm install && npm run build

# 启动服务
./bin/registry
```

## 步骤 2: 访问 Web 界面

1. 打开浏览器访问 http://localhost:3000
2. 您将看到登录页面

## 步骤 3: 创建账户

1. 点击「立即注册」链接
2. 填写注册信息：
   - 用户名（3-20个字符）
   - 邮箱（有效邮箱地址）
   - 密码（至少8位，包含数字和字母）
3. 点击「注册」按钮

## 步骤 4: 创建项目

登录后，按照以下步骤创建您的第一个项目：

### 通过 Web 界面

1. 在左侧导航栏点击「项目管理」
2. 点击「新建项目」按钮
3. 填写项目信息：
   - 项目名称（必填，英文或数字）
   - 项目描述（可选）
   - 可见性（公开/私有）
   - 存储配额（可选）
4. 点击「创建」按钮

### 通过 API

```bash
# 获取 Access Token
# 先登录获取 token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "your-username", "password": "your-password"}' | jq -r '.data.accessToken')

# 创建项目
curl -X POST http://localhost:8080/api/v1/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-first-project",
    "description": "我的第一个项目",
    "isPublic": false
  }'
```

## 步骤 5: 配置 Docker 客户端

### 配置 insecure registry（开发环境）

```bash
# 编辑 Docker daemon 配置
sudo vim /etc/docker/daemon.json
```

添加以下内容：
```json
{
  "insecure-registries": ["localhost:3000"]
}
```

```bash
# 重启 Docker
sudo systemctl restart docker
```

### 登录镜像仓库

```bash
# 登录到镜像仓库
docker login localhost:3000

# 输入用户名和密码
```

## 步骤 6: 推送和拉取镜像

### 推送镜像到仓库

```bash
# 1. 从官方镜像仓库拉取一个基础镜像
docker pull alpine:latest

# 2. 重命名镜像以匹配您的项目
docker tag alpine:latest localhost:3000/my-first-project/alpine:latest

# 3. 推送镜像到您的仓库
docker push localhost:3000/my-first-project/alpine:latest
```

### 从仓库拉取镜像

```bash
# 拉取镜像
docker pull localhost:3000/my-first-project/alpine:latest

# 运行容器
docker run -it --rm localhost:3000/my-first-project/alpine:latest sh
```

## 步骤 7: 使用漏洞扫描功能

CYP-Registry 内置了漏洞扫描功能，可以扫描容器镜像中的安全漏洞。

### 启动扫描

1. 进入「漏洞扫描」页面
2. 选择要扫描的项目和镜像
3. 点击「立即扫描」按钮

### 查看扫描结果

扫描完成后，您可以查看：
- 漏洞摘要（按严重程度分类）
- 详细漏洞列表（CVE 编号、修复建议）
- 扫描报告

## 步骤 8: 配置 Webhook

Webhook 允许您在特定事件发生时接收通知。

### 创建 Webhook

1. 进入「Webhook 管理」页面
2. 点击「创建 Webhook」按钮
3. 配置以下信息：
   - 名称：Webhook 的名称
   - URL：接收通知的端点
   - 事件类型：选择需要通知的事件（如镜像推送、扫描完成等）
   - 密钥：用于验证请求签名的密钥（可选）

### 支持的事件类型

| 事件类型 | 说明 |
|---------|------|
| image_push | 镜像推送完成 |
| image_pull | 镜像拉取完成 |
| scan_completed | 漏洞扫描完成 |
| scan_failed | 漏洞扫描失败 |
| webhook_test | Webhook 测试请求 |

## 步骤 9: 使用 API

### 认证

所有 API 请求都需要在 Header 中包含 Access Token：

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

#### 漏洞扫描
- `POST /api/v1/scans` - 启动扫描
- `GET /api/v1/scans/:id` - 获取扫描结果
- `GET /api/v1/scans` - 获取扫描历史

完整的 API 文档请访问 http://localhost:3000/docs

## 步骤 10: 个性化设置

### 主题设置

在「系统设置」→「外观设置」中，您可以：
- 选择浅色/深色/自动主题
- 设置界面语言
- 配置时区

### 通知设置

在「系统设置」→「通知设置」中，您可以：
- 启用/禁用邮件通知
- 配置扫描完成通知
- 配置安全警报
- 配置 Webhook 通知

### 访问令牌

在「系统设置」→「访问令牌」中，您可以：
- 创建 Personal Access Token (PAT)
- 设置令牌权限范围
- 设置令牌过期时间
- 撤销不再使用的令牌

## 常见问题

### Q: Docker 登录失败怎么办？

A: 请确保您已正确配置 insecure registry，并且 Docker 服务已重启。

### Q: 无法推送镜像怎么办？

A: 请检查：
1. 存储服务是否正常运行
2. 项目存储配额是否已满
3. 您是否有该项目的推送权限

### Q: 漏洞扫描很慢怎么办？

A: 首次扫描需要下载漏洞数据库，可能需要较长时间。建议保持服务运行以缓存数据。

### Q: 如何备份数据？

A: 参考部署文档中的「备份与恢复」章节。

## 下一步

- 阅读 [部署文档](./DEPLOYMENT.md) 了解生产环境部署
- 阅读 [运维手册](./OPERATIONS.md) 了解日常运维操作
- 访问 [API 文档](http://localhost:3000/docs) 了解更多 API 用法

## 获取帮助

如果您遇到问题，可以通过以下方式获取帮助：
- 提交 GitHub Issue
- 发送邮件至 nasDSSCYP@outlook.com

