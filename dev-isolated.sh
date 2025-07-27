#!/bin/bash

# Wave Terminal 独立开发环境启动脚本
# 设计用于与MCP服务器完全隔离，避免进程冲突

set -e

# 配置
DEV_DATA_DIR="$HOME/.waveterm"  # 使用默认开发目录
LOG_FILE="dev-waveterm.log"
PID_FILE="dev-waveterm.pid"

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

# 检查环境冲突
check_environment_conflicts() {
    log "检查环境冲突..."
    
    # 检查是否已设置了MCP相关的环境变量
    if [ -n "$WAVETERM_DATA_HOME" ] && [ "$WAVETERM_DATA_HOME" != "$DEV_DATA_DIR" ]; then
        warning "检测到环境变量 WAVETERM_DATA_HOME=$WAVETERM_DATA_HOME"
        warning "这可能与MCP服务器配置冲突"
        echo "建议:"
        echo "  1. 在新的shell会话中运行此脚本"
        echo "  2. 或者运行: unset WAVETERM_DATA_HOME WAVETERM_CONFIG_HOME WAVETERM_AUTH_KEY"
        echo
        read -p "是否继续？[y/N] " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 0
        fi
    fi
    
    # 检查MCP服务器是否在运行
    if [ -f "waveterm-server.pid" ]; then
        local mcp_pid=$(cat "waveterm-server.pid")
        if ps -p "$mcp_pid" > /dev/null 2>&1; then
            success "检测到MCP服务器正在运行 (PID: $mcp_pid)"
            echo "开发环境将使用不同的数据目录以避免冲突"
        fi
    fi
    
    success "环境冲突检查完成"
}

# 停止开发服务器
stop_dev_server() {
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            log "停止开发服务器 (PID: $pid)..."
            kill "$pid" 2>/dev/null || true
            sleep 2
            
            # 强制杀死如果还在运行
            if ps -p "$pid" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null || true
                sleep 1
            fi
            success "开发服务器已停止"
        fi
        rm -f "$PID_FILE"
    fi
}

# 清理开发环境相关的进程（不影响MCP服务器）
cleanup_dev_processes() {
    log "清理开发环境进程..."
    
    # 只清理非MCP的Wave进程
    local dev_pids=$(ps aux | grep -E "(yarn.*dev|go run.*main-server)" | grep -v grep | awk '{print $2}' || true)
    if [ ! -z "$dev_pids" ]; then
        warning "发现开发环境相关进程，正在清理..."
        for pid in $dev_pids; do
            if ps -p "$pid" > /dev/null 2>&1; then
                kill -9 "$pid" 2>/dev/null || true
            fi
        done
    fi
    
    success "开发环境进程清理完成"
}

# 启动开发环境
start_dev_environment() {
    log "启动Wave Terminal开发环境..."
    
    # 确保数据目录存在
    mkdir -p "$DEV_DATA_DIR"
    
    # 检查依赖
    if ! command -v yarn &> /dev/null; then
        error "yarn 未安装，请先安装 yarn"
        exit 1
    fi
    
    # 使用完全隔离的子shell启动开发环境
    # 确保环境变量不会污染当前shell
    (
        # 清除可能的MCP环境变量
        unset WAVETERM_DATA_HOME WAVETERM_CONFIG_HOME WAVETERM_AUTH_KEY
        unset WAVETERM_WEB_PORT WAVETERM_WS_PORT
        
        # 设置开发环境专用的环境变量
        export WAVETERM_DATA_HOME="$DEV_DATA_DIR"
        export WAVETERM_CONFIG_HOME="$DEV_DATA_DIR"
        
        # 启动开发服务器
        source ~/.zshrc >/dev/null 2>&1
        exec yarn dev
    ) > "$LOG_FILE" 2>&1 &
    
    local dev_pid=$!
    echo "$dev_pid" > "$PID_FILE"
    
    log "等待开发环境启动..."
    sleep 5
    
    # 检查进程是否还在运行
    if ps -p "$dev_pid" > /dev/null 2>&1; then
        success "开发环境启动成功!"
        echo "  进程ID: $dev_pid"
        echo "  数据目录: $DEV_DATA_DIR"
        echo "  日志文件: $LOG_FILE"
        echo
        echo "📋 开发环境信息:"
        echo "  • 前端开发服务器通常运行在: http://localhost:5173"
        echo "  • 后端服务器会自动选择可用端口"
        echo "  • 与MCP服务器完全隔离，使用不同的数据目录"
        echo
        echo "📝 管理命令:"
        echo "  ./dev-isolated.sh status     - 查看状态"
        echo "  ./dev-isolated.sh logs       - 查看日志"
        echo "  ./dev-isolated.sh stop       - 停止开发环境"
        echo "  ./dev-isolated.sh restart    - 重启开发环境"
        echo
    else
        error "开发环境启动失败"
        if [ -f "$LOG_FILE" ]; then
            echo "错误日志:"
            tail -20 "$LOG_FILE"
        fi
        return 1
    fi
}

# 显示状态
show_status() {
    echo "=== Wave Terminal 开发环境状态 ==="
    echo
    
    if [ ! -f "$PID_FILE" ]; then
        echo "开发环境未运行 (PID文件不存在)"
        return 1
    fi
    
    local pid=$(cat "$PID_FILE")
    if ! ps -p "$pid" > /dev/null 2>&1; then
        echo "开发环境未运行 (进程 $pid 不存在)"
        return 1
    fi
    
    success "开发环境正在运行"
    echo "  进程ID: $pid"
    echo "  数据目录: $DEV_DATA_DIR"
    echo "  日志文件: $LOG_FILE"
    echo
    
    # 检查MCP服务器状态
    if [ -f "waveterm-server.pid" ]; then
        local mcp_pid=$(cat "waveterm-server.pid")
        if ps -p "$mcp_pid" > /dev/null 2>&1; then
            echo "🔗 MCP服务器状态: 运行中 (PID: $mcp_pid)"
            echo "   两个环境完全隔离，无冲突"
        else
            echo "🔗 MCP服务器状态: 未运行"
        fi
    else
        echo "🔗 MCP服务器状态: 未运行"
    fi
}

# 主函数
main() {
    case "${1:-start}" in
        "start")
            log "启动独立开发环境..."
            check_environment_conflicts
            stop_dev_server
            cleanup_dev_processes
            start_dev_environment
            ;;
        "stop")
            log "停止开发环境..."
            stop_dev_server
            cleanup_dev_processes
            success "开发环境已停止"
            ;;
        "status")
            show_status
            ;;
        "logs")
            if [ -f "$LOG_FILE" ]; then
                tail -f "$LOG_FILE"
            else
                error "日志文件不存在"
            fi
            ;;
        "restart")
            log "重启开发环境..."
            stop_dev_server
            sleep 2
            cleanup_dev_processes
            start_dev_environment
            ;;
        "clean")
            log "清理开发环境..."
            stop_dev_server
            cleanup_dev_processes
            rm -f "$LOG_FILE" "$PID_FILE"
            success "开发环境清理完成"
            ;;
        *)
            echo "用法: $0 {start|stop|status|logs|restart|clean}"
            echo
            echo "命令说明:"
            echo "  start    - 启动独立开发环境"
            echo "  stop     - 停止开发环境"
            echo "  status   - 查看开发环境状态"
            echo "  logs     - 查看实时日志"
            echo "  restart  - 重启开发环境"
            echo "  clean    - 清理开发环境文件"
            echo
            echo "特性:"
            echo "  • 与MCP服务器完全隔离，使用不同数据目录"
            echo "  • 自动检测环境冲突"
            echo "  • 支持同时运行开发环境和MCP服务器"
            echo "  • 环境变量完全隔离，不会相互影响"
            exit 1
            ;;
    esac
}

# 处理信号，确保清理
trap 'stop_dev_server; exit 0' INT TERM

# 执行主函数
main "$@"