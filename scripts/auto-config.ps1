## ============================================
## CYP-Registry 自动配置脚本 (Windows PowerShell)
## 强制要求：任何环境/任何平台/任何系统下，均可一键自动生成可运行的配置
## 目标：
## - 若项目根目录不存在 .env，则自动生成（含必要默认值）
## - 为 DB_PASSWORD / REDIS_PASSWORD / JWT_SECRET 生成强随机值（生产可直接使用）
## - 不修改已存在的 .env（保证可重复执行）
## ============================================

$ErrorActionPreference = "Stop"

function Write-Info { param([string]$Message) Write-Host "INFO  $Message" -ForegroundColor Cyan }
function Write-Success { param([string]$Message) Write-Host "OK    $Message" -ForegroundColor Green }
function Write-Warning { param([string]$Message) Write-Host "WARN  $Message" -ForegroundColor Yellow }

function New-RandomHex {
    param([int]$Bytes = 32)
    $buf = New-Object byte[] $Bytes
    [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($buf)
    return ($buf | ForEach-Object { $_.ToString("x2") }) -join ""
}

$root = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $root ".env"
$frontendEnvFile = Join-Path $root "web\.env.local"

if (Test-Path $envFile) {
    Write-Success ".env 已存在，跳过自动生成：$envFile"
} else {
    $jwt = "jwt_" + (New-RandomHex -Bytes 32)
    $dbPassword = (New-RandomHex -Bytes 32)
    $redisPassword = (New-RandomHex -Bytes 32)

    # 说明：
    # - 这里给出“可运行”的默认值，满足“全局配置中心 + 任意环境可自动配置”要求
    # - DB_PASSWORD / REDIS_PASSWORD / JWT_SECRET 已生成强随机值，可直接用于生产环境
    # - 数据库/Redis 主机与端口在此处给出默认值，容器编排环境可通过 docker-compose 覆盖
    $content = @"
# ============================================
# CYP-Registry 全局配置中心（根级 .env）
# 生成时间：$(Get-Date -Format "yyyy-MM-dd HH:mm:ss")
# 说明：
# - 本文件作为全局配置中心唯一源头，前端/后端/容器均应从此派生配置
# - DB_PASSWORD / REDIS_PASSWORD / JWT_SECRET 由脚本自动生成为强随机值，可直接用于生产环境
# ============================================

# Application
APP_NAME=CYP-Registry
APP_HOST=0.0.0.0
APP_PORT=8080
APP_ENV=production

# Database (默认用于本地/单机环境；容器编排环境可通过 docker-compose 覆盖)
DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=registry
DB_NAME=registry_db
DB_SSLMODE=disable

# Redis (默认用于本地/单机环境；容器编排环境可通过 docker-compose 覆盖)
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_DB=0

# API / Web Endpoints
API_BASE_URL=http://localhost:8080
WEB_BASE_URL=http://localhost:3000

# Database / Redis Passwords
DB_PASSWORD=$dbPassword
REDIS_PASSWORD=$redisPassword

# JWT (AUTO-GENERATED)
JWT_SECRET=$jwt

# Storage
STORAGE_TYPE=local

# MinIO (仅当 STORAGE_TYPE=minio 时需要)
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000

# Grafana
GRAFANA_ADMIN_USER=admin
GRAFANA_ADMIN_PASSWORD=admin

# Server Shutdown & Cleanup
# CLEANUP_ON_SHUTDOWN: 控制服务器关闭时是否清理所有数据
#   1 = 清理所有数据（删除模式）- 会永久删除所有用户数据、项目数据、镜像文件、缓存数据
#   0 或不设置 = 保留数据（停止模式）- 仅关闭服务，保留所有数据
# ⚠️ 警告：设置为 1 时，关闭服务器会永久删除所有数据，此操作不可恢复！
# 生产环境强烈建议设置为 0 或不设置，避免误操作导致数据丢失
CLEANUP_ON_SHUTDOWN=0
"@

    Set-Content -Path $envFile -Value $content -Encoding UTF8
    Write-Success "全局配置中心已初始化：$envFile"
    Write-Success "已为 DB_PASSWORD / REDIS_PASSWORD / JWT_SECRET 生成强随机值，可直接用于生产环境。"
}

# --------------------------------------------
# 前端配置自动初始化（从全局配置中心派生）
# --------------------------------------------
if (-not (Test-Path $frontendEnvFile)) {
    # 读取全局配置中心中的关键字段（若缺失则使用与上方写入保持一致的默认值）
    $envLines = Get-Content -Path $envFile -ErrorAction SilentlyContinue

    $appName = ($envLines | Where-Object { $_ -match '^APP_NAME=' } | Select-Object -First 1).Split('=')[1]
    $appEnv  = ($envLines | Where-Object { $_ -match '^APP_ENV=' }  | Select-Object -First 1).Split('=')[1]
    $apiBase = ($envLines | Where-Object { $_ -match '^API_BASE_URL=' } | Select-Object -First 1).Split('=')[1]

    if (-not $appName) { $appName = "CYP-Registry" }
    if (-not $appEnv)  { $appEnv  = "production" }
    if (-not $apiBase) { $apiBase = "http://localhost:8080" }

    $frontendContent = @"
# ============================================
# CYP-Registry 前端环境变量（由全局配置中心自动生成）
# 生成时间：$(Get-Date -Format "yyyy-MM-dd HH:mm:ss")
# 说明：
# - 本文件由项目根级 .env 派生（全局配置中心）
# - 前端仅需关心 VITE_* 变量，不直接修改根级 .env
# ============================================

VITE_APP_NAME=$appName
VITE_APP_ENV=$appEnv
VITE_API_BASE_URL=$apiBase
"@

    Set-Content -Path $frontendEnvFile -Value $frontendContent -Encoding UTF8
    Write-Success "前端配置已初始化：$frontendEnvFile"
} else {
    Write-Info "前端配置已存在：$frontendEnvFile，跳过自动初始化"
}

