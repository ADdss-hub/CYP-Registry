# 运维手册

CYP-Registry 容器镜像仓库平台的日常运维指南。

## 日常运维任务

### 1. 服务状态检查

#### 检查所有服务状态
```bash
# 使用 Docker Compose
docker-compose ps

# 检查健康端点
curl http://localhost:8080/health

# 检查进程状态
ps aux | grep registry
```

#### 检查资源使用情况
```bash
# CPU 和内存使用
docker stats

# 磁盘使用
df -h

# Docker 磁盘使用
docker system df
```

### 2. 日志管理

#### 查看应用日志
```bash
# 实时查看应用日志
docker-compose logs -f app

# 查看最近 100 行日志
docker-compose logs --tail 100 app

# 查看错误日志
docker-compose logs app 2>&1 | grep -i error
```

#### 日志轮转配置
创建 `/etc/logrotate.d/registry`:
```
/opt/registry/logs/*.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    create 0640 www-data www-data
    sharedscripts
    postrotate
        docker kill --signal=HUP registry-app-1 2>/dev/null || true
    endscript
}
```

### 3. 数据库维护

#### PostgreSQL 维护
```bash
# 连接数据库
docker exec -it registry-postgres-1 psql -U registry

# 分析表性能
docker exec registry-postgres-1 psql -U registry -c "ANALYZE;"

# 清理死元组
docker exec registry-postgres-1 psql -U registry -c "VACUUM ANALYZE;"

# 检查表大小
docker exec registry-postgres-1 psql -U registry -c "SELECT pg_size_pretty(pg_database_size('registry_db'));"

# 查看慢查询
docker exec registry-postgres-1 psql -U registry -c "SELECT query, call_time FROM pg_stat_statements ORDER BY call_time DESC LIMIT 10;"
```

#### Redis 维护
```bash
# 连接 Redis
docker exec -it registry-redis-1 redis-cli

# 查看内存使用
INFO memory

# 清除过期键
BGSAVE

# 查看键统计
INFO stats

# 清除所有键（慎用）
FLUSHALL
```

## 性能优化

### 1. 数据库优化

#### PostgreSQL 配置优化
编辑 `postgresql.conf`:
```ini
# 连接配置
max_connections = 200

# 内存配置
shared_buffers = 4GB
effective_cache_size = 12GB
work_mem = 64MB
maintenance_work_mem = 1GB

# 日志配置
log_min_duration_statement = 1000
log_lock_waits = on

# 性能优化
fsync = on
synchronous_commit = remote_write
```

#### 索引优化
```sql
-- 创建常用索引
CREATE INDEX CONCURRENTLY idx_projects_owner_id ON projects(owner_id);
CREATE INDEX CONCURRENTLY idx_images_project_id ON images(project_id);
CREATE INDEX CONCURRENTLY idx_scans_project_id ON scans(project_id);
CREATE INDEX CONCURRENTLY idx_webhooks_project_id ON webhooks(project_id);

-- 查询未使用的索引
SELECT indexrelname FROM pg_stat_user_indexes WHERE idx_scan = 0;
```

### 2. 缓存优化

#### Redis 缓存配置
```bash
# 编辑 redis.conf
maxmemory 2gb
maxmemory-policy allkeys-lru

# 启用 AOF 持久化
appendonly yes
appendfsync everysec
```

#### 应用层缓存策略
- 会话缓存：24 小时过期
- 项目列表缓存：5 分钟过期
- 扫描结果缓存：1 小时过期

### 3. 存储优化

#### MinIO 存储优化
```bash
# 启用对象锁定
mc anonymous set public myminio/public

# 配置生命周期策略
mc admin bucket quota myminio/registry-storage --hard 500GB

# 查看存储使用
mc admin info myminio
```

## 安全运维

### 1. 用户安全管理

#### 创建管理员账户
```bash
# 通过 API 创建管理员
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "SecurePassword123!",
    "isAdmin": true
  }'
```

#### 查看活跃用户
```bash
# 查看最近活跃用户
docker exec registry-postgres-1 psql -U registry -c "
SELECT username, email, last_login_at, is_active 
FROM users 
ORDER BY last_login_at DESC 
LIMIT 20;
"
```

#### 禁用用户
```bash
# 禁用用户
docker exec registry-postgres-1 psql -U registry -c "
UPDATE users SET is_active = false WHERE username = 'username';
"
```

### 2. 访问控制

#### 查看权限列表
```bash
docker exec registry-postgres-1 psql -U registry -c "
SELECT r.name as role, p.name as permission 
FROM roles r
JOIN role_permissions rp ON r.id = rp.role_id
JOIN permissions p ON rp.permission_id = p.id;
"
```

#### 审计日志查询
```bash
# 查看最近的认证日志
docker exec registry-postgres-1 psql -U registry -c "
SELECT user_id, action, ip_address, created_at 
FROM audit_logs 
ORDER BY created_at DESC 
LIMIT 100;
"
```

### 3. 安全加固

#### SSL/TLS 配置
```bash
# 生成自签名证书（开发环境）
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /opt/registry/ssl/server.key \
  -out /opt/registry/ssl/server.crt \
  -subj "/C=CN/ST=Beijing/L=Beijing/O=CYP-Registry/CN=registry.example.com"
```

#### 配置 HTTPS
```yaml
# docker-compose.single.yml（单镜像）
services:
  app:
    ports:
      - "8080:8080"
    volumes:
      - ./ssl:/opt/registry/ssl:ro
    environment:
      - SSL_ENABLED=true
      - SSL_CERT=/opt/registry/ssl/server.crt
      - SSL_KEY=/opt/registry/ssl/server.key
```

## 监控配置

### 1. Prometheus 监控

#### 配置服务发现
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'registry'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance
```

#### 关键指标
| 指标名称 | 说明 | 告警阈值 |
|---------|------|---------|
| http_requests_total | 请求总数 | - |
| http_request_duration_seconds | 请求延迟 | > 1s |
| registry_images_total | 镜像数量 | > 10000 |
| registry_storage_used_bytes | 存储使用 | > 80% |
| postgres_connections | 数据库连接 | > 180 |

### 2. Grafana 仪表盘

#### 导入仪表盘
1. 访问 Grafana → Dashboards → Import
2. 上传 `deploy/grafana/registry-dashboard.json`
3. 选择数据源

#### 推荐面板
- 服务健康状态
- 请求速率与延迟
- 错误率分布
- 存储使用趋势
- 数据库连接池

### 3. 告警配置

#### Prometheus 告警规则
```yaml
groups:
  - name: registry-alerts
    rules:
      - alert: RegistryHighMemory
        expr: container_memory_usage_bytes{container="registry-app"} / container_spec_memory_limit_bytes > 0.85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Registry 内存使用率过高"
          
      - alert: RegistryHighCPU
        expr: rate(container_cpu_usage_seconds_total{container="registry-app"}[5m]) > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Registry CPU 使用率过高"
          
      - alert: PostgresDown
        expr: up{job="postgres"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "PostgreSQL 服务不可用"
```

## 故障处理

### 1. 常见故障排查

#### 服务无法启动
```bash
# 检查端口占用
netstat -tlnp | grep 8080

# 检查配置文件语法
docker exec registry-app ./registry validate

# 查看详细错误
docker-compose logs --tail 200 app
```

#### 数据库连接失败
```bash
# 测试数据库连接
docker exec registry-app nc -zv postgres 5432

# 检查数据库状态
docker exec registry-postgres-1 pg_isready -U registry

# 检查连接数
docker exec registry-postgres-1 psql -U registry -c "SELECT count(*) FROM pg_stat_activity;"
```

#### 镜像上传失败
```bash
# 检查 MinIO 连接
docker exec registry-app mc alias set minio http://minio:9000 minioadmin minioadmin
docker exec registry-app mc admin info minio

# 检查存储配额
docker exec registry-app mc admin bucket quota myminio/registry-storage

# 检查磁盘空间
docker exec registry-app df -h /storage
```

### 2. 紧急恢复流程

#### 回滚到上一版本
```bash
# 停止当前服务
docker-compose down

# 回滚代码
git checkout v1.0.0

# 重启服务
docker-compose up -d

# 验证服务
curl http://localhost:8080/health
```

#### 从备份恢复
```bash
# 停止服务
docker-compose down

# 恢复 PostgreSQL
docker exec -i registry-postgres-1 psql -U registry registry_db < backup_20240115.sql

# 恢复 Redis
docker cp backup.rdb registry-redis-1:/data/dump.rdb

# 启动服务
docker-compose up -d
```

#### 紧急停机
```bash
# 优雅停止所有服务
docker-compose down

# 如果无法优雅停止，强制停止
docker-compose kill
docker-compose down -v
```

## 容量规划

### 1. 存储容量计算

| 资源 | 计算公式 | 示例 |
|------|---------|------|
| 元数据存储 | 50KB × 用户数 + 100KB × 项目数 | 1000 用户 + 100 项目 = 150MB |
| 镜像存储 | 平均镜像大小 × 镜像数量 × 冗余系数 | 500MB × 10000 × 1.5 = 7.5TB |
| 日志存储 | 100MB/天 × 保留天数 | 100MB × 30 = 3GB |
| 备份存储 | 每周全量 × 保留周数 | 500MB × 4 = 2GB |

### 2. 性能容量规划

| 用户规模 | CPU | 内存 | 数据库连接 |
|---------|-----|------|-----------|
| < 100 用户 | 2 Core | 4 GB | 50 |
| 100-1000 用户 | 4 Core | 8 GB | 100 |
| > 1000 用户 | 8 Core | 16 GB | 200 |

## 运维检查清单

### 每日检查
- [ ] 检查所有服务状态
- [ ] 查看错误日志
- [ ] 检查磁盘空间
- [ ] 验证备份完成

### 每周检查
- [ ] 分析慢查询
- [ ] 检查用户活跃度
- [ ] 查看存储使用趋势
- [ ] 测试告警通道

### 每月检查
- [ ] 审查安全日志
- [ ] 清理过期数据
- [ ] 更新证书
- [ ] 容量规划评估

## 联系方式

- **技术支持**: nasDSSCYP@outlook.com
- **紧急联系**: 请提交 GitHub Issue
- **文档更新**: 欢迎提交 Pull Request

