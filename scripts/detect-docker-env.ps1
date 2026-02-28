# ============================================
# Docker环境自动检测脚本 (Windows PowerShell)
# 遵循《全平台通用容器开发设计规范》3.3节
# 使用方法: .\scripts\detect-docker-env.ps1
# ============================================

$ErrorActionPreference = "Stop"

# 颜色输出函数
function Write-Info {
    param([string]$Message)
    Write-Host "ℹ️  $Message" -ForegroundColor Cyan
}

function Write-Success {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠️  $Message" -ForegroundColor Yellow
}

function Write-ErrorMsg {
    param([string]$Message)
    Write-Host "❌ $Message" -ForegroundColor Red
}

# 检测报告文件
$ReportFile = if ($env:REPORT_FILE) { $env:REPORT_FILE } else { "$env:TEMP\docker_env_detect_report.json" }

# ============================================
# 1. 操作系统版本检测
# ============================================
Write-Info "检测操作系统版本..."

try {
    $osInfo = Get-CimInstance Win32_OperatingSystem -ErrorAction Stop
    $osVersion = $osInfo.Caption + " " + $osInfo.Version
    Write-Success "操作系统: $osVersion"
} catch {
    $osVersion = "Unknown"
    Write-Warning "无法检测操作系统版本"
}

# ============================================
# 2. 硬件资源检测（精准提取并统一单位）
# ============================================
Write-Info "检测硬件资源..."

try {
    # CPU信息
    $cpuInfo = (Get-CimInstance Win32_Processor -ErrorAction Stop | Select-Object -First 1).Name
    $cpuCores = (Get-CimInstance Win32_Processor -ErrorAction Stop | Measure-Object -Property NumberOfCores -Sum).Sum
    Write-Success "CPU: $cpuInfo (核心数: $cpuCores)"
} catch {
    $cpuInfo = "Unknown"
    $cpuCores = "Unknown"
    Write-Warning "无法检测CPU信息"
}

try {
    # 内存信息（统一单位为GiB）
    $ramRaw = Get-CimInstance Win32_PhysicalMemory -ErrorAction Stop
    $memTotalGB = [math]::Round(($ramRaw | Measure-Object -Property Capacity -Sum).Sum / 1GB, 2)
    $memTotal = "$memTotalGB GiB"
    Write-Success "内存: $memTotal"
} catch {
    $memTotal = "Unknown"
    Write-Warning "无法检测内存信息"
}

# ============================================
# 3. 容器环境判断
# ============================================
Write-Info "检测容器环境..."

$envType = "Windows Host"
$containerEngine = "None"

# 检查是否在容器内（Windows容器）
if (Test-Path "C:\Windows\System32\config\systemprofile\AppData\Local\Docker") {
    $envType = "Container"
    $containerEngine = "Docker"
    Write-Success "环境类型: $envType"
    Write-Success "容器引擎: $containerEngine"
} else {
    Write-Success "环境类型: $envType"
}

# ============================================
# 4. Docker检测
# ============================================
Write-Info "检测容器引擎..."

$dockerAvailable = $false
$dockerVersion = ""
$dockerRunning = $false
$composeAvailable = $false
$composeVersion = ""

# 检测Docker Desktop
if (Get-Command docker -ErrorAction SilentlyContinue) {
    $dockerAvailable = $true
    try {
        $dockerVersion = (docker --version 2>&1 | Out-String).Trim()
        Write-Success "Docker已安装: $dockerVersion"
        
        # 检测Docker服务状态
        $dockerInfo = docker info 2>&1
        if ($LASTEXITCODE -eq 0) {
            $dockerRunning = $true
            Write-Success "Docker服务运行中"
        } else {
            Write-ErrorMsg "Docker服务未运行"
        }
    } catch {
        Write-Warning "无法获取Docker版本信息"
    }
} else {
    Write-Warning "Docker未安装"
}

# 检测Docker Compose
if (Get-Command docker-compose -ErrorAction SilentlyContinue) {
    $composeAvailable = $true
    try {
        $composeVersion = (docker-compose --version 2>&1 | Out-String).Trim()
        Write-Success "Docker Compose已安装: $composeVersion"
    } catch {
        Write-Warning "无法获取Docker Compose版本信息"
    }
} elseif (docker compose version 2>&1 | Out-Null) {
    $composeAvailable = $true
    try {
        $composeVersion = (docker compose version 2>&1 | Out-String).Trim()
        Write-Success "Docker Compose已安装: $composeVersion"
    } catch {
        Write-Warning "无法获取Docker Compose版本信息"
    }
} else {
    Write-Warning "Docker Compose未安装"
}

# ============================================
# 5. 网络配置检测
# ============================================
Write-Info "检测网络配置..."

try {
    $ipAddress = (Get-NetIPAddress -AddressFamily IPv4 -ErrorAction Stop | 
                  Where-Object { $_.InterfaceAlias -notlike "*Loopback*" -and $_.IPAddress -notlike "169.254.*" } | 
                  Select-Object -First 1).IPAddress
    Write-Success "IP地址: $ipAddress"
} catch {
    $ipAddress = "Unknown"
    Write-Warning "无法检测IP地址"
}

# ============================================
# 6. 存储路径检测
# ============================================
Write-Info "检测存储路径..."

# Windows Docker Desktop默认路径
$storagePath = "$env:LOCALAPPDATA\Docker\wsl\data"
$storagePermission = "Unknown"

if (Test-Path $storagePath) {
    try {
        $acl = Get-Acl $storagePath
        $storagePermission = $acl.AccessToString
        Write-Success "存储路径: $storagePath"
    } catch {
        Write-Warning "无法获取存储路径权限信息"
    }
} else {
    Write-Warning "存储路径不存在: $storagePath"
}

# ============================================
# 7. 生成检测报告
# ============================================
Write-Info "生成检测报告: $ReportFile"

$report = [PSCustomObject]@{
    os_version = $osVersion
    cpu_info = $cpuInfo
    cpu_cores = $cpuCores
    mem_total = $memTotal
    env_type = $envType
    container_engine = $containerEngine
    docker_available = $dockerAvailable
    docker_version = $dockerVersion
    docker_running = $dockerRunning
    podman_available = $false
    podman_version = ""
    compose_available = $composeAvailable
    compose_version = $composeVersion
    ip_address = $ipAddress
    nas_model = "None"
    storage_path = $storagePath
    storage_permission = $storagePermission
    detect_time = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")
}

$report | ConvertTo-Json -Depth 10 | Out-File -FilePath $ReportFile -Encoding UTF8
Write-Success "检测完成！报告已保存至: $ReportFile"

# ============================================
# 8. 输出摘要
# ============================================
Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  环境检测摘要" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "操作系统: $osVersion"
Write-Host "环境类型: $envType"
Write-Host "CPU: $cpuInfo ($cpuCores 核心)"
Write-Host "内存: $memTotal"
Write-Host "IP地址: $ipAddress"
if ($dockerAvailable) {
    $dockerStatus = if ($dockerRunning) { "运行中" } else { "未运行" }
    Write-Host "Docker: $dockerVersion ($dockerStatus)"
}
if ($composeAvailable) {
    Write-Host "Docker Compose: $composeVersion"
}
Write-Host "存储路径: $storagePath"
Write-Host "============================================" -ForegroundColor Cyan

exit 0


