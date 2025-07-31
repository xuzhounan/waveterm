#!/bin/bash

# 清理过时脚本和临时文件
# 这个脚本会移除项目中不再需要的临时脚本文件

echo "🧹 清理过时的脚本和临时文件..."

# 定义要清理的文件列表
OLD_SCRIPTS=(
    "test-widget-api.sh"
    "test-widget-api-workspace-name.sh"
    "start-server.sh"
    "fix-env.sh"
    "setup-dev.sh" 
    "reinstall-deps.sh"
    "fix-node-version.sh"
    "clean-and-reinstall.sh"
    "fix-proxy.sh"
    "permanent-proxy-fix.sh"
    "force-cleanup.sh"
    "dev-isolated.sh"
    "env-conflict-detector.sh"
    "test-reverse-conflict-fix.sh"
    "reverse-conflict-validator.sh"
    "cleanup-wrong-dirs.sh"
    "test-cors.sh"
    "start-with-fixed-ports.sh"
    "deploy-claude-code.sh"
    "setup-event-bridge.sh"
)

# 定义要清理的文档文件
OLD_DOCS=(
    "widget-api-workspace-name-example.md"
    "PROCESS_MANAGEMENT_GUIDE.md"
    "CORS-FIX-VERIFICATION.md"
    "MCP-API-DOCUMENTATION.md"
    "test-widget-api.md"
    "DEPLOYMENT.md"
    "ARCHITECTURE_REVIEW.md"
)

# 创建备份目录
BACKUP_DIR="old-files-backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "📦 创建备份目录: $BACKUP_DIR"

# 备份和移除脚本文件
for script in "${OLD_SCRIPTS[@]}"; do
    if [ -f "$script" ]; then
        echo "  🗃️  备份并移除: $script"
        mv "$script" "$BACKUP_DIR/"
    fi
done

# 备份和移除过时文档
for doc in "${OLD_DOCS[@]}"; do
    if [ -f "$doc" ]; then
        echo "  📄 备份并移除: $doc"
        mv "$doc" "$BACKUP_DIR/"
    fi
done

# Git提供版本控制，不需要本地备份
if [ -d "$BACKUP_DIR" ] && [ -n "$(ls -A "$BACKUP_DIR")" ]; then
    echo "✅ 清理完成！文件已从项目中移除"
    echo "🗑️  删除本地备份（Git提供版本控制）..."
    rm -rf "$BACKUP_DIR"
    echo "📚 可通过git历史恢复任何文件：git checkout HEAD~1 -- filename"
else
    echo "✅ 没有找到需要清理的文件"
fi

echo ""
echo "🎯 推荐使用的标准命令："
echo "  📦 打包应用:    task package"
echo "  🚀 开发模式:    task dev"  
echo "  🔨 构建后端:    task build:backend"
echo "  ⚡ 构建前端:    npm run build:prod"
echo "  📚 查看文档:    BUILD.md"