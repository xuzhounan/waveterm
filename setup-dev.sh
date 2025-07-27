#!/bin/bash

echo "=== Wave Terminal 开发环境设置 ==="

# 1. 设置 PATH 环境变量
export PATH="/opt/homebrew/bin:$PATH"
echo "✓ 设置 homebrew PATH"

# 2. 启用 corepack 管理包管理器版本
corepack enable
echo "✓ 启用 corepack"

# 3. 安装并激活 yarn 4.6.0
corepack prepare yarn@4.6.0 --activate
echo "✓ 准备 yarn 4.6.0"

# 4. 验证工具版本
echo ""
echo "=== 工具版本验证 ==="
echo "Go版本: $(go version)"
echo "Yarn版本: $(yarn --version)"
echo "Task版本: $(task --version)"
echo "Node版本: $(node --version)"

echo ""
echo "✓ 环境设置完成！"