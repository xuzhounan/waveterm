# Wave Terminal MCP 开发指南

## 概述

Wave Terminal集成了MCP (Model Context Protocol) 协议，为AI助手提供强大的终端和工作空间管理能力。本文档详细介绍Wave Terminal的MCP实现架构和开发指南。

## 🏗️ MCP架构概览

### 核心组件

1. **MCP Bridge Server** (`mcp-bridge.cjs`)
   - 主要的MCP协议实现
   - 提供所有Wave Terminal工具调用
   - 基于`@modelcontextprotocol/sdk`

2. **前端MCP集成**
   - `frontend/app/store/mcpapi.ts` - MCP API接口定义
   - `frontend/app/element/mcpservercontrol.tsx` - MCP服务器控制组件
   - `frontend/app/element/mcpclient.tsx` - MCP客户端界面

3. **后端集成**
   - Widget API服务作为MCP工具的底层实现
   - 认证和权限管理

## 🔧 MCP工具列表

Wave Terminal提供以下MCP工具，都以`mcp__wave-terminal__`前缀：

### 工作空间管理
- `list_workspaces` - 列出所有工作空间
- `get_workspace_by_name` - 根据名称获取工作空间
- `get_workspace` - 获取工作空间详情

### Widget管理
- `create_widget` - 创建新的widget
- `get_widget_types` - 获取支持的widget类型
- `list_tabs` - 列出工作空间中的标签页
- `create_tab` - 创建新标签页
- `set_active_tab` - 设置活跃标签页

### 终端控制
- `send_terminal_input` - 向终端发送输入
- `get_block_content` - 获取block内容
- `get_block_status` - 获取block状态
- `list_blocks` - 列出blocks

### 系统管理
- `check_server_status` - 检查服务器状态
- `restart_mcp_server` - 重启MCP服务器
- `fix_workspace_data` - 修复工作空间数据

## 📁 关键文件结构

```
Wave Terminal MCP 实现
├── mcp-bridge.cjs                    # MCP服务器主文件
├── frontend/app/store/
│   └── mcpapi.ts                     # MCP API接口定义
├── frontend/app/element/
│   ├── mcpservercontrol.tsx          # MCP服务器控制
│   ├── mcpservercontrol.scss         # 相关样式
│   ├── mcpclient.tsx                 # MCP客户端组件
│   └── mcpclient.scss                # 客户端样式
└── pkg/service/widgetapiservice/     # 底层Widget API服务
```

## 🚀 MCP工具实现模式

### 基本工具结构

```javascript
// mcp-bridge.cjs 中的工具定义示例
{
    name: "mcp__wave-terminal__create_widget",
    description: "在Wave Terminal工作区中创建新的widget",
    inputSchema: {
        type: "object",
        properties: {
            workspace_id: {
                type: "string",
                description: "工作空间ID"
            },
            widget_type: {
                type: "string", 
                enum: ["terminal", "web", "files", "ai", "sysinfo", "help", "tips"],
                description: "Widget类型"
            },
            title: {
                type: "string",
                description: "Widget标题（可选）"
            },
            meta: {
                type: "object",
                description: "Widget元数据（可选）"
            }
        },
        required: ["workspace_id", "widget_type"]
    }
}
```

### 工具实现逻辑

```javascript
async function handleCreateWidget(args) {
    try {
        // 1. 参数验证
        const { workspace_id, widget_type, title, meta } = args;
        
        // 2. 调用Wave Terminal Widget API
        const response = await fetch(`${WAVE_API_BASE}/api/v1/widgets`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${AUTH_TOKEN}`
            },
            body: JSON.stringify({
                workspace_id,
                widget_type,
                title,
                meta
            })
        });
        
        // 3. 处理响应
        const result = await response.json();
        return {
            content: [{
                type: "text",
                text: `Widget创建成功: ${result.block_id}`
            }]
        };
        
    } catch (error) {
        return {
            content: [{
                type: "text", 
                text: `创建Widget失败: ${error.message}`
            }],
            isError: true
        };
    }
}
```

## 🔒 认证和安全

### AuthKey认证
Wave Terminal使用AuthKey进行API认证：

```javascript
const AUTH_KEY = '83958e47ddc89fae695a7e1eb429899871e80334bd58cfc2d17a80388791f073';

const headers = {
    'X-AuthKey': AUTH_KEY,
    'Content-Type': 'application/json'
};
```

### 安全考虑
- AuthKey在生产环境中应该动态生成
- 所有API调用都需要认证
- 输入参数需要严格验证

## 🌐 网络配置

### 端点管理
MCP服务器通过以下方式获取Wave Terminal端点：

```javascript
function getWaveTerminalEndpoint() {
    // 优先使用环境变量
    if (process.env.WAVE_API_ENDPOINT) {
        return process.env.WAVE_API_ENDPOINT;
    }
    
    // 默认本地端点
    return 'http://127.0.0.1:8090';
}
```

### 连接处理
- 自动重试机制
- 超时设置（5秒）
- 错误恢复

## 🎯 Widget类型支持

### 支持的Widget类型

1. **terminal** - 交互式终端
   ```json
   {
     "widget_type": "terminal",
     "meta": {
       "cwd": "/project/path",
       "env": {"NODE_ENV": "development"}
     }
   }
   ```

2. **web** - Web浏览器
   ```json
   {
     "widget_type": "web", 
     "meta": {
       "url": "https://docs.waveterm.dev"
     }
   }
   ```

3. **files** - 文件浏览器
   ```json
   {
     "widget_type": "files",
     "meta": {
       "file": "/home/user/project"
     }
   }
   ```

4. **ai** - AI助手界面
5. **sysinfo** - 系统信息
6. **help** - 帮助文档  
7. **tips** - 快速提示

## 🔄 状态管理

### MCP服务器状态
通过`mcpservercontrol.tsx`组件管理：

```typescript
interface MCPServerStatus {
    isRunning: boolean;
    port?: number;
    authKey?: string; 
    error?: string;
    lastCheck: number;
}
```

### 状态监控
- 定期健康检查
- 错误自动恢复
- 状态可视化指示

## 🛠️ 开发和调试

### 本地开发
1. 启动Wave Terminal开发服务器
2. 运行MCP bridge服务器：
   ```bash
   node mcp-bridge.cjs
   ```
3. 配置Claude Code使用本地MCP服务器

### 调试工具
- MCP服务器日志：`mcp-bridge.log`
- Wave Terminal开发者工具
- 网络请求监控

### 测试工具调用
```bash
# 测试创建Widget
curl -X POST http://localhost:8090/api/v1/widgets \
  -H "Content-Type: application/json" \
  -H "X-AuthKey: YOUR_AUTH_KEY" \
  -d '{
    "workspace_id": "ws-123", 
    "widget_type": "terminal"
  }'
```

## 📊 性能优化

### 缓存策略
- 工作空间信息缓存
- Widget类型定义缓存
- 状态查询结果缓存

### 网络优化
- 连接复用
- 请求批处理
- 超时优化

## 🚨 错误处理

### 常见错误类型
1. **连接错误** - Wave Terminal服务器不可达
2. **认证错误** - AuthKey无效或过期
3. **参数错误** - 工具调用参数不正确
4. **资源不存在** - 工作空间或Widget不存在

### 错误处理策略
```javascript
try {
    const result = await callWaveAPI(endpoint, params);
    return formatSuccessResponse(result);
} catch (error) {
    if (error.code === 'AUTH_FAILED') {
        return formatAuthError(error);
    } else if (error.code === 'NOT_FOUND') {
        return formatNotFoundError(error);
    } else {
        return formatGenericError(error);
    }
}
```

## 🔮 扩展开发

### 添加新的MCP工具
1. 在`mcp-bridge.cjs`中定义工具Schema
2. 实现工具处理函数
3. 注册到MCP服务器
4. 添加相应的测试

### 自定义Widget类型
1. 扩展Widget API支持新类型
2. 更新MCP工具的enum定义
3. 添加相应的元数据Schema

## 📚 相关资源

- [MCP协议官方文档](https://github.com/modelcontextprotocol/specification)
- [Wave Terminal Widget API文档](./widget-api.md)
- [系统架构对比文档](./mcp-vs-widget-api.md)

---

> 💡 **提示**: MCP协议设计用于AI助手，确保所有工具调用都提供清晰的描述和合适的错误处理。