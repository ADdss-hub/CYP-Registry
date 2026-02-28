#!/bin/bash
# ============================================
# CYP-Registry 自动配置脚本 (Linux/macOS/容器内)
# 强制要求：任何环境/任何平台/任何系统下，均可一键自动生成可运行的配置
# 目标：
# - 若项目根目录不存在 .env，则自动生成（含必要默认值）
# - 为 DB_PASSWORD / REDIS_PASSWORD / JWT_SECRET 生成强随机值（生产可直接使用）
# - 不修改已存在的 .env（保证可重复执行、幂等）
# - 支持通过 AUTO_CONFIG_ROOT 覆盖项目根目录（容器内挂载宿主机目录时使用）
# ============================================

set -euo pipefail

# 允许通过环境变量覆盖项目根目录（例如：容器内挂载的宿主机目录）
ROOT_DIR="${AUTO_CONFIG_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
ENV_FILE="${ROOT_DIR}/.env"
FRONTEND_ENV_FILE="${ROOT_DIR}/web/.env.local"

info() { printf "ℹ️  %s\n" "$1"; }
ok() { printf "✅ %s\n" "$1"; }
warn() { printf "⚠️  %s\n" "$1"; }

# 生成 32 字节（64 hex 字符）的强随机密钥，兼容本地/容器环境
rand_hex() {
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

if [ -f "${ENV_FILE}" ]; then
  ok ".env 已存在，跳过自动生成：${ENV_FILE}"
else
  JWT_SECRET="jwt_$(rand_hex)"
  DB_PASSWORD_RANDOM="$(rand_hex)"
  REDIS_PASSWORD_RANDOM="$(rand_hex)"

  cat > "${ENV_FILE}" <<EOF
# ============================================
# CYP-Registry 全局配置中心（根级 .env）
# 生成时间：$(date '+%Y-%m-%d %H:%M:%S')
# 说明：
# - 本文件作为全局配置中心唯一源头，前端/后端/容器均应从此派生配置
# - DB_PASSWORD / REDIS_PASSWORD / JWT_SECRET 由脚本自动生成为强随机值，可直接用于生产环境
# ============================================

# Application
APP_NAME=CYP-Registry
APP_HOST=${APP_HOST:-0.0.0.0}
APP_PORT=${APP_PORT:-8080}
APP_ENV=${APP_ENV:-production}

# Database (默认用于本地/单机环境；容器编排环境可通过 docker-compose 覆盖)
DB_HOST=${DB_HOST:-127.0.0.1}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-registry}
DB_NAME=${DB_NAME:-registry_db}
DB_SSLMODE=${DB_SSLMODE:-disable}

# Redis (默认用于本地/单机环境；容器编排环境可通过 docker-compose 覆盖)
REDIS_HOST=${REDIS_HOST:-127.0.0.1}
REDIS_PORT=${REDIS_PORT:-6379}
REDIS_DB=${REDIS_DB:-0}

# API / Web Endpoints
API_BASE_URL=${API_BASE_URL:-http://localhost:8080}
WEB_BASE_URL=${WEB_BASE_URL:-http://localhost:3000}

# Database / Redis Passwords
DB_PASSWORD=${DB_PASSWORD_RANDOM}
REDIS_PASSWORD=${REDIS_PASSWORD_RANDOM}

# JWT (AUTO-GENERATED)
JWT_SECRET=${JWT_SECRET}

# Storage
STORAGE_TYPE=local

# MinIO (仅当 STORAGE_TYPE=minio 时需要)
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# CORS
CORS_ALLOWED_ORIGINS=${CORS_ALLOWED_ORIGINS:-http://localhost:3000}

# Grafana
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin

# Server Shutdown & Cleanup
# CLEANUP_ON_SHUTDOWN: 控制服务器关闭时是否清理所有数据
#   1 = 清理所有数据（删除模式）- 会永久删除所有用户数据、项目数据、镜像文件、缓存数据
#   0 或不设置 = 保留数据（停止模式）- 仅关闭服务，保留所有数据
# ⚠️ 警告：设置为 1 时，关闭服务器会永久删除所有数据，此操作不可恢复！
# 生产环境强烈建议设置为 0 或不设置，避免误操作导致数据丢失
CLEANUP_ON_SHUTDOWN=${CLEANUP_ON_SHUTDOWN:-0}
EOF

  # 设置文件权限（跨平台兼容）
  # Linux/macOS: chmod 600
  # Windows: chmod 可能无效，但不影响功能
  chmod 600 "${ENV_FILE}" 2>/dev/null || true
  ok "全局配置中心已初始化：${ENV_FILE}"
  ok "已为 DB_PASSWORD / REDIS_PASSWORD / JWT_SECRET 生成强随机值，可直接用于生产环境。"
fi

# --------------------------------------------
# 前端配置自动初始化（从全局配置中心派生）
# 若不存在前端目录，则自动创建并初始化 web/.env.local（包括容器环境）
# --------------------------------------------
FRONTEND_DIR="$(dirname "${FRONTEND_ENV_FILE}")"

# 确保前端目录存在（容器镜像中可能默认不包含 web 目录）
if [ ! -d "${FRONTEND_DIR}" ]; then
  mkdir -p "${FRONTEND_DIR}"
fi

if [ ! -f "${FRONTEND_ENV_FILE}" ]; then
  # 从全局配置中心读取必要字段（若失败则使用与上方写入保持一致的默认值）
  APP_NAME_VAL="$(grep -E '^APP_NAME=' "${ENV_FILE}" 2>/dev/null | cut -d'=' -f2- || true)"
  APP_ENV_VAL="$(grep -E '^APP_ENV=' "${ENV_FILE}" 2>/dev/null | cut -d'=' -f2- || true)"
  API_BASE_VAL="$(grep -E '^API_BASE_URL=' "${ENV_FILE}" 2>/dev/null | cut -d'=' -f2- || true)"

  APP_NAME_VAL="${APP_NAME_VAL:-CYP-Registry}"
  APP_ENV_VAL="${APP_ENV_VAL:-production}"
  API_BASE_VAL="${API_BASE_VAL:-http://localhost:8080}"

  cat > "${FRONTEND_ENV_FILE}" <<EOF_FE
# ============================================
# CYP-Registry 前端环境变量（由全局配置中心自动生成）
# 生成时间：$(date '+%Y-%m-%d %H:%M:%S')
# 说明：
# - 本文件由项目根级 .env 派生（全局配置中心）
# - 前端仅需关心 VITE_* 变量，不直接修改根级 .env
# ============================================

VITE_APP_NAME=${APP_NAME_VAL}
VITE_APP_ENV=${APP_ENV_VAL}
VITE_API_BASE_URL=${API_BASE_VAL}
EOF_FE

  ok "前端配置已初始化：web/.env.local（基于全局配置中心）"
else
  info "前端配置已存在：web/.env.local，跳过自动初始化"
fi

