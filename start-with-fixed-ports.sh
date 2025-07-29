#!/bin/bash

# Wave Terminal 固定端口启动脚本
# 确保开发和生产环境使用相同的端口配置

set -e

# 固定端口配置
export WAVETERM_WEB_PORT="8090"
export WAVETERM_WS_PORT="8091"

echo "🚀 启动Wave Terminal，使用固定端口配置:"
echo "   Web端口: $WAVETERM_WEB_PORT"
echo "   WebSocket端口: $WAVETERM_WS_PORT"
echo ""

# 启动开发服务器
exec yarn dev