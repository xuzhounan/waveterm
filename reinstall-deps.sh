#!/bin/bash

echo "=== Wave Terminal 依赖重新安装 ==="

# 设置环境
source ./setup-dev.sh

echo ""
echo "=== 清理现有依赖 ==="

# 1. 清理 yarn 缓存
echo "清理 yarn 缓存..."
yarn cache clean

# 2. 清理 Go modules
echo "清理 Go modules..."
go clean -modcache

# 3. 手动删除 node_modules (如果可以的话)
if [ -d "node_modules" ]; then
    echo "发现 node_modules 目录，请手动删除："
    echo "手动运行: rm -rf node_modules"
    echo "然后继续执行此脚本"
    # exit 1
fi

echo ""
echo "=== 重新安装依赖 ==="

# 4. 安装 Go 依赖
echo "安装 Go 依赖..."
go mod download
go mod tidy

# 5. 安装 Node.js 依赖
echo "安装 Node.js 依赖..."
yarn install

echo ""
echo "=== 构建后端组件 ==="

# 6. 构建后端
task build:backend

echo ""
echo "✓ 依赖安装完成！现在可以运行 'task dev'"