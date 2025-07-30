# Wave Terminal 架构审查报告

## 🔍 问题根源回顾

通过深入调试"MCP创建widget需要刷新才显示"的问题，我们发现了根本原因：**多实例管理混乱**导致MCP连接到错误的服务器实例。

## 🏗️ 当前架构分析

### ✅ 架构优点

1. **完整的事件驱动系统**
   - `wps.go` 提供了健壮的发布-订阅机制
   - WebSocket实时通信良好
   - 前端状态管理（Jotai）设计合理

2. **模块化设计**
   - MCP bridge独立进程
   - HTTP API与WebSocket分离
   - 前后端职责清晰

3. **扩展性良好**
   - Widget系统支持多种类型
   - Block创建流程标准化
   - Layout Action机制灵活

### ⚠️ 发现的架构问题

## 1. 🚨 多实例管理混乱

### 问题描述
- **无统一进程管理**：同时存在多个wavesrv实例
- **端口冲突检测不足**：8090/8091端口可能被多个实例占用
- **PID文件不统一**：不同脚本使用不同PID文件名

### 当前状况
```bash
# 发现的进程管理文件
persistent-server.sh -> waveterm-server.pid
dev-isolated.sh -> dev-waveterm.pid
# 实际运行的二进制
wavesrv.arm64 (开发环境)
go run main-server.go (MCP服务器)
```

### 影响
- MCP连接到错误实例
- 开发调试困难
- 数据同步问题

## 2. 🔧 配置管理不一致

### 问题描述
- **环境变量优先级混乱**
- **默认端口分配冲突**
- **数据目录不统一**

### 当前配置冲突
```bash
# persistent-server.sh
FIXED_WEB_PORT="60289"
DATA_DIR="$HOME/Library/Application Support/waveterm"

# dev-isolated.sh  
DEV_DATA_DIR="$HOME/.waveterm"
# 端口自动分配

# mcp-bridge.cjs
WAVETERM_WEB_PORT || '8090'  # 默认8090

# 开发环境
实际使用8090/8091端口
```

## 3. 🔍 服务发现机制缺失

### 问题描述
- **MCP无法自动发现正确服务器**
- **客户端连接检测不完善**
- **健康检查机制不足**

## 🛠️ 架构改进建议

## 1. 统一进程管理系统

### 建议实现
```bash
# 新增：wave-process-manager.sh
./wave-process-manager.sh list       # 显示所有实例
./wave-process-manager.sh dev        # 启动开发环境
./wave-process-manager.sh mcp        # 启动MCP服务器
./wave-process-manager.sh cleanup    # 清理所有实例
./wave-process-manager.sh status     # 统一状态检查
```

### 特性
- **统一PID管理**：所有实例注册到中央PID注册表
- **端口冲突检测**：启动前检查端口占用
- **实例隔离**：基于用途自动分配端口范围
- **优雅停止**：支持级联停止和清理

## 2. 配置管理系统重构

### 建议结构
```
config/
├── development.env      # 开发环境配置
├── mcp-server.env      # MCP服务器配置  
├── production.env      # 生产环境配置
└── default.env         # 默认配置
```

### 端口分配策略
```bash
# 端口范围规划
开发环境：    8090-8099
MCP服务器：   60289-60299  
生产环境：    动态分配
测试环境：    9090-9099
```

## 3. 服务发现与注册

### 服务注册表
```json
{
  "instances": {
    "dev": {
      "pid": 12345,
      "web_port": 8090,
      "ws_port": 8091,
      "data_dir": "~/.waveterm",
      "status": "running"
    },
    "mcp": {
      "pid": 12346, 
      "web_port": 60289,
      "ws_port": 60290,
      "data_dir": "~/Library/Application Support/waveterm",
      "status": "running"
    }
  }
}
```

### MCP自动发现
```javascript
// mcp-bridge.cjs 改进
async discoverWaveTerminal() {
    const registry = await loadServiceRegistry();
    
    // 优先级：环境变量 > 开发实例 > MCP实例
    if (process.env.WAVE_TERMINAL_URL) {
        return process.env.WAVE_TERMINAL_URL;
    }
    
    if (registry.instances.dev?.status === 'running') {
        return `http://127.0.0.1:${registry.instances.dev.web_port}`;
    }
    
    if (registry.instances.mcp?.status === 'running') {
        return `http://127.0.0.1:${registry.instances.mcp.web_port}`;
    }
    
    throw new Error('No Wave Terminal instance found');
}
```

## 4. 健康检查与监控

### 实例健康检查
```bash
# 每30秒检查一次
check_instance_health() {
    local instance_type=$1
    local pid=$(get_instance_pid $instance_type)
    local port=$(get_instance_port $instance_type)
    
    # 进程检查
    if ! ps -p "$pid" > /dev/null 2>&1; then
        mark_instance_dead $instance_type
        return 1
    fi
    
    # 端口检查  
    if ! curl -s -f "http://127.0.0.1:$port/health" > /dev/null; then
        mark_instance_unhealthy $instance_type
        return 1
    fi
    
    mark_instance_healthy $instance_type
}
```

### MCP连接状态监控
```javascript
// 连接状态追踪
class ConnectionMonitor {
    async checkConnection() {
        try {
            const response = await fetch(`${this.waveTerminalUrl}/health`);
            if (response.ok) {
                this.connectionStatus = 'healthy';
                this.lastSuccessfulCheck = Date.now();
            } else {
                this.connectionStatus = 'unhealthy';
            }
        } catch (error) {
            this.connectionStatus = 'disconnected';
            console.error('[MCP] Connection check failed:', error.message);
        }
    }
}
```

## 5. 开发体验改进

### 统一启动命令
```bash
# 替代当前的复杂脚本
npm run dev          # 启动开发环境
npm run mcp          # 启动MCP服务器  
npm run dev:full     # 同时启动开发环境和MCP
npm run status       # 查看所有服务状态
npm run cleanup      # 清理所有服务
```

### 智能端口分配
```javascript
// 自动寻找可用端口
async function findAvailablePort(basePort, range = 10) {
    for (let i = 0; i < range; i++) {
        const port = basePort + i;
        if (await isPortAvailable(port)) {
            return port;
        }
    }
    throw new Error(`No available port in range ${basePort}-${basePort + range}`);
}
```

## 6. 错误处理与调试

### 统一日志系统
```bash
# 日志文件标准化
logs/
├── wave-dev.log         # 开发环境日志
├── wave-mcp.log         # MCP服务器日志
├── mcp-bridge.log       # MCP bridge日志
└── process-manager.log  # 进程管理日志
```

### 智能错误诊断
```bash
# 新增：wave-doctor.sh
./wave-doctor.sh check          # 全面健康检查
./wave-doctor.sh diagnose       # 问题诊断
./wave-doctor.sh fix-common     # 修复常见问题
./wave-doctor.sh port-conflicts # 检查端口冲突
```

## 📋 实施优先级

### 阶段1：紧急修复（1-2天）
1. ✅ **统一PID文件管理**
2. ✅ **端口冲突检测**
3. ✅ **MCP自动发现机制**

### 阶段2：架构改进（3-5天）
1. **进程管理器实现**
2. **配置管理系统**
3. **服务注册表**

### 阶段3：开发体验（1-2天）
1. **统一启动命令**
2. **智能端口分配**
3. **健康检查系统**

## 🎯 预期效果

实施这些改进后：

1. **彻底消除多实例问题**
2. **MCP自动连接到正确服务器**
3. **开发调试体验显著提升**
4. **减少90%的环境配置问题**
5. **支持同时运行多个隔离环境**

## 📝 结论

当前架构的核心功能设计良好，主要问题在于**进程管理和配置管理的不一致**。通过系统性的改进，可以在保持现有架构优势的同时，彻底解决多实例管理混乱的问题，为项目的持续发展奠定坚实基础。

这次深入调试暴露的问题实际上为项目架构的进一步完善提供了宝贵机会。