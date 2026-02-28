# 自动推送脚本
# 使用方法：在Git Bash中执行此脚本

cd /e/kf/CYP-Registry
git remote set-url origin git@github.com:ADdss-hub/CYP-Registry.git

# 如果仓库不存在，需要先在GitHub网页上创建
# 然后执行：git push -u origin main

echo "准备推送..."
git push -u origin main
