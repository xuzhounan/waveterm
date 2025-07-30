#!/bin/bash

# Wave Terminal 统一进程管理器
# 解决多实例管理混乱问题，提供统一的进程生命周期管理

set -e

# 配置
REGISTRY_FILE="$HOME/.waveterm/process-registry.json"
LOCK_FILE="$HOME/.waveterm/process-manager.lock"
MANAGER_LOG="$HOME/.waveterm/process-manager.log"

# 端口范围规划
DEV_PORT_BASE=8090      # 开发环境：8090-8099
MCP_PORT_BASE=60289     # MCP服务器：60289-60299
TEST_PORT_BASE=9090     # 测试环境：9090-9099

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$MANAGER_LOG"
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

info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

# 初始化管理器
init_manager() {
    # 创建必要目录
    mkdir -p "$(dirname "$REGISTRY_FILE")"
    mkdir -p "$(dirname "$MANAGER_LOG")"
    
    # 创建空的注册表（如果不存在）
    if [ ! -f "$REGISTRY_FILE" ]; then
        cat > "$REGISTRY_FILE" << 'EOF'
{
  "instances": {},
  "last_updated": 0,
  "manager_version": "1.0.0"
}
EOF
    fi
}

# 获取互斥锁
acquire_lock() {
    local timeout=${1:-10}
    local count=0
    
    while [ $count -lt $timeout ]; do
        if ( set -C; echo $$ > "$LOCK_FILE" ) 2>/dev/null; then
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done
    
    error "获取锁失败，可能有其他进程管理器实例在运行"
    return 1
}

# 释放互斥锁
release_lock() {
    rm -f "$LOCK_FILE"
}

# 清理函数
cleanup() {
    release_lock
    exit 0
}

# 设置信号处理
trap cleanup INT TERM EXIT

# 检查端口是否可用
is_port_available() {
    local port=$1
    ! lsof -i ":$port" >/dev/null 2>&1
}

# 找到可用端口
find_available_port() {
    local base_port=$1
    local range=${2:-10}
    
    for i in $(seq 0 $((range - 1))); do
        local port=$((base_port + i))
        if is_port_available $port; then
            echo $port
            return 0
        fi
    done
    
    return 1
}

# 更新服务注册表
update_registry() {
    local instance_type=$1
    local action=$2
    local data="$3"
    
    if ! acquire_lock; then
        return 1
    fi
    
    local temp_file=$(mktemp)
    
    if [ "$action" = "register" ]; then
        # 注册新实例
        jq --arg type "$instance_type" --argjson data "$data" \
           '.instances[$type] = $data | .last_updated = now' \
           "$REGISTRY_FILE" > "$temp_file"
    elif [ "$action" = "unregister" ]; then
        # 注销实例
        jq --arg type "$instance_type" \
           'del(.instances[$type]) | .last_updated = now' \
           "$REGISTRY_FILE" > "$temp_file"
    elif [ "$action" = "update_status" ]; then
        # 更新状态
        jq --arg type "$instance_type" --arg status "$data" \
           '.instances[$type].status = $status | .last_updated = now' \
           "$REGISTRY_FILE" > "$temp_file"
    fi
    
    mv "$temp_file" "$REGISTRY_FILE"
    release_lock
}

# 获取实例信息
get_instance_info() {
    local instance_type=$1
    jq -r ".instances.\"$instance_type\" // empty" "$REGISTRY_FILE"
}

# 检查实例健康状态
check_instance_health() {
    local instance_type=$1
    local instance_info=$(get_instance_info "$instance_type")
    
    if [ -z "$instance_info" ]; then
        return 1
    fi
    
    local pid=$(echo "$instance_info" | jq -r '.pid // empty')
    local web_port=$(echo "$instance_info" | jq -r '.web_port // empty')
    
    # 进程检查
    if [ -n "$pid" ] && ! ps -p "$pid" > /dev/null 2>&1; then
        update_registry "$instance_type" "update_status" "dead"
        return 1
    fi
    
    # 端口检查
    if [ -n "$web_port" ]; then
        if curl -s -f --max-time 3 "http://127.0.0.1:$web_port/health" > /dev/null 2>&1 || \
           curl -s -f --max-time 3 "http://127.0.0.1:$web_port/api/v1/widgets" > /dev/null 2>&1; then
            update_registry "$instance_type" "update_status" "healthy"
            return 0
        else
            update_registry "$instance_type" "update_status" "unhealthy"
            return 1
        fi
    fi
    
    return 0
}

# 启动开发环境
start_dev() {
    log "启动开发环境..."
    
    # 检查是否已经运行
    if check_instance_health "dev"; then
        warning "开发环境已在运行"
        show_instance_status "dev"
        return 0
    fi
    
    # 查找可用端口
    local web_port
    local ws_port
    
    web_port=$(find_available_port $DEV_PORT_BASE)
    if [ -z "$web_port" ]; then
        error "无法找到可用的Web端口（范围：$DEV_PORT_BASE-$((DEV_PORT_BASE + 9))）"
        return 1
    fi
    
    ws_port=$(find_available_port $((web_port + 1)))
    if [ -z "$ws_port" ]; then
        error "无法找到可用的WebSocket端口"
        return 1
    fi
    
    info "分配端口 - Web: $web_port, WebSocket: $ws_port"
    
    # 创建日志目录
    mkdir -p logs
    local log_file="logs/wave-dev.log"
    
    # 启动开发环境
    (
        export WAVETERM_DATA_HOME="$HOME/Library/Application Support/waveterm"
        export WAVETERM_CONFIG_HOME="$HOME/Library/Application Support/waveterm"
        export WCLOUD_ENDPOINT="https://api.waveterm.dev/central"
        export WCLOUD_WS_ENDPOINT="wss://wsapi.waveterm.dev/"
        export WAVETERM_WEB_PORT="$web_port"
        export WAVETERM_WS_PORT="$ws_port"
        
        source ~/.zshrc >/dev/null 2>&1
        yarn dev
    ) > "$log_file" 2>&1 &
    
    local pid=$!
    
    # 注册实例
    local instance_data=$(cat <<EOF
{
  "pid": $pid,
  "web_port": $web_port,
  "ws_port": $ws_port,
  "data_dir": "$HOME/.waveterm",
  "log_file": "$log_file",
  "status": "starting",
  "started_at": $(date +%s),
  "type": "development"
}
EOF
)
    
    update_registry "dev" "register" "$instance_data"
    
    # 等待启动
    log "等待开发环境启动..."
    local max_wait=30
    local count=0
    
    while [ $count -lt $max_wait ]; do
        if check_instance_health "dev"; then
            success "开发环境启动成功！"
            show_instance_status "dev"
            return 0
        fi
        
        if ! ps -p "$pid" > /dev/null 2>&1; then
            error "开发环境进程意外退出"
            update_registry "dev" "unregister"
            return 1
        fi
        
        sleep 1
        count=$((count + 1))
        echo -n "."
    done
    
    echo
    error "开发环境启动超时"
    kill "$pid" 2>/dev/null || true
    update_registry "dev" "unregister"
    return 1
}

# 启动MCP服务器
start_mcp() {
    log "启动MCP服务器..."
    
    # 检查是否已经运行
    if check_instance_health "mcp"; then
        warning "MCP服务器已在运行"
        show_instance_status "mcp"
        return 0
    fi
    
    # 使用固定端口（与persistent-server.sh保持一致）
    local web_port=$MCP_PORT_BASE
    local ws_port=$((MCP_PORT_BASE + 1))
    
    # 检查端口是否可用
    if ! is_port_available $web_port; then
        error "MCP Web端口 $web_port 被占用"
        return 1
    fi
    
    if ! is_port_available $ws_port; then
        error "MCP WebSocket端口 $ws_port 被占用"
        return 1
    fi
    
    info "使用端口 - Web: $web_port, WebSocket: $ws_port"
    
    # 创建数据目录和日志
    local data_dir="$HOME/Library/Application Support/waveterm"
    mkdir -p "$data_dir"
    mkdir -p logs
    local log_file="logs/wave-mcp.log"
    
    # 启动MCP服务器
    (
        export WAVETERM_DATA_HOME="$data_dir"
        export WAVETERM_CONFIG_HOME="$data_dir"
        export WAVETERM_WEB_PORT="$web_port"
        export WAVETERM_WS_PORT="$ws_port"
        export WAVETERM_AUTH_KEY="83958e47ddc89fae695a7e1eb429899871e80334bd58cfc2d17a80388791f073"
        
        exec go run cmd/server/main-server.go
    ) > "$log_file" 2>&1 &
    
    local pid=$!
    
    # 注册实例
    local instance_data=$(cat <<EOF
{
  "pid": $pid,
  "web_port": $web_port,
  "ws_port": $ws_port,
  "data_dir": "$data_dir",
  "log_file": "$log_file",
  "status": "starting",
  "started_at": $(date +%s),
  "type": "mcp-server"
}
EOF
)
    
    update_registry "mcp" "register" "$instance_data"
    
    # 等待启动
    log "等待MCP服务器启动..."
    local max_wait=30
    local count=0
    
    while [ $count -lt $max_wait ]; do
        if check_instance_health "mcp"; then
            success "MCP服务器启动成功！"
            show_instance_status "mcp"
            return 0
        fi
        
        if ! ps -p "$pid" > /dev/null 2>&1; then
            error "MCP服务器进程意外退出"
            update_registry "mcp" "unregister"
            return 1
        fi
        
        sleep 1
        count=$((count + 1))
        echo -n "."
    done
    
    echo
    error "MCP服务器启动超时"
    kill "$pid" 2>/dev/null || true
    update_registry "mcp" "unregister"
    return 1
}

# 停止实例
stop_instance() {
    local instance_type=$1
    local instance_info=$(get_instance_info "$instance_type")
    
    if [ -z "$instance_info" ]; then
        warning "$instance_type 实例未注册"
        return 0
    fi
    
    local pid=$(echo "$instance_info" | jq -r '.pid // empty')
    
    if [ -n "$pid" ] && ps -p "$pid" > /dev/null 2>&1; then
        log "停止 $instance_type 实例 (PID: $pid)..."
        kill "$pid" 2>/dev/null || true
        sleep 2
        
        # 强制杀死
        if ps -p "$pid" > /dev/null 2>&1; then
            kill -9 "$pid" 2>/dev/null || true
            sleep 1
        fi
        
        success "$instance_type 实例已停止"
    else
        warning "$instance_type 实例进程不存在"
    fi
    
    update_registry "$instance_type" "unregister"
}

# 显示实例状态
show_instance_status() {
    local instance_type=$1
    local instance_info=$(get_instance_info "$instance_type")
    
    if [ -z "$instance_info" ]; then
        echo "  $instance_type: 未运行"
        return
    fi
    
    local pid=$(echo "$instance_info" | jq -r '.pid // "N/A"')
    local web_port=$(echo "$instance_info" | jq -r '.web_port // "N/A"')
    local ws_port=$(echo "$instance_info" | jq -r '.ws_port // "N/A"')
    local status=$(echo "$instance_info" | jq -r '.status // "unknown"')
    local started_at=$(echo "$instance_info" | jq -r '.started_at // 0')
    
    # 计算运行时间
    local uptime="N/A"
    if [ "$started_at" != "0" ]; then
        local current_time=$(date +%s)
        local uptime_seconds=$((current_time - started_at))
        uptime="${uptime_seconds}s"
    fi
    
    # 状态颜色
    local status_color="$NC"
    case "$status" in
        "healthy") status_color="$GREEN" ;;
        "unhealthy") status_color="$YELLOW" ;;
        "dead") status_color="$RED" ;;
        "starting") status_color="$BLUE" ;;
    esac
    
    echo -e "  ${CYAN}$instance_type${NC}:"
    echo -e "    状态: ${status_color}$status${NC}"
    echo -e "    PID: $pid"
    echo -e "    端口: Web($web_port) WebSocket($ws_port)"
    echo -e "    运行时间: $uptime"
    if [ "$web_port" != "N/A" ]; then
        echo -e "    API: http://127.0.0.1:$web_port/api/v1/widgets"
    fi
}

# 显示所有状态
show_status() {
    echo "=== Wave Terminal 进程管理器状态 ==="
    echo
    
    # 更新所有实例的健康状态
    local instances=$(jq -r '.instances | keys[]' "$REGISTRY_FILE" 2>/dev/null || echo "")
    
    if [ -z "$instances" ]; then
        info "没有运行的实例"
        echo
        echo "可用命令:"
        echo "  $0 dev     # 启动开发环境"
        echo "  $0 mcp     # 启动MCP服务器" 
        echo "  $0 full    # 同时启动开发环境和MCP服务器"
        return
    fi
    
    for instance_type in $instances; do
        check_instance_health "$instance_type" > /dev/null 2>&1 || true
        show_instance_status "$instance_type"
        echo
    done
    
    # 显示端口占用情况
    echo "=== 端口占用情况 ==="
    echo "开发环境端口范围: $DEV_PORT_BASE-$((DEV_PORT_BASE + 9))"
    echo "MCP服务器端口范围: $MCP_PORT_BASE-$((MCP_PORT_BASE + 9))"
    echo
    lsof -i ":8090-8099,60289-60299" 2>/dev/null | head -10 || echo "无相关端口占用"
}

# 清理所有实例
cleanup_all() {
    log "清理所有Wave Terminal实例..."
    
    local instances=$(jq -r '.instances | keys[]' "$REGISTRY_FILE" 2>/dev/null || echo "")
    
    for instance_type in $instances; do
        stop_instance "$instance_type"
    done
    
    # 清理孤儿进程
    local orphan_pids=$(ps aux | grep -E "(go run.*main-server|yarn.*dev)" | grep -v grep | awk '{print $2}' || true)
    if [ -n "$orphan_pids" ]; then
        warning "发现孤儿进程，正在清理..."
        for pid in $orphan_pids; do
            if ps -p "$pid" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null || true
            fi
        done
    fi
    
    # 清理锁文件和临时文件
    rm -f "$HOME/.waveterm/wave.lock" "$HOME/.waveterm/wave.sock" 2>/dev/null || true
    rm -f "$HOME/Library/Application Support/waveterm/wave.lock" 2>/dev/null || true
    
    success "所有实例已清理完成"
}

# 启动完整环境（开发环境 + MCP服务器）
start_full() {
    log "启动完整开发环境（开发环境 + MCP服务器）..."
    
    # 启动MCP服务器
    if ! start_mcp; then
        error "MCP服务器启动失败"
        return 1
    fi
    
    # 等待MCP服务器稳定
    sleep 2
    
    # 启动开发环境
    if ! start_dev; then
        error "开发环境启动失败"
        return 1
    fi
    
    echo
    success "完整开发环境启动成功！"
    echo
    echo "📋 环境信息："
    show_status
    
    echo
    echo "🔗 MCP连接信息："
    local mcp_info=$(get_instance_info "mcp")
    local mcp_port=$(echo "$mcp_info" | jq -r '.web_port // "N/A"')
    echo "  Claude Code MCP应连接到: http://127.0.0.1:$mcp_port"
    echo "  环境变量: WAVETERM_WEB_PORT=$mcp_port"
}

# 健康检查所有实例
health_check() {
    echo "=== 健康检查 ==="
    
    local instances=$(jq -r '.instances | keys[]' "$REGISTRY_FILE" 2>/dev/null || echo "")
    local all_healthy=true
    
    for instance_type in $instances; do
        echo -n "检查 $instance_type... "
        if check_instance_health "$instance_type"; then
            echo -e "${GREEN}健康${NC}"
        else
            echo -e "${RED}不健康${NC}"
            all_healthy=false
        fi
    done
    
    if $all_healthy; then
        success "所有实例运行正常"
    else
        warning "存在不健康的实例"
        return 1
    fi
}

# 主函数
main() {
    init_manager
    
    case "${1:-status}" in
        "dev")
            start_dev
            ;;
        "mcp")
            start_mcp
            ;;
        "full")
            start_full
            ;;
        "stop")
            if [ -n "$2" ]; then
                stop_instance "$2"
            else
                cleanup_all
            fi
            ;;
        "status"|"list")
            show_status
            ;;
        "cleanup")
            cleanup_all
            ;;
        "health")
            health_check
            ;;
        "logs")
            local instance_type=${2:-dev}
            local instance_info=$(get_instance_info "$instance_type")
            if [ -n "$instance_info" ]; then
                local log_file=$(echo "$instance_info" | jq -r '.log_file // empty')
                if [ -n "$log_file" ] && [ -f "$log_file" ]; then
                    tail -f "$log_file"
                else
                    error "$instance_type 实例的日志文件不存在"
                fi
            else
                error "$instance_type 实例未运行"
            fi
            ;;
        "restart")
            local instance_type=${2:-all}
            if [ "$instance_type" = "all" ]; then
                cleanup_all
                sleep 2
                start_full
            else
                stop_instance "$instance_type"
                sleep 2
                if [ "$instance_type" = "dev" ]; then
                    start_dev
                elif [ "$instance_type" = "mcp" ]; then
                    start_mcp
                fi
            fi
            ;;
        *)
            echo "Wave Terminal 统一进程管理器"
            echo
            echo "用法: $0 <command> [args]"
            echo
            echo "命令:"
            echo "  dev              启动开发环境"
            echo "  mcp              启动MCP服务器"
            echo "  full             启动完整环境（开发环境 + MCP服务器）"
            echo "  stop [type]      停止实例（不指定type则停止所有）"
            echo "  status           显示所有实例状态"
            echo "  cleanup          清理所有实例和孤儿进程"
            echo "  health           健康检查"
            echo "  logs [type]      查看实例日志（默认dev）"
            echo "  restart [type]   重启实例（默认all）"
            echo
            echo "实例类型:"
            echo "  dev              开发环境"
            echo "  mcp              MCP服务器"
            echo
            echo "端口分配:"
            echo "  开发环境: $DEV_PORT_BASE-$((DEV_PORT_BASE + 9))"
            echo "  MCP服务器: $MCP_PORT_BASE-$((MCP_PORT_BASE + 9))"
            echo
            echo "日志文件位置:"
            echo "  管理器日志: $MANAGER_LOG"
            echo "  实例日志: logs/"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"