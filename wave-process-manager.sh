#!/bin/bash

# Wave Terminal 进程管理工具
# 提供统一的进程状态检查、清理和管理功能

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
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

info() {
    echo -e "${CYAN}ℹ️  $1${NC}"
}

# 显示所有Wave相关进程
show_all_processes() {
    echo "=== Wave Terminal 进程状态总览 ==="
    echo
    
    # 查找所有Wave相关进程
    local wave_processes=$(ps aux | grep -E "(wavesrv|main-server|yarn.*dev|go run.*main-server)" | grep -v grep || true)
    
    if [ -z "$wave_processes" ]; then
        info "未发现任何Wave Terminal相关进程"
        echo
    else
        echo "发现的Wave相关进程:"
        echo "USER       PID  %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND"
        echo "$wave_processes"
        echo
    fi
    
    # 检查PID文件状态
    echo "=== PID文件状态 ==="
    check_pid_file "waveterm-server.pid" "MCP服务器"
    check_pid_file "dev-waveterm.pid" "开发环境"
    check_pid_file "server.pid" "通用服务器"
    check_pid_file "waveterm-server.pid" "Wave服务器"
    echo
    
    # 检查锁文件和套接字
    echo "=== 锁文件和套接字状态 ==="
    check_lock_files
    echo
    
    # 检查端口占用
    echo "=== 端口占用状态 ==="
    check_port_usage
    echo
    
    # 检查环境变量
    echo "=== 环境变量状态 ==="
    check_environment_variables
}

# 检查PID文件
check_pid_file() {
    local pid_file="$1"
    local description="$2"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            success "$description: 运行中 (PID: $pid)"
            # 显示进程详细信息
            ps -p "$pid" -o pid,ppid,user,%cpu,%mem,vsz,rss,tty,stat,start,time,command | tail -n +2
        else
            warning "$description: PID文件存在但进程不存在 (PID: $pid)"
            echo "  建议清理: rm -f $pid_file"
        fi
    else
        info "$description: 未运行 (无PID文件)"
    fi
    echo
}

# 检查锁文件和套接字
check_lock_files() {
    local lock_paths=(
        "/tmp/waveterm-mcp/wave.lock"
        "/tmp/waveterm-mcp/wave.sock"
        "$HOME/.waveterm/wave.lock"
        "$HOME/.waveterm/wave.sock"
        "/tmp/wave.lock"
        "/tmp/wave.sock"
    )
    
    for lock_path in "${lock_paths[@]}"; do
        if [ -e "$lock_path" ]; then
            warning "发现锁文件/套接字: $lock_path"
            ls -la "$lock_path"
        fi
    done
    
    info "如果发现孤立的锁文件，可使用 '$0 clean-locks' 清理"
}

# 检查端口占用
check_port_usage() {
    local common_ports=(5173 8080 3000 60289 60290)
    
    for port in "${common_ports[@]}"; do
        local process=$(lsof -ti :$port 2>/dev/null || true)
        if [ -n "$process" ]; then
            success "端口 $port 被占用 (PID: $process)"
            ps -p "$process" -o pid,user,command | tail -n +2
        else
            info "端口 $port 空闲"
        fi
    done
}

# 检查环境变量
check_environment_variables() {
    local env_vars=("WAVETERM_DATA_HOME" "WAVETERM_CONFIG_HOME" "WAVETERM_AUTH_KEY" "WAVETERM_WEB_PORT" "WAVETERM_WS_PORT")
    
    for var in "${env_vars[@]}"; do
        if [ -n "${!var}" ]; then
            warning "$var=${!var}"
        else
            info "$var: 未设置"
        fi
    done
    
    if [ -n "$WAVETERM_DATA_HOME" ]; then
        warning "检测到环境变量污染，这可能导致进程冲突"
        echo "建议在启动新进程前运行: unset WAVETERM_DATA_HOME WAVETERM_CONFIG_HOME WAVETERM_AUTH_KEY"
    fi
}

# 智能清理所有Wave进程
smart_cleanup() {
    log "开始智能清理Wave进程..."
    
    # 首先尝试优雅停止
    graceful_stop_all
    
    sleep 2
    
    # 强制清理残留进程
    force_cleanup_processes
    
    # 清理锁文件
    clean_lock_files
    
    success "智能清理完成"
}

# 优雅停止所有服务
graceful_stop_all() {
    log "尝试优雅停止所有服务..."
    
    # 停止MCP服务器
    if [ -f "persistent-server.sh" ]; then
        log "停止MCP服务器..."
        ./persistent-server.sh stop 2>/dev/null || true
    fi
    
    # 停止开发环境
    if [ -f "dev-isolated.sh" ]; then
        log "停止开发环境..."
        ./dev-isolated.sh stop 2>/dev/null || true
    fi
    
    # 停止其他可能的服务
    for pid_file in waveterm-server.pid dev-waveterm.pid server.pid; do
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if ps -p "$pid" > /dev/null 2>&1; then
                log "停止进程 $pid ($pid_file)..."
                kill "$pid" 2>/dev/null || true
            fi
        fi
    done
}

# 强制清理进程
force_cleanup_processes() {
    log "强制清理残留进程..."
    
    # 查找所有Wave相关进程
    local wave_pids=$(ps aux | grep -E "(wavesrv|main-server|yarn.*dev|go run.*main-server)" | grep -v grep | awk '{print $2}' || true)
    
    if [ -n "$wave_pids" ]; then
        warning "发现残留进程，强制清理..."
        for pid in $wave_pids; do
            if ps -p "$pid" > /dev/null 2>&1; then
                log "强制终止进程 $pid"
                kill -9 "$pid" 2>/dev/null || true
            fi
        done
    else
        success "未发现残留进程"
    fi
}

# 清理锁文件
clean_lock_files() {
    log "清理锁文件和套接字..."
    
    local lock_patterns=(
        "/tmp/waveterm-mcp/wave.*"
        "$HOME/.waveterm/wave.*"
        "/tmp/wave.*"
    )
    
    for pattern in "${lock_patterns[@]}"; do
        for file in $pattern; do
            if [ -e "$file" ]; then
                log "删除: $file"
                rm -f "$file" 2>/dev/null || true
            fi
        done
    done
    
    # 清理PID文件
    local pid_files=(waveterm-server.pid dev-waveterm.pid server.pid)
    for pid_file in "${pid_files[@]}"; do
        if [ -f "$pid_file" ]; then
            log "删除PID文件: $pid_file"
            rm -f "$pid_file"
        fi
    done
    
    success "锁文件清理完成"
}

# 检测环境冲突
detect_conflicts() {
    echo "=== 环境冲突检测 ==="
    echo
    
    local conflicts=0
    
    # 检查环境变量污染
    if [ -n "$WAVETERM_DATA_HOME" ]; then
        warning "环境变量污染: WAVETERM_DATA_HOME=$WAVETERM_DATA_HOME"
        ((conflicts++))
    fi
    
    # 检查多个服务使用相同数据目录
    local running_services=()
    if [ -f "waveterm-server.pid" ] && ps -p "$(cat waveterm-server.pid)" > /dev/null 2>&1; then
        running_services+=("MCP服务器")
    fi
    if [ -f "dev-waveterm.pid" ] && ps -p "$(cat dev-waveterm.pid)" > /dev/null 2>&1; then
        running_services+=("开发环境")
    fi
    
    if [ ${#running_services[@]} -gt 1 ]; then
        warning "多个服务同时运行: ${running_services[*]}"
        echo "检查是否使用了不同的数据目录"
        ((conflicts++))
    fi
    
    # 检查端口冲突
    local used_ports=()
    for port in 5173 8080 3000 60289 60290; do
        if lsof -ti :$port >/dev/null 2>&1; then
            used_ports+=($port)
        fi
    done
    
    if [ ${#used_ports[@]} -gt 0 ]; then
        info "使用中的端口: ${used_ports[*]}"
    fi
    
    # 检查孤立的锁文件
    local orphaned_locks=()
    for lock_file in "/tmp/waveterm-mcp/wave.lock" "$HOME/.waveterm/wave.lock" "/tmp/wave.lock"; do
        if [ -f "$lock_file" ]; then
            # 检查是否有对应的进程
            local lock_found=false
            for pid_file in waveterm-server.pid dev-waveterm.pid server.pid; do
                if [ -f "$pid_file" ] && ps -p "$(cat $pid_file)" > /dev/null 2>&1; then
                    lock_found=true
                    break
                fi
            done
            
            if [ "$lock_found" = false ]; then
                orphaned_locks+=("$lock_file")
            fi
        fi
    done
    
    if [ ${#orphaned_locks[@]} -gt 0 ]; then
        warning "发现孤立的锁文件: ${orphaned_locks[*]}"
        ((conflicts++))
    fi
    
    if [ $conflicts -eq 0 ]; then
        success "未检测到环境冲突"
    else
        warning "检测到 $conflicts 个潜在冲突"
        echo
        echo "建议解决方案:"
        echo "1. 运行 '$0 smart-cleanup' 清理所有进程"
        echo "2. 使用 'dev-isolated.sh' 启动开发环境"
        echo "3. 使用 'persistent-server.sh' 启动MCP服务器"
    fi
    
    echo
}

# 主函数
main() {
    case "${1:-status}" in
        "status"|"show")
            show_all_processes
            ;;
        "conflicts"|"detect")
            detect_conflicts
            ;;
        "smart-cleanup"|"cleanup")
            smart_cleanup
            ;;
        "clean-locks")
            clean_lock_files
            ;;
        "kill-all"|"force-stop")
            force_cleanup_processes
            clean_lock_files
            ;;
        "help"|"-h"|"--help")
            echo "Wave Terminal 进程管理工具"
            echo
            echo "用法: $0 {status|conflicts|smart-cleanup|clean-locks|kill-all|help}"
            echo
            echo "命令说明:"
            echo "  status         - 显示所有Wave进程状态 (默认)"
            echo "  conflicts      - 检测环境冲突"
            echo "  smart-cleanup  - 智能清理所有Wave进程"
            echo "  clean-locks    - 仅清理锁文件和套接字"
            echo "  kill-all       - 强制终止所有Wave进程"
            echo "  help           - 显示此帮助信息"
            echo
            echo "常用操作:"
            echo "  检查状态:     $0 status"
            echo "  解决冲突:     $0 smart-cleanup"
            echo "  快速诊断:     $0 conflicts"
            echo
            ;;
        *)
            error "未知命令: $1"
            echo "使用 '$0 help' 查看可用命令"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"