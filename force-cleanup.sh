#!/bin/bash

# Wave Terminal 强力清理脚本
# 彻底解决锁文件和进程冲突问题

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

log "🧹 开始Wave Terminal强力清理..."

# 1. 停止所有相关进程
log "🔍 查找所有Wave相关进程..."
WAVE_PIDS=$(ps aux | grep -E "(wave|main-server|wavesrv)" | grep -v grep | grep -v force-cleanup | awk '{print $2}' || true)

if [ ! -z "$WAVE_PIDS" ]; then
    warning "发现以下Wave相关进程:"
    ps aux | grep -E "(wave|main-server|wavesrv)" | grep -v grep | grep -v force-cleanup
    
    log "🛑 停止所有Wave相关进程..."
    for pid in $WAVE_PIDS; do
        if ps -p "$pid" > /dev/null 2>&1; then
            echo "  停止进程 $pid..."
            kill -TERM "$pid" 2>/dev/null || true
        fi
    done
    
    # 等待进程正常退出
    sleep 3
    
    # 强制杀死仍在运行的进程
    log "🔥 强制清理残留进程..."
    for pid in $WAVE_PIDS; do
        if ps -p "$pid" > /dev/null 2>&1; then
            echo "  强制杀死进程 $pid..."
            kill -9 "$pid" 2>/dev/null || true
        fi
    done
    
    success "所有Wave进程已清理"
else
    success "未发现Wave相关进程"
fi

# 2. 清理锁文件和socket文件
log "🗂️  清理锁文件和socket文件..."
LOCK_FILES=(
    "/tmp/waveterm-mcp/wave.lock"
    "/tmp/waveterm-mcp/wave.sock"
    "/private/tmp/waveterm-mcp/wave.lock"
    "/private/tmp/waveterm-mcp/wave.sock"
)

for file in "${LOCK_FILES[@]}"; do
    if [ -f "$file" ] || [ -S "$file" ]; then
        echo "  删除 $file"
        rm -f "$file" 2>/dev/null || true
    fi
done

# 3. 清理PID文件
log "📄 清理PID和端口文件..."
rm -f waveterm-server.pid
rm -f waveterm-server.port

# 4. 检查端口占用
log "🔌 检查端口占用情况..."
OCCUPIED_PORTS=$(lsof -ti :58616,58617,55893,55894 2>/dev/null || true)
if [ ! -z "$OCCUPIED_PORTS" ]; then
    warning "发现占用端口的进程: $OCCUPIED_PORTS"
    for port_pid in $OCCUPIED_PORTS; do
        echo "  释放端口进程 $port_pid..."
        kill -9 "$port_pid" 2>/dev/null || true
    done
    success "端口已释放"
else
    success "无端口占用"
fi

# 5. 清理开发环境相关进程（如果需要）
if [ "$1" = "--dev" ]; then
    log "🔧 清理开发环境进程..."
    pkill -f "electron-vite" 2>/dev/null || true
    pkill -f "esbuild" 2>/dev/null || true
    success "开发环境已清理"
fi

# 6. 验证清理结果
log "🔍 验证清理结果..."
REMAINING=$(ps aux | grep -E "(wave|main-server|wavesrv)" | grep -v grep | grep -v force-cleanup || true)
if [ -z "$REMAINING" ]; then
    success "所有Wave进程已清理完毕"
else
    warning "仍有残留进程:"
    echo "$REMAINING"
fi

# 7. 检查锁文件状态
if [ ! -f "/tmp/waveterm-mcp/wave.lock" ]; then
    success "锁文件已清理"
else
    error "锁文件仍然存在，可能需要手动清理"
fi

log "🧹 清理完成！"
echo
echo "📝 现在可以安全重启Wave Terminal:"
echo "  ./persistent-server.sh start"
echo
echo "💡 如果问题仍然存在，请尝试:"
echo "  ./force-cleanup.sh --dev  # 同时清理开发环境"
echo "  sudo lsof /tmp/waveterm-mcp/wave.lock  # 检查系统级锁定"