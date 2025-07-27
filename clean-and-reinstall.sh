#!/bin/bash

echo "=== Wave Terminal 依赖完整清理和重新安装 ==="

# 确保使用正确的Node.js版本
echo "当前 Node.js 版本: $(node --version)"
echo "当前 NPM 版本: $(npm --version)"

if [[ "$(node --version)" != "v20"* ]]; then
    echo "⚠️  警告: 当前不是 Node.js 20.x 版本，请先运行 'nvm use 20'"
    exit 1
fi

# 1. 设置环境变量
export PATH="/opt/homebrew/bin:$PATH"
echo "✓ 设置 PATH"

# 2. 启用 corepack 并设置 yarn
corepack enable
corepack prepare yarn@4.6.0 --activate
echo "✓ 启用 corepack 和 yarn 4.6.0"

echo ""
echo "=== 清理依赖 ==="

# 3. 清理 yarn 缓存
echo "清理 yarn 缓存..."
yarn cache clean

# 4. 清理 npm 缓存 (预防措施)
echo "清理 npm 缓存..."
npm cache clean --force

# 5. 提示用户手动删除目录
echo ""
echo "=== 需要手动删除以下目录 ==="
echo "请手动运行以下命令："
echo ""
echo "rm -rf node_modules"
echo "rm -rf .yarn/cache"
echo "rm -rf .yarn/install-state.gz"
echo "rm -f yarn.lock"
echo ""
echo "删除完成后，按任意键继续..."
read -n 1 -s

# 6. 验证目录已删除
if [ -d "node_modules" ]; then
    echo "❌ node_modules 目录仍然存在，请手动删除"
    exit 1
fi

echo "✓ 目录清理完成"

echo ""
echo "=== 重新安装依赖 ==="

# 7. 重新安装 Node.js 依赖
echo "重新安装 Node.js 依赖..."
yarn install

echo ""
echo "=== 清理 Go 依赖 ==="

# 8. 清理并重新安装 Go 依赖
go clean -modcache
go mod download
go mod tidy

echo ""
echo "✓ 依赖清理和重新安装完成！"
echo ""
echo "现在可以运行: task dev"