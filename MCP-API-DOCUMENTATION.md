# Wave Terminal MCP API Documentation

这是为Model Context Protocol (MCP)集成准备的Wave Terminal API文档。通过这些API，AI agents可以持续监控和获取Wave Terminal的状态信息。

## 🚀 服务器启动

### 启动持久化服务器
```bash
./persistent-server.sh start
```

### 管理命令
```bash
./persistent-server.sh status   # 查看服务器状态
./persistent-server.sh test     # 测试所有API端点
./persistent-server.sh logs     # 查看实时日志
./persistent-server.sh stop     # 停止服务器
./persistent-server.sh restart  # 重启服务器
```

## 🔗 API端点

### 基础信息
- **服务器地址**: `http://localhost:[动态端口]`
- **认证方式**: X-AuthKey header (当前已禁用用于开发)
- **响应格式**: JSON
- **认证密钥**: `83958e47ddc89fae695a7e1eb429899871e80334bd58cfc2d17a80388791f073`

### 1. 列出所有工作区
**端点**: `GET /api/v1/widgets/workspaces`

**描述**: 获取Wave Terminal中所有可用工作区的列表

**响应示例**:
```json
{
  "success": true,
  "workspaces": [
    {
      "workspace_id": "886a7057-bd51-4c6a-b5f1-540dccfb6c4f",
      "name": "Starter workspace",
      "tab_ids": [],
      "active_tab_id": "ac154ac7-d422-4904-b554-3ab42d20b631"
    },
    {
      "workspace_id": "79838cdd-f971-40b2-84a8-d143f0d39254",
      "name": "waveterm",
      "tab_ids": ["0d212362-9a4a-4b5d-b0c9-0d75e55e1745"],
      "active_tab_id": "0d212362-9a4a-4b5d-b0c9-0d75e55e1745"
    }
  ]
}
```

**使用场景**:
- 获取所有可用工作区概览
- 监控工作区数量变化
- 获取工作区基本信息用于后续操作

### 2. 根据名称获取工作区信息
**端点**: `GET /api/v1/widgets/workspace/name/{workspace_name}`

**描述**: 通过工作区名称查找特定工作区的详细信息

**参数**:
- `workspace_name`: 工作区名称（支持URL编码，大小写不敏感）

**成功响应示例**:
```json
{
  "success": true,
  "workspace": {
    "workspace_id": "79838cdd-f971-40b2-84a8-d143f0d39254",
    "name": "waveterm",
    "tab_ids": ["0d212362-9a4a-4b5d-b0c9-0d75e55e1745"],
    "active_tab_id": "0d212362-9a4a-4b5d-b0c9-0d75e55e1745"
  }
}
```

**失败响应示例**:
```json
{
  "success": false,
  "error": "workspace with name 'NonExistent' not found"
}
```

**使用场景**:
- 通过友好的名称查找工作区
- 验证工作区是否存在
- 获取工作区ID用于后续操作

### 3. 获取工作区可用的Widgets
**端点**: `GET /api/v1/widgets/workspace/{workspace_id}`

**描述**: 获取指定工作区中可用的widget配置

**参数**:
- `workspace_id`: 工作区UUID

**响应示例**:
```json
{
  "success": true,
  "widgets": {
    "terminal": {
      "name": "Terminal",
      "description": "Interactive terminal session",
      "icon": "square-terminal"
    },
    "web": {
      "name": "Web Browser", 
      "description": "Web browser widget",
      "icon": "globe"
    }
  }
}
```

### 4. 创建Widget
**端点**: `POST /api/v1/widgets`

**描述**: 在指定工作区创建新的widget

**请求体示例**:
```json
{
  "workspace_id": "79838cdd-f971-40b2-84a8-d143f0d39254",
  "widget_type": "terminal",
  "title": "My Terminal",
  "meta": {
    "cwd": "/home/user"
  }
}
```

**响应示例**:
```json
{
  "success": true,
  "block_id": "new-block-uuid",
  "message": "Widget 'terminal' created successfully",
  "widget": {
    "block_id": "new-block-uuid",
    "tab_id": "tab-uuid",
    "workspace_id": "workspace-uuid",
    "widget_type": "terminal",
    "title": "My Terminal",
    "created_at": 1706123456789
  }
}
```

### 5. 获取Widget类型信息
**端点**: `GET /api/v1/widgets`

**描述**: 获取所有可用的widget类型和它们的描述

**响应示例**:
```json
{
  "success": true,
  "widget_types": {
    "terminal": {
      "name": "Terminal",
      "description": "Interactive terminal session",
      "icon": "square-terminal",
      "meta_fields": {
        "controller": "shell controller type",
        "cwd": "working directory",
        "env": "environment variables"
      }
    },
    "web": {
      "name": "Web Browser",
      "description": "Web browser widget for browsing websites", 
      "icon": "globe",
      "meta_fields": {
        "url": "initial URL to load"
      }
    }
  },
  "endpoints": {
    "get_workspace_by_name": {
      "method": "GET",
      "path": "/api/v1/widgets/workspace/name/{workspace_name}",
      "description": "Get workspace information by name (case-insensitive)"
    }
  }
}
```

## 🤖 MCP集成示例

### Python示例
```python
import requests
import json

class WaveTerminalMCP:
    def __init__(self, base_url):
        self.base_url = base_url
        self.session = requests.Session()
        # 禁用代理以避免连接问题
        self.session.proxies = {'http': None, 'https': None}
    
    def list_workspaces(self):
        """获取所有工作区"""
        response = self.session.get(f"{self.base_url}/api/v1/widgets/workspaces")
        return response.json()
    
    def get_workspace_by_name(self, name):
        """根据名称获取工作区"""
        response = self.session.get(f"{self.base_url}/api/v1/widgets/workspace/name/{name}")
        return response.json()
    
    def create_terminal_widget(self, workspace_id, title="AI Terminal", cwd="/"):
        """创建终端widget"""
        data = {
            "workspace_id": workspace_id,
            "widget_type": "terminal",
            "title": title,
            "meta": {"cwd": cwd}
        }
        response = self.session.post(f"{self.base_url}/api/v1/widgets", json=data)
        return response.json()
    
    def get_workspace_id_by_name(self, name):
        """便捷方法：通过名称获取工作区ID"""
        result = self.get_workspace_by_name(name)
        if result.get("success") and result.get("workspace"):
            return result["workspace"]["workspace_id"]
        return None

# 使用示例
mcp = WaveTerminalMCP("http://localhost:61269")

# 获取waveterm工作区的ID
workspace_id = mcp.get_workspace_id_by_name("waveterm")
print(f"Workspace ID: {workspace_id}")

# 在该工作区创建终端
if workspace_id:
    result = mcp.create_terminal_widget(workspace_id, "AI Assistant Terminal")
    print(f"Created widget: {result}")
```

### JavaScript/Node.js示例
```javascript
class WaveTerminalMCP {
    constructor(baseUrl) {
        this.baseUrl = baseUrl;
    }
    
    async request(endpoint, options = {}) {
        const url = `${this.baseUrl}${endpoint}`;
        const response = await fetch(url, {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        });
        return response.json();
    }
    
    async listWorkspaces() {
        return this.request('/api/v1/widgets/workspaces');
    }
    
    async getWorkspaceByName(name) {
        return this.request(`/api/v1/widgets/workspace/name/${encodeURIComponent(name)}`);
    }
    
    async createWidget(workspaceId, widgetType, options = {}) {
        return this.request('/api/v1/widgets', {
            method: 'POST',
            body: JSON.stringify({
                workspace_id: workspaceId,
                widget_type: widgetType,
                ...options
            })
        });
    }
}

// 使用示例
const mcp = new WaveTerminalMCP('http://localhost:61269');

async function example() {
    // 获取工作区信息
    const workspace = await mcp.getWorkspaceByName('waveterm');
    console.log('Workspace:', workspace);
    
    // 创建Web widget
    if (workspace.success) {
        const result = await mcp.createWidget(
            workspace.workspace.workspace_id,
            'web',
            {
                title: 'Documentation',
                meta: { url: 'https://docs.waveterm.dev' }
            }
        );
        console.log('Created widget:', result);
    }
}
```

## 🔍 常见使用场景

### 1. 监控工作区状态
```bash
# 定期检查工作区数量和状态
curl -s "http://localhost:61269/api/v1/widgets/workspaces" | jq '.workspaces | length'
```

### 2. 自动化工作区管理
```bash
# 检查特定工作区是否存在，不存在则创建相关资源
WORKSPACE_ID=$(curl -s "http://localhost:61269/api/v1/widgets/workspace/name/Development" | jq -r '.workspace.workspace_id // empty')

if [ -z "$WORKSPACE_ID" ]; then
    echo "Development workspace not found"
else
    echo "Development workspace ID: $WORKSPACE_ID"
fi
```

### 3. 集成到AI Workflow
- **状态查询**: AI可以实时查询当前活跃的工作区
- **资源创建**: 根据用户需求在特定工作区创建终端或浏览器widget
- **环境监控**: 跟踪工作区的标签页数量和活动状态

## 🛠️ 故障排除

### 1. 连接问题
- 确保服务器正在运行: `./persistent-server.sh status`
- 检查端口是否被占用: `lsof -i :端口号`
- 禁用HTTP代理: `unset http_proxy && unset https_proxy`

### 2. API错误
- 检查请求格式是否正确
- 验证工作区ID是否存在
- 查看服务器日志: `./persistent-server.sh logs`

### 3. 权限问题
- 确保数据目录有写权限
- 检查认证密钥设置（如果启用认证）

## 📊 性能特征

- **响应时间**: 通常 < 50ms
- **并发支持**: 支持多个同时连接
- **内存使用**: 约 20-50MB（取决于工作区数量）
- **持久化**: 数据存储在SQLite数据库中

## 🔄 更新和维护

服务器支持热重启而不丢失数据：
```bash
./persistent-server.sh restart
```

定期检查服务器健康状态：
```bash
./persistent-server.sh test
```

## 📝 日志记录

所有API调用都会记录在服务器日志中：
- 请求URL和方法
- 响应状态
- 数据库操作
- 错误信息

查看实时日志：
```bash
./persistent-server.sh logs
```

这个API系统为MCP集成提供了完整的Wave Terminal状态访问能力，支持AI agents进行智能的工作区管理和资源创建。