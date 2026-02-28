# 部署文档

CYP-Registry 容器镜像仓库平台的部署指南。

## 环境要求

### 基础环境
- **操作系统**: Linux (Ubuntu 20.04/22.04 LTS 推荐), CentOS 7+, 或 Windows Server 2019+
- **Docker**: 20.10+ 
- **Docker Compose**: 2.0+
- **Git**: 2.0+

### 硬件配置
| 组件 | 最低配置 | 推荐配置 |
|------|----------|----------|
| CPU | 2 Core | 4 Core+ |
| 内存 | 4 GB | 8 GB+ |
| 存储 | 50 GB | 500 GB+ SSD |
| 网络 | 100 Mbps | 1 Gbps |

### 依赖服务
- **PostgreSQL**: 13+ (用于存储元数据)
- **Redis**: 6+ (用于缓存和会话)
- **MinIO/S3**: 用于存储镜像层 (可选)

> 单机/离线场景可使用 **单镜像模式（All-in-One）**：一个容器内置 PostgreSQL + Redis + 应用，避免拉取多个外部服务镜像。该模式主要用于开发/演示/离线环境；生产环境建议仍采用“服务拆分”架构。

## 快速部署

### 1. 克隆项目
```bash
git clone https://github.com/cyp-registry/registry.git
cd registry
```

### 2. 配置环境变量
```bash
# 复制环境变量模板
cp .env.example .env

# 编辑配置
vim .env
```

`.env` 文件配置示例:
```env
# 应用配置
APP_NAME=CYP-Registry
APP_HOST=0.0.0.0
APP_PORT=8080
APP_DEBUG=false

# 数据库配置
DB_HOST=localhost
DB_PORT=5432
DB_USER=registry
DB_PASSWORD=your_secure_password
DB_NAME=registry_db
DB_SSLMODE=disable

# Redis配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT认证配置
JWT_SECRET=your-super-secret-jwt-key
JWT_EXPIRY_HOURS=24
REFRESH_TOKEN_EXPIRY_DAYS=7

# 存储配置 (MinIO/S3)
STORAGE_ENDPOINT=localhost:9000
STORAGE_ACCESS_KEY=minioadmin
STORAGE_SECRET_KEY=minioadmin
STORAGE_BUCKET=registry-storage
STORAGE_USE_SSL=false

# 安全配置
BCRYPT_COST=12
CORS_ALLOW_ORIGINS=http://localhost:3000
```

### 3. 使用 Docker Compose 启动（单镜像模式为主）

> 生产环境提示（单镜像模式）：
> - 若未显式提供 `DB_PASSWORD` / `JWT_SECRET`，单镜像容器会在**首次启动**自动生成强随机值并持久化到数据卷（后续重启不会改变，且“已自动生成”的提示日志不会重复刷屏）。
> - 需要查看当前自动生成的值时，可在容器内读取：
>   - `cat /var/lib/postgresql/data/.cyp_registry_db_password`
>   - `cat /var/lib/postgresql/data/.cyp_registry_jwt_secret`
> - 为便于审计/轮换，生产环境仍推荐通过 `.env` 或外部 Secret 显式注入上述敏感值。

#### 创建必要目录
```bash
# 创建数据目录
mkdir -p data/postgres data/redis data/minio

# 创建日志目录
mkdir -p logs
```

#### 启动服务
```bash
# 构建并启动单镜像容器（一个镜像=全部模块）
docker compose -f docker-compose.single.yml up -d --build

# 查看服务状态
docker compose -f docker-compose.single.yml ps

# 查看日志
docker compose -f docker-compose.single.yml logs -f
```

#### 单镜像模式（All-in-One，不拉取外部服务镜像）

```bash
# 构建并启动单镜像容器（一个容器内置 PostgreSQL + Redis + 应用）
docker compose -f docker-compose.single.yml up -d --build

# 查看状态
docker compose -f docker-compose.single.yml ps
```

### 4. 使用 Docker Desktop 图形界面（Compose 项目）部署（可选）

> 说明：本项目底层仍然使用 `docker compose`，如果你习惯通过 Docker Desktop / Rancher Desktop 等图形化工具来管理 Compose 项目，可以直接导入本仓库中的 `docker-compose.single.yml` 文件，而无需手写命令行。

以 Windows 版 Docker Desktop 为例，步骤如下：

1. 打开 Docker Desktop，在左侧导航中选择 **Compose**（或类似入口）。
2. 点击右上角 **「新建项目」** 按钮，弹出「创建项目」对话框。
3. **项目名称**：例如填写 `cyp-registry`，用于在 Docker Desktop 中标识该 Compose 项目。
4. **路径**：选择本仓库在宿主机上的目录，例如 `C:\path\to\registry`（需要能够访问到 `docker-compose.single.yml`）。
5. **来源**：
   - 选择「上传 docker-compose.yml」/「使用现有 docker-compose 文件」；
   - 选择仓库根目录下的 `docker-compose.single.yml` 作为 Compose 定义文件。
6. （可选）如需修改端口映射、数据卷路径或环境变量：
   - 可以在导入前直接编辑 `docker-compose.single.yml`；
   - 或者在 Docker Desktop 提供的 YAML 编辑器中进行修改，常见调整包括：
     - 修改 `ports` 中的宿主机端口（例如将 `3000:3000` 改为 `8080:3000`）；
     - 修改 `volumes` 挂载到宿主机的目录路径；
     - 按需覆盖环境变量（如数据库密码、JWT 密钥等敏感配置）。
7. 确认无误后点击 **「确认 / 创建」**，Docker Desktop 会在后台执行等价的：
   - `docker compose -f docker-compose.single.yml up -d`  
   并在图形界面中展示服务状态、日志和健康检查结果。

如你在其他项目中使用过类似的 Compose 定义（例如包含 `services`、`depends_on`、`healthcheck`、`volumes`、`networks` 等字段的 YAML，如 Halo、PostgreSQL 等组合服务），CYP-Registry 的单镜像 Compose 文件也可以以完全相同的方式被导入和管理。

### 5. 验证部署
```bash
# 健康检查
curl http://localhost:8080/health

# 预期响应
{
  "status": "healthy",
  "service": "CYP-Registry"
}
```

## 生产环境部署

### 使用 Systemd 服务

#### 创建服务文件
```bash
sudo vim /etc/systemd/system/registry.service
```

```ini
[Unit]
Description=CYP-Registry Container Registry
After=docker.service
Requires=docker.service

[Service]
Type=simple
User=registry
Group=registry
WorkingDirectory=/opt/registry
ExecStart=/usr/bin/docker compose -f /opt/registry/docker-compose.single.yml up
ExecStop=/usr/bin/docker compose -f /opt/registry/docker-compose.single.yml down
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

#### 启动服务
```bash
sudo systemctl daemon-reload
sudo systemctl enable registry
sudo systemctl start registry
sudo systemctl status registry
```

### Nginx 反向代理配置

```nginx
server {
    listen 80;
    server_name registry.example.com;

    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name registry.example.com;

    ssl_certificate /etc/ssl/certs/registry.example.com.crt;
    ssl_certificate_key /etc/ssl/private/registry.example.com.key;
    ssl_protocols TLSv1.2 TLSv1.3;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket 支持 (用于实时扫描状态)
    location /ws {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### SSL/TLS 证书配置

使用 Let's Encrypt 免费证书:
```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d registry.example.com
```

## Docker 镜像仓库配置

### 配置 Docker 信任仓库
```bash
# 编辑 Docker daemon 配置
sudo vim /etc/docker/daemon.json
```

```json
{
  "insecure-registries": ["registry.example.com"],
  "registry-mirrors": ["https://registry.example.com"]
}
```

```bash
# 重启 Docker
sudo systemctl restart docker
```

### 登录镜像仓库
```bash
docker login registry.example.com
```

### 推送镜像示例
```bash
# 打标签
docker tag myapp:latest registry.example.com/myproject/myapp:latest

# 推送
docker push registry.example.com/myproject/myapp:latest

# 拉取
docker pull registry.example.com/myproject/myapp:latest
```

## 高可用部署

### 集群架构
```
                    ┌─────────────────┐
                    │    Nginx LB     │
                    │   (Keepalived)  │
                    └────────┬────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  Registry-1   │    │  Registry-2   │    │  Registry-3   │
│  :8080        │    │  :8080        │    │  :8080        │
└───────┬───────┘    └───────┬───────┘    └───────┬───────┘
        │                    │                    │
        └────────────────────┼────────────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              ▼              ▼
    ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
    │  PostgreSQL     │ │     Redis       │ │     MinIO       │
    │  (主从复制)      │ │   (Sentinel)    │ │   (分布式)       │
    └─────────────────┘ └─────────────────┘ └─────────────────┘
```

### PostgreSQL 主从复制
```yaml
# docker-compose.replica.yml
version: '3.8'

services:
  postgres-primary:
    image: postgres:15
    environment:
      POSTGRES_USER: registry
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: registry_db
    volumes:
      - postgres_primary:/var/lib/postgresql/data
    command: >
      postgres
      -c synchronous_commit=remote_write
      -c synchronous_standby_names='*'
    networks:
      - registry-network

  postgres-replica:
    image: postgres:15
    environment:
      POSTGRES_USER: registry
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: registry_db
    volumes:
      - postgres_replica:/var/lib/postgresql/data
    command: >
      postgres
      -c primary_conninfo='host=postgres-primary port=5432 user=registry password=${DB_PASSWORD}'
      -c hot_standby=on
    depends_on:
      - postgres-primary
    networks:
      - registry-network

networks:
  registry-network:
    driver: bridge

volumes:
  postgres_primary:
  postgres_replica:
```

## 监控与告警

### Prometheus 配置
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'registry'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
```

### Grafana 仪表盘

导入预配置的 Grafana 仪表盘:
1. 访问 http://grafana.example.com
2. 进入 Dashboard → Import
3. 上传 `deploy/grafana/registry-dashboard.json`

### 告警规则示例
```yaml
groups:
  - name: registry-alerts
    rules:
      - alert: RegistryDown
        expr: up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Registry 服务已宕机"
          
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "错误率超过 5%"
```

## 备份与恢复

### 自动备份脚本
```bash
#!/bin/bash
# backup.sh

BACKUP_DIR="/opt/registry/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# 备份 PostgreSQL
docker exec registry-postgres-1 pg_dump -U registry registry_db > "$BACKUP_DIR/registry_$DATE.sql"

# 备份 Redis
docker exec registry-redis-1 redis-cli BGSAVE
docker cp registry-redis-1:/data/dump.rdb "$BACKUP_DIR/redis_$DATE.rdb"

# 保留最近 7 天的备份
find "$BACKUP_DIR" -name "*.sql" -mtime +7 -delete
find "$BACKUP_DIR" -name "*.rdb" -mtime +7 -delete

echo "Backup completed: $DATE"
```

### 定时任务
```bash
# 添加到 crontab
0 2 * * * /opt/registry/scripts/backup.sh >> /var/log/registry-backup.log 2>&1
```

### 恢复数据
```bash
# 恢复 PostgreSQL
docker exec -i registry-postgres-1 psql -U registry registry_db < backup_file.sql

# 恢复 Redis
docker cp backup.rdb registry-redis-1:/data/dump.rdb
docker restart registry-redis-1
```

## 故障排查

### 常见问题

#### 1. 服务无法启动
```bash
# 检查 Docker 状态
docker-compose logs app

# 检查端口占用
netstat -tlnp | grep 8080

# 检查配置文件
docker exec registry-app ls -la /app/config.yaml
```

#### 2. 数据库连接失败
```bash
# 测试数据库连接
docker exec registry-app ping postgres

# 检查数据库日志
docker logs registry-postgres-1
```

#### 3. 镜像上传失败
```bash
# 检查 MinIO 连接
docker exec registry-app mc alias set minio http://minio:9000 minioadmin minioadmin
docker exec registry-app mc ls minio

# 检查存储配额
docker exec registry-app df -h /storage
```

### 日志位置
```
/opt/registry/logs/
├── app.log          # 应用日志
├── access.log       # 访问日志
└── error.log        # 错误日志
```

## 升级指南

### 版本升级
```bash
# 拉取最新代码
git fetch origin
git checkout v1.1.0

# 更新依赖
docker compose -f docker-compose.single.yml down
docker compose -f docker-compose.single.yml up -d --build

# 运行数据库迁移
docker exec registry-app ./registry migrate
```

### 回滚
```bash
# 回滚到上一版本
git checkout v1.0.0
docker compose -f docker-compose.single.yml down
docker compose -f docker-compose.single.yml up -d --build
```

