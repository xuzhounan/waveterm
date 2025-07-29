#!/bin/bash

# Wave Terminal Claude Code 部署脚本
# 一键部署Wave Terminal与Claude Code的MCP集成

set -e

echo "🚀 Wave Terminal Claude Code 部署脚本"
echo "======================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查系统依赖..."
    
    # 检查Node.js
    if ! command -v node &> /dev/null; then
        log_error "Node.js 未安装，请先安装 Node.js 18+"
        exit 1
    fi
    
    local node_version=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
    if [ "$node_version" -lt 18 ]; then
        log_error "Node.js 版本过低 (当前: $(node -v))，需要 18+"
        exit 1
    fi
    log_success "Node.js 版本: $(node -v)"
    
    # 检查Go
    if ! command -v go &> /dev/null; then
        log_error "Go 未安装，请先安装 Go 1.21+"
        exit 1
    fi
    log_success "Go 版本: $(go version)"
    
    # 检查npm
    if ! command -v npm &> /dev/null; then
        log_error "npm 未安装"
        exit 1
    fi
    log_success "npm 版本: $(npm -v)"
}

# 构建Wave Terminal
build_wave_terminal() {
    log_info "构建 Wave Terminal..."
    
    # 安装前端依赖
    log_info "安装前端依赖..."
    npm install
    
    # 构建前端
    log_info "构建前端资源..."
    npm run build:dev
    
    # 检查构建结果
    if [ ! -d "dist/frontend" ]; then
        log_error "前端构建失败，未找到 dist/frontend 目录"
        exit 1
    fi
    
    log_success "Wave Terminal 构建完成"
}

# 安装MCP依赖
install_mcp_dependencies() {
    log_info "安装 MCP SDK 依赖..."
    
    # 检查是否已安装MCP SDK
    if ! npm list @modelcontextprotocol/sdk &> /dev/null; then
        log_info "安装 @modelcontextprotocol/sdk..."
        npm install @modelcontextprotocol/sdk
    else
        log_success "MCP SDK 已安装"
    fi
}

# 生成认证密钥
generate_auth_key() {
    log_info "生成认证密钥..."
    
    # 尝试不同的方法生成密钥
    if command -v openssl &> /dev/null; then
        AUTH_KEY=$(openssl rand -hex 32)
    elif command -v python3 &> /dev/null; then
        AUTH_KEY=$(python3 -c "import secrets; print(secrets.token_hex(32))")
    else
        # Fallback: 使用固定密钥
        AUTH_KEY="83958e47ddc89fae695a7e1eb429899871e80334bd58cfc2d17a80388791f073"
        log_warning "使用默认认证密钥"
    fi
    
    export WAVETERM_AUTH_KEY="$AUTH_KEY"
    log_success "认证密钥已生成"
}

# 启动Wave Terminal服务器
start_wave_terminal() {
    log_info "启动 Wave Terminal 服务器..."
    
    # 检查是否已有服务器在运行
    if pgrep -f "main-server" > /dev/null; then
        log_warning "Wave Terminal 服务器已在运行"
        return
    fi
    
    # 使用持久化脚本启动
    if [ -f "persistent-server.sh" ]; then
        chmod +x persistent-server.sh
        ./persistent-server.sh start
    else
        log_warning "未找到 persistent-server.sh，使用直接启动"
        export WAVETERM_DATA_DIR="/tmp/waveterm-mcp"
        mkdir -p "$WAVETERM_DATA_DIR"
        
        # 后台启动服务器
        nohup go run cmd/server/main-server.go > waveterm-server.log 2>&1 &
        sleep 3
    fi
    
    # 检查服务器是否启动成功
    local max_attempts=10
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s http://localhost:8090/api/v1/widgets/workspaces > /dev/null; then
            log_success "Wave Terminal 服务器启动成功"
            return
        fi
        
        log_info "等待服务器启动... ($attempt/$max_attempts)"
        sleep 2
        ((attempt++))
    done
    
    log_error "Wave Terminal 服务器启动失败"
    exit 1
}

# 配置Claude Code
configure_claude_code() {
    log_info "配置 Claude Code MCP 集成..."
    
    # 检查Claude Code配置目录
    local claude_config_dir="$HOME/.config/claude-desktop"
    if [ ! -d "$claude_config_dir" ]; then
        mkdir -p "$claude_config_dir"
        log_info "创建 Claude Code 配置目录: $claude_config_dir"
    fi
    
    local config_file="$claude_config_dir/claude_desktop_config.json"
    local current_dir=$(pwd)
    
    # 生成配置文件
    cat > "$config_file" << EOF
{
  "mcpServers": {
    "wave-terminal": {
      "command": "node",
      "args": ["$current_dir/mcp-bridge.js"],
      "env": {
        "WAVE_TERMINAL_URL": "http://localhost:8090",
        "WAVE_TERMINAL_AUTH_KEY": "$WAVETERM_AUTH_KEY"
      }
    }
  }
}
EOF
    
    log_success "Claude Code 配置文件已创建: $config_file"
}

# 测试MCP连接
test_mcp_connection() {
    log_info "测试 MCP 连接..."
    
    # 测试MCP桥接脚本
    export WAVE_TERMINAL_URL="http://localhost:8090"
    export WAVE_TERMINAL_AUTH_KEY="$WAVETERM_AUTH_KEY"
    
    log_info "测试 Wave Terminal API..."
    if curl -s -H "X-AuthKey: $WAVETERM_AUTH_KEY" http://localhost:8090/api/v1/widgets/workspaces > /dev/null; then
        log_success "Wave Terminal API 连接正常"
    else
        log_warning "Wave Terminal API 连接失败，检查认证配置"
    fi
    
    # 测试MCP桥接脚本语法
    log_info "测试 MCP 桥接脚本..."
    if node -c mcp-bridge.js; then
        log_success "MCP 桥接脚本语法正确"
    else
        log_error "MCP 桥接脚本语法错误"
        exit 1
    fi
}

# 显示部署信息
show_deployment_info() {
    log_success "🎉 Wave Terminal Claude Code 部署完成！"
    echo ""
    echo "📋 部署信息:"
    echo "  Wave Terminal URL: http://localhost:8090"
    echo "  WebSocket URL: ws://localhost:8091"
    echo "  认证密钥: $WAVETERM_AUTH_KEY"
    echo ""
    echo "🔧 Claude Code 配置:"
    echo "  配置文件: $HOME/.config/claude-desktop/claude_desktop_config.json"
    echo "  MCP 服务器: wave-terminal"
    echo ""
    echo "🚀 可用工具:"
    echo "  • create_widget - 创建新的 widget"
    echo "  • list_workspaces - 列出所有工作区"
    echo "  • get_workspace_by_name - 根据名称获取工作区"
    echo "  • get_widget_types - 获取 widget 类型"
    echo "  • check_server_status - 检查服务器状态"
    echo ""
    echo "💡 使用方法:"
    echo "  1. 重启 Claude Code 应用"
    echo "  2. 在对话中使用工具，例如："
    echo "     \"请列出所有可用的工作区\""
    echo "     \"在 waveterm 工作区创建一个新终端\""
    echo ""
    echo "🔍 状态监控:"
    echo "  • 呼吸灯显示服务器状态 (绿色=正常, 红色=异常)"
    echo "  • MCP 客户端显示连接的服务器数量"
    echo ""
    echo "📝 日志文件:"
    echo "  • Wave Terminal: waveterm-server.log"
    echo "  • 进程状态: ./persistent-server.sh status"
}

# 主函数
main() {
    echo ""
    log_info "开始部署 Wave Terminal Claude Code 集成..."
    echo ""
    
    check_dependencies
    echo ""
    
    build_wave_terminal
    echo ""
    
    install_mcp_dependencies
    echo ""
    
    generate_auth_key
    echo ""
    
    start_wave_terminal
    echo ""
    
    configure_claude_code
    echo ""
    
    test_mcp_connection
    echo ""
    
    show_deployment_info
}

# 错误处理
trap 'log_error "部署过程中发生错误，请检查日志"; exit 1' ERR

# 参数处理
case "${1:-}" in
    "help"|"-h"|"--help")
        echo "Wave Terminal Claude Code 部署脚本"
        echo ""
        echo "用法: $0 [选项]"
        echo ""
        echo "选项:"
        echo "  help, -h, --help    显示此帮助信息"
        echo "  status              检查部署状态"
        echo "  clean               清理部署环境"
        echo ""
        echo "环境变量:"
        echo "  WAVE_TERMINAL_URL   Wave Terminal URL (默认: http://localhost:8090)"
        echo "  WAVETERM_AUTH_KEY   认证密钥 (自动生成)"
        exit 0
        ;;
    "status")
        log_info "检查部署状态..."
        
        # 检查服务器状态
        if pgrep -f "main-server" > /dev/null; then
            log_success "Wave Terminal 服务器正在运行"
        else
            log_warning "Wave Terminal 服务器未运行"
        fi
        
        # 检查API
        if curl -s http://localhost:8090/api/v1/widgets/workspaces > /dev/null; then
            log_success "Wave Terminal API 响应正常"
        else
            log_warning "Wave Terminal API 无响应"
        fi
        
        # 检查配置文件
        if [ -f "$HOME/.config/claude-desktop/claude_desktop_config.json" ]; then
            log_success "Claude Code 配置文件存在"
        else
            log_warning "Claude Code 配置文件不存在"
        fi
        
        exit 0
        ;;
    "clean")
        log_info "清理部署环境..."
        
        # 停止服务器
        if [ -f "persistent-server.sh" ]; then
            ./persistent-server.sh stop
        else
            pkill -f "main-server" || true
        fi
        
        # 清理文件
        rm -f waveterm-server.log waveterm-server.pid waveterm-server.port
        rm -rf /tmp/waveterm-mcp
        
        log_success "清理完成"
        exit 0
        ;;
    "")
        # 默认执行主函数
        main
        ;;
    *)
        log_error "未知参数: $1"
        log_info "使用 '$0 help' 查看帮助信息"
        exit 1
        ;;
esac