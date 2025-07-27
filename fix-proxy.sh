#!/bin/bash

# 修正代理设置脚本

echo "🔧 修正代理设置..."

# 检查当前代理设置
echo "当前代理设置:"
env | grep -i proxy

echo ""
echo "修正代理设置..."

# 修正 http_proxy（从 109000 改为 10900）
export http_proxy="http://127.0.0.1:10900"
export https_proxy="http://127.0.0.1:10900"

echo "✅ 代理设置已修正："
echo "  http_proxy=$http_proxy"
echo "  https_proxy=$https_proxy"

# 测试代理连接
echo ""
echo "🔍 测试代理连接..."
if curl -s --connect-timeout 5 --proxy "$http_proxy" "http://www.google.com" > /dev/null 2>&1; then
    echo "✅ 代理连接测试成功"
else
    echo "⚠️  代理连接测试失败，请检查代理服务器是否运行"
fi

echo ""
echo "📝 要永久修正代理设置，请在你的 ~/.zshrc 或 ~/.bashrc 中修改："
echo "export http_proxy=\"http://127.0.0.1:10900\""
echo "export https_proxy=\"http://127.0.0.1:10900\""