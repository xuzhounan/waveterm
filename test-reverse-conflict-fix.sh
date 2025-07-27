#!/bin/bash

# 测试反向锁冲突修复的验证脚本
# 用于验证MCP服务器启动时不会影响已运行的开发环境

set -e

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

# 模拟开发环境进程
simulate_dev_environment() {
    log "模拟开发环境进程..."
    
    # 创建模拟的开发环境数据目录
    local dev_data_dir="$HOME/.waveterm-dev-test"
    mkdir -p "$dev_data_dir"
    
    # 启动模拟的yarn dev进程（使用sleep模拟）
    echo "模拟yarn dev进程" > yarn-dev-test.log
    sleep 300 > yarn-dev-test.log 2>&1 &
    local yarn_pid=$!
    echo "$yarn_pid" > dev-yarn.pid
    
    # 启动模拟的main-server进程（使用不同数据目录）
    echo "模拟开发环境main-server进程" > dev-main-server-test.log
    WAVETERM_DATA_HOME="$dev_data_dir" sleep 300 > dev-main-server-test.log 2>&1 &
    local dev_server_pid=$!
    echo "$dev_server_pid" > dev-main-server.pid
    
    success "模拟开发环境启动成功"
    echo "  Yarn进程 PID: $yarn_pid"
    echo "  开发服务器 PID: $dev_server_pid"
    echo "  开发数据目录: $dev_data_dir"
    
    return 0
}

# 检查进程状态
check_process_status() {
    local desc="$1"
    local pid_file="$2"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p "$pid" > /dev/null 2>&1; then
            success "$desc 正在运行 (PID: $pid)"
            return 0
        else
            error "$desc 已停止 (PID: $pid)"
            return 1
        fi
    else
        warning "$desc PID文件不存在"
        return 1
    fi
}

# 清理测试环境
cleanup_test() {
    log "清理测试环境..."
    
    # 停止模拟进程
    for pid_file in dev-yarn.pid dev-main-server.pid; do
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if ps -p "$pid" > /dev/null 2>&1; then
                kill "$pid" 2>/dev/null || true
            fi
            rm -f "$pid_file"
        fi
    done
    
    # 清理日志文件
    rm -f yarn-dev-test.log dev-main-server-test.log
    
    # 清理测试数据目录
    rm -rf "$HOME/.waveterm-dev-test"
    
    success "测试环境清理完成"
}

# 运行测试
run_test() {
    log "开始反向锁冲突修复测试..."
    echo
    
    # 清理可能存在的测试残留
    cleanup_test
    
    # 第一步：模拟开发环境
    log "=== 第一步：启动模拟开发环境 ==="
    simulate_dev_environment
    sleep 2
    
    # 检查开发环境状态
    log "=== 检查开发环境初始状态 ==="
    check_process_status "Yarn Dev" "dev-yarn.pid"
    check_process_status "开发服务器" "dev-main-server.pid"
    echo
    
    # 第二步：启动MCP服务器（这里会触发修复的逻辑）
    log "=== 第二步：启动MCP服务器 ==="
    log "注意：这里会测试修复后的stop_server()函数是否正确保护开发环境"
    
    # 运行persistent-server.sh的start命令（它会调用stop_server）
    if ./persistent-server.sh start; then
        success "MCP服务器启动成功"
    else
        error "MCP服务器启动失败"
        cleanup_test
        return 1
    fi
    echo
    
    # 第三步：检查开发环境是否受影响
    log "=== 第三步：检查开发环境是否受影响 ==="
    local yarn_ok=false
    local server_ok=false
    
    if check_process_status "Yarn Dev" "dev-yarn.pid"; then
        yarn_ok=true
    fi
    
    if check_process_status "开发服务器" "dev-main-server.pid"; then
        server_ok=true
    fi
    
    echo
    
    # 第四步：测试结果
    log "=== 测试结果 ==="
    if $yarn_ok && $server_ok; then
        success "✅ 反向锁冲突修复成功！"
        echo "   开发环境进程在MCP服务器启动后仍然正常运行"
        echo "   修复方案有效"
    else
        error "❌ 反向锁冲突修复失败！"
        echo "   开发环境进程被误杀"
        echo "   需要进一步改进修复方案"
    fi
    
    echo
    
    # 第五步：清理
    log "=== 清理测试环境 ==="
    
    # 停止MCP服务器
    if ./persistent-server.sh stop; then
        success "MCP服务器已停止"
    fi
    
    # 清理测试进程
    cleanup_test
    
    echo
    if $yarn_ok && $server_ok; then
        success "🎉 测试通过！反向锁冲突问题已解决"
        return 0
    else
        error "💥 测试失败！需要进一步修复"
        return 1
    fi
}

# 显示帮助信息
show_help() {
    echo "反向锁冲突修复测试脚本"
    echo
    echo "用法: $0 [选项]"
    echo
    echo "选项:"
    echo "  test     - 运行完整的反向冲突测试"
    echo "  cleanup  - 清理测试环境"
    echo "  help     - 显示此帮助信息"
    echo
    echo "测试流程:"
    echo "  1. 启动模拟的开发环境进程"
    echo "  2. 启动MCP服务器（触发stop_server函数）"
    echo "  3. 检查开发环境进程是否被误杀"
    echo "  4. 验证修复方案的有效性"
    echo
}

# 主函数
main() {
    case "${1:-test}" in
        "test")
            run_test
            ;;
        "cleanup")
            cleanup_test
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
}

# 处理信号，确保清理
trap 'cleanup_test; exit 0' INT TERM

# 执行主函数
main "$@"