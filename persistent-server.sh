#!/bin/bash

# Wave Terminal 持久化服务器启动脚本
# 用于MCP集成，保持服务器持续运行

set -e

# 配置
AUTH_KEY="83958e47ddc89fae695a7e1eb429899871e80334bd58cfc2d17a80388791f073"
DATA_DIR="/tmp/waveterm-mcp"
LOG_FILE="waveterm-server.log"
PID_FILE="waveterm-server.pid"
PORT_FILE="waveterm-server.port"

# 代理配置
PROXY_HOST="127.0.0.1"
PROXY_PORT="10900"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

error() {
    echo -e "${RED}❌ $1${NC}"
}

# 设置代理配置用于curl命令
setup_proxy_for_curl() {
    # 修正代理设置 - 确保端口号正确
    export http_proxy="http://${PROXY_HOST}:${PROXY_PORT}"
    export https_proxy="http://${PROXY_HOST}:${PROXY_PORT}"
    
    log "设置代理配置用于外部请求: ${PROXY_HOST}:${PROXY_PORT}"
}

# 禁用代理用于本地API调用
disable_proxy_for_local() {
    unset http_proxy
    unset https_proxy
    unset HTTP_PROXY
    unset HTTPS_PROXY
    
    log "禁用代理用于本地API调用"
}

# 停止现有服务器
stop_server() {
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            log "停止现有服务器 (PID: $pid)..."
            kill "$pid" 2>/dev/null || true
            sleep 2
            
            # 强制杀死如果还在运行
            if ps -p "$pid" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null || true
                sleep 1
            fi
        fi
        rm -f "$PID_FILE"
    fi
    
    # 增强清理：查找并清理所有相关进程
    local wave_pids=$(ps aux | grep -E "(main-server|wavesrv)" | grep -v grep | awk '{print $2}' || true)
    if [ ! -z "$wave_pids" ]; then
        warning "发现残留的Wave进程，正在清理..."
        for pid in $wave_pids; do
            if ps -p "$pid" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null || true
            fi
        done
    fi
    
    # 清理锁文件
    rm -f /tmp/waveterm-mcp/wave.lock /tmp/waveterm-mcp/wave.sock 2>/dev/null || true
    
    sleep 1
}

# 创建目录和环境
setup_environment() {
    log "设置环境..."
    
    # 创建数据目录
    mkdir -p "$DATA_DIR"
    
    # 设置环境变量
    export WAVETERM_DATA_HOME="$DATA_DIR"
    export WAVETERM_CONFIG_HOME="$DATA_DIR"
    export WAVETERM_AUTH_KEY="$AUTH_KEY"
    
    success "环境设置完成"
    echo "  数据目录: $DATA_DIR"
    echo "  认证密钥: $AUTH_KEY"
}

# 启动服务器
start_server() {
    log "启动Wave Terminal服务器..."
    
    # 使用fifo管道保持stdin开放，防止服务器因EOF而关闭
    local fifo_path="/tmp/waveterm-stdin-$$"
    mkfifo "$fifo_path"
    
    # 在后台保持fifo开放
    exec 3<>"$fifo_path"
    rm "$fifo_path"  # 删除文件系统中的文件，但fd仍然开放
    
    # 启动服务器，使用fifo作为stdin
    WAVETERM_DATA_HOME="$DATA_DIR" \
    WAVETERM_CONFIG_HOME="$DATA_DIR" \
    WAVETERM_AUTH_KEY="$AUTH_KEY" \
    go run cmd/server/main-server.go <&3 > "$LOG_FILE" 2>&1 &
    
    local server_pid=$!
    echo "$server_pid" > "$PID_FILE"
    
    log "等待服务器启动..."
    
    # 等待服务器启动并获取端口信息
    local max_wait=30
    local wait_count=0
    local web_port=""
    local ws_port=""
    
    while [ $wait_count -lt $max_wait ]; do
        if [ -f "$LOG_FILE" ]; then
            web_port=$(grep "Server \[web\] listening" "$LOG_FILE" 2>/dev/null | grep -o "127.0.0.1:[0-9]*" | cut -d: -f2 | tail -1)
            ws_port=$(grep "Server \[websocket\] listening" "$LOG_FILE" 2>/dev/null | grep -o "127.0.0.1:[0-9]*" | cut -d: -f2 | tail -1)
            
            if [ ! -z "$web_port" ] && [ ! -z "$ws_port" ]; then
                break
            fi
        fi
        
        # 检查进程是否还在运行
        if ! ps -p "$server_pid" > /dev/null 2>&1; then
            error "服务器进程意外退出"
            if [ -f "$LOG_FILE" ]; then
                echo "最后的日志信息:"
                tail -10 "$LOG_FILE"
            fi
            return 1
        fi
        
        sleep 1
        wait_count=$((wait_count + 1))
        echo -n "."
    done
    
    echo  # 换行
    
    if [ -z "$web_port" ]; then
        error "服务器启动超时或失败"
        if [ -f "$LOG_FILE" ]; then
            echo "日志信息:"
            tail -20 "$LOG_FILE"
        fi
        return 1
    fi
    
    # 保存端口信息
    echo "web_port=$web_port" > "$PORT_FILE"
    echo "ws_port=$ws_port" >> "$PORT_FILE"
    
    success "Wave Terminal服务器启动成功!"
    echo "  进程ID: $server_pid"
    echo "  Web端口: $web_port"
    echo "  WebSocket端口: $ws_port"
    echo "  API基础URL: http://localhost:$web_port"
    echo "  日志文件: $LOG_FILE"
    
    return 0
}

# 测试API端点
test_api() {
    if [ ! -f "$PORT_FILE" ]; then
        error "端口文件不存在，服务器可能未启动"
        return 1
    fi
    
    source "$PORT_FILE"
    
    log "测试API端点..."
    
    # 禁用代理用于本地API调用
    disable_proxy_for_local
    
    # 测试基础API
    local api_url="http://localhost:$web_port/api/v1/widgets"
    if curl -s -f "$api_url" > /dev/null; then
        success "API端点可访问: $api_url"
    else
        warning "API端点不可访问: $api_url"
        return 1
    fi
    
    # 测试工作区列表API
    local workspaces_url="http://localhost:$web_port/api/v1/widgets/workspaces"
    log "测试工作区列表API: $workspaces_url"
    echo "响应数据:"
    curl -s "$workspaces_url" | jq '.' 2>/dev/null || curl -s "$workspaces_url"
    echo
    
    # 测试工作区名称查找API
    local workspace_name_url="http://localhost:$web_port/api/v1/widgets/workspace/name/waveterm"
    log "测试工作区名称查找API: $workspace_name_url"
    echo "响应数据:"
    curl -s "$workspace_name_url" | jq '.' 2>/dev/null || curl -s "$workspace_name_url"
    echo
    
    # 测试不存在的工作区
    local nonexist_url="http://localhost:$web_port/api/v1/widgets/workspace/name/NonExistent"
    log "测试不存在工作区API: $nonexist_url"
    echo "响应数据:"
    curl -s "$nonexist_url" | jq '.' 2>/dev/null || curl -s "$nonexist_url"
    echo
    
    success "API测试完成"
    
    # 测试完成后，如果需要外部网络访问，可以重新启用代理
    if [ "$1" = "--enable-proxy-after" ]; then
        setup_proxy_for_curl
        success "代理已重新启用用于外部请求"
    fi
}

# 显示服务器状态
show_status() {
    if [ ! -f "$PID_FILE" ]; then
        echo "服务器未运行 (PID文件不存在)"
        return 1
    fi
    
    local pid=$(cat "$PID_FILE")
    if ! ps -p "$pid" > /dev/null 2>&1; then
        echo "服务器未运行 (进程 $pid 不存在)"
        return 1
    fi
    
    if [ -f "$PORT_FILE" ]; then
        source "$PORT_FILE"
        success "服务器正在运行"
        echo "  进程ID: $pid"
        echo "  Web端口: $web_port"
        echo "  WebSocket端口: $ws_port"
        echo "  API基础URL: http://localhost:$web_port"
        echo "  认证密钥: $AUTH_KEY"
    else
        echo "服务器运行中但端口信息不可用"
    fi
}

# 主函数
main() {
    case "${1:-start}" in
        "start")
            log "启动Wave Terminal持久化服务器..."
            stop_server
            setup_environment
            if start_server; then
                echo
                success "服务器启动完成！"
                echo
                echo "📋 可用的API端点:"
                source "$PORT_FILE"
                echo "  • 列出工作区: http://localhost:$web_port/api/v1/widgets/workspaces"
                echo "  • 按名称查找工作区: http://localhost:$web_port/api/v1/widgets/workspace/name/{name}"
                echo "  • 获取工作区widgets: http://localhost:$web_port/api/v1/widgets/workspace/{id}"
                echo "  • 创建widget: http://localhost:$web_port/api/v1/widgets (POST)"
                echo
                echo "🔑 认证信息 (如果需要):"
                echo "  Header: X-AuthKey: $AUTH_KEY"
                echo
                echo "📝 管理命令:"
                echo "  ./persistent-server.sh status          - 查看状态"
                echo "  ./persistent-server.sh test            - 测试API（自动禁用代理）"
                echo "  ./persistent-server.sh test-with-proxy - 测试API并重新启用代理"
                echo "  ./persistent-server.sh setup-proxy     - 设置代理"
                echo "  ./persistent-server.sh disable-proxy   - 禁用代理"
                echo "  ./persistent-server.sh logs            - 查看日志"
                echo "  ./persistent-server.sh stop            - 停止服务器"
                echo
                echo "🌐 代理配置:"
                echo "  代理服务器: ${PROXY_HOST}:${PROXY_PORT}"
                echo "  当前状态: $([ -n "$http_proxy" ] && echo "已启用 ($http_proxy)" || echo "已禁用")"
                echo
            else
                error "服务器启动失败"
                exit 1
            fi
            ;;
        "stop")
            log "停止服务器..."
            stop_server
            success "服务器已停止"
            ;;
        "status")
            show_status
            ;;
        "test")
            test_api
            ;;
        "test-with-proxy")
            test_api --enable-proxy-after
            ;;
        "setup-proxy")
            setup_proxy_for_curl
            success "代理已设置为 ${PROXY_HOST}:${PROXY_PORT}"
            ;;
        "disable-proxy")
            disable_proxy_for_local
            success "代理已禁用"
            ;;
        "force-cleanup")
            if [ -f "./force-cleanup.sh" ]; then
                log "运行强力清理..."
                ./force-cleanup.sh
            else
                error "force-cleanup.sh 脚本不存在"
                exit 1
            fi
            ;;
        "logs")
            if [ -f "$LOG_FILE" ]; then
                tail -f "$LOG_FILE"
            else
                error "日志文件不存在"
            fi
            ;;
        "restart")
            log "重启服务器..."
            stop_server
            sleep 2
            setup_environment
            start_server
            ;;
        *)
            echo "用法: $0 {start|stop|status|test|test-with-proxy|setup-proxy|disable-proxy|force-cleanup|logs|restart}"
            echo
            echo "命令说明:"
            echo "  start           - 启动服务器"
            echo "  stop            - 停止服务器"
            echo "  status          - 查看服务器状态"
            echo "  test            - 测试API端点（自动禁用代理）"
            echo "  test-with-proxy - 测试API端点并在完成后重新启用代理"
            echo "  setup-proxy     - 设置代理为 ${PROXY_HOST}:${PROXY_PORT}"
            echo "  disable-proxy   - 禁用代理设置"
            echo "  force-cleanup   - 强力清理所有Wave进程和锁文件"
            echo "  logs            - 查看实时日志"
            echo "  restart         - 重启服务器"
            echo
            echo "代理配置:"
            echo "  当前代理设置: ${PROXY_HOST}:${PROXY_PORT}"
            echo "  注意: API测试会自动禁用代理，外部请求需要重新启用代理"
            echo
            echo "故障排除:"
            echo "  如果遇到锁文件问题，使用: $0 force-cleanup"
            exit 1
            ;;
    esac
}

# 处理信号，确保清理
trap 'stop_server; exit 0' INT TERM

# 执行主函数
main "$@"