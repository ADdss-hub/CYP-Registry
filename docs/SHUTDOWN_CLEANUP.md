# 服务器关闭与数据清理说明

## 概述

CYP-Registry 服务器支持两种关闭模式：
1. **停止模式（Stop）**：正常关闭服务器，保留所有数据
2. **删除模式（Cleanup）**：关闭服务器并清理所有数据

## 停止模式（默认）

### 行为
- 关闭 HTTP 服务器
- 停止扫描器服务
- 关闭数据库连接
- 关闭缓存连接
- **保留所有数据**（数据库、文件、缓存）

### 使用方法

#### Linux/macOS
```bash
# 发送 SIGINT 信号（Ctrl+C）
kill -INT <pid>

# 或发送 SIGTERM 信号
kill -TERM <pid>
```

#### Windows
```powershell
# 在运行服务器的终端按 Ctrl+C
```

#### Docker
```bash
# 停止容器（保留数据卷）
docker stop <container_name>

# 或使用 docker-compose
docker-compose stop
```

## 删除模式（清理所有数据）

### 行为
- 执行停止模式的所有操作
- **清理数据库数据**（删除所有表的数据，保留表结构）
- **清理文件存储**（删除所有镜像文件）
- **清理上传文件**（删除所有头像等上传文件）
- **清理缓存数据**（清空 Redis 缓存）

### ⚠️ 警告
删除模式会**永久删除所有数据**，包括：
- 所有用户数据
- 所有项目数据
- 所有镜像文件
- 所有缓存数据
- 所有上传文件

**此操作不可恢复！**

### 使用方法

#### 方法一：全局配置中心（推荐）

**`CLEANUP_ON_SHUTDOWN` 由全局配置中心（根级 `.env` 文件）统一控制。**

在根级 `.env` 文件中设置：

```bash
# 服务器关闭时是否清理所有数据
# 1 = 清理所有数据（删除模式）
# 0 或不设置 = 保留数据（停止模式）
CLEANUP_ON_SHUTDOWN=1
```

**说明**：
- 容器启动时，`single-entrypoint.sh` 会自动从 `.env` 文件加载此环境变量
- 服务器启动时会自动检测并显示当前配置状态
- 修改配置后需要重启容器才能生效
- 这是推荐的配置方式，符合全局配置中心规范

#### 方法二：环境变量（临时设置）

如果需要临时设置（不推荐用于生产环境）：

```bash
# Linux/macOS
export CLEANUP_ON_SHUTDOWN=1
./registry-server

# 或在启动命令中设置
CLEANUP_ON_SHUTDOWN=1 ./registry-server
```

```powershell
# Windows PowerShell
$env:CLEANUP_ON_SHUTDOWN="1"
.\registry-server.exe

# Windows CMD
set CLEANUP_ON_SHUTDOWN=1
registry-server.exe
```

```bash
# Docker（临时设置）
docker run -e CLEANUP_ON_SHUTDOWN=1 <image_name>

# docker-compose（临时设置）
# 在 docker-compose.yml 中添加：
# environment:
#   - CLEANUP_ON_SHUTDOWN=1
```

#### 方法二：在容器启动脚本中设置

修改启动脚本，在关闭时自动清理：

```bash
#!/bin/bash
export CLEANUP_ON_SHUTDOWN=1
exec ./registry-server
```

## 关闭流程说明

### 停止模式流程
1. 接收停止信号（SIGINT/SIGTERM）
2. 停止扫描器服务
3. 优雅关闭 HTTP 服务器（等待正在处理的请求完成，最多 10 秒）
4. 关闭缓存连接
5. 关闭数据库连接
6. 完成关闭，**数据保留**

### 删除模式流程
1. 接收停止信号（SIGINT/SIGTERM，且 CLEANUP_ON_SHUTDOWN=1）
2. 停止扫描器服务
3. 优雅关闭 HTTP 服务器
4. **清理数据库数据**（TRUNCATE 所有表）
5. **清理文件存储**（删除所有存储文件）
6. **清理上传文件**（删除上传目录）
7. **清理缓存数据**（清空 Redis）
8. 关闭缓存连接
9. 关闭数据库连接
10. 完成关闭，**所有数据已清理**

## 使用场景

### 停止模式适用于
- 正常维护和重启
- 更新服务器版本
- 临时停止服务
- 生产环境日常操作

### 删除模式适用于
- 测试环境重置
- 开发环境清理
- 容器镜像重建前的清理
- 完全重置系统

## 示例

### 示例 1：正常停止（保留数据）
```bash
# 启动服务器
./registry-server

# 在另一个终端停止（保留数据）
kill -TERM $(pgrep registry-server)
```

### 示例 2：停止并清理数据
```bash
# 启动服务器（设置清理标志）
CLEANUP_ON_SHUTDOWN=1 ./registry-server

# 在另一个终端停止（会清理所有数据）
kill -TERM $(pgrep registry-server)
```

### 示例 3：Docker 容器停止并清理
```bash
# 启动容器（设置清理标志）
docker run -d \
  -e CLEANUP_ON_SHUTDOWN=1 \
  -v /data/storage:/data/storage \
  -v /data/db:/var/lib/postgresql/data \
  registry-server:latest

# 停止容器（会清理所有数据）
docker stop <container_id>
```

## 注意事项

1. **数据备份**：在执行删除模式前，请确保已备份重要数据
2. **生产环境**：生产环境**强烈建议**不要使用删除模式
3. **容器卷**：如果使用 Docker 卷挂载，删除模式不会删除卷中的数据文件，只会清理数据库记录
4. **权限**：确保服务器有权限删除存储目录和上传目录
5. **时间**：清理大量数据可能需要较长时间，请耐心等待

## 故障排查

### 清理失败
如果清理过程中出现错误，服务器会记录警告日志但继续执行后续步骤。检查日志以了解具体失败原因。

### 数据未清理
- 检查环境变量是否正确设置
- 检查日志确认是否进入清理流程
- 检查文件系统权限
- 检查数据库连接是否正常

## 相关配置

### 全局配置中心控制

- **`CLEANUP_ON_SHUTDOWN`**：控制关闭时是否清理数据（由全局配置中心 `.env` 文件控制）
  - `1` = 清理所有数据（删除模式）
  - `0` 或不设置 = 保留数据（停止模式）
  - 配置位置：根级 `.env` 文件
  - 服务器启动时会自动检测并显示当前配置状态

### 其他配置

- `UPLOADS_DIR`：上传文件目录（用于清理上传文件）
- 存储配置：文件存储路径（用于清理存储文件）
