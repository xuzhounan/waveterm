#!/bin/bash

# 设置环境变量
export WAVETERM_DATA_HOME="/tmp/waveterm-server"
export WAVETERM_CONFIG_HOME="/tmp/waveterm-server"
export WAVETERM_AUTH_KEY="83958e47ddc89fae695a7e1eb429899871e80334bd58cfc2d17a80388791f073"

# 创建目录
mkdir -p /tmp/waveterm-server

# 启动服务器，将stdin重定向到/dev/null以避免EOF问题
go run cmd/server/main-server.go < /dev/null > server.log 2>&1 &

# 保存进程ID
echo $! > server.pid

echo "服务器已启动，PID: $(cat server.pid)"
echo "认证密钥: $WAVETERM_AUTH_KEY"
echo "数据目录: $WAVETERM_DATA_HOME"

# 等待服务器启动
sleep 5

# 显示端口信息
echo "正在查找服务器端口..."
if [ -f server.log ]; then
    WEB_PORT=$(grep "Server \[web\] listening" server.log | grep -o "127.0.0.1:[0-9]*" | cut -d: -f2)
    WS_PORT=$(grep "Server \[websocket\] listening" server.log | grep -o "127.0.0.1:[0-9]*" | cut -d: -f2)
    
    if [ ! -z "$WEB_PORT" ]; then
        echo "✅ Web服务器运行在端口: $WEB_PORT"
        echo "   API测试地址: http://localhost:$WEB_PORT/api/v1/widgets"
    fi
    
    if [ ! -z "$WS_PORT" ]; then
        echo "✅ WebSocket服务器运行在端口: $WS_PORT"
    fi
else
    echo "❌ 服务器日志文件未找到"
fi