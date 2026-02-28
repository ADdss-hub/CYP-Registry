#!/usr/bin/env bash
# ============================================
# CYP-Registry 单镜像(All-in-One) 入口脚本
# - 启动内置 Postgres + Redis
# - 首次启动初始化 DB（创建用户/库，执行 init-scripts/01-schema.sql）
# - 启动 registry-server
# ============================================

set -euo pipefail

log() { echo "[single] $*"; }
error() { echo "[single] ERROR: $*" >&2; }
warn() { echo "[single] WARN: $*" >&2; }
debug() { [[ "${DEBUG:-0}" == "1" ]] && echo "[single] DEBUG: $*" >&2 || true; }

# ----------------------------
# 跨平台基础能力层（命令/输出差异统一封装）
# ----------------------------
have_cmd() { command -v "$1" >/dev/null 2>&1; }

# 读取 stdin 的第一行（兼容 GNU/BSD/BusyBox）
first_line() {
  head -n 1 2>/dev/null || head -1 2>/dev/null || sed -n '1p' 2>/dev/null || true
}

# 输出文件末尾 N 行（兼容 GNU/BSD/BusyBox）
tail_lines() {
  local n="${1:-50}"
  local file="${2:-}"
  [[ -z "${file}" ]] && return 0
  if tail -n "${n}" "${file}" 2>/dev/null; then
    return 0
  fi
  if tail "-${n}" "${file}" 2>/dev/null; then
    return 0
  fi
  tail "${file}" 2>/dev/null || true
}

# 取某个 glob 最新的一个文件（依赖 ls -t；跨平台 head 差异由 first_line 处理）
latest_file() {
  # 注意：传入 glob（不能整体加引号）
  ls -t $1 2>/dev/null | first_line || true
}

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
# 数据库业务账号用户名：默认从 APP_NAME 生成（与默认管理员账号保持一致）
# 若 APP_NAME 未设置或生成失败，则回退到 "registry"
if [[ -z "${DB_USER:-}" ]]; then
  # 从 APP_NAME 生成数据库用户名（与默认管理员账号逻辑一致）
  # 跨平台兼容：使用 tr 和 sed（POSIX 标准命令）
  # Linux/macOS/Alpine: tr 和 sed 都可用
  # Windows (Git Bash): tr 和 sed 都可用
  app_name="${APP_NAME:-CYP-Registry}"
  # 使用 tr 进行大小写转换（跨平台兼容）
  # 使用 sed 进行字符替换（跨平台兼容，但注意不同 sed 实现的差异）
  db_user_from_app_name=$(echo "$app_name" | tr '[:upper:]' '[:lower:]' 2>/dev/null | \
    sed 's/[^a-z0-9]/-/g' 2>/dev/null | \
    sed 's/--*/-/g' 2>/dev/null | \
    sed 's/^-\|-$//g' 2>/dev/null || echo "registry")
  # 确保以字母/数字开头，长度 3-64
  if [[ -z "$db_user_from_app_name" ]] || [[ ! "$db_user_from_app_name" =~ ^[a-z0-9] ]]; then
    db_user_from_app_name="cyp-${db_user_from_app_name}"
  fi
  if [[ ${#db_user_from_app_name} -gt 64 ]]; then
    db_user_from_app_name="${db_user_from_app_name:0:64}"
  fi
  if [[ ${#db_user_from_app_name} -lt 3 ]]; then
    db_user_from_app_name="cyp-registry"
  fi
  # 最终校验（仅允许字母数字及 _ . -，且需以字母或数字开头）
  if [[ ! "$db_user_from_app_name" =~ ^[a-z0-9][a-z0-9_.-]{2,63}$ ]]; then
    db_user_from_app_name="cyp-registry"
  fi
  export DB_USER="$db_user_from_app_name"
else
  export DB_USER
fi
export DB_NAME="${DB_NAME:-registry_db}"
export DB_SSLMODE="${DB_SSLMODE:-disable}"

# ----------------------------
# PostgreSQL 数据目录：支持挂载点场景（Windows Docker 卷挂载）
# 提前检测并确定数据目录，确保密钥文件路径正确
# ----------------------------
determine_pg_data_dir() {
  local base_dir="/var/lib/postgresql/data"
  
  # 如果目录不存在，直接返回基础路径
  if [[ ! -d "${base_dir}" ]]; then
    debug "数据目录不存在，使用基础路径: ${base_dir}"
    echo "${base_dir}"
    return
  fi
  
  # 如果已经初始化过 PostgreSQL（存在 PG_VERSION），直接使用基础路径
  if [[ -f "${base_dir}/PG_VERSION" ]] && [[ -s "${base_dir}/PG_VERSION" ]]; then
    debug "检测到已初始化的 PostgreSQL 数据目录: ${base_dir}"
    echo "${base_dir}"
    return
  fi
  
  # 检查目录是否可写（处理只读挂载点场景）
  if [[ ! -w "${base_dir}" ]]; then
    warn "数据目录不可写，尝试使用子目录"
    echo "${base_dir}/pgdata"
    return
  fi
  
  # 检查目录内容：NAS 环境（群晖/QNAP）挂载点可能包含系统隐藏文件
  # Linux/macOS/Windows Docker 卷挂载也可能包含系统文件
  # 如果目录不为空且没有 PostgreSQL 文件，可能是挂载点，需要使用子目录
  local file_count=0
  local has_pg_file=false
  local has_system_files_only=true
  
  # 统计目录中的文件（包括隐藏文件）
  # 使用 find 命令更可靠，兼容各种文件系统（ext4、xfs、btrfs、zfs 等）
  # Linux 发行版兼容性：
  # - Ubuntu/Debian: GNU findutils，支持 -print0
  # - CentOS/RHEL: GNU findutils，支持 -print0
  # - Alpine: BusyBox find，支持 -print0（但功能可能有限）
  # - SUSE: GNU findutils，支持 -print0
  if command -v find >/dev/null 2>&1; then
    while IFS= read -r -d '' item; do
      [[ -z "${item}" ]] && continue
      file_count=$((file_count + 1))
      local basename_item=$(basename "${item}")
      
      # 检查是否是 PostgreSQL 初始化文件
      if [[ "${basename_item}" == "PG_VERSION" ]] || \
         [[ "${basename_item}" == "postgresql.conf" ]] || \
         [[ "${basename_item}" == "pg_hba.conf" ]] || \
         ([[ -d "${item}" ]] && [[ "${basename_item}" == "base" ]]) || \
         ([[ -d "${item}" ]] && [[ "${basename_item}" == "global" ]]); then
        has_pg_file=true
        has_system_files_only=false
        break
      fi
      
      # 检查是否是系统隐藏文件（NAS/Windows 常见）
      # 如果发现非系统文件，标记为需要子目录
      if [[ ! "${basename_item}" =~ ^\.(@__|DS_Store|@__thumb|@__qnap|@__syno) ]]; then
        has_system_files_only=false
      fi
    done < <(find "${base_dir}" -mindepth 1 -maxdepth 1 -print0 2>/dev/null || true)
  else
    # fallback: 使用 ls（兼容性更好但可能不够准确）
    local items
    items=$(ls -A "${base_dir}" 2>/dev/null || true)
    if [[ -n "${items}" ]]; then
      while IFS= read -r item; do
        [[ -z "${item}" ]] && continue
        file_count=$((file_count + 1))
        local basename_item="${item}"
        
        if [[ "${basename_item}" == "PG_VERSION" ]] || \
           [[ "${basename_item}" == "postgresql.conf" ]] || \
           [[ "${basename_item}" == "pg_hba.conf" ]] || \
           [[ "${basename_item}" == "base" ]] || \
           [[ "${basename_item}" == "global" ]]; then
          has_pg_file=true
          has_system_files_only=false
          break
        fi
        
        if [[ ! "${basename_item}" =~ ^\.(@__|DS_Store|@__thumb|@__qnap|@__syno) ]]; then
          has_system_files_only=false
        fi
      done <<< "${items}"
    fi
  fi
  
  # 如果目录不为空但没有 PostgreSQL 文件，说明是挂载点（NAS/Windows 环境常见情况）
  # 需要在子目录中初始化 PostgreSQL
  if [[ ${file_count} -gt 0 ]] && [[ "${has_pg_file}" == false ]]; then
    debug "检测到挂载点场景（文件数: ${file_count}, 仅系统文件: ${has_system_files_only}），使用子目录"
    echo "${base_dir}/pgdata"
  else
    debug "使用基础数据目录: ${base_dir}"
    echo "${base_dir}"
  fi
}

PG_DATA_DIR="${PG_DATA_DIR:-$(determine_pg_data_dir)}"

# ----------------------------
# 关键密钥：自动生成 + 持久化（避免 pre-start-check 阻断启动）
# 说明：
# - 生产环境"最佳实践"仍然是外部显式注入；但为了避免因为缺失而导致服务不可用，
#   单镜像模式下我们会在缺失时自动生成并写入数据卷内的 secrets 文件。
# - 持久化位置选择 PostgreSQL 数据目录（compose 中为持久卷），确保重启/升级不变。
# ----------------------------
SECRETS_DIR="${PG_DATA_DIR}"
DB_PASSWORD_FILE="${SECRETS_DIR}/.cyp_registry_db_password"
JWT_SECRET_FILE="${SECRETS_DIR}/.cyp_registry_jwt_secret"

ensure_secret_file() {
  local file="$1"
  local value="$2"
  # 保存当前 umask
  local old_umask=$(umask)
  # 设置严格的 umask（077 = 仅所有者可读写）
  umask 077
  mkdir -p "${SECRETS_DIR}" 2>/dev/null || true
  # 仅当文件不存在或为空时写入，避免覆盖既有密钥
  if [[ ! -s "${file}" ]]; then
    printf '%s' "${value}" > "${file}"
    # 确保文件权限为 600（仅所有者可读写）
    # Linux 环境下，umask 077 应该已经设置了正确的权限，但显式设置更安全
    chmod 600 "${file}" 2>/dev/null || true
  fi
  # 恢复原始 umask
  umask "${old_umask}" 2>/dev/null || true
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
  log "WARN: 未设置 DB_PASSWORD，已自动生成并持久化到 ${DB_PASSWORD_FILE}（该提示仅首次显示一次；建议生产环境显式注入以便审计/轮换）"
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
  # 跨平台随机数生成兼容性：
  # - Linux/macOS: openssl rand, /dev/urandom + od
  # - Alpine/BusyBox: openssl(若安装), /dev/urandom + od
  # - Windows (Git Bash/WSL): openssl(若安装), /dev/urandom(WSL), date + sha256sum/shasum
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32 2>/dev/null || echo ""
  elif [ -r /dev/urandom ] && command -v od >/dev/null 2>&1; then
    od -An -N32 -tx1 /dev/urandom 2>/dev/null | tr -d ' \n' || echo ""
  elif command -v date >/dev/null 2>&1 && command -v sha256sum >/dev/null 2>&1; then
    date +%s 2>/dev/null | sha256sum 2>/dev/null | awk '{print $1}' || echo ""
  elif command -v date >/dev/null 2>&1 && command -v shasum >/dev/null 2>&1; then
    date +%s 2>/dev/null | shasum -a 256 2>/dev/null | awk '{print $1}' || echo ""
  else
    # 最后兜底：简单时间戳 + /dev/urandom 组合（若可用）
    {
      date +%s 2>/dev/null || echo "0"
      if [ -r /dev/urandom ] && command -v od >/dev/null 2>&1; then
        od -An -N32 -tx1 /dev/urandom 2>/dev/null | tr -d ' \n'
      fi
    } | tr -d '\n' | head -c 64
  fi
}

# JWT_SECRET：必须通过 .env/外部配置显式提供，缺失会在 pre-start-check.sh 中被视为致命错误
if [[ -z "${JWT_SECRET:-}" ]]; then
  JWT_SECRET="$(read_secret_file "${JWT_SECRET_FILE}")"
fi
if [[ -z "${JWT_SECRET:-}" ]]; then
  JWT_SECRET="$(gen_random_hex)"
  ensure_secret_file "${JWT_SECRET_FILE}" "${JWT_SECRET}"
  log "WARN: 未设置 JWT_SECRET，已自动生成并持久化到 ${JWT_SECRET_FILE}（该提示仅首次显示一次；建议生产环境显式注入以便审计/轮换）"
fi
export JWT_SECRET

# ----------------------------
# config.yaml：若未挂载/不存在则自动生成（单镜像默认不提交 config.yaml）
# 说明：
# - 后端会尝试加载 config.yaml；即使不存在也会回退到默认配置，但 pre-start-check.sh 会严格要求文件存在
# - 因此在单镜像入口脚本中保证 /app/config.yaml 一定存在
# - 生产环境建议通过 volume 挂载自定义 config.yaml 覆盖本文件
# ----------------------------
CONFIG_FILE="${CONFIG_FILE:-/app/config.yaml}"
# 生成日志只提示一次（持久化到数据卷；避免每次容器重建/误删配置后反复刷屏）
CONFIG_GENERATED_MARKER="${SECRETS_DIR}/.cyp_registry_config_generated"

yaml_squote() {
  # YAML 单引号字符串：内部单引号需要写成两个单引号
  local s="${1:-}"
  s="${s//\'/''}"
  printf "'%s'" "$s"
}

generate_config_yaml_if_missing() {
  if [[ -f "${CONFIG_FILE}" ]]; then
    return 0
  fi

  if [[ ! -f "${CONFIG_GENERATED_MARKER}" ]]; then
    log "未检测到配置文件 ${CONFIG_FILE}，将根据当前环境变量自动生成（单镜像默认；该提示仅首次显示）"
    # 保存当前 umask
    local old_umask=$(umask)
    # 配置文件不需要严格权限，使用默认 umask
    umask 0022
    touch "${CONFIG_GENERATED_MARKER}" 2>/dev/null || true
    # 恢复原始 umask
    umask "${old_umask}" 2>/dev/null || true
  fi
  mkdir -p "$(dirname "${CONFIG_FILE}")" 2>/dev/null || true

  # 注意：这里写入的是“可运行的默认配置”，具体值仍可被环境变量覆盖（src/pkg/config/applyEnvOverrides）
  cat > "${CONFIG_FILE}" <<YAML
# CYP-Registry 配置文件（由单镜像入口脚本自动生成）
app:
  name: $(yaml_squote "${APP_NAME}")
  host: $(yaml_squote "${APP_HOST}")
  port: ${APP_PORT}
  env: $(yaml_squote "${APP_ENV}")
  debug: false

database:
  host: $(yaml_squote "${DB_HOST}")
  port: ${DB_PORT}
  username: $(yaml_squote "${DB_USER}")
  password: $(yaml_squote "${DB_PASSWORD}")
  name: $(yaml_squote "${DB_NAME}")
  sslmode: $(yaml_squote "${DB_SSLMODE}")
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 300

redis:
  host: $(yaml_squote "${REDIS_HOST}")
  port: ${REDIS_PORT}
  password: $(yaml_squote "${REDIS_PASSWORD:-}")
  db: ${REDIS_DB}
  pool_size: 100
  min_idle_conns: 10
  key_prefix: $(yaml_squote "cyp:registry:")

auth:
  jwt:
    access_token_expire: 7200
    refresh_token_expire: 604800
    secret: $(yaml_squote "${JWT_SECRET}")
  pat:
    prefix: $(yaml_squote "pat_v1_")
    expire: 2592000
  bcrypt_cost: 12

storage:
  type: $(yaml_squote "${STORAGE_TYPE:-local}")
  local:
    root_path: $(yaml_squote "${STORAGE_LOCAL_ROOT_PATH:-/data/storage}")
  minio:
    endpoint: $(yaml_squote "${STORAGE_MINIO_ENDPOINT:-localhost:9000}")
    access_key: $(yaml_squote "${STORAGE_MINIO_ACCESS_KEY:-minioadmin}")
    secret_key: $(yaml_squote "${STORAGE_MINIO_SECRET_KEY:-minioadmin}")
    bucket: $(yaml_squote "${STORAGE_MINIO_BUCKET:-registry}")
    use_ssl: false

registry:
  max_layer_size: 107374182400
  allow_anonymous: false
  token_expire: 300

security:
  rate_limit:
    enabled: true
    requests_per_second: 100
    burst: 200
  brute_force:
    max_attempts_per_minute: 10
    lockout_duration: 86400
    max_attempts_per_ip: 50
  cors:
    allowed_origins:
      - $(yaml_squote "http://localhost:3000")
      - $(yaml_squote "http://localhost:8080")
    allowed_methods:
      - $(yaml_squote "GET")
      - $(yaml_squote "POST")
      - $(yaml_squote "PUT")
      - $(yaml_squote "DELETE")
      - $(yaml_squote "OPTIONS")
    allowed_headers:
      - $(yaml_squote "Authorization")
      - $(yaml_squote "Content-Type")
      - $(yaml_squote "X-Requested-With")

logging:
  level: $(yaml_squote "info")
  format: $(yaml_squote "json")
  output: $(yaml_squote "stdout")
  file:
    path: $(yaml_squote "./logs/app.log")
    max_size: 100
    max_age: 30
    max_backups: 10
  trace:
    enabled: true
    sample_rate: 1.0

scanner:
  enabled: true
  severity:
    - $(yaml_squote "CRITICAL")
    - $(yaml_squote "HIGH")
    - $(yaml_squote "MEDIUM")
  block_on_critical: true
  async: true

webhook:
  max_retries: 3
  timeout: 30
  signature_secret: $(yaml_squote "")
YAML
}

fix_permissions() {
  # 确保基础目录存在（不依赖 PG_DATA_DIR，因为可能在 start_postgres 中会调整）
  local dirs=("/data/storage" "/data/redis" "/app/logs" "/var/lib/postgresql" "/run/postgresql")
  for dir in "${dirs[@]}"; do
    if ! mkdir -p "${dir}" 2>/dev/null; then
      warn "无法创建目录: ${dir}（某些环境可能不需要）"
    fi
  done
  
  # 设置 umask 确保新创建的文件有正确的权限（Linux 环境兼容性）
  # 保存当前 umask，避免影响后续操作
  local old_umask=$(umask)
  umask 0022  # 默认：目录 755，文件 644
  
  # 设置应用用户目录权限（多次尝试，兼容不同的权限模型）
  # 兼容性说明：
  # - Ubuntu/Debian: 标准 chown/chmod
  # - CentOS/RHEL: 可能受 SELinux 影响，需要 chcon（容器内通常不需要）
  # - Alpine: BusyBox 工具集，命令参数可能略有不同
  # - SUSE: 标准 Linux 工具
  local app_dirs=("/data/storage" "/data/redis" "/app/logs")
  for dir in "${app_dirs[@]}"; do
    local attempts=0
    while [[ ${attempts} -lt 3 ]]; do
      # 优先使用 chown，失败时尝试 chmod（兼容只读文件系统或特殊权限模型）
      if chown -R appuser:appuser "${dir}" 2>/dev/null; then
        # 确保目录权限正确（Linux 标准：目录 755）
        chmod 755 "${dir}" 2>/dev/null || true
        break
      elif chmod -R 755 "${dir}" 2>/dev/null; then
        # Fallback：仅设置权限，不改变所有者（某些容器环境可能不需要 chown）
        debug "使用 chmod 设置 ${dir} 权限（chown 不可用）"
        break
      fi
      attempts=$((attempts + 1))
      [[ ${attempts} -lt 3 ]] && sleep 0.3 || debug "无法设置 ${dir} 权限（某些环境可能不需要）"
    done
  done
  
  # 确保 PostgreSQL 相关目录权限正确（包括挂载点场景）
  # PostgreSQL 需要 postgres 用户和组，这在 Alpine 镜像中已预配置
  local pg_dirs=("/var/lib/postgresql" "/run/postgresql")
  for dir in "${pg_dirs[@]}"; do
    local attempts=0
    while [[ ${attempts} -lt 3 ]]; do
      # PostgreSQL 目录需要 postgres:postgres 所有者和 700 权限（安全）
      if chown -R postgres:postgres "${dir}" 2>/dev/null; then
        chmod 700 "${dir}" 2>/dev/null || true
        break
      elif chmod 700 "${dir}" 2>/dev/null; then
        debug "使用 chmod 设置 ${dir} 权限（chown 不可用）"
        break
      fi
      attempts=$((attempts + 1))
      [[ ${attempts} -lt 3 ]] && sleep 0.3 || debug "无法设置 ${dir} 权限（某些环境可能不需要）"
    done
  done
  
  # 恢复原始 umask
  umask "${old_umask}" 2>/dev/null || true
}

start_postgres() {
  log "启动内置 PostgreSQL..."

  # PostgreSQL 数据目录已在脚本开头确定（支持 NAS/Windows 挂载点场景）
  # 如果检测到挂载点场景，记录日志
  if [[ "${PG_DATA_DIR}" == */pgdata ]]; then
    log "检测到挂载点场景（NAS/Windows Docker 卷），使用子目录：${PG_DATA_DIR}"
    log "提示：这是正常行为，用于避免在挂载点直接初始化 PostgreSQL 数据目录"
  fi

  # 确保数据目录存在且权限正确
  if ! mkdir -p "${PG_DATA_DIR}" /run/postgresql; then
    error "无法创建数据目录: ${PG_DATA_DIR}"
    exit 1
  fi
  
  # 设置权限（多次尝试，兼容不同的权限模型）
  # 先设置数据目录和 socket 目录权限
  local chown_attempts=0
  while [[ ${chown_attempts} -lt 3 ]]; do
    if chown -R postgres:postgres "${PG_DATA_DIR}" /run/postgresql 2>/dev/null; then
      break
    fi
    chown_attempts=$((chown_attempts + 1))
    if [[ ${chown_attempts} -lt 3 ]]; then
      sleep 0.5
    else
      warn "无法设置数据目录权限，继续尝试（某些环境可能不需要）"
    fi
  done
  
  # 设置日志目录（在数据目录下）
  local pg_log_dir="${PG_DATA_DIR}/log"
  if ! mkdir -p "${pg_log_dir}" 2>/dev/null; then
    warn "无法创建日志目录: ${pg_log_dir}，将使用数据目录"
    pg_log_dir="${PG_DATA_DIR}"
  fi
  
  # 设置日志目录权限
  local log_chown_attempts=0
  while [[ ${log_chown_attempts} -lt 3 ]]; do
    chown postgres:postgres "${pg_log_dir}" 2>/dev/null || true
    chmod 755 "${pg_log_dir}" 2>/dev/null || true
    # 使用 postgres 用户实际验证可写性（避免 root 的 -w 误判）
    if su-exec postgres:postgres sh -c "test -w '${pg_log_dir}'" >/dev/null 2>&1; then
      break
    fi
    log_chown_attempts=$((log_chown_attempts + 1))
    [[ ${log_chown_attempts} -lt 3 ]] && sleep 0.3 || warn "无法设置日志目录权限，继续尝试"
  done
  
  # 若数据目录下日志目录仍不可写，则回退到 /app/logs/postgres 或 /tmp（更适合 NAS/Windows 权限模型）
  if ! su-exec postgres:postgres sh -c "test -w '${pg_log_dir}'" >/dev/null 2>&1; then
    warn "数据目录下日志目录对 postgres 不可写，回退到 /app/logs/postgres 或 /tmp"
    if mkdir -p /app/logs/postgres 2>/dev/null; then
      chown -R postgres:postgres /app/logs/postgres 2>/dev/null || true
      chmod 755 /app/logs/postgres 2>/dev/null || true
      if su-exec postgres:postgres sh -c "test -w /app/logs/postgres" >/dev/null 2>&1; then
        pg_log_dir="/app/logs/postgres"
      fi
    fi
  fi
  if ! su-exec postgres:postgres sh -c "test -w '${pg_log_dir}'" >/dev/null 2>&1; then
    if mkdir -p /tmp/postgres-logs 2>/dev/null; then
      chown -R postgres:postgres /tmp/postgres-logs 2>/dev/null || true
      chmod 755 /tmp/postgres-logs 2>/dev/null || true
      if su-exec postgres:postgres sh -c "test -w /tmp/postgres-logs" >/dev/null 2>&1; then
        pg_log_dir="/tmp/postgres-logs"
      fi
    fi
  fi

  # 初始化数据库目录（仅首次）
  if [[ ! -s "${PG_DATA_DIR}/PG_VERSION" ]]; then
    log "PostgreSQL 数据目录未初始化，正在 initdb..."
    
    # 检查目录是否可写
    if [[ ! -w "${PG_DATA_DIR}" ]]; then
      error "数据目录不可写: ${PG_DATA_DIR}"
      exit 1
    fi
    
    # 使用更安全的认证方式初始化：
    # - 本地连接使用 peer（系统用户 postgres 通过本地 Unix Socket 免密登录）
    # - TCP 连接使用 scram-sha-256（应用通过密码访问）
    if ! su-exec postgres:postgres initdb \
      --auth-local=peer \
      --auth-host=scram-sha-256 \
      -D "${PG_DATA_DIR}" >/dev/null 2>&1; then
      error "PostgreSQL initdb 失败，请检查数据目录权限和磁盘空间"
      error "数据目录: ${PG_DATA_DIR}"
      exit 1
    fi

    # 允许本地连接（单机容器内）
    if ! echo "listen_addresses = '127.0.0.1'" >> "${PG_DATA_DIR}/postgresql.conf" 2>/dev/null; then
      error "无法写入 postgresql.conf"
      exit 1
    fi
    
    if ! echo "port = ${DB_PORT}" >> "${PG_DATA_DIR}/postgresql.conf" 2>/dev/null; then
      error "无法写入 postgresql.conf"
      exit 1
    fi

    # 确保 127.0.0.1 使用基于密码的安全认证（应用侧使用 registry/DB_PASSWORD 登录）
    if ! echo "host all all 127.0.0.1/32 scram-sha-256" >> "${PG_DATA_DIR}/pg_hba.conf" 2>/dev/null; then
      error "无法写入 pg_hba.conf"
      exit 1
    fi
    
    log "PostgreSQL 数据目录初始化完成"
  fi

  # 启动 PostgreSQL（后台运行）
  # 关键：优先保证“能启动”，日志按可用性降级（避免因日志文件权限导致 postgres 根本起不来）
  local pg_log_file="${pg_log_dir}/postgresql-$(date +%Y%m%d-%H%M%S).log"
  local can_use_log_file=0
  if su-exec postgres:postgres sh -c "umask 0077; : >'${pg_log_file}'" >/dev/null 2>&1; then
    can_use_log_file=1
  fi
  if [[ ${can_use_log_file} -eq 1 ]]; then
    log "启动 PostgreSQL，日志文件: ${pg_log_file}"
  else
    warn "无法在 ${pg_log_dir} 创建/写入 PostgreSQL 日志文件，将输出到容器标准输出"
  fi

  # 配置 PostgreSQL 日志（仅当日志目录可写时启用 collector；否则强制 stderr）
  if [[ -f "${PG_DATA_DIR}/postgresql.conf" ]]; then
    if [[ ${can_use_log_file} -eq 1 ]]; then
      if [[ "${pg_log_dir}" == "${PG_DATA_DIR}/log" ]]; then
        grep -q "^log_directory" "${PG_DATA_DIR}/postgresql.conf" 2>/dev/null || \
          su-exec postgres:postgres sh -c "echo \"log_directory = 'log'\" >>'${PG_DATA_DIR}/postgresql.conf'" 2>/dev/null || true
      else
        grep -q "^log_directory" "${PG_DATA_DIR}/postgresql.conf" 2>/dev/null || \
          su-exec postgres:postgres sh -c "echo \"log_directory = '${pg_log_dir}'\" >>'${PG_DATA_DIR}/postgresql.conf'" 2>/dev/null || true
      fi
      grep -q "^logging_collector" "${PG_DATA_DIR}/postgresql.conf" 2>/dev/null || \
        su-exec postgres:postgres sh -c "echo \"logging_collector = on\" >>'${PG_DATA_DIR}/postgresql.conf'" 2>/dev/null || true
    else
      grep -q "^logging_collector" "${PG_DATA_DIR}/postgresql.conf" 2>/dev/null || \
        su-exec postgres:postgres sh -c "echo \"logging_collector = off\" >>'${PG_DATA_DIR}/postgresql.conf'" 2>/dev/null || true
      grep -q "^log_destination" "${PG_DATA_DIR}/postgresql.conf" 2>/dev/null || \
        su-exec postgres:postgres sh -c "echo \"log_destination = 'stderr'\" >>'${PG_DATA_DIR}/postgresql.conf'" 2>/dev/null || true
    fi
  fi

  # Socket 目录权限（部分环境对 700/继承权限敏感，统一到 775 更稳）
  chmod 775 /run/postgresql 2>/dev/null || true
  chown postgres:postgres /run/postgresql 2>/dev/null || true

  log "执行启动命令: postgres -D ${PG_DATA_DIR} -k /run/postgresql"
  if [[ ${can_use_log_file} -eq 1 ]]; then
    # 使用 sh -c 包装命令，确保重定向在各种 su-exec/BusyBox 环境下工作
    su-exec postgres:postgres sh -c "exec postgres -D '${PG_DATA_DIR}' -k /run/postgresql >>'${pg_log_file}' 2>&1" &
  else
    su-exec postgres:postgres postgres -D "${PG_DATA_DIR}" -k /run/postgresql &
  fi
  
  PG_PID=$!
  
  # 验证进程是否真的启动
  if [[ -z "${PG_PID}" ]] || [[ "${PG_PID}" -eq 0 ]]; then
    error "无法获取 PostgreSQL 进程 PID"
    error "请检查 su-exec 和 postgres 命令是否可用"
    
    # 诊断信息
    if ! command -v su-exec >/dev/null 2>&1; then
      error "su-exec 命令不存在"
    fi
    if ! command -v postgres >/dev/null 2>&1; then
      error "postgres 命令不存在"
    fi
    if ! id postgres >/dev/null 2>&1; then
      error "postgres 用户不存在"
    fi
    
    exit 1
  fi
  
  log "PostgreSQL 进程已启动，PID: ${PG_PID}"
  
  # 等待进程启动（Windows/NAS 环境可能需要更长时间）
  log "等待 PostgreSQL 进程启动..."
  sleep 3
  
  # 检查进程是否真的在运行（兼容 Alpine/BusyBox）
  # 使用 kill -0 作为主要检测方式（最兼容）
  if ! kill -0 "${PG_PID}" 2>/dev/null; then
    # 尝试使用 ps 命令作为辅助检测（如果可用）
    local ps_check=false
    if command -v ps >/dev/null 2>&1; then
      # 跨平台进程检测兼容性：
      # - Linux (GNU coreutils): ps aux, ps -p, ps -o pid=
      # - Alpine/BusyBox: ps aux (可能不支持 ps -p)
      # - macOS: ps aux, ps -p (BSD ps)
      # 优先使用 ps aux（兼容性最好），fallback 到 ps -p
      if ps aux >/dev/null 2>&1; then
        # 使用 ps aux（最兼容）
        if ps aux 2>/dev/null | grep -q "[[:space:]]${PG_PID}[[:space:]]"; then
          ps_check=true
        fi
      elif ps -o pid= -p "${PG_PID}" >/dev/null 2>&1; then
        # Fallback: 使用 ps -o pid=（GNU/BSD ps）
        ps_check=true
      elif ps -p "${PG_PID}" >/dev/null 2>&1; then
        # Fallback: 使用 ps -p（某些系统）
        ps_check=true
      fi
    fi
    
    if [[ "${ps_check}" == false ]]; then
      error "PostgreSQL 进程未运行（PID: ${PG_PID}）"
      error "请检查日志文件: ${pg_log_file}"
      if [[ -f "${pg_log_file}" ]]; then
        error "日志内容："
        cat "${pg_log_file}" >&2 || true
      fi
      exit 1
    fi
  fi
  
  # 检查进程是否还在运行
  if ! kill -0 "${PG_PID}" 2>/dev/null; then
    error "PostgreSQL 进程启动后立即退出（PID: ${PG_PID}）"
    error "数据目录: ${PG_DATA_DIR}"
    error "日志文件: ${pg_log_file}"
    
    # 输出日志文件内容（最后 50 行）
    if [[ -f "${pg_log_file}" ]]; then
      error "PostgreSQL 启动日志（最后 50 行）："
      # 跨平台 tail 兼容性：优先使用 tail -n，fallback 到 tail
      if tail -n 50 "${pg_log_file}" >&2 2>/dev/null; then
        : # 成功
      elif tail -50 "${pg_log_file}" >&2 2>/dev/null; then
        : # Fallback: BSD tail 格式
      else
        tail "${pg_log_file}" >&2 2>/dev/null || true
      fi
    else
      # 如果没有日志文件，尝试检查标准错误
      error "未找到日志文件，可能的原因："
      error "1. 数据目录权限问题: ${PG_DATA_DIR}"
      error "2. 日志目录权限问题: ${pg_log_dir}"
      error "3. postgres 用户无法执行 postgres 命令"
      
      # 检查数据目录权限（兼容不同 Linux 发行版的 stat 命令）
      # Linux 发行版兼容性：
      # - Ubuntu/Debian/CentOS/RHEL/SUSE: GNU coreutils，使用 stat -c "%a"
      # - Alpine: BusyBox stat，可能不支持 -c，使用 ls 作为 fallback
      # - macOS: BSD stat，使用 stat -f "%OLp"
      if [[ -d "${PG_DATA_DIR}" ]]; then
        local dir_perm="unknown"
        # 尝试使用 stat 命令（GNU/Linux 格式 - 大多数 Linux 发行版）
        if stat -c "%a" "${PG_DATA_DIR}" >/dev/null 2>&1; then
          dir_perm=$(stat -c "%a" "${PG_DATA_DIR}" 2>/dev/null)
        # 尝试使用 stat 命令（BSD/macOS 格式）
        elif stat -f "%OLp" "${PG_DATA_DIR}" >/dev/null 2>&1; then
          dir_perm=$(stat -f "%OLp" "${PG_DATA_DIR}" 2>/dev/null)
        # Fallback：使用 ls 命令（Alpine/BusyBox 兼容）
        else
          dir_perm=$(ls -ld "${PG_DATA_DIR}" 2>/dev/null | awk '{print $1}' || echo "unknown")
        fi
        error "数据目录权限: ${dir_perm}"
        
        # 检查目录是否可写
        if [[ ! -w "${PG_DATA_DIR}" ]]; then
          error "数据目录不可写: ${PG_DATA_DIR}"
        fi
      fi
      
      # 检查 postgres 用户和命令
      if ! id postgres >/dev/null 2>&1; then
        error "postgres 用户不存在"
      fi
      
      if ! command -v postgres >/dev/null 2>&1; then
        error "postgres 命令不存在"
      fi
    fi
    
    exit 1
  fi
  
  log "PostgreSQL 进程已启动（PID: ${PG_PID}）"

  # 等待就绪（增加重试次数和详细日志）
  local max_attempts=60
  local attempt=0
  log "等待 PostgreSQL 就绪（最多 ${max_attempts} 秒）..."
  
  while [[ ${attempt} -lt ${max_attempts} ]]; do
    if su-exec postgres:postgres pg_isready -h /run/postgresql -p "${DB_PORT}" >/dev/null 2>&1; then
      log "PostgreSQL 就绪（耗时 ${attempt} 秒）"
      break
    fi
    
    # 检查进程是否还在运行
    if ! kill -0 "${PG_PID}" 2>/dev/null; then
      error "PostgreSQL 进程意外退出（PID: ${PG_PID}）"
      error "数据目录: ${PG_DATA_DIR}"
      
      # 查找最新的日志文件（跨平台兼容）
      local latest_log=""
      if [[ -d "${pg_log_dir}" ]]; then
        latest_log="$(latest_file "${pg_log_dir}"/*.log)"
      fi
      if [[ -n "${latest_log}" ]] && [[ -f "${latest_log}" ]]; then
        error "PostgreSQL 日志文件: ${latest_log}"
        error "日志内容（最后 30 行）："
        # 跨平台 tail 兼容性
        if tail -n 30 "${latest_log}" >&2 2>/dev/null; then
          : # 成功
        elif tail -30 "${latest_log}" >&2 2>/dev/null; then
          : # Fallback: BSD tail 格式
        else
          tail "${latest_log}" >&2 2>/dev/null || true
        fi
      else
        error "未找到 PostgreSQL 日志文件"
        error "请检查数据目录权限: ${PG_DATA_DIR}"
      fi
      
      exit 1
    fi
    
    attempt=$((attempt + 1))
    if [[ $((attempt % 10)) -eq 0 ]]; then
      debug "PostgreSQL 仍在启动中... (${attempt}/${max_attempts})"
    fi
    sleep 1
  done

  if ! su-exec postgres:postgres pg_isready -h /run/postgresql -p "${DB_PORT}" >/dev/null 2>&1; then
    error "PostgreSQL 启动超时（${max_attempts} 秒后仍未就绪）"
    error "数据目录: ${PG_DATA_DIR}"
    error "进程 PID: ${PG_PID}"
    
    # 检查进程状态
    if kill -0 "${PG_PID}" 2>/dev/null; then
      error "PostgreSQL 进程仍在运行，但无法连接"
      error "可能的原因："
      error "1. PostgreSQL 配置问题（端口、监听地址）"
      error "2. Socket 文件权限问题: /run/postgresql"
      error "3. 网络连接问题"
    else
      error "PostgreSQL 进程已退出"
    fi
    
    # 输出日志（跨平台兼容）
    local latest_log=""
    if [[ -d "${pg_log_dir}" ]]; then
      latest_log="$(latest_file "${pg_log_dir}"/*.log)"
    fi
    
    if [[ -n "${latest_log}" ]] && [[ -f "${latest_log}" ]]; then
      error "PostgreSQL 日志文件: ${latest_log}"
      error "日志内容（最后 50 行）："
      # 跨平台 tail 兼容性
      if tail -n 50 "${latest_log}" >&2 2>/dev/null; then
        : # 成功
      elif tail -50 "${latest_log}" >&2 2>/dev/null; then
        : # Fallback: BSD tail 格式
      else
        tail "${latest_log}" >&2 2>/dev/null || true
      fi
    elif [[ -f "${pg_log_file}" ]]; then
      error "PostgreSQL 日志文件: ${pg_log_file}"
      error "日志内容（最后 50 行）："
      # 跨平台 tail 兼容性
      if tail -n 50 "${pg_log_file}" >&2 2>/dev/null; then
        : # 成功
      elif tail -50 "${pg_log_file}" >&2 2>/dev/null; then
        : # Fallback: BSD tail 格式
      else
        tail "${pg_log_file}" >&2 2>/dev/null || true
      fi
    fi
    
    exit 1
  fi

  export PG_PID
}

init_db_schema_once() {
  # 用一个 marker 文件确保只初始化一次
  # 使用 PG_DATA_DIR（在 start_postgres 中已设置）
  local marker="${PG_DATA_DIR:-/var/lib/postgresql/data}/.cyp_registry_initialized"
  if [[ -f "$marker" ]]; then
    debug "数据库已初始化，跳过初始化步骤"
    return 0
  fi

  log "初始化数据库用户/库并执行 schema（仅首次）..."

  # 等待 PostgreSQL 完全就绪（额外等待，确保可以接受连接）
  local wait_count=0
  while [[ ${wait_count} -lt 10 ]]; do
    if su-exec postgres:postgres psql -h /run/postgresql -p "${DB_PORT}" -d postgres -c "SELECT 1" >/dev/null 2>&1; then
      break
    fi
    wait_count=$((wait_count + 1))
    sleep 1
  done

  # 创建用户（若不存在）
  if ! su-exec postgres:postgres psql -h /run/postgresql -p "${DB_PORT}" -d postgres -v ON_ERROR_STOP=1 <<SQL
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '${DB_USER}') THEN
    CREATE ROLE ${DB_USER} LOGIN PASSWORD '${DB_PASSWORD}';
  END IF;
END
\$\$;
SQL
  then
    error "创建数据库用户失败: ${DB_USER}"
    exit 1
  fi

  # 执行初始化脚本（包含 CREATE DATABASE / \c）
  if [[ ! -f /app/init-scripts/01-schema.sql ]]; then
    error "初始化脚本不存在: /app/init-scripts/01-schema.sql"
    exit 1
  fi
  
  if ! su-exec postgres:postgres psql -h /run/postgresql -p "${DB_PORT}" -d postgres -v ON_ERROR_STOP=1 -f /app/init-scripts/01-schema.sql; then
    error "执行数据库初始化脚本失败"
    exit 1
  fi

  # 创建标记文件
  if ! touch "$marker" 2>/dev/null; then
    warn "无法创建初始化标记文件: $marker（可能影响后续启动）"
  fi
  
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
  
  # 确保 Redis 数据目录存在
  if ! mkdir -p /data/redis; then
    error "无法创建 Redis 数据目录: /data/redis"
    exit 1
  fi
  
  # 设置权限
  local attempts=0
  while [[ ${attempts} -lt 3 ]]; do
    if chown -R appuser:appuser /data/redis 2>/dev/null || chmod 755 /data/redis 2>/dev/null; then
      break
    fi
    attempts=$((attempts + 1))
    [[ ${attempts} -lt 3 ]] && sleep 0.3 || warn "无法设置 Redis 数据目录权限（继续尝试）"
  done
  
  # 配置持久化（AOF），数据写到 /data/redis
  # 如果显式设置了 REDIS_PASSWORD，则启用 requirepass，保证与全局配置中心一致
  # - 未设置则保持无密码（默认与 docker-compose.single.yml 一致）
  REDIS_AUTH_ARGS=()
  if [[ -n "${REDIS_PASSWORD:-}" ]]; then
    REDIS_AUTH_ARGS+=(--requirepass "${REDIS_PASSWORD}")
  fi

  # 创建 Redis 日志目录（使用局部变量，确保作用域正确）
  local redis_log_dir="/data/redis/log"
  if ! mkdir -p "${redis_log_dir}" 2>/dev/null; then
    warn "无法创建 Redis 日志目录: ${redis_log_dir}，将使用数据目录"
    redis_log_dir="/data/redis"
  fi
  
  # 设置日志目录权限
  local log_chown_attempts=0
  while [[ ${log_chown_attempts} -lt 3 ]]; do
    if chown appuser:appuser "${redis_log_dir}" 2>/dev/null || chmod 755 "${redis_log_dir}" 2>/dev/null; then
      break
    fi
    log_chown_attempts=$((log_chown_attempts + 1))
    [[ ${log_chown_attempts} -lt 3 ]] && sleep 0.3 || warn "无法设置 Redis 日志目录权限，继续尝试"
  done
  
  local redis_log_file="${redis_log_dir}/redis-$(date +%Y%m%d-%H%M%S).log"
  log "启动 Redis，日志文件: ${redis_log_file}"
  
  # 确保日志文件可以被创建
  touch "${redis_log_file}" 2>/dev/null || true
  chown appuser:appuser "${redis_log_file}" 2>/dev/null || chmod 666 "${redis_log_file}" 2>/dev/null || true
  
  # 启动 Redis（后台运行，输出到日志文件）
  redis-server \
    --bind 127.0.0.1 \
    --port "${REDIS_PORT}" \
    --dir /data/redis \
    --appendonly yes \
    --appendfilename "appendonly.aof" \
    --save 900 1 --save 300 10 --save 60 10000 \
    "${REDIS_AUTH_ARGS[@]}" \
    >>"${redis_log_file}" 2>&1 &
  
  export REDIS_PID=$!
  
  # 验证进程是否真的启动
  if [[ -z "${REDIS_PID}" ]] || [[ "${REDIS_PID}" -eq 0 ]]; then
    error "无法获取 Redis 进程 PID"
    error "请检查 redis-server 命令是否可用"
    
    if ! command -v redis-server >/dev/null 2>&1; then
      error "redis-server 命令不存在"
    fi
    
    exit 1
  fi
  
  log "Redis 进程已启动，PID: ${REDIS_PID}"
  
  # 等待进程启动（Windows/NAS 环境可能需要更长时间）
  sleep 2
  
  # 检查进程是否还在运行（使用 kill -0，最兼容）
  if ! kill -0 "${REDIS_PID}" 2>/dev/null; then
    error "Redis 进程启动后立即退出（PID: ${REDIS_PID}）"
    error "日志文件: ${redis_log_file}"
    
    if [[ -f "${redis_log_file}" ]]; then
      error "Redis 启动日志（最后 30 行）："
      # 跨平台 tail 兼容性
      if tail -n 30 "${redis_log_file}" >&2 2>/dev/null; then
        : # 成功
      elif tail -30 "${redis_log_file}" >&2 2>/dev/null; then
        : # Fallback: BSD tail 格式
      else
        tail "${redis_log_file}" >&2 2>/dev/null || true
      fi
    else
      error "未找到 Redis 日志文件"
      error "请检查 Redis 数据目录权限: /data/redis"
    fi
    
    exit 1
  fi
  
  # 等待 Redis 就绪
  local max_attempts=30
  local attempt=0
  while [[ ${attempt} -lt ${max_attempts} ]]; do
    if redis-cli -h 127.0.0.1 -p "${REDIS_PORT}" ping >/dev/null 2>&1; then
      log "Redis 就绪（耗时 ${attempt} 秒）"
      break
    fi
    
    # 检查进程是否还在运行
    if ! kill -0 "${REDIS_PID}" 2>/dev/null; then
      error "Redis 进程意外退出（PID: ${REDIS_PID}）"
      
      # 查找最新的日志文件（跨平台兼容）
      local latest_redis_log=""
      if [[ -d "${redis_log_dir}" ]]; then
        latest_redis_log="$(latest_file "${redis_log_dir}"/*.log)"
      fi
      
      if [[ -n "${latest_redis_log}" ]] && [[ -f "${latest_redis_log}" ]]; then
        error "Redis 日志文件: ${latest_redis_log}"
        error "日志内容（最后 30 行）："
        # 跨平台 tail 兼容性
        if tail -n 30 "${latest_redis_log}" >&2 2>/dev/null; then
          : # 成功
        elif tail -30 "${latest_redis_log}" >&2 2>/dev/null; then
          : # Fallback: BSD tail 格式
        else
          tail "${latest_redis_log}" >&2 2>/dev/null || true
        fi
      elif [[ -f "${redis_log_file}" ]]; then
        error "Redis 日志文件: ${redis_log_file}"
        error "日志内容（最后 30 行）："
        # 跨平台 tail 兼容性
        if tail -n 30 "${redis_log_file}" >&2 2>/dev/null; then
          : # 成功
        elif tail -30 "${redis_log_file}" >&2 2>/dev/null; then
          : # Fallback: BSD tail 格式
        else
          tail "${redis_log_file}" >&2 2>/dev/null || true
        fi
      fi
      
      exit 1
    fi
    
    attempt=$((attempt + 1))
    sleep 1
  done
  
  if ! redis-cli -h 127.0.0.1 -p "${REDIS_PORT}" ping >/dev/null 2>&1; then
    error "Redis 启动超时（${max_attempts} 秒后仍未就绪）"
    error "进程 PID: ${REDIS_PID}"
    
    # 检查进程状态
    if kill -0 "${REDIS_PID}" 2>/dev/null; then
      error "Redis 进程仍在运行，但无法连接"
      error "可能的原因："
      error "1. Redis 配置问题（端口、绑定地址）"
      error "2. 网络连接问题"
    else
      error "Redis 进程已退出"
    fi
    
    # 输出日志
    local latest_redis_log=""
    if [[ -d "${redis_log_dir}" ]]; then
      latest_redis_log="$(latest_file "${redis_log_dir}"/*.log)"
    fi
    
    if [[ -n "${latest_redis_log}" ]] && [[ -f "${latest_redis_log}" ]]; then
      error "Redis 日志文件: ${latest_redis_log}"
      error "日志内容（最后 50 行）："
      # 跨平台 tail 兼容性
      if tail -n 50 "${latest_redis_log}" >&2 2>/dev/null; then
        : # 成功
      elif tail -50 "${latest_redis_log}" >&2 2>/dev/null; then
        : # Fallback: BSD tail 格式
      else
        tail "${latest_redis_log}" >&2 2>/dev/null || true
      fi
    elif [[ -f "${redis_log_file}" ]]; then
      error "Redis 日志文件: ${redis_log_file}"
      error "日志内容（最后 50 行）："
      # 跨平台 tail 兼容性
      if tail -n 50 "${redis_log_file}" >&2 2>/dev/null; then
        : # 成功
      elif tail -50 "${redis_log_file}" >&2 2>/dev/null; then
        : # Fallback: BSD tail 格式
      else
        tail "${redis_log_file}" >&2 2>/dev/null || true
      fi
    fi
    
    exit 1
  fi
}

shutdown() {
  log "收到退出信号，正在停止服务..."
  
  # 优雅停止 Redis（跨平台信号处理）
  # Linux/macOS: SIGTERM, SIGKILL
  # Windows (WSL): SIGTERM, SIGKILL (通过 kill 命令)
  # Alpine/BusyBox: SIGTERM, SIGKILL
  if [[ -n "${REDIS_PID:-}" ]] && kill -0 "${REDIS_PID}" 2>/dev/null; then
    log "停止 Redis (PID: ${REDIS_PID})..."
    # 优先使用 redis-cli SHUTDOWN（优雅关闭）
    if command -v redis-cli >/dev/null 2>&1; then
      if [[ -n "${REDIS_PASSWORD:-}" ]]; then
        redis-cli -h 127.0.0.1 -p "${REDIS_PORT:-6379}" -a "${REDIS_PASSWORD}" SHUTDOWN SAVE >/dev/null 2>&1 || \
          kill -TERM "${REDIS_PID}" 2>/dev/null || true
      else
        redis-cli -h 127.0.0.1 -p "${REDIS_PORT:-6379}" SHUTDOWN SAVE >/dev/null 2>&1 || \
          kill -TERM "${REDIS_PID}" 2>/dev/null || true
      fi
    else
      # Fallback: 直接发送 SIGTERM
      kill -TERM "${REDIS_PID}" 2>/dev/null || true
    fi
    
    # 等待进程退出（最多 10 秒）
    local wait_count=0
    while [[ ${wait_count} -lt 10 ]] && kill -0 "${REDIS_PID}" 2>/dev/null; do
      sleep 1
      wait_count=$((wait_count + 1))
    done
    
    # 如果还在运行，强制终止（SIGKILL）
    if kill -0 "${REDIS_PID}" 2>/dev/null; then
      warn "Redis 未正常退出，强制终止"
      kill -KILL "${REDIS_PID}" 2>/dev/null || kill -9 "${REDIS_PID}" 2>/dev/null || true
    fi
  fi
  
  # 优雅停止 PostgreSQL（跨平台信号处理）
  if [[ -n "${PG_PID:-}" ]] && kill -0 "${PG_PID}" 2>/dev/null; then
    log "停止 PostgreSQL (PID: ${PG_PID})..."
    # 优先使用 pg_ctl stop（优雅关闭）
    if command -v pg_ctl >/dev/null 2>&1 && [[ -n "${PG_DATA_DIR:-}" ]]; then
      su-exec postgres:postgres pg_ctl stop -D "${PG_DATA_DIR}" -m fast >/dev/null 2>&1 || \
        kill -TERM "${PG_PID}" 2>/dev/null || true
    else
      # Fallback: 直接发送 SIGTERM
      kill -TERM "${PG_PID}" 2>/dev/null || true
    fi
    
    # 等待进程退出（最多 15 秒）
    local wait_count=0
    while [[ ${wait_count} -lt 15 ]] && kill -0 "${PG_PID}" 2>/dev/null; do
      sleep 1
      wait_count=$((wait_count + 1))
    done
    
    # 如果还在运行，强制终止（SIGKILL）
    if kill -0 "${PG_PID}" 2>/dev/null; then
      warn "PostgreSQL 未正常退出，强制终止"
      kill -KILL "${PG_PID}" 2>/dev/null || kill -9 "${PG_PID}" 2>/dev/null || true
    fi
  fi
  
  log "服务已停止"
}

# 注册信号处理（跨平台兼容）
# Linux/macOS/Alpine: SIGTERM, SIGINT, SIGHUP
# Windows (WSL): SIGTERM, SIGINT
# 注意：某些环境可能不支持 trap，使用 || true 避免失败
trap shutdown SIGTERM SIGINT SIGHUP EXIT 2>/dev/null || true

fix_permissions

# 确保 /app/config.yaml 存在（避免 pre-start-check 阻断启动）
generate_config_yaml_if_missing

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
  # 跨平台兼容性：
  # - Linux: ip route (GNU iputils) 或 hostname -i
  # - Alpine/BusyBox: hostname -i (如果安装了 iputils)
  # - macOS Docker Desktop: hostname -i
  CONTAINER_IP=""
  # 优先使用 ip 命令（更可靠）
  if have_cmd ip; then
    CONTAINER_IP="$(ip route get 8.8.8.8 2>/dev/null | awk '{print $7}' | first_line || echo "")"
  fi
  # Fallback: 使用 hostname -i
  if [[ -z "${CONTAINER_IP}" ]] && command -v hostname >/dev/null 2>&1; then
    CONTAINER_IP=$(hostname -i 2>/dev/null | awk '{print $1}' || echo "")
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

