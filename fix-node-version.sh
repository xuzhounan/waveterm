#!/bin/bash

echo "=== 修复 Node.js 版本兼容性问题 ==="

# 1. 切换到 Node.js 18 (更稳定)
echo "安装并切换到 Node.js 18..."
nvm install 18
nvm use 18

# 2. 重新启用 corepack
export PATH="/opt/homebrew/bin:$PATH"
corepack enable
corepack prepare yarn@4.6.0 --activate

# 3. 验证版本
echo ""
echo "=== 版本验证 ==="
echo "Node.js: $(node --version)"
echo "Yarn: $(yarn --version)"

# 4. 清理并重新安装依赖
echo ""
echo "=== 清理并重新安装依赖 ==="
yarn cache clean

echo ""
echo "现在请手动删除 node_modules 然后重新安装："
echo "rm -rf node_modules"
echo "yarn install"