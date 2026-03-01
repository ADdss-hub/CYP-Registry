# ARM设备支持说明

## 概述

CYP-Registry 现已全面支持 ARM 架构设备，包括 ARM64 (aarch64) 和 ARMv7。本文档说明如何在 ARM 设备上部署和使用 CYP-Registry。

## 支持的ARM架构

| 架构 | 支持状态 | 说明 |
|------|---------|------|
| **ARM64 (aarch64)** | ✅ 完全支持 | 推荐架构，所有功能完整支持 |
| **ARMv7** | ✅ 支持 | 需要自行构建，部分功能可能受限 |
| **ARMv6** | ❌ 不支持 | 性能不足，不支持 |

## ARM设备部署方式

### 方式1: 使用预构建镜像（推荐）

#### GitHub Container Registry (GHCR)

```bash
# 拉取ARM64镜像
docker pull ghcr.io/addss-hub/cyp-registry:latest --platform linux/arm64

# 运行容器
docker run -d \
  --name cyp-registry \
  --platform linux/arm64 \
  -p 8080:8080 \
  -v cyp-registry-pg-data:/var/lib/postgresql/data \
  -v cyp-registry-redis-data:/data/redis \
  -v cyp-registry-storage:/data/storage \
  -v cyp-registry-logs:/app/logs \
  ghcr.io/addss-hub/cyp-registry:latest
```

#### 指定架构标签

```bash
# ARM64架构
docker pull ghcr.io/addss-hub/cyp-registry:v1.0.8-linux-arm64

# AMD64架构
docker pull ghcr.io/addss-hub/cyp-registry:v1.0.8-linux-amd64
```

### 方式2: 从源码构建（ARM设备本地构建）

#### ARM64设备构建

```bash
# 克隆项目
git clone https://github.com/ADdss-hub/CYP-Registry.git
cd CYP-Registry

# 构建ARM64镜像
docker buildx build \
  --platform linux/arm64 \
  -f Dockerfile.single \
  -t cyp-registry:arm64 \
  --load \
  .

# 运行容器
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-registry-data:/data \
  cyp-registry:arm64
```

#### ARMv7设备构建

```bash
# 构建ARMv7镜像
docker buildx build \
  --platform linux/arm/v7 \
  -f Dockerfile.single \
  -t cyp-registry:armv7 \
  --load \
  .
```

### 方式3: 使用Docker Compose（ARM设备）

```yaml
version: '3.8'

services:
  cyp-registry:
    image: ghcr.io/addss-hub/cyp-registry:latest
    platform: linux/arm64  # 指定ARM64架构
    container_name: cyp-registry
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - APP_ENV=production
      - DB_PASSWORD=${DB_PASSWORD}
      - JWT_SECRET=${JWT_SECRET}
    volumes:
      - pg_data:/var/lib/postgresql/data
      - redis_data:/data/redis
      - storage_data:/data/storage
      - logs_data:/app/logs

volumes:
  pg_data:
  redis_data:
  storage_data:
  logs_data:
```

## ARM设备兼容性

### 操作系统支持

| 操作系统 | ARM64支持 | ARMv7支持 | 说明 |
|---------|----------|----------|------|
| **Raspberry Pi OS** | ✅ | ✅ | Raspberry Pi 4+ (ARM64), Pi 3 (ARMv7) |
| **Ubuntu ARM** | ✅ | ✅ | Ubuntu Server for ARM |
| **Debian ARM** | ✅ | ✅ | Debian for ARM |
| **Alpine Linux ARM** | ✅ | ✅ | Alpine Linux ARM版本 |
| **macOS (Apple Silicon)** | ✅ | ❌ | M1/M2/M3芯片 |

### 容器运行时支持

| 运行时 | ARM64支持 | ARMv7支持 | 说明 |
|--------|----------|----------|------|
| **Docker** | ✅ | ✅ | Docker Desktop for ARM |
| **Podman** | ✅ | ✅ | Podman for ARM |
| **containerd** | ✅ | ✅ | containerd for ARM |

### 依赖组件支持

| 组件 | ARM64支持 | ARMv7支持 | 说明 |
|------|----------|----------|------|
| **PostgreSQL 15** | ✅ | ✅ | Alpine包管理器提供ARM版本 |
| **Redis 7** | ✅ | ✅ | Alpine包管理器提供ARM版本 |
| **Go 1.24** | ✅ | ✅ | 官方支持ARM架构 |
| **Node.js 20** | ✅ | ✅ | 官方支持ARM架构 |

## 性能优化建议

### ARM设备资源要求

#### 最低配置（ARM64）

| 资源 | 要求 | 说明 |
|------|------|------|
| **CPU** | 4核心 | ARM64架构，推荐Cortex-A72或更高 |
| **内存** | 2GB | 推荐4GB+ |
| **磁盘** | 10GB | 根据镜像数量调整 |
| **网络** | 100Mbps | 局域网部署 |

#### 推荐配置（ARM64）

| 资源 | 要求 | 说明 |
|------|------|------|
| **CPU** | 8核心 | ARM64架构，推荐Cortex-A78或更高 |
| **内存** | 4GB+ | 推荐8GB |
| **磁盘** | 50GB+ | SSD推荐 |
| **网络** | 1Gbps | 生产环境推荐 |

### ARM设备优化配置

#### Docker配置优化

```json
{
  "builder": {
    "gc": {
      "enabled": true,
      "defaultKeepStorage": "20GB"
    }
  },
  "experimental": false,
  "features": {
    "buildkit": true
  }
}
```

#### 系统优化

```bash
# 增加文件描述符限制
echo "* soft nofile 65535" >> /etc/security/limits.conf
echo "* hard nofile 65535" >> /etc/security/limits.conf

# 优化内核参数（ARM设备）
echo "vm.swappiness=10" >> /etc/sysctl.conf
echo "vm.vfs_cache_pressure=50" >> /etc/sysctl.conf
```

## 常见ARM设备部署示例

### Raspberry Pi 4/5 (ARM64)

```bash
# 1. 更新系统
sudo apt update && sudo apt upgrade -y

# 2. 安装Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# 3. 拉取ARM64镜像
docker pull ghcr.io/addss-hub/cyp-registry:latest --platform linux/arm64

# 4. 运行容器
docker run -d \
  --name cyp-registry \
  --platform linux/arm64 \
  -p 8080:8080 \
  -v cyp-registry-data:/data \
  ghcr.io/addss-hub/cyp-registry:latest
```

### Apple Silicon (M1/M2/M3)

```bash
# 使用Docker Desktop for Mac（已内置ARM64支持）
docker pull ghcr.io/addss-hub/cyp-registry:latest

# 运行容器（自动使用ARM64架构）
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-registry-data:/data \
  ghcr.io/addss-hub/cyp-registry:latest
```

### 树莓派3 (ARMv7)

```bash
# 构建ARMv7镜像
docker buildx build \
  --platform linux/arm/v7 \
  -f Dockerfile.single \
  -t cyp-registry:armv7 \
  --load \
  .

# 运行容器
docker run -d \
  --name cyp-registry \
  -p 8080:8080 \
  -v cyp-registry-data:/data \
  cyp-registry:armv7
```

## CI/CD中的ARM支持

### 多架构构建

CI/CD流程已配置多架构构建，自动为以下架构构建镜像：

- `linux/amd64` - x86_64架构
- `linux/arm64` - ARM64架构

### ARM64测试

CI流程中包含ARM64架构的交叉编译测试，确保代码在ARM64架构上可以正常编译。

## 故障排查

### 常见问题

1. **镜像拉取失败**
   ```bash
   # 明确指定平台
   docker pull ghcr.io/addss-hub/cyp-registry:latest --platform linux/arm64
   ```

2. **构建失败**
   ```bash
   # 检查Docker Buildx是否启用
   docker buildx version
   
   # 创建buildx builder
   docker buildx create --name multiarch --use
   docker buildx inspect --bootstrap
   ```

3. **性能问题**
   ```bash
   # 检查容器资源使用
   docker stats cyp-registry
   
   # 检查系统负载
   top
   ```

## 总结

- ✅ **ARM64完全支持**: 所有功能完整支持，推荐使用
- ✅ **ARMv7支持**: 需要自行构建，部分功能可能受限
- ✅ **多架构构建**: CI/CD自动构建AMD64和ARM64镜像
- ✅ **性能优化**: 针对ARM设备提供优化建议
- ✅ **文档完善**: 提供详细的部署和使用说明

## 相关文档

- [系统平台环境架构完整文档](./系统平台环境架构完整文档.md) - CPU架构支持说明
- [CI工具检查报告](./CI工具检查报告.md) - CI/CD中的ARM支持
- [README.md](../README.md) - 快速开始指南
