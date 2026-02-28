#!/bin/bash
# ============================================
# 生产环境启动前自动检查脚本
# 遵循《全平台通用容器开发设计规范》2.2节
# 使用方法: ./scripts/pre-start-check.sh
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================
# 公共辅助：生成随机密钥（与单镜像入口脚本保持一致）
# ============================================
gen_random_hex() {
    # 32 bytes -> 64 hex chars
    if command -v openssl >/dev/null 2>&1; then
        openssl rand -hex 32 2>/dev/null || true
    elif [ -r /dev/urandom ] && command -v od >/dev/null 2>&1; then
        od -An -N32 -tx1 /dev/urandom 2>/dev/null | tr -d ' \n' || true
    elif command -v date >/dev/null 2>&1 && command -v sha256sum >/dev/null 2>&1; then
        date +%s 2>/dev/null | sha256sum 2>/dev/null | awk '{print $1}' || true
    elif command -v date >/dev/null 2>&1 && command -v shasum >/dev/null 2>&1; then
        date +%s 2>/dev/null | shasum -a 256 2>/dev/null | awk '{print $1}' || true
    else
        printf '%s' "$(date +%s 2>/dev/null || echo 0)"
    fi
}

# 输出函数
print_step() {
    echo -e "${BLUE}🔍 $1${NC}"
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

# 检查结果
CHECKS_PASSED=0
CHECKS_FAILED=0
CHECKS_WARNINGS=0

# ============================================
# 检查函数
# ============================================
check_pass() {
    print_success "$1"
    ((CHECKS_PASSED++))
}

check_fail() {
    print_error "$1"
    ((CHECKS_FAILED++))
    return 1
}

check_warn() {
    print_warning "$1"
    ((CHECKS_WARNINGS++))
}

# ============================================
# 1. 宿主机与容器网络连通性检查
# ============================================
check_network_connectivity() {
    print_step "检查网络连通性..."
    
    # 检查本地回环（跨平台兼容）
    # Linux/macOS: ping -c
    # Windows (Git Bash): ping -n
    # Alpine/BusyBox: ping -c
    if ping -c 1 127.0.0.1 &> /dev/null 2>&1 || ping -n 1 127.0.0.1 &> /dev/null 2>&1; then
        check_pass "本地回环网络正常"
    else
        check_warn "本地回环网络检测失败（某些环境可能不支持 ping）"
    fi
    
    # 检查外部网络（如果不在容器内）
    # 容器内通常不需要外部网络（单镜像模式）
    if [ ! -f /.dockerenv ] && ([ ! -f /proc/1/cgroup ] || ! grep -q "docker\|podman" /proc/1/cgroup 2>/dev/null); then
        if ping -c 1 8.8.8.8 &> /dev/null 2>&1 || ping -n 1 8.8.8.8 &> /dev/null 2>&1; then
            check_pass "外部网络连通正常"
        else
            check_warn "外部网络不可达（可能影响镜像拉取，单镜像模式通常不需要）"
        fi
    fi
    
    # 检查DNS解析（跨平台兼容）
    # Linux: nslookup, getent hosts
    # macOS: nslookup, getent hosts (如果安装了)
    # Windows: nslookup
    # Alpine/BusyBox: nslookup (如果安装了 bind-tools)
    if (command -v nslookup >/dev/null 2>&1 && nslookup google.com &> /dev/null 2>&1) || \
       (command -v getent >/dev/null 2>&1 && getent hosts google.com &> /dev/null 2>&1); then
        check_pass "DNS解析正常"
    else
        check_warn "DNS解析检测失败（某些环境可能不支持 DNS 检测工具）"
    fi
}

# ============================================
# 2. 数据库服务可用性检查
# ============================================
check_database() {
    print_step "检查数据库服务..."
    
    DB_HOST="${DB_HOST:-postgres}"
    DB_PORT="${DB_PORT:-5432}"
    DB_USER="${DB_USER:-registry}"
    DB_NAME="${DB_NAME:-registry_db}"
    
    # 检查数据库端口是否可达（跨平台兼容）
    # Linux/Alpine: nc (netcat) 或 bash TCP 检测
    # macOS: nc (通常已安装) 或 bash TCP 检测
    # Windows: 可能没有 nc，使用 bash TCP 检测（如果可用）
    if command -v nc >/dev/null 2>&1; then
        # GNU netcat 或 BusyBox netcat
        if nc -z -w 3 "$DB_HOST" "$DB_PORT" 2>/dev/null || \
           nc -z "$DB_HOST" "$DB_PORT" 2>/dev/null; then
            check_pass "数据库端口 $DB_HOST:$DB_PORT 可达"
        else
            check_fail "数据库端口 $DB_HOST:$DB_PORT 不可达"
            return 1
        fi
    elif command -v timeout >/dev/null 2>&1 && command -v bash >/dev/null 2>&1; then
        # Fallback: 使用 bash 内置 TCP 检测（Linux/macOS/Git Bash）
        if timeout 3 bash -c "echo > /dev/tcp/$DB_HOST/$DB_PORT" 2>/dev/null; then
            check_pass "数据库端口 $DB_HOST:$DB_PORT 可达"
        else
            check_fail "数据库端口 $DB_HOST:$DB_PORT 不可达"
            return 1
        fi
    else
        check_warn "无法检测数据库端口连通性（nc/timeout未安装，单镜像模式会自动检查）"
    fi
    
    # 如果pg_isready可用，检查数据库就绪状态
    if command -v pg_isready &> /dev/null; then
        if pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" &> /dev/null; then
            check_pass "数据库服务就绪"
        else
            check_fail "数据库服务未就绪"
            return 1
        fi
    fi
}

# ============================================
# 3. 依赖服务运行状态检查
# ============================================
check_dependencies() {
    print_step "检查依赖服务..."
    
    # 检查Redis（跨平台端口检测）
    REDIS_HOST="${REDIS_HOST:-redis}"
    REDIS_PORT="${REDIS_PORT:-6379}"
    
    if command -v nc >/dev/null 2>&1; then
        if nc -z -w 3 "$REDIS_HOST" "$REDIS_PORT" 2>/dev/null || \
           nc -z "$REDIS_HOST" "$REDIS_PORT" 2>/dev/null; then
            check_pass "Redis服务 $REDIS_HOST:$REDIS_PORT 可达"
        else
            check_warn "Redis服务 $REDIS_HOST:$REDIS_PORT 不可达（将使用内存缓存）"
        fi
    elif command -v timeout >/dev/null 2>&1 && command -v bash >/dev/null 2>&1; then
        # Fallback: 使用 bash TCP 检测
        if timeout 3 bash -c "echo > /dev/tcp/$REDIS_HOST/$REDIS_PORT" 2>/dev/null; then
            check_pass "Redis服务 $REDIS_HOST:$REDIS_PORT 可达"
        else
            check_warn "Redis服务 $REDIS_HOST:$REDIS_PORT 不可达（将使用内存缓存）"
        fi
    elif command -v redis-cli >/dev/null 2>&1; then
        if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping &> /dev/null; then
            check_pass "Redis服务正常"
        else
            check_warn "Redis服务异常（将使用内存缓存）"
        fi
    else
        check_warn "无法检测Redis服务（将使用内存缓存）"
    fi
    
    # 检查MinIO（如果使用）
    if [ "${STORAGE_TYPE:-local}" == "minio" ]; then
        MINIO_ENDPOINT="${STORAGE_MINIO_ENDPOINT:-minio:9000}"
        MINIO_HOST=$(echo "$MINIO_ENDPOINT" | cut -d: -f1)
        MINIO_PORT=$(echo "$MINIO_ENDPOINT" | cut -d: -f2)
        
        if command -v nc &> /dev/null; then
            if nc -z -w 3 "$MINIO_HOST" "$MINIO_PORT" 2>/dev/null; then
                check_pass "MinIO服务 $MINIO_HOST:$MINIO_PORT 可达"
            else
                check_fail "MinIO服务 $MINIO_HOST:$MINIO_PORT 不可达"
                return 1
            fi
        else
            check_warn "无法检测MinIO服务连通性"
        fi
    fi
}

# ============================================
# 4. 配置文件完整性与权限检查
# ============================================
check_config_files() {
    print_step "检查配置文件..."
    
    CONFIG_FILE="${CONFIG_FILE:-/app/config.yaml}"
    
    if [ -f "$CONFIG_FILE" ]; then
        check_pass "配置文件存在: $CONFIG_FILE"
        
        # 检查文件权限
        if [ -r "$CONFIG_FILE" ]; then
            check_pass "配置文件可读"
        else
            check_fail "配置文件不可读: $CONFIG_FILE"
            return 1
        fi
        
        # 检查YAML语法（如果yq或python可用）
        if command -v yq &> /dev/null; then
            if yq eval '.' "$CONFIG_FILE" &> /dev/null; then
                check_pass "配置文件YAML语法正确"
            else
                check_fail "配置文件YAML语法错误"
                return 1
            fi
        elif command -v python3 &> /dev/null; then
            if python3 -c "import yaml; yaml.safe_load(open('$CONFIG_FILE'))" 2>/dev/null; then
                check_pass "配置文件YAML语法正确"
            else
                check_fail "配置文件YAML语法错误"
                return 1
            fi
        else
            check_warn "无法验证YAML语法（yq/python3未安装）"
        fi
    else
        check_fail "配置文件不存在: $CONFIG_FILE"
        return 1
    fi
}

# ============================================
# 5. 存储目录可读写性检查
# ============================================
check_storage() {
    print_step "检查存储目录..."
    
    if [ "${STORAGE_TYPE:-local}" == "local" ]; then
        STORAGE_PATH="${STORAGE_LOCAL_ROOT_PATH:-/data/storage}"
    else
        STORAGE_PATH="/data/storage"
    fi
    
    # 检查目录是否存在
    if [ ! -d "$STORAGE_PATH" ]; then
        print_warning "存储目录不存在，尝试创建: $STORAGE_PATH"
        if mkdir -p "$STORAGE_PATH" 2>/dev/null; then
            check_pass "已创建存储目录: $STORAGE_PATH"
        else
            check_fail "无法创建存储目录: $STORAGE_PATH"
            return 1
        fi
    else
        check_pass "存储目录存在: $STORAGE_PATH"
    fi
    
    # 检查目录权限
    if [ -r "$STORAGE_PATH" ] && [ -w "$STORAGE_PATH" ]; then
        check_pass "存储目录可读写"
    else
        check_fail "存储目录权限不足: $STORAGE_PATH"
        return 1
    fi
    
    # 测试写入
    TEST_FILE="$STORAGE_PATH/.write_test_$$"
    if touch "$TEST_FILE" 2>/dev/null && rm -f "$TEST_FILE" 2>/dev/null; then
        check_pass "存储目录写入测试通过"
    else
        check_fail "存储目录写入测试失败"
        return 1
    fi
}

# ============================================
# 6. 生产环境关键配置检查（强制）
# ============================================
check_prod_required_secrets() {
    print_step "检查生产环境关键配置..."

    # 运行环境（默认 production）
    APP_ENV="${APP_ENV:-production}"

    # 必须配置 JWT_SECRET / DB_PASSWORD
    if [ -z "${JWT_SECRET:-}" ]; then
        check_fail "JWT_SECRET 未设置（必须由全局配置中心/.env 显式提供）"
        return 1
    fi
    if [ -z "${DB_PASSWORD:-}" ]; then
        check_fail "DB_PASSWORD 未设置（必须设置，且必须与数据库实际密码一致）"
        return 1
    fi

    # 使用 MinIO 时必须配置密钥
    if [ "${STORAGE_TYPE:-local}" == "minio" ]; then
        if [ -z "${MINIO_ACCESS_KEY:-}" ] || [ -z "${MINIO_SECRET_KEY:-}" ]; then
            check_fail "使用 MinIO 存储但未设置 MINIO_ACCESS_KEY / MINIO_SECRET_KEY（必须设置）"
            return 1
        fi
    fi

    check_pass "关键配置检查通过（APP_ENV=$APP_ENV）"
}

# ============================================
# 6. 镜像版本一致性检查
# ============================================
check_image_versions() {
    print_step "检查镜像版本..."
    
    # 如果是在容器内，检查当前镜像信息
    if [ -f /.dockerenv ] || (grep -q "docker\|podman" /proc/1/cgroup 2>/dev/null); then
        if [ -f /etc/os-release ]; then
            OS_VERSION=$(grep "PRETTY_NAME" /etc/os-release | cut -d'"' -f2)
            check_pass "容器OS版本: $OS_VERSION"
        fi
    fi
    
    # 检查应用版本（如果存在版本文件）
    if [ -f /app/VERSION ] || [ -f /app/version.txt ]; then
        VERSION_FILE=$(ls /app/VERSION /app/version.txt 2>/dev/null | head -1)
        APP_VERSION=$(cat "$VERSION_FILE" 2>/dev/null || echo "Unknown")
        check_pass "应用版本: $APP_VERSION"
    fi
}

# ============================================
# 7. 资源配额检查
# ============================================
check_resources() {
    print_step "检查资源配额..."
    
    # 检查内存（跨平台兼容）
    # Linux: /proc/meminfo
    # macOS: sysctl hw.memsize (容器内通常不会用到)
    if [ -f /proc/meminfo ]; then
        MEM_AVAILABLE=$(grep MemAvailable /proc/meminfo 2>/dev/null | awk '{print $2}' || echo "")
        MEM_TOTAL=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}' || echo "")
        
        if [ -n "$MEM_AVAILABLE" ] && [ -n "$MEM_TOTAL" ] && [ "$MEM_AVAILABLE" != "0" ] && [ "$MEM_TOTAL" != "0" ]; then
            # 使用 awk 进行整数运算（兼容性更好）
            MEM_PERCENT=$(awk "BEGIN {printf \"%.0f\", $MEM_AVAILABLE * 100 / $MEM_TOTAL}" 2>/dev/null || echo "0")
            if [ "$MEM_PERCENT" -lt 10 ]; then
                check_fail "可用内存不足（${MEM_PERCENT}%）"
                return 1
            elif [ "$MEM_PERCENT" -lt 20 ]; then
                check_warn "可用内存较低（${MEM_PERCENT}%）"
            else
                check_pass "内存充足（可用: ${MEM_PERCENT}%）"
            fi
        fi
    fi
    
    # 检查磁盘空间（跨平台兼容）
    # Linux/macOS: df
    # Alpine/BusyBox: df (可能不支持某些选项)
    STORAGE_PATH="${STORAGE_LOCAL_ROOT_PATH:-/data/storage}"
    if [ -d "$STORAGE_PATH" ]; then
        # 优先使用 df -BG（GNU df，大多数 Linux 发行版）
        if df -BG "$STORAGE_PATH" >/dev/null 2>&1; then
            DISK_AVAILABLE=$(df -BG "$STORAGE_PATH" 2>/dev/null | tail -1 | awk '{print $4}' | sed 's/G//' || echo "")
            DISK_TOTAL=$(df -BG "$STORAGE_PATH" 2>/dev/null | tail -1 | awk '{print $2}' | sed 's/G//' || echo "")
        else
            # Fallback: 使用 df -h（Alpine/BusyBox 兼容）
            DISK_AVAILABLE=$(df -h "$STORAGE_PATH" 2>/dev/null | tail -1 | awk '{print $4}' || echo "")
            DISK_TOTAL=$(df -h "$STORAGE_PATH" 2>/dev/null | tail -1 | awk '{print $2}' || echo "")
        fi
        
        if [ -n "$DISK_AVAILABLE" ] && [ -n "$DISK_TOTAL" ] && [ "$DISK_AVAILABLE" != "0" ] && [ "$DISK_TOTAL" != "0" ]; then
            # 使用 awk 进行整数运算（兼容性更好）
            # 注意：如果 df -h 返回的是 "10G" 格式，需要先转换
            DISK_PERCENT=$(awk "BEGIN {printf \"%.0f\", ($DISK_AVAILABLE + 0) * 100 / ($DISK_TOTAL + 0)}" 2>/dev/null || echo "0")
            if [ "$DISK_PERCENT" -lt 10 ]; then
                check_fail "磁盘空间不足（可用: ${DISK_PERCENT}%）"
                return 1
            elif [ "$DISK_PERCENT" -lt 20 ]; then
                check_warn "磁盘空间较低（可用: ${DISK_PERCENT}%）"
            else
                check_pass "磁盘空间充足（可用: ${DISK_PERCENT}%）"
            fi
        fi
    fi
}

# ============================================
# 8. 自动修复功能
# ============================================
auto_fix() {
    print_step "尝试自动修复..."
    
    FIXED=0
    
    # 修复存储目录
    STORAGE_PATH="${STORAGE_LOCAL_ROOT_PATH:-/data/storage}"
    if [ ! -d "$STORAGE_PATH" ]; then
        if mkdir -p "$STORAGE_PATH" 2>/dev/null; then
            chmod 755 "$STORAGE_PATH" 2>/dev/null
            print_success "已创建存储目录: $STORAGE_PATH"
            ((FIXED++))
        fi
    fi
    
    # 修复配置文件权限
    CONFIG_FILE="${CONFIG_FILE:-/app/config.yaml}"
    if [ -f "$CONFIG_FILE" ] && [ ! -r "$CONFIG_FILE" ]; then
        chmod 644 "$CONFIG_FILE" 2>/dev/null && print_success "已修复配置文件权限" && ((FIXED++))
    fi
    
    if [ $FIXED -gt 0 ]; then
        print_success "自动修复完成（修复 $FIXED 项）"
        return 0
    else
        print_warning "无需修复或修复失败"
        return 1
    fi
}

# ============================================
# 主检查流程
# ============================================
main() {
    echo "============================================"
    echo "  生产环境启动前自动检查"
    echo "============================================"
    echo ""
    
    # 执行所有检查
    check_network_connectivity || true
    check_database || true
    check_dependencies || true
    check_config_files || true
    check_storage || true
    check_prod_required_secrets
    check_image_versions || true
    check_resources || true
    
    echo ""
    echo "============================================"
    echo "  检查结果摘要"
    echo "============================================"
    echo "✅ 通过: $CHECKS_PASSED"
    echo "⚠️  警告: $CHECKS_WARNINGS"
    echo "❌ 失败: $CHECKS_FAILED"
    echo "============================================"
    
    # 如果有失败项，尝试自动修复
    if [ $CHECKS_FAILED -gt 0 ]; then
        echo ""
        auto_fix
        
        # 重新检查关键项
        echo ""
        print_step "重新检查关键项..."
        check_config_files || true
        check_storage || true
    fi
    
    # 最终判断
    if [ $CHECKS_FAILED -gt 0 ]; then
        echo ""
        print_error "检查未完全通过，请手动修复后重试"
        exit 1
    elif [ $CHECKS_WARNINGS -gt 0 ]; then
        echo ""
        print_warning "检查通过，但有警告项，建议检查"
        exit 0
    else
        echo ""
        print_success "所有检查通过，可以启动服务"
        exit 0
    fi
}

# 执行主流程
main


