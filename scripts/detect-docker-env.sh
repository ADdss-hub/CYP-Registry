#!/bin/bash
# ============================================
# Docker环境自动检测脚本 (Linux/macOS)
# 遵循《全平台通用容器开发设计规范》3.3节
# 使用方法: ./scripts/detect-docker-env.sh
# ============================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 输出函数
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

# 检测报告文件
REPORT_FILE="${REPORT_FILE:-/tmp/docker_env_detect_report.json}"

# ============================================
# 1. 系统版本检测（主备命令交叉验证）
# ============================================
print_info "检测操作系统版本..."

if command -v lsb_release &> /dev/null; then
    OS_VERSION=$(lsb_release -a 2>/dev/null | grep "Description" | awk -F: '{print $2}' | sed 's/^ //' || echo "Unknown")
elif [ -f /etc/os-release ]; then
    OS_VERSION=$(grep "PRETTY_NAME" /etc/os-release | cut -d'"' -f2 || echo "Unknown")
elif [ -f /etc/redhat-release ]; then
    OS_VERSION=$(cat /etc/redhat-release)
elif [ "$(uname)" == "Darwin" ]; then
    OS_VERSION=$(sw_vers -productName)" "$(sw_vers -productVersion)
else
    OS_VERSION="Unknown"
fi

print_success "操作系统: $OS_VERSION"

# ============================================
# 2. 硬件资源检测（统一单位为Gi）
# ============================================
print_info "检测硬件资源..."

# CPU信息
if [ "$(uname)" == "Darwin" ]; then
    CPU_INFO=$(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo "Unknown")
    CPU_CORES=$(sysctl -n hw.ncpu 2>/dev/null || echo "Unknown")
else
    if command -v lscpu &> /dev/null; then
        CPU_INFO=$(lscpu | grep "Model name" | awk -F: '{print $2}' | sed 's/^ //' || echo "Unknown")
        CPU_CORES=$(lscpu | grep "^CPU(s):" | awk '{print $2}' || echo "Unknown")
    else
        CPU_INFO="Unknown"
        CPU_CORES=$(nproc 2>/dev/null || echo "Unknown")
    fi
fi

# 内存信息
if [ "$(uname)" == "Darwin" ]; then
    MEM_TOTAL_GB=$(sysctl -n hw.memsize 2>/dev/null | awk '{printf "%.2f", $1/1024/1024/1024}' || echo "0")
    MEM_TOTAL="${MEM_TOTAL_GB}Gi"
else
    if command -v free &> /dev/null; then
        MEM_TOTAL_MB=$(free -m | grep Mem | awk '{print $2}' || echo "0")
        MEM_TOTAL_GB=$(echo "$MEM_TOTAL_MB / 1024" | bc -l 2>/dev/null | awk '{printf "%.2f", $1}' || echo "0")
        MEM_TOTAL="${MEM_TOTAL_GB}Gi"
    else
        MEM_TOTAL="Unknown"
    fi
fi

print_success "CPU: $CPU_INFO (核心数: $CPU_CORES)"
print_success "内存: $MEM_TOTAL"

# ============================================
# 3. 容器环境判断
# ============================================
print_info "检测容器环境..."

if [ -f /proc/1/cgroup ]; then
    if grep -q "docker\|podman" /proc/1/cgroup 2>/dev/null; then
        ENV_TYPE="Container"
        # 检测容器引擎类型
        if grep -q "docker" /proc/1/cgroup; then
            CONTAINER_ENGINE="Docker"
        elif grep -q "podman" /proc/1/cgroup; then
            CONTAINER_ENGINE="Podman"
        else
            CONTAINER_ENGINE="Unknown"
        fi
    else
        ENV_TYPE="Host"
        CONTAINER_ENGINE="None"
    fi
elif [ -f /.dockerenv ]; then
    ENV_TYPE="Container"
    CONTAINER_ENGINE="Docker"
else
    ENV_TYPE="Host"
    CONTAINER_ENGINE="None"
fi

print_success "环境类型: $ENV_TYPE"
if [ "$ENV_TYPE" == "Container" ]; then
    print_success "容器引擎: $CONTAINER_ENGINE"
fi

# ============================================
# 4. Docker/Podman检测
# ============================================
print_info "检测容器引擎..."

DOCKER_AVAILABLE=false
PODMAN_AVAILABLE=false
DOCKER_VERSION=""
PODMAN_VERSION=""

if command -v docker &> /dev/null; then
    DOCKER_AVAILABLE=true
    DOCKER_VERSION=$(docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//' || echo "Unknown")
    print_success "Docker已安装: $DOCKER_VERSION"
else
    print_warning "Docker未安装"
fi

if command -v podman &> /dev/null; then
    PODMAN_AVAILABLE=true
    PODMAN_VERSION=$(podman --version 2>/dev/null | awk '{print $3}' || echo "Unknown")
    print_success "Podman已安装: $PODMAN_VERSION"
else
    print_warning "Podman未安装"
fi

if command -v docker-compose &> /dev/null || docker compose version &> /dev/null; then
    COMPOSE_AVAILABLE=true
    if docker compose version &> /dev/null; then
        COMPOSE_VERSION=$(docker compose version 2>/dev/null | awk '{print $4}' || echo "Unknown")
    else
        COMPOSE_VERSION=$(docker-compose --version 2>/dev/null | awk '{print $3}' | sed 's/,//' || echo "Unknown")
    fi
    print_success "Docker Compose已安装: $COMPOSE_VERSION"
else
    COMPOSE_AVAILABLE=false
    print_warning "Docker Compose未安装"
fi

# ============================================
# 5. 网络配置检测
# ============================================
print_info "检测网络配置..."

if [ "$(uname)" == "Darwin" ]; then
    IP_ADDRESS=$(ifconfig | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | head -1 || echo "Unknown")
else
    IP_ADDRESS=$(ip route get 8.8.8.8 2>/dev/null | awk '{print $7}' | head -1 || \
                 hostname -I 2>/dev/null | awk '{print $1}' || \
                 echo "Unknown")
fi

print_success "IP地址: $IP_ADDRESS"

# ============================================
# 6. 存储路径检测
# ============================================
print_info "检测存储路径..."

# 检测NAS环境
NAS_MODEL=""
if command -v synogear &> /dev/null || [ -f /etc/synoinfo.conf ]; then
    NAS_MODEL="Synology"
    STORAGE_PATH="/volume1/docker"
elif command -v getcfg &> /dev/null || [ -f /etc/config/qpkg.conf ]; then
    NAS_MODEL="QNAP"
    STORAGE_PATH="/share/Container/docker"
else
    NAS_MODEL="None"
    STORAGE_PATH="/var/lib/docker"
fi

# 检查存储路径是否存在，不存在则创建
if [ "$ENV_TYPE" == "Host" ] && [ ! -d "$STORAGE_PATH" ]; then
    print_warning "存储路径不存在: $STORAGE_PATH"
    if [ "$NAS_MODEL" != "None" ]; then
        print_info "尝试创建存储路径..."
        mkdir -p "$STORAGE_PATH" 2>/dev/null && chmod 755 "$STORAGE_PATH" 2>/dev/null && \
        print_success "已创建存储路径: $STORAGE_PATH" || \
        print_warning "无法创建存储路径，可能需要root权限"
    fi
fi

if [ -d "$STORAGE_PATH" ]; then
    # 跨平台权限检测（兼容不同 Linux 发行版）
    # Linux: stat -c "%a" (GNU coreutils)
    # macOS: stat -f "%OLp" (BSD stat)
    # Alpine/BusyBox: 可能不支持 stat -c，使用 ls 作为 fallback
    if stat -c "%a" "$STORAGE_PATH" >/dev/null 2>&1; then
        STORAGE_PERM=$(stat -c "%a" "$STORAGE_PATH" 2>/dev/null)
    elif stat -f "%OLp" "$STORAGE_PATH" >/dev/null 2>&1; then
        STORAGE_PERM=$(stat -f "%OLp" "$STORAGE_PATH" 2>/dev/null)
    else
        # Fallback: 使用 ls 命令（Alpine/BusyBox 兼容）
        STORAGE_PERM=$(ls -ld "$STORAGE_PATH" 2>/dev/null | awk '{print $1}' || echo "Unknown")
    fi
    print_success "存储路径: $STORAGE_PATH (权限: $STORAGE_PERM)"
else
    print_warning "存储路径不可访问: $STORAGE_PATH"
fi

# ============================================
# 7. Docker服务状态检测
# ============================================
print_info "检测Docker服务状态..."

if [ "$DOCKER_AVAILABLE" == "true" ]; then
    if docker info &> /dev/null; then
        DOCKER_RUNNING=true
        print_success "Docker服务运行中"
    else
        DOCKER_RUNNING=false
        print_error "Docker服务未运行"
    fi
else
    DOCKER_RUNNING=false
fi

# ============================================
# 8. 生成检测报告
# ============================================
print_info "生成检测报告: $REPORT_FILE"

# 创建JSON格式的检测报告
cat > "$REPORT_FILE" << EOF
{
  "os_version": "$OS_VERSION",
  "cpu_info": "$CPU_INFO",
  "cpu_cores": "$CPU_CORES",
  "mem_total": "$MEM_TOTAL",
  "env_type": "$ENV_TYPE",
  "container_engine": "$CONTAINER_ENGINE",
  "docker_available": $DOCKER_AVAILABLE,
  "docker_version": "$DOCKER_VERSION",
  "docker_running": $DOCKER_RUNNING,
  "podman_available": $PODMAN_AVAILABLE,
  "podman_version": "$PODMAN_VERSION",
  "compose_available": $COMPOSE_AVAILABLE,
  "compose_version": "$COMPOSE_VERSION",
  "ip_address": "$IP_ADDRESS",
  "nas_model": "$NAS_MODEL",
  "storage_path": "$STORAGE_PATH",
  "storage_permission": "$STORAGE_PERM",
  "detect_time": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

print_success "检测完成！报告已保存至: $REPORT_FILE"

# ============================================
# 9. 输出摘要
# ============================================
echo ""
echo "============================================"
echo "  环境检测摘要"
echo "============================================"
echo "操作系统: $OS_VERSION"
echo "环境类型: $ENV_TYPE"
echo "CPU: $CPU_INFO ($CPU_CORES 核心)"
echo "内存: $MEM_TOTAL"
echo "IP地址: $IP_ADDRESS"
if [ "$DOCKER_AVAILABLE" == "true" ]; then
    echo "Docker: $DOCKER_VERSION ($([ "$DOCKER_RUNNING" == "true" ] && echo "运行中" || echo "未运行"))"
fi
if [ "$COMPOSE_AVAILABLE" == "true" ]; then
    echo "Docker Compose: $COMPOSE_VERSION"
fi
if [ "$NAS_MODEL" != "None" ]; then
    echo "NAS型号: $NAS_MODEL"
fi
echo "存储路径: $STORAGE_PATH"
echo "============================================"

exit 0


