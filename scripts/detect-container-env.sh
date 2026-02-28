#!/bin/bash
# ============================================
# 容器内环境自动检测脚本
# 遵循《全平台通用容器开发设计规范》2.2节和3.3节
# 在容器启动时自动执行
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# ============================================
# 为常用环境变量提供“检测阶段默认值”
# 说明：
# - 不会覆盖已经由外部 .env / 编排系统传入的值（:= 语法只在为空时赋值）
# - 这样即使在容器内手动执行本脚本，也能得到与正式启动时相同的默认配置
# - 默认值与 Docker 运行镜像的 /app/docker-entrypoint.sh / 单镜像入口脚本保持一致
# ============================================

: "${APP_NAME:=CYP-Registry}"
: "${APP_HOST:=0.0.0.0}"
: "${APP_PORT:=8080}"
: "${APP_ENV:=production}"

# 多容器部署场景：数据库/Redis 通常通过服务名访问（postgres/redis）
# 单机 All-in-One 镜像会在自己的入口脚本中将 DB_HOST/REDIS_HOST 设为 127.0.0.1
: "${DB_HOST:=postgres}"
: "${DB_PORT:=5432}"
: "${DB_USER:=registry}"
: "${DB_NAME:=registry_db}"
: "${DB_SSLMODE:=disable}"

: "${REDIS_HOST:=redis}"
: "${REDIS_PORT:=6379}"
: "${REDIS_DB:=0}"

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# ============================================
# 检测容器与全局配置中心环境
# ============================================
detect_container_env() {
    print_info "检测容器环境..."
    
    # 检测容器引擎
    if [ -f /.dockerenv ]; then
        CONTAINER_ENGINE="Docker"
    elif [ -f /proc/1/cgroup ] && grep -q "docker" /proc/1/cgroup 2>/dev/null; then
        CONTAINER_ENGINE="Docker"
    elif [ -f /proc/1/cgroup ] && grep -q "podman" /proc/1/cgroup 2>/dev/null; then
        CONTAINER_ENGINE="Podman"
    else
        CONTAINER_ENGINE="Unknown"
    fi
    
    print_success "容器引擎: $CONTAINER_ENGINE"
    
    # 检测容器ID
    if [ -f /etc/hostname ]; then
        CONTAINER_ID=$(cat /etc/hostname)
        print_success "容器ID: $CONTAINER_ID"
    fi
    
    # 检测容器镜像
    if [ -f /.dockerenv ]; then
        print_success "运行在容器环境中"
    fi
}

# ============================================
# 检测网络配置
# ============================================
detect_network() {
    print_info "检测网络配置..."
    
    # 检测容器IP
    if command -v hostname &> /dev/null; then
        HOSTNAME=$(hostname)
        print_success "主机名: $HOSTNAME"
    fi
    
    # 检测网络接口
    if command -v ip &> /dev/null; then
        IP_ADDRESS=$(ip route get 8.8.8.8 2>/dev/null | awk '{print $7}' | head -1 || echo "Unknown")
        print_success "IP地址: $IP_ADDRESS"
    elif command -v hostname &> /dev/null; then
        IP_ADDRESS=$(hostname -i 2>/dev/null | awk '{print $1}' || echo "Unknown")
        print_success "IP地址: $IP_ADDRESS"
    fi
}

# ============================================
# 检测环境变量配置
# ============================================
detect_env_vars() {
    print_info "检测环境变量配置..."
    
    REQUIRED_VARS=("APP_NAME" "APP_HOST" "APP_PORT")
    MISSING_VARS=()
    
    for var in "${REQUIRED_VARS[@]}"; do
        if [ -z "${!var}" ]; then
            MISSING_VARS+=("$var")
        else
            print_success "$var=${!var}"
        fi
    done
    
    if [ ${#MISSING_VARS[@]} -gt 0 ]; then
        print_warning "缺少环境变量: ${MISSING_VARS[*]}（请检查全局配置中心 .env 或部署配置）"
    fi
    
    # 检测数据库配置
    if [ -z "${DB_HOST}" ]; then
        print_warning "数据库配置未设置（DB_HOST 为空），请检查全局配置中心 .env 中的数据库相关配置"
    else
        print_success "数据库主机: $DB_HOST:${DB_PORT:-5432}"
    fi
    
    # 检测Redis配置
    if [ -z "${REDIS_HOST}" ]; then
        print_warning "Redis配置未设置（REDIS_HOST 为空），请检查全局配置中心 .env 中的 Redis 相关配置"
    else
        print_success "Redis主机: $REDIS_HOST:${REDIS_PORT:-6379}"
    fi
}

# ============================================
# 检测存储配置
# ============================================
detect_storage() {
    print_info "检测存储配置..."
    
    STORAGE_TYPE="${STORAGE_TYPE:-local}"
    print_success "存储类型: $STORAGE_TYPE"
    
    if [ "$STORAGE_TYPE" == "local" ]; then
        STORAGE_PATH="${STORAGE_LOCAL_ROOT_PATH:-/data/storage}"
        if [ -d "$STORAGE_PATH" ]; then
            if [ -r "$STORAGE_PATH" ] && [ -w "$STORAGE_PATH" ]; then
                print_success "存储路径可读写: $STORAGE_PATH"
            else
                print_error "存储路径权限不足: $STORAGE_PATH"
                return 1
            fi
        else
            print_warning "存储路径不存在: $STORAGE_PATH"
            if mkdir -p "$STORAGE_PATH" 2>/dev/null; then
                chmod 755 "$STORAGE_PATH"
                print_success "已创建存储路径: $STORAGE_PATH"
            else
                print_error "无法创建存储路径: $STORAGE_PATH"
                return 1
            fi
        fi
    elif [ "$STORAGE_TYPE" == "minio" ]; then
        print_success "MinIO端点: ${STORAGE_MINIO_ENDPOINT:-minio:9000}"
    fi
}

# ============================================
# 检测依赖服务连通性
# ============================================
detect_dependencies() {
    print_info "检测依赖服务连通性..."
    
    # 检测数据库
    if [ -n "${DB_HOST}" ]; then
        if command -v nc &> /dev/null; then
            if nc -z -w 3 "${DB_HOST}" "${DB_PORT:-5432}" 2>/dev/null; then
                print_success "数据库服务可达: ${DB_HOST}:${DB_PORT:-5432}"
            else
                print_warning "数据库服务不可达: ${DB_HOST}:${DB_PORT:-5432}"
            fi
        fi
    fi
    
    # 检测Redis（为避免单机 All-in-One 场景下的“刚启动就检测”误报，这里做短暂重试）
    if [ -n "${REDIS_HOST}" ]; then
        if command -v nc &> /dev/null; then
            local retries=5
            local ok=0
            local i
            for i in $(seq 1 "${retries}"); do
                if nc -z -w 2 "${REDIS_HOST}" "${REDIS_PORT:-6379}" 2>/dev/null; then
                    ok=1
                    break
                fi
                sleep 1
            done
            if [ "$ok" -eq 1 ]; then
                print_success "Redis服务可达: ${REDIS_HOST}:${REDIS_PORT:-6379}"
            else
                print_warning "Redis服务不可达: ${REDIS_HOST}:${REDIS_PORT:-6379}"
            fi
        fi
    fi
    
    # 检测MinIO
    if [ "${STORAGE_TYPE:-local}" == "minio" ] && [ -n "${STORAGE_MINIO_ENDPOINT}" ]; then
        MINIO_HOST=$(echo "${STORAGE_MINIO_ENDPOINT}" | cut -d: -f1)
        MINIO_PORT=$(echo "${STORAGE_MINIO_ENDPOINT}" | cut -d: -f2)
        if command -v nc &> /dev/null; then
            if nc -z -w 3 "$MINIO_HOST" "$MINIO_PORT" 2>/dev/null; then
                print_success "MinIO服务可达: ${STORAGE_MINIO_ENDPOINT}"
            else
                print_warning "MinIO服务不可达: ${STORAGE_MINIO_ENDPOINT}"
            fi
        fi
    fi
}

# ============================================
# 主函数
# ============================================
main() {
    echo "============================================"
    echo "  容器与全局配置中心环境自动检测"
    echo "============================================"
    echo ""
    
    detect_container_env
    detect_network
    detect_env_vars
    detect_storage
    detect_dependencies
    
    echo ""
    echo "============================================"
    print_success "环境检测完成"
    echo "============================================"
}

# 执行检测
main


