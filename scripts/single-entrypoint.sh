#!/usr/bin/env bash
# ============================================
# CYP-Registry 单镜像(All-in-One) 入口脚本
# - 启动内置 Postgres + Redis
# - 首次启动初始化 DB（创建用户/库，执行 init-scripts/01-schema.sql）
# - 启动 registry-server
# ============================================

set -euo pipefail

log() { echo "[single] $*"; }

# ----------------------------
# 在任何启动逻辑之前：自动生成并加载宿主机 .env
# ----------------------------
# APP_ROOT：容器内应用根目录（镜像内固定为 /app）
APP_ROOT="${APP_ROOT:-/app}"
# HOST_PROJECT_ROOT：宿主机项目根目录在容器内的挂载路径
# 默认使用 /workspace，可通过环境变量覆盖
HOST_PROJECT_ROOT="${HOST_PROJECT_ROOT:-/workspace}"
AUTO_CONFIG_SCRIPT="${APP_ROOT}/scripts/auto-config.sh"

# 若宿主机项目根目录已挂载，则在启动前自动生成 .env（如不存在）并按需加载；
# 若未挂载，则在容器内 APP_ROOT(/app) 目录下生成并加载 .env（仅对当前容器/卷生效，不回写宿主机）。
if [[ -d "${HOST_PROJECT_ROOT}" ]]; then
  ROOT_FOR_ENV="${HOST_PROJECT_ROOT}"
  log "检测到宿主机项目根目录挂载：${HOST_PROJECT_ROOT}（优先在此生成/读取 .env）"
else
  ROOT_FOR_ENV="${APP_ROOT}"
  log "INFO: 未检测到宿主机项目根目录挂载：${HOST_PROJECT_ROOT}，将在容器内 ${ROOT_FOR_ENV} 下自动生成并加载 .env（不会回写宿主机）。"
fi

# 自动生成 .env（如不存在）
if [[ ! -f "${ROOT_FOR_ENV}/.env" ]]; then
  if [[ -x "${AUTO_CONFIG_SCRIPT}" ]]; then
    log "未检测到 .env，使用 auto-config.sh 自动生成（ROOT=${ROOT_FOR_ENV}）..."
    # 入口脚本已在前面将 APP_ENV 默认设置为 production；
    # 这里直接透传当前 APP_ENV，确保单镜像容器默认始终以生产环境运行。
    AUTO_CONFIG_ROOT="${ROOT_FOR_ENV}" bash "${AUTO_CONFIG_SCRIPT}" || \
      log "ERROR: auto-config.sh 执行失败，请检查脚本或文件系统权限。"
  else
    log "ERROR: 未找到可执行的 auto-config.sh (${AUTO_CONFIG_SCRIPT})，无法自动生成 .env。"
  fi
else
  log "检测到已有 .env，保持不覆盖：${ROOT_FOR_ENV}/.env"
fi

# 从 .env 加载环境变量：仅填充当前环境中“未显式设置”的变量
# 兼容 Windows BOM/CRLF、前导空格、export 前缀
if [[ -f "${ROOT_FOR_ENV}/.env" ]]; then
  log "从 ${ROOT_FOR_ENV}/.env 加载默认环境变量（不覆盖已存在的环境变量）"
  while IFS= read -r line; do
    # 去除 CRLF 与 UTF-8 BOM
    line="${line%$'\r'}"
    line="${line#$'\xEF\xBB\xBF'}"
    # 去除前导空格
    line="${line#"${line%%[![:space:]]*}"}"
    # 跳过空行与注释
    [[ -z "${line}" ]] && continue
    [[ "${line}" == \#* ]] && continue
    # 支持 "export KEY=VALUE"
    [[ "${line}" == export\ * ]] && line="${line#export }"
    # 仅处理 KEY=VALUE 形式
    [[ "${line}" != *"="* ]] && continue

    key="${line%%=*}"
    value="${line#*=}"
    # key 合法性校验（避免非法变量名导致脚本退出）
    if [[ ! "${key}" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
      continue
    fi
    # 仅在当前环境未显式设置时赋值（保持 docker-compose environment 等更高优先级）
    if [[ -z "${!key+x}" ]]; then
      export "${key}=${value}"
    fi
  done < "${ROOT_FOR_ENV}/.env"
fi

#
# ----------------------------
# 默认值（允许通过环境变量覆盖）
# 说明：
# - DB_PASSWORD / JWT_SECRET 必须通过 .env/外部配置显式传入
# ----------------------------
export APP_NAME="${APP_NAME:-CYP-Registry}"
export APP_HOST="${APP_HOST:-0.0.0.0}"
export APP_PORT="${APP_PORT:-8080}"
export APP_ENV="${APP_ENV:-production}"
# 强制设置 GIN_MODE=release，避免 Gin 框架输出 debug 模式警告
export GIN_MODE="release"

export DB_HOST="${DB_HOST:-127.0.0.1}"
export DB_PORT="${DB_PORT:-5432}"
export DB_USER="${DB_USER:-registry}"
export DB_NAME="${DB_NAME:-registry_db}"
export DB_SSLMODE="${DB_SSLMODE:-disable}"

# ----------------------------
# 关键密钥：自动生成 + 持久化（避免 pre-start-check 阻断启动）
# 说明：
# - 生产环境“最佳实践”仍然是外部显式注入；但为了避免因为缺失而导致服务不可用，
#   单镜像模式下我们会在缺失时自动生成并写入数据卷内的 secrets 文件。
# - 持久化位置选择 /var/lib/postgresql/data（compose 中为持久卷），确保重启/升级不变。
# ----------------------------
SECRETS_DIR="/var/lib/postgresql/data"
DB_PASSWORD_FILE="${SECRETS_DIR}/.cyp_registry_db_password"
JWT_SECRET_FILE="${SECRETS_DIR}/.cyp_registry_jwt_secret"

ensure_secret_file() {
  local file="$1"
  local value="$2"
  umask 077
  mkdir -p "${SECRETS_DIR}" 2>/dev/null || true
  # 仅当文件不存在或为空时写入，避免覆盖既有密钥
  if [[ ! -s "${file}" ]]; then
    printf '%s' "${value}" > "${file}"
    chmod 600 "${file}" 2>/dev/null || true
  fi
}

read_secret_file() {
  local file="$1"
  if [[ -f "${file}" ]]; then
    # 去除 CRLF，避免 Windows 卷回写导致的 \r
    tr -d '\r\n' < "${file}"
  fi
}

# DB_PASSWORD：缺失则自动生成并持久化（供内置 Postgres 账号使用）
if [[ -z "${DB_PASSWORD:-}" ]]; then
  DB_PASSWORD="$(read_secret_file "${DB_PASSWORD_FILE}")"
fi
if [[ -z "${DB_PASSWORD:-}" ]]; then
  DB_PASSWORD="$(gen_random_hex)"
  ensure_secret_file "${DB_PASSWORD_FILE}" "${DB_PASSWORD}"
  log "WARN: 未设置 DB_PASSWORD，已自动生成并持久化到 ${DB_PASSWORD_FILE}（建议生产环境显式注入以便审计/轮换）"
fi
export DB_PASSWORD

export REDIS_HOST="${REDIS_HOST:-127.0.0.1}"
export REDIS_PORT="${REDIS_PORT:-6379}"
export REDIS_DB="${REDIS_DB:-0}"

# 兼容规范命名（APP_DB_* / APP_REDIS_*）
export APP_DB_HOST="${APP_DB_HOST:-$DB_HOST}"
export APP_DB_PORT="${APP_DB_PORT:-$DB_PORT}"
export APP_DB_USER="${APP_DB_USER:-$DB_USER}"
export APP_DB_PASSWORD="${APP_DB_PASSWORD:-$DB_PASSWORD}"
export APP_DB_NAME="${APP_DB_NAME:-$DB_NAME}"
export APP_DB_SSLMODE="${APP_DB_SSLMODE:-$DB_SSLMODE}"

export APP_REDIS_HOST="${APP_REDIS_HOST:-$REDIS_HOST}"
export APP_REDIS_PORT="${APP_REDIS_PORT:-$REDIS_PORT}"
export APP_REDIS_DB="${APP_REDIS_DB:-$REDIS_DB}"
export APP_REDIS_PASSWORD="${APP_REDIS_PASSWORD:-${REDIS_PASSWORD:-}}"

gen_random_hex() {
  # 32 bytes -> 64 hex chars
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  elif [ -r /dev/urandom ]; then
    od -An -N32 -tx1 /dev/urandom | tr -d ' \n'
  else
    date +%s | sha256sum | awk '{print $1}'
  fi
}

# JWT_SECRET：必须通过 .env/外部配置显式提供，缺失会在 pre-start-check.sh 中被视为致命错误
if [[ -z "${JWT_SECRET:-}" ]]; then
  JWT_SECRET="$(read_secret_file "${JWT_SECRET_FILE}")"
fi
if [[ -z "${JWT_SECRET:-}" ]]; then
  JWT_SECRET="$(gen_random_hex)"
  ensure_secret_file "${JWT_SECRET_FILE}" "${JWT_SECRET}"
  log "WARN: 未设置 JWT_SECRET，已自动生成并持久化到 ${JWT_SECRET_FILE}（建议生产环境显式注入以便审计/轮换）"
fi
export JWT_SECRET

fix_permissions() {
  mkdir -p /data/storage /data/redis /app/logs /var/lib/postgresql/data /run/postgresql || true
  chown -R appuser:appuser /data/storage /data/redis /app/logs 2>/dev/null || true
  chown -R postgres:postgres /var/lib/postgresql /run/postgresql 2>/dev/null || true
}

start_postgres() {
  log "启动内置 PostgreSQL..."

  # 初始化数据库目录（仅首次）
  if [[ ! -s /var/lib/postgresql/data/PG_VERSION ]]; then
    log "PostgreSQL 数据目录未初始化，正在 initdb..."
    # 使用更安全的认证方式初始化：
    # - 本地连接使用 peer（系统用户 postgres 通过本地 Unix Socket 免密登录）
    # - TCP 连接使用 scram-sha-256（应用通过密码访问）
    su-exec postgres:postgres initdb \
      --auth-local=peer \
      --auth-host=scram-sha-256 \
      -D /var/lib/postgresql/data >/dev/null

    # 允许本地连接（单机容器内）
    echo "listen_addresses = '127.0.0.1'" >> /var/lib/postgresql/data/postgresql.conf
    echo "port = ${DB_PORT}" >> /var/lib/postgresql/data/postgresql.conf

    # 确保 127.0.0.1 使用基于密码的安全认证（应用侧使用 registry/DB_PASSWORD 登录）
    echo "host all all 127.0.0.1/32 scram-sha-256" >> /var/lib/postgresql/data/pg_hba.conf
  fi

  su-exec postgres:postgres postgres -D /var/lib/postgresql/data -k /run/postgresql &
  PG_PID=$!

  # 等待就绪
  for i in $(seq 1 60); do
    if su-exec postgres:postgres pg_isready -h /run/postgresql -p "${DB_PORT}" >/dev/null 2>&1; then
      log "PostgreSQL 就绪"
      break
    fi
    sleep 1
  done

  if ! su-exec postgres:postgres pg_isready -h /run/postgresql -p "${DB_PORT}" >/dev/null 2>&1; then
    log "ERROR: PostgreSQL 启动失败"
    exit 1
  fi

  export PG_PID
}

init_db_schema_once() {
  # 用一个 marker 文件确保只初始化一次
  local marker="/var/lib/postgresql/data/.cyp_registry_initialized"
  if [[ -f "$marker" ]]; then
    return 0
  fi

  log "初始化数据库用户/库并执行 schema（仅首次）..."

  # 创建用户（若不存在）
  su-exec postgres:postgres psql -h /run/postgresql -p "${DB_PORT}" -d postgres -v ON_ERROR_STOP=1 <<SQL
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${DB_USER}') THEN
    CREATE ROLE ${DB_USER} LOGIN PASSWORD '${DB_PASSWORD}';
  END IF;
END
\$\$;
SQL

  # 执行初始化脚本（包含 CREATE DATABASE / \c）
  su-exec postgres:postgres psql -h /run/postgresql -p "${DB_PORT}" -d postgres -v ON_ERROR_STOP=1 -f /app/init-scripts/01-schema.sql

  touch "$marker"
  log "数据库初始化完成"
}

fix_db_permissions_and_schema() {
  # 每次启动都执行一次“幂等修复”，解决：
  # - 历史库表缺少 deleted_at 导致 GORM 查询报错
  # - registry 业务账号缺少 schema/table 权限导致 permission denied
  log "修复数据库权限与历史 schema 兼容性（幂等）..."

  # schema usage + default privileges（未来新表）
  su-exec postgres:postgres psql -h /run/postgresql -p "${DB_PORT}" -d "${DB_NAME}" -v ON_ERROR_STOP=1 <<SQL
GRANT USAGE ON SCHEMA public TO ${DB_USER};
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO ${DB_USER};
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO ${DB_USER};
SQL

  # 修复现有表/序列权限 + owner（需要 postgres）
  su-exec postgres:postgres psql -h /run/postgresql -p "${DB_PORT}" -d "${DB_NAME}" -v ON_ERROR_STOP=1 <<SQL
SET cyp_registry.db_user = '${DB_USER}';
DO \$\$
DECLARE
  r record;
  db_user text := current_setting('cyp_registry.db_user');
BEGIN
  IF db_user IS NULL OR db_user = '' THEN
    RAISE EXCEPTION 'db_user 未设置';
  END IF;

  -- 表：补齐 deleted_at（如果缺失），并修复 owner/权限
  FOR r IN
    SELECT t.table_name
    FROM information_schema.tables t
    WHERE t.table_schema = 'public' AND t.table_type = 'BASE TABLE'
  LOOP
    -- 补齐 created_at / updated_at / deleted_at（兼容历史表结构）
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns c
      WHERE c.table_schema='public' AND c.table_name=r.table_name AND c.column_name='created_at'
    ) THEN
      EXECUTE format('ALTER TABLE %I ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP', r.table_name);
    END IF;

    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns c
      WHERE c.table_schema='public' AND c.table_name=r.table_name AND c.column_name='updated_at'
    ) THEN
      EXECUTE format('ALTER TABLE %I ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP', r.table_name);
    END IF;

    -- 补齐 deleted_at（兼容历史表结构）
    IF NOT EXISTS (
      SELECT 1 FROM information_schema.columns c
      WHERE c.table_schema='public' AND c.table_name=r.table_name AND c.column_name='deleted_at'
    ) THEN
      EXECUTE format('ALTER TABLE %I ADD COLUMN deleted_at TIMESTAMP', r.table_name);
    END IF;

    -- owner/权限（避免 permission denied）
    EXECUTE format('ALTER TABLE %I OWNER TO %I', r.table_name, db_user);
    EXECUTE format('GRANT ALL PRIVILEGES ON TABLE %I TO %I', r.table_name, db_user);
  END LOOP;

  -- 特定表结构兼容：registry_users.first_login（默认账号首次登录提示）
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns c
    WHERE c.table_schema='public' AND c.table_name='registry_users' AND c.column_name='first_login'
  ) THEN
    EXECUTE 'ALTER TABLE registry_users ADD COLUMN first_login BOOLEAN DEFAULT FALSE';
  END IF;

  -- 序列权限（如果存在）
  FOR r IN
    SELECT sequence_name
    FROM information_schema.sequences
    WHERE sequence_schema='public'
  LOOP
    EXECUTE format('ALTER SEQUENCE %I OWNER TO %I', r.sequence_name, db_user);
    EXECUTE format('GRANT ALL PRIVILEGES ON SEQUENCE %I TO %I', r.sequence_name, db_user);
  END LOOP;
END \$\$;
SQL
}

start_redis() {
  log "启动内置 Redis..."
  # 配置持久化（AOF），数据写到 /data/redis
  # 如果显式设置了 REDIS_PASSWORD，则启用 requirepass，保证与全局配置中心一致
  # - 未设置则保持无密码（默认与 docker-compose.single.yml 一致）
  REDIS_AUTH_ARGS=()
  if [[ -n "${REDIS_PASSWORD:-}" ]]; then
    REDIS_AUTH_ARGS+=(--requirepass "${REDIS_PASSWORD}")
  fi

  redis-server \
    --bind 127.0.0.1 \
    --port "${REDIS_PORT}" \
    --dir /data/redis \
    --appendonly yes \
    --appendfilename "appendonly.aof" \
    --save 900 1 --save 300 10 --save 60 10000 \
    "${REDIS_AUTH_ARGS[@]}" \
    &
  export REDIS_PID=$!
}

shutdown() {
  log "收到退出信号，正在停止..."
  if [[ -n "${REDIS_PID:-}" ]]; then
    kill "${REDIS_PID}" 2>/dev/null || true
  fi
  if [[ -n "${PG_PID:-}" ]]; then
    kill "${PG_PID}" 2>/dev/null || true
  fi
}

trap shutdown SIGTERM SIGINT

fix_permissions

# 先启动内置 Postgres/Redis，再做环境/连通性检测，避免“数据库/Redis 不可达”的误报
start_postgres
init_db_schema_once
fix_db_permissions_and_schema
start_redis

# 兼容已有“自动检测/启动前检查”脚本：
# - detect-container-env.sh 负责打印当前容器环境与依赖状态（仅告警，不阻断启动）
if [[ -f /app/scripts/detect-container-env.sh ]]; then
  /app/scripts/detect-container-env.sh || true
fi

# 执行严格的启动前检查：
# 单镜像模式需要先启动内置 Postgres/Redis，否则连通性检查才有意义。
if [[ -f /app/scripts/pre-start-check.sh ]]; then
  bash /app/scripts/pre-start-check.sh
fi

# 显示服务访问地址信息（简化版，详细信息由主程序输出）
log ""
log "╔════════════════════════════════════════════════════════════╗"
log "║               CYP-Registry 服务启动中                      ║"
log "╠════════════════════════════════════════════════════════════╣"
log "║  应用名称: ${APP_NAME}"
if [[ "${APP_HOST}" == "0.0.0.0" ]]; then
  # 如果监听所有接口，尝试获取容器IP（只获取一次，避免重复）
  CONTAINER_IP=""
  if command -v hostname &> /dev/null; then
    CONTAINER_IP=$(hostname -i 2>/dev/null | awk '{print $1}' || echo "")
  fi
  if [[ -z "${CONTAINER_IP}" ]] && command -v ip &> /dev/null; then
    CONTAINER_IP=$(ip route get 8.8.8.8 2>/dev/null | awk '{print $7}' | head -1 || echo "")
  fi
  if [[ -n "${CONTAINER_IP}" ]]; then
    log "║  容器IP:   ${CONTAINER_IP}"
    log "║  外部访问: http://${CONTAINER_IP}:${APP_PORT}"
  fi
  log "║  本地访问: http://localhost:${APP_PORT}"
else
  log "║  访问地址: http://${APP_HOST}:${APP_PORT}"
fi
log "╚════════════════════════════════════════════════════════════╝"
log ""

log "启动 registry-server..."
exec su-exec appuser:appuser /app/registry-server

