# CYP-Registry 自动推送到GitHub脚本
# 作者：CYP
# 用途：自动配置并推送代码到GitHub公开仓库

Write-Host "=== CYP-Registry 自动推送到GitHub ===" -ForegroundColor Cyan
Write-Host ""

# 切换到项目目录
Set-Location -Path $PSScriptRoot

# 检查Git配置
Write-Host "[1/5] 检查Git配置..." -ForegroundColor Yellow
$userName = git config user.name
$userEmail = git config user.email
Write-Host "  用户名: $userName" -ForegroundColor Green
Write-Host "  邮箱: $userEmail" -ForegroundColor Green

# 检查远程仓库配置
Write-Host "`n[2/5] 检查远程仓库配置..." -ForegroundColor Yellow
$remoteUrl = git remote get-url origin 2>$null
if (-not $remoteUrl) {
    Write-Host "  配置远程仓库..." -ForegroundColor Yellow
    git remote add origin https://github.com/ADdss-hub/CYP-Registry.git
} else {
    Write-Host "  远程仓库: $remoteUrl" -ForegroundColor Green
    git remote set-url origin https://github.com/ADdss-hub/CYP-Registry.git
}

# 检查本地提交
Write-Host "`n[3/5] 检查本地提交..." -ForegroundColor Yellow
$commitCount = (git log --oneline | Measure-Object -Line).Lines
Write-Host "  提交数量: $commitCount" -ForegroundColor Green

# 配置Git凭据助手
Write-Host "`n[4/5] 配置Git凭据助手..." -ForegroundColor Yellow
git config --global credential.helper manager-core
Write-Host "  已配置Windows Credential Manager" -ForegroundColor Green

# 尝试推送
Write-Host "`n[5/5] 推送到GitHub..." -ForegroundColor Yellow
Write-Host "  如果提示输入凭据，请使用GitHub用户名和Personal Access Token" -ForegroundColor Cyan
Write-Host ""

$pushResult = git push -u origin main 2>&1

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✅ 推送成功！" -ForegroundColor Green
    Write-Host "  仓库地址: https://github.com/ADdss-hub/CYP-Registry" -ForegroundColor Cyan
} else {
    $errorMsg = $pushResult -join "`n"
    if ($errorMsg -match "Repository not found") {
        Write-Host "`n❌ 错误：仓库不存在" -ForegroundColor Red
        Write-Host "`n请先在GitHub上创建仓库：" -ForegroundColor Yellow
        Write-Host "  1. 访问: https://github.com/new" -ForegroundColor Cyan
        Write-Host "  2. 仓库名称: CYP-Registry" -ForegroundColor Cyan
        Write-Host "  3. 设置为公开仓库" -ForegroundColor Cyan
        Write-Host "  4. 不要初始化README、.gitignore或license" -ForegroundColor Cyan
        Write-Host "  5. 创建后重新运行此脚本" -ForegroundColor Cyan
    } elseif ($errorMsg -match "Permission denied|authentication") {
        Write-Host "`n❌ 错误：认证失败" -ForegroundColor Red
        Write-Host "`n解决方案：" -ForegroundColor Yellow
        Write-Host "  1. 创建GitHub Personal Access Token:" -ForegroundColor Cyan
        Write-Host "     https://github.com/settings/tokens" -ForegroundColor Cyan
        Write-Host "  2. 权限选择: repo (全部)" -ForegroundColor Cyan
        Write-Host "  3. 推送时用户名输入: ADdss-hub" -ForegroundColor Cyan
        Write-Host "  4. 密码输入: Personal Access Token" -ForegroundColor Cyan
    } else {
        Write-Host "`n❌ 推送失败：" -ForegroundColor Red
        Write-Host $errorMsg -ForegroundColor Red
    }
}

Write-Host "`n=== 脚本执行完成 ===" -ForegroundColor Cyan
