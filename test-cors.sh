#!/bin/bash

# CORS测试脚本
# 测试各种CORS场景以确保修复有效

set -e

# 配置
API_BASE="http://127.0.0.1:60289"
DEV_ORIGIN="http://localhost:5173"
ALT_ORIGIN="http://127.0.0.1:5173"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[CORS Test]${NC} $1"
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

# 测试CORS预检请求
test_preflight() {
    local origin="$1"
    local url="$2"
    
    log "测试CORS预检请求: Origin=$origin, URL=$url"
    
    local response=$(curl -s -I -X OPTIONS \
        -H "Origin: $origin" \
        -H "Access-Control-Request-Method: POST" \
        -H "Access-Control-Request-Headers: Content-Type,X-AuthKey" \
        "$url" 2>/dev/null || echo "FAILED")
    
    if [[ "$response" == "FAILED" ]]; then
        error "预检请求失败: $url"
        return 1
    fi
    
    # 检查CORS头
    local allow_origin=$(echo "$response" | grep -i "access-control-allow-origin:" | cut -d: -f2- | tr -d '\r\n ' || echo "")
    local allow_methods=$(echo "$response" | grep -i "access-control-allow-methods:" | cut -d: -f2- | tr -d '\r\n ' || echo "")
    local allow_headers=$(echo "$response" | grep -i "access-control-allow-headers:" | cut -d: -f2- | tr -d '\r\n ' || echo "")
    
    echo "  Access-Control-Allow-Origin: $allow_origin"
    echo "  Access-Control-Allow-Methods: $allow_methods"
    echo "  Access-Control-Allow-Headers: $allow_headers"
    
    # 验证CORS头
    if [[ "$allow_origin" == "*" ]] || [[ "$allow_origin" == "$origin" ]]; then
        success "CORS Origin头正确"
    else
        error "CORS Origin头不正确: 期望 '$origin' 或 '*', 得到 '$allow_origin'"
        return 1
    fi
    
    if [[ "$allow_methods" == *"POST"* ]] && [[ "$allow_methods" == *"OPTIONS"* ]]; then
        success "CORS Methods头正确"
    else
        error "CORS Methods头不完整: $allow_methods"
        return 1
    fi
    
    if [[ "$allow_headers" == *"Content-Type"* ]] && [[ "$allow_headers" == *"X-AuthKey"* ]]; then
        success "CORS Headers头正确"
    else
        error "CORS Headers头不完整: $allow_headers"
        return 1
    fi
    
    return 0
}

# 测试实际API请求
test_api_request() {
    local origin="$1"
    local url="$2"
    
    log "测试API请求: Origin=$origin, URL=$url"
    
    local response=$(curl -s -w "\n%{http_code}" \
        -H "Origin: $origin" \
        -H "Content-Type: application/json" \
        "$url" 2>/dev/null || echo -e "\nFAILED")
    
    local body=$(echo "$response" | head -n -1)
    local status=$(echo "$response" | tail -n 1)
    
    if [[ "$status" == "FAILED" ]]; then
        error "API请求失败: $url"
        return 1
    fi
    
    echo "  HTTP状态: $status"
    echo "  响应长度: $(echo "$body" | wc -c) 字节"
    
    if [[ "$status" == "200" ]]; then
        success "API请求成功"
        return 0
    else
        error "API请求失败，状态码: $status"
        return 1
    fi
}

# 主测试函数
run_cors_tests() {
    log "开始CORS测试..."
    echo
    
    # 检查服务器是否运行
    if ! curl -s "$API_BASE/api/v1/widgets" > /dev/null; then
        error "API服务器不可访问: $API_BASE"
        echo "请先启动服务器: ./persistent-server.sh start"
        exit 1
    fi
    
    success "API服务器可访问: $API_BASE"
    echo
    
    # 测试用例
    local test_endpoints=(
        "/api/v1/widgets"
        "/api/v1/widgets/workspaces"
        "/api/v1/widgets/mcp/status"
    )
    
    local test_origins=(
        "$DEV_ORIGIN"
        "$ALT_ORIGIN"
        "http://localhost:3000"
    )
    
    local passed=0
    local total=0
    
    # 测试预检请求
    log "=== 测试CORS预检请求 ==="
    for origin in "${test_origins[@]}"; do
        for endpoint in "${test_endpoints[@]}"; do
            total=$((total + 1))
            if test_preflight "$origin" "$API_BASE$endpoint"; then
                passed=$((passed + 1))
            fi
            echo
        done
    done
    
    # 测试实际请求
    log "=== 测试实际API请求 ==="
    for origin in "${test_origins[@]}"; do
        for endpoint in "${test_endpoints[@]}"; do
            total=$((total + 1))
            if test_api_request "$origin" "$API_BASE$endpoint"; then
                passed=$((passed + 1))
            fi
            echo
        done
    done
    
    # 输出测试结果
    echo "=================================="
    if [[ $passed -eq $total ]]; then
        success "所有CORS测试通过! ($passed/$total)"
        echo
        echo "🎉 CORS配置已正确修复!"
        echo "前端现在应该能够正常访问API端点。"
    else
        error "部分CORS测试失败: $passed/$total"
        echo
        echo "需要进一步检查CORS配置。"
        exit 1
    fi
}

# 显示使用说明
show_usage() {
    echo "CORS测试脚本"
    echo
    echo "用法: $0 [选项]"
    echo
    echo "选项:"
    echo "  test     - 运行完整的CORS测试套件 (默认)"
    echo "  quick    - 运行快速测试"
    echo "  preflight- 只测试CORS预检请求"
    echo "  api      - 只测试API请求"
    echo
    echo "示例:"
    echo "  $0              # 运行完整测试"
    echo "  $0 quick        # 快速测试"
    echo "  $0 preflight    # 只测试预检"
}

# 快速测试
quick_test() {
    log "运行快速CORS测试..."
    
    if ! curl -s "$API_BASE/api/v1/widgets" > /dev/null; then
        error "API服务器不可访问"
        exit 1
    fi
    
    if test_preflight "$DEV_ORIGIN" "$API_BASE/api/v1/widgets/mcp/status"; then
        success "快速CORS测试通过!"
    else
        error "快速CORS测试失败"
        exit 1
    fi
}

# 主函数
main() {
    case "${1:-test}" in
        "test")
            run_cors_tests
            ;;
        "quick")
            quick_test
            ;;
        "preflight")
            test_preflight "$DEV_ORIGIN" "$API_BASE/api/v1/widgets/mcp/status"
            ;;
        "api")
            test_api_request "$DEV_ORIGIN" "$API_BASE/api/v1/widgets/mcp/status"
            ;;
        "help"|"-h"|"--help")
            show_usage
            ;;
        *)
            error "未知选项: $1"
            show_usage
            exit 1
            ;;
    esac
}

main "$@"