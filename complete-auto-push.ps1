# CYP-Registry 完整自动化推送到GitHub
# 功能：自动创建仓库并推送代码（需要GitHub Personal Access Token）

param(
    [string]$GitHubToken = $env:GITHUB_TOKEN
)

Write-Host "=== CYP-Registry 完整自动化推送 ===" -ForegroundColor Cyan
Write-Host ""

Set-Location -Path $PSScriptRoot

# 步骤1：检查GitHub Token
Write-Host "[1/6] 检查GitHub Token..." -ForegroundColor Yellow
if (-not $GitHubToken) {
    Write-Host "  ⚠️  未找到GitHub Token" -ForegroundColor Yellow
    Write-Host "`n  请提供GitHub Personal Access Token：" -ForegroundColor Cyan
    Write-Host "  1. 访问: https://github.com/settings/tokens" -ForegroundColor White
    Write-Host "  2. 点击 'Generate new token (classic)'" -ForegroundColor White
    Write-Host "  3. 权限选择: repo (全部权限)" -ForegroundColor White
    Write-Host "  4. 生成后，执行以下命令之一：" -ForegroundColor White
    Write-Host "     - `$env:GITHUB_TOKEN='你的token'; .\complete-auto-push.ps1" -ForegroundColor Green
    Write-Host "     - .\complete-auto-push.ps1 -GitHubToken '你的token'" -ForegroundColor Green
    exit 1
}
Write-Host "  ✅ 找到GitHub Token" -ForegroundColor Green

# 步骤2：检查仓库是否存在
Write-Host "`n[2/6] 检查仓库是否存在..." -ForegroundColor Yellow
$headers = @{
    "Authorization" = "token $GitHubToken"
    "Accept" = "application/vnd.github.v3+json"
}

try {
    $repo = Invoke-RestMethod -Uri "https://api.github.com/repos/ADdss-hub/CYP-Registry" -Headers $headers -ErrorAction Stop
    Write-Host "  ✅ 仓库已存在: $($repo.html_url)" -ForegroundColor Green
    $repoExists = $true
} catch {
    if ($_.Exception.Response.StatusCode -eq 404) {
        Write-Host "  ⚠️  仓库不存在，将自动创建" -ForegroundColor Yellow
        $repoExists = $false
    } else {
        Write-Host "  ❌ 检查失败: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }
}

# 步骤3：创建仓库（如果不存在）
if (-not $repoExists) {
    Write-Host "`n[3/6] 创建GitHub仓库..." -ForegroundColor Yellow
    $body = @{
        name = "CYP-Registry"
        description = "CYP Registry - Container Registry System"
        private = $false
        auto_init = $false
    } | ConvertTo-Json

    try {
        $newRepo = Invoke-RestMethod -Uri "https://api.github.com/user/repos" -Method Post -Headers $headers -Body $body -ContentType "application/json"
        Write-Host "  ✅ 仓库创建成功: $($newRepo.html_url)" -ForegroundColor Green
    } catch {
        Write-Host "  ❌ 创建失败: $($_.Exception.Message)" -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "`n[3/6] 跳过创建（仓库已存在）" -ForegroundColor Gray
}

# 步骤4：配置Git
Write-Host "`n[4/6] 配置Git..." -ForegroundColor Yellow
git config --global credential.helper manager-core
git remote remove origin 2>$null
git remote add origin https://github.com/ADdss-hub/CYP-Registry.git
Write-Host "  ✅ Git配置完成" -ForegroundColor Green

# 步骤5：配置推送凭据
Write-Host "`n[5/6] 配置推送凭据..." -ForegroundColor Yellow
$credentialUrl = "https://${GitHubToken}@github.com"
git remote set-url origin $credentialUrl
Write-Host "  ✅ 凭据配置完成" -ForegroundColor Green

# 步骤6：推送代码
Write-Host "`n[6/6] 推送到GitHub..." -ForegroundColor Yellow
$pushOutput = git push -u origin main 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✅✅✅ 推送成功！✅✅✅" -ForegroundColor Green
    Write-Host "`n仓库地址: https://github.com/ADdss-hub/CYP-Registry" -ForegroundColor Cyan
    Write-Host "`n提交信息:" -ForegroundColor Yellow
    git log --oneline -n 1
} else {
    Write-Host "`n❌ 推送失败：" -ForegroundColor Red
    Write-Host $pushOutput -ForegroundColor Red
    exit 1
}

Write-Host "`n=== 完成 ===" -ForegroundColor Cyan
