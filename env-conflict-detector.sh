#!/bin/bash

# Wave Terminal 环境冲突检测工具
# 专门用于检测和解决环境变量作用域污染问题

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[环境检测]${NC} $1"
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

critical() {
    echo -e "${MAGENTA}🚨 $1${NC}"
}

# 检测环境变量污染
detect_env_pollution() {
    echo "=== Wave Terminal 环境污染检测 ==="
    echo
    
    local pollution_detected=false
    local recommendations=()
    
    # 检查关键环境变量
    local critical_vars=("WAVETERM_DATA_HOME" "WAVETERM_CONFIG_HOME" "WAVETERM_AUTH_KEY")
    local port_vars=("WAVETERM_WEB_PORT" "WAVETERM_WS_PORT")
    local all_vars=("${critical_vars[@]}" "${port_vars[@]}")
    
    echo "🔍 检查环境变量状态:"
    for var in "${all_vars[@]}"; do
        if [ -n "${!var}" ]; then
            if [[ " ${critical_vars[@]} " =~ " ${var} " ]]; then
                warning "$var=${!var}"
                pollution_detected=true
            else
                info "$var=${!var}"
            fi
        else
            success "$var: 未设置 (正常)"
        fi
    done
    echo
    
    # 检查进程间的环境变量冲突
    echo "🔍 检查进程环境变量冲突:"
    local mcp_running=false
    local dev_running=false
    
    if [ -f "waveterm-server.pid" ]; then
        local mcp_pid=$(cat "waveterm-server.pid")
        if ps -p "$mcp_pid" > /dev/null 2>&1; then
            mcp_running=true
            info "MCP服务器运行中 (PID: $mcp_pid)"
            
            # 检查MCP服务器的环境变量
            local mcp_env=$(ps eww -p "$mcp_pid" 2>/dev/null | grep -o 'WAVETERM_[^=]*=[^[:space:]]*' || true)
            if [ -n "$mcp_env" ]; then
                echo "  MCP服务器环境变量:"
                echo "$mcp_env" | while read line; do
                    echo "    $line"
                done
            fi
        fi
    fi
    
    if [ -f "dev-waveterm.pid" ]; then
        local dev_pid=$(cat "dev-waveterm.pid")
        if ps -p "$dev_pid" > /dev/null 2>&1; then
            dev_running=true
            info "开发环境运行中 (PID: $dev_pid)"
            
            # 检查开发环境的环境变量
            local dev_env=$(ps eww -p "$dev_pid" 2>/dev/null | grep -o 'WAVETERM_[^=]*=[^[:space:]]*' || true)
            if [ -n "$dev_env" ]; then
                echo "  开发环境环境变量:"
                echo "$dev_env" | while read line; do
                    echo "    $line"
                done
            fi
        fi
    fi
    echo
    
    # 分析潜在冲突
    echo "🔍 分析潜在冲突:"
    
    if [ "$pollution_detected" = true ]; then
        critical "检测到环境变量污染！"
        echo "当前shell环境中存在Wave Terminal环境变量，这可能导致："
        echo "  • 新启动的进程继承错误的配置"
        echo "  • 多个Wave实例争夺同一个锁文件"
        echo "  • 数据目录冲突"
        echo "  • 端口冲突"
        recommendations+=("清理当前shell的环境变量")
        echo
    fi
    
    if [ "$mcp_running" = true ] && [ "$dev_running" = true ]; then
        warning "MCP服务器和开发环境同时运行"
        echo "需要确保它们使用不同的数据目录"
        recommendations+=("检查两个服务是否使用了不同的数据目录")
        echo
    fi
    
    if [ "$pollution_detected" = true ] && { [ "$mcp_running" = true ] || [ "$dev_running" = true ]; }; then
        critical "高风险: 环境污染 + 服务运行"
        echo "这是最容易导致锁文件冲突的情况！"
        recommendations+=("立即停止所有服务并清理环境")
        echo
    fi
    
    # 检查数据目录冲突
    echo "🔍 检查数据目录冲突:"
    local data_dirs=()
    
    if [ -n "$WAVETERM_DATA_HOME" ]; then
        data_dirs+=("当前环境: $WAVETERM_DATA_HOME")
    fi
    
    # 检查MCP配置
    if [ -f "persistent-server.sh" ]; then
        local mcp_data_dir=$(grep 'DATA_DIR=' persistent-server.sh | head -1 | cut -d'"' -f2)
        if [ -n "$mcp_data_dir" ]; then
            data_dirs+=("MCP服务器: $mcp_data_dir")
        fi
    fi
    
    # 检查开发环境配置
    if [ -f "dev-isolated.sh" ]; then
        local dev_data_dir=$(grep 'DEV_DATA_DIR=' dev-isolated.sh | head -1 | cut -d'"' -f2)
        if [ -n "$dev_data_dir" ]; then
            data_dirs+=("开发环境: $dev_data_dir")
        fi
    fi
    
    if [ ${#data_dirs[@]} -gt 0 ]; then
        echo "发现的数据目录配置:"
        for dir in "${data_dirs[@]}"; do
            echo "  $dir"
        done
        
        # 检查是否有重复
        local unique_dirs=$(printf '%s\n' "${data_dirs[@]}" | cut -d':' -f2 | sort -u)
        local unique_count=$(echo "$unique_dirs" | wc -l)
        local total_count=${#data_dirs[@]}
        
        if [ $unique_count -lt $total_count ]; then
            critical "检测到数据目录冲突！"
            echo "多个服务使用相同的数据目录，这会导致锁文件争夺"
            recommendations+=("确保每个服务使用独立的数据目录")
        else
            success "数据目录配置正确，没有冲突"
        fi
    else
        info "未检测到明确的数据目录配置"
    fi
    echo
    
    # 提供解决建议
    if [ ${#recommendations[@]} -gt 0 ]; then
        echo "📋 解决建议:"
        for i in "${!recommendations[@]}"; do
            echo "$((i+1)). ${recommendations[i]}"
        done
        echo
        
        echo "🛠️  具体操作:"
        echo "清理环境变量:"
        echo "  unset WAVETERM_DATA_HOME WAVETERM_CONFIG_HOME WAVETERM_AUTH_KEY"
        echo "  unset WAVETERM_WEB_PORT WAVETERM_WS_PORT"
        echo
        echo "安全重启所有服务:"
        echo "  ./wave-process-manager.sh smart-cleanup"
        echo "  # 等待几秒"
        echo "  ./persistent-server.sh start    # 启动MCP服务器"
        echo "  ./dev-isolated.sh start         # 在新shell中启动开发环境"
        echo
    else
        success "环境配置良好，未检测到冲突风险"
    fi
}

# 检查锁文件状态
check_lock_conflicts() {
    echo "=== 锁文件冲突检测 ==="
    echo
    
    local lock_files=(
        "/tmp/waveterm-mcp/wave.lock"
        "$HOME/.waveterm/wave.lock"
        "/tmp/wave.lock"
    )
    
    local socket_files=(
        "/tmp/waveterm-mcp/wave.sock"
        "$HOME/.waveterm/wave.sock"
        "/tmp/wave.sock"
    )
    
    local conflicts=false
    
    echo "🔍 检查锁文件:"
    for lock_file in "${lock_files[@]}"; do
        if [ -f "$lock_file" ]; then
            warning "发现锁文件: $lock_file"
            ls -la "$lock_file"
            
            # 尝试确定哪个进程创建了这个锁文件
            local lock_owner="未知"
            if [[ "$lock_file" == *"waveterm-mcp"* ]]; then
                lock_owner="可能是MCP服务器"
            elif [[ "$lock_file" == *".waveterm"* ]]; then
                lock_owner="可能是开发环境"
            fi
            echo "  $lock_owner"
            conflicts=true
        else
            success "锁文件不存在: $lock_file"
        fi
    done
    echo
    
    echo "🔍 检查套接字文件:"
    for socket_file in "${socket_files[@]}"; do
        if [ -S "$socket_file" ]; then
            warning "发现套接字: $socket_file"
            ls -la "$socket_file"
            conflicts=true
        else
            success "套接字不存在: $socket_file"
        fi
    done
    echo
    
    if [ "$conflicts" = true ]; then
        critical "检测到锁文件/套接字冲突"
        echo "建议操作:"
        echo "1. 停止所有Wave服务"
        echo "2. 清理锁文件: ./wave-process-manager.sh clean-locks"
        echo "3. 重新启动服务"
    else
        success "无锁文件冲突"
    fi
}

# 生成环境报告
generate_report() {
    local report_file="wave-env-report-$(date +%Y%m%d-%H%M%S).txt"
    
    echo "生成环境冲突检测报告..."
    
    {
        echo "Wave Terminal 环境冲突检测报告"
        echo "生成时间: $(date)"
        echo "=========================================="
        echo
        
        echo "1. 环境变量状态:"
        for var in WAVETERM_DATA_HOME WAVETERM_CONFIG_HOME WAVETERM_AUTH_KEY WAVETERM_WEB_PORT WAVETERM_WS_PORT; do
            echo "   $var=${!var:-<未设置>}"
        done
        echo
        
        echo "2. 运行中的进程:"
        ps aux | grep -E "(wavesrv|main-server|yarn.*dev|go run.*main-server)" | grep -v grep || echo "   无Wave进程运行"
        echo
        
        echo "3. PID文件状态:"
        for pid_file in waveterm-server.pid dev-waveterm.pid server.pid; do
            if [ -f "$pid_file" ]; then
                local pid=$(cat "$pid_file")
                if ps -p "$pid" > /dev/null 2>&1; then
                    echo "   $pid_file: 活动 (PID: $pid)"
                else
                    echo "   $pid_file: 孤立 (PID: $pid, 进程不存在)"
                fi
            else
                echo "   $pid_file: 不存在"
            fi
        done
        echo
        
        echo "4. 锁文件状态:"
        for lock_file in "/tmp/waveterm-mcp/wave.lock" "$HOME/.waveterm/wave.lock" "/tmp/wave.lock"; do
            if [ -f "$lock_file" ]; then
                echo "   $lock_file: 存在 ($(ls -la "$lock_file"))"
            else
                echo "   $lock_file: 不存在"
            fi
        done
        echo
        
        echo "5. 端口占用:"
        for port in 5173 8080 3000 60289 60290; do
            local process=$(lsof -ti :$port 2>/dev/null || true)
            if [ -n "$process" ]; then
                echo "   端口 $port: 被占用 (PID: $process)"
            else
                echo "   端口 $port: 空闲"
            fi
        done
        echo
        
    } > "$report_file"
    
    success "报告已生成: $report_file"
}

# 交互式修复助手
interactive_fix() {
    echo "=== Wave Terminal 交互式修复助手 ==="
    echo
    
    # 先做快速检测
    local has_env_pollution=false
    local has_running_services=false
    local has_lock_conflicts=false
    
    # 检查环境污染
    if [ -n "$WAVETERM_DATA_HOME" ] || [ -n "$WAVETERM_CONFIG_HOME" ] || [ -n "$WAVETERM_AUTH_KEY" ]; then
        has_env_pollution=true
    fi
    
    # 检查运行中的服务
    if ([ -f "waveterm-server.pid" ] && ps -p "$(cat waveterm-server.pid)" > /dev/null 2>&1) || \
       ([ -f "dev-waveterm.pid" ] && ps -p "$(cat dev-waveterm.pid)" > /dev/null 2>&1); then
        has_running_services=true
    fi
    
    # 检查锁文件
    if [ -f "/tmp/waveterm-mcp/wave.lock" ] || [ -f "$HOME/.waveterm/wave.lock" ] || [ -f "/tmp/wave.lock" ]; then
        has_lock_conflicts=true
    fi
    
    echo "🔍 快速检测结果:"
    echo "  环境变量污染: $([ "$has_env_pollution" = true ] && echo "是" || echo "否")"
    echo "  运行中的服务: $([ "$has_running_services" = true ] && echo "是" || echo "否")"
    echo "  锁文件冲突: $([ "$has_lock_conflicts" = true ] && echo "是" || echo "否")"
    echo
    
    if [ "$has_env_pollution" = false ] && [ "$has_running_services" = false ] && [ "$has_lock_conflicts" = false ]; then
        success "恭喜！未检测到任何问题"
        return 0
    fi
    
    echo "🛠️  修复选项:"
    echo "1. 一键智能修复 (推荐)"
    echo "2. 仅清理环境变量"
    echo "3. 仅停止服务"
    echo "4. 仅清理锁文件"
    echo "5. 生成详细报告"
    echo "6. 退出"
    echo
    
    read -p "请选择操作 [1-6]: " choice
    
    case $choice in
        1)
            echo "执行一键智能修复..."
            ./wave-process-manager.sh smart-cleanup
            echo
            echo "环境变量清理命令 (请手动执行):"
            echo "unset WAVETERM_DATA_HOME WAVETERM_CONFIG_HOME WAVETERM_AUTH_KEY WAVETERM_WEB_PORT WAVETERM_WS_PORT"
            ;;
        2)
            echo "环境变量清理命令 (请手动执行):"
            echo "unset WAVETERM_DATA_HOME WAVETERM_CONFIG_HOME WAVETERM_AUTH_KEY WAVETERM_WEB_PORT WAVETERM_WS_PORT"
            ;;
        3)
            echo "停止所有服务..."
            ./wave-process-manager.sh smart-cleanup
            ;;
        4)
            echo "清理锁文件..."
            ./wave-process-manager.sh clean-locks
            ;;
        5)
            generate_report
            ;;
        6)
            echo "退出"
            ;;
        *)
            warning "无效选择"
            ;;
    esac
}

# 主函数
main() {
    case "${1:-detect}" in
        "detect"|"check")
            detect_env_pollution
            check_lock_conflicts
            ;;
        "locks")
            check_lock_conflicts
            ;;
        "report")
            generate_report
            ;;
        "interactive"|"fix")
            interactive_fix
            ;;
        "help"|"-h"|"--help")
            echo "Wave Terminal 环境冲突检测工具"
            echo
            echo "用法: $0 {detect|locks|report|interactive|help}"
            echo
            echo "命令说明:"
            echo "  detect      - 全面检测环境冲突 (默认)"
            echo "  locks       - 仅检测锁文件冲突"
            echo "  report      - 生成详细检测报告"
            echo "  interactive - 交互式修复助手"
            echo "  help        - 显示此帮助信息"
            echo
            echo "常用操作:"
            echo "  快速检测:   $0 detect"
            echo "  交互修复:   $0 interactive"
            echo "  生成报告:   $0 report"
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