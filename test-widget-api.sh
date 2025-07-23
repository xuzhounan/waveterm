#!/bin/bash

# Wave Terminal Widget API 测试脚本
# 使用curl测试新开发的Widget API功能

set -e

# 配置
WAVE_URL="${WAVE_URL:-http://localhost:8080}"
API_BASE="$WAVE_URL/api/v1/widgets"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 测试函数
test_connection() {
    log_info "测试服务器连接..."
    
    if curl -s --connect-timeout 5 "$WAVE_URL/wave/service" > /dev/null 2>&1; then
        log_success "服务器连接正常"
        return 0
    else
        log_error "无法连接到Wave Terminal服务器 ($WAVE_URL)"
        log_warning "请确保Wave Terminal正在运行并监听端口8080"
        return 1
    fi
}

test_list_widget_types() {
    log_info "测试获取widget类型列表..."
    
    local response
    response=$(curl -s -w "\n%{http_code}" "$API_BASE")
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n -1)
    
    echo "HTTP状态码: $http_code"
    
    if [ "$http_code" = "200" ]; then
        log_success "获取widget类型列表成功"
        echo "响应内容:"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        
        # 检查响应格式
        if echo "$body" | jq -e '.success and .widget_types' > /dev/null 2>&1; then
            log_success "响应格式正确"
            
            # 显示支持的widget类型
            local widget_types
            widget_types=$(echo "$body" | jq -r '.widget_types | keys[]' 2>/dev/null)
            log_info "支持的widget类型: $(echo $widget_types | tr '\n' ' ')"
        else
            log_warning "响应格式可能不正确"
        fi
    else
        log_error "获取widget类型列表失败 (HTTP $http_code)"
        echo "$body"
    fi
    
    echo
}

test_list_workspaces() {
    log_info "测试获取工作空间列表..."
    
    local response
    response=$(curl -s -w "\n%{http_code}" "$API_BASE/workspaces")
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n -1)
    
    echo "HTTP状态码: $http_code"
    
    if [ "$http_code" = "200" ]; then
        log_success "获取工作空间列表成功"
        echo "响应内容:"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        
        # 提取第一个工作空间ID
        if echo "$body" | jq -e '.success and .workspaces' > /dev/null 2>&1; then
            FIRST_WORKSPACE_ID=$(echo "$body" | jq -r '.workspaces[0].workspace_id' 2>/dev/null)
            if [ "$FIRST_WORKSPACE_ID" != "null" ] && [ -n "$FIRST_WORKSPACE_ID" ]; then
                log_info "找到工作空间: $FIRST_WORKSPACE_ID"
            fi
        fi
    else
        log_error "获取工作空间列表失败 (HTTP $http_code)"
        echo "$body"
    fi
    
    echo
}

test_get_workspace_widgets() {
    local workspace_id="$1"
    
    if [ -z "$workspace_id" ]; then
        log_warning "跳过获取工作空间widget配置测试（无有效工作空间ID）"
        return
    fi
    
    log_info "测试获取工作空间widget配置..."
    
    local response
    response=$(curl -s -w "\n%{http_code}" "$API_BASE/workspace/$workspace_id")
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n -1)
    
    echo "HTTP状态码: $http_code"
    
    if [ "$http_code" = "200" ]; then
        log_success "获取工作空间widget配置成功"
        echo "响应内容:"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
    else
        log_error "获取工作空间widget配置失败 (HTTP $http_code)"
        echo "$body"
    fi
    
    echo
}

test_create_widget() {
    local workspace_id="$1"
    local widget_type="${2:-terminal}"
    
    if [ -z "$workspace_id" ]; then
        log_warning "跳过创建widget测试（无有效工作空间ID）"
        return
    fi
    
    log_info "测试创建 $widget_type widget..."
    
    # 构建请求数据
    local request_data
    case "$widget_type" in
        "terminal")
            request_data='{
                "workspace_id": "'$workspace_id'",
                "widget_type": "terminal",
                "title": "Test Terminal Widget",
                "meta": {
                    "cwd": "/tmp"
                }
            }'
            ;;
        "web")
            request_data='{
                "workspace_id": "'$workspace_id'",
                "widget_type": "web",
                "title": "Test Web Widget",
                "meta": {
                    "url": "https://www.waveterm.dev"
                }
            }'
            ;;
        "files")
            request_data='{
                "workspace_id": "'$workspace_id'",
                "widget_type": "files",
                "title": "Test Files Widget",
                "meta": {
                    "file": "~"
                }
            }'
            ;;
        *)
            request_data='{
                "workspace_id": "'$workspace_id'",
                "widget_type": "'$widget_type'",
                "title": "Test '$widget_type' Widget"
            }'
            ;;
    esac
    
    echo "请求数据:"
    echo "$request_data" | jq '.' 2>/dev/null || echo "$request_data"
    
    local response
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$request_data" \
        "$API_BASE")
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n -1)
    
    echo "HTTP状态码: $http_code"
    
    if [ "$http_code" = "201" ]; then
        log_success "创建 $widget_type widget 成功"
        echo "响应内容:"
        echo "$body" | jq '.' 2>/dev/null || echo "$body"
        
        # 提取Block ID
        if echo "$body" | jq -e '.success and .block_id' > /dev/null 2>&1; then
            local block_id
            block_id=$(echo "$body" | jq -r '.block_id' 2>/dev/null)
            log_info "创建的Block ID: $block_id"
        fi
    else
        log_error "创建 $widget_type widget 失败 (HTTP $http_code)"
        echo "$body"
    fi
    
    echo
}

test_error_handling() {
    log_info "测试错误处理..."
    
    # 测试1: 无效的workspace_id
    log_info "  测试1: 无效的workspace_id"
    local response
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"workspace_id": "invalid-workspace", "widget_type": "terminal"}' \
        "$API_BASE")
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "400" ] || [ "$http_code" = "404" ]; then
        log_success "  正确处理无效workspace_id (HTTP $http_code)"
    else
        log_warning "  期望400/404状态码，实际: $http_code"
    fi
    
    # 测试2: 缺少必需参数
    log_info "  测试2: 缺少必需参数"
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"workspace_id": "test"}' \
        "$API_BASE")
    
    http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "400" ]; then
        log_success "  正确处理缺少参数 (HTTP $http_code)"
    else
        log_warning "  期望400状态码，实际: $http_code"
    fi
    
    # 测试3: 无效的widget类型
    log_info "  测试3: 无效的widget类型"
    response=$(curl -s -w "\n%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d '{"workspace_id": "test-workspace", "widget_type": "invalid-type"}' \
        "$API_BASE")
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "400" ]; then
        log_success "  正确处理无效widget类型 (HTTP $http_code)"
        
        # 检查错误响应格式
        if echo "$body" | jq -e '.success == false and .error' > /dev/null 2>&1; then
            log_success "  错误响应格式正确"
        else
            log_warning "  错误响应格式可能不正确"
        fi
    else
        log_warning "  期望400状态码，实际: $http_code"
    fi
    
    echo
}

# 主测试函数
main() {
    echo "🚀 开始Wave Terminal Widget API 测试"
    echo "============================================"
    echo "测试URL: $WAVE_URL"
    echo
    
    # 检查依赖
    if ! command -v curl &> /dev/null; then
        log_error "curl命令未找到，请安装curl"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        log_warning "jq命令未找到，JSON格式化将不可用"
    fi
    
    # 测试连接
    if ! test_connection; then
        exit 1
    fi
    
    echo
    
    # 运行API测试
    test_list_widget_types
    test_list_workspaces
    
    # 如果找到工作空间，继续测试
    if [ -n "$FIRST_WORKSPACE_ID" ] && [ "$FIRST_WORKSPACE_ID" != "null" ]; then
        test_get_workspace_widgets "$FIRST_WORKSPACE_ID"
        
        # 测试创建不同类型的widget
        for widget_type in terminal web files ai; do
            test_create_widget "$FIRST_WORKSPACE_ID" "$widget_type"
            sleep 1  # 避免请求过于频繁
        done
    fi
    
    # 测试错误处理
    test_error_handling
    
    echo "============================================"
    log_success "Widget API 测试完成"
    
    echo
    echo "📝 注意事项:"
    echo "   • 确保Wave Terminal服务器正在运行"
    echo "   • 某些测试可能因为实际数据不同而显示警告"
    echo "   • 创建的widget会实际添加到工作空间中"
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --url)
            WAVE_URL="$2"
            API_BASE="$WAVE_URL/api/v1/widgets"
            shift 2
            ;;
        --help|-h)
            echo "用法: $0 [选项]"
            echo "选项:"
            echo "  --url URL    指定Wave Terminal服务器URL (默认: http://localhost:8080)"
            echo "  --help       显示此帮助信息"
            exit 0
            ;;
        *)
            log_error "未知选项: $1"
            echo "使用 --help 查看帮助信息"
            exit 1
            ;;
    esac
done

# 运行测试
main