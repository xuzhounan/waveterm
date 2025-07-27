#!/bin/bash

# 永久修正代理设置脚本

echo "🔧 永久修正代理配置..."

# 检查当前shell配置文件
if [ -f ~/.zshrc ]; then
    CONFIG_FILE=~/.zshrc
elif [ -f ~/.bashrc ]; then
    CONFIG_FILE=~/.bashrc
else
    CONFIG_FILE=~/.profile
fi

echo "📝 配置文件: $CONFIG_FILE"

# 备份原始配置文件
cp "$CONFIG_FILE" "${CONFIG_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
echo "✅ 已备份原始配置文件"

# 移除错误的代理设置
echo "🔍 移除错误的代理设置..."
sed -i.bak '/export http_proxy.*109000/d' "$CONFIG_FILE"
sed -i.bak '/export HTTP_PROXY.*109000/d' "$CONFIG_FILE"

# 添加或更新正确的代理设置
echo "📝 添加正确的代理设置..."

# 检查是否已经有正确的设置
if ! grep -q "export http_proxy=\"http://127.0.0.1:10900\"" "$CONFIG_FILE"; then
    echo "" >> "$CONFIG_FILE"
    echo "# Proxy settings - corrected port" >> "$CONFIG_FILE"
    echo "export http_proxy=\"http://127.0.0.1:10900\"" >> "$CONFIG_FILE"
    echo "export https_proxy=\"http://127.0.0.1:10900\"" >> "$CONFIG_FILE"
fi

echo "✅ 代理配置已永久修正"
echo ""
echo "📋 修正的设置:"
echo "  http_proxy=http://127.0.0.1:10900"
echo "  https_proxy=http://127.0.0.1:10900"
echo ""
echo "🔄 请运行以下命令使设置生效:"
echo "  source $CONFIG_FILE"
echo ""
echo "或者重启终端窗口"