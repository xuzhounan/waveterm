# Widget API 测试文档

## API概述

为Wave Terminal开发的REST API用于通过后端请求在工作区创建widget。

## API端点

### 1. 获取widget类型列表
```bash
GET /api/v1/widgets
```

**响应示例:**
```json
{
  "success": true,
  "widget_types": {
    "terminal": {
      "name": "Terminal",
      "description": "Interactive terminal session",
      "icon": "square-terminal",
      "meta_fields": {
        "controller": "shell controller type (default: 'shell')",
        "cwd": "working directory",
        "env": "environment variables"
      }
    },
    "web": {
      "name": "Web Browser",
      "description": "Web browser widget for browsing websites",
      "icon": "globe",
      "meta_fields": {
        "url": "initial URL to load (default: 'https://www.waveterm.dev')"
      }
    }
  }
}
```

### 2. 列出所有工作空间
```bash
GET /api/v1/widgets/workspaces
```

**响应示例:**
```json
{
  "success": true,
  "workspaces": [
    {
      "workspace_id": "workspace-123",
      "name": "Default Workspace",
      "tab_ids": ["tab-1", "tab-2"],
      "active_tab_id": "tab-1"
    }
  ]
}
```

### 3. 获取工作空间的widget配置
```bash
GET /api/v1/widgets/workspace/{workspace_id}
```

**响应示例:**
```json
{
  "success": true,
  "widgets": {
    "defwidget@terminal": {
      "display:order": -5,
      "icon": "square-terminal",
      "label": "terminal",
      "blockdef": {
        "meta": {
          "view": "term",
          "controller": "shell"
        }
      }
    }
  }
}
```

### 4. 创建widget
```bash
POST /api/v1/widgets
Content-Type: application/json
```

**请求体示例:**
```json
{
  "workspace_id": "workspace-123",
  "tab_id": "tab-1",
  "widget_type": "terminal",
  "title": "My Custom Terminal",
  "icon": "terminal",
  "meta": {
    "cwd": "/home/user/projects"
  },
  "position": {
    "target_block_id": "block-456",
    "action": "splitright"
  },
  "magnified": false
}
```

**响应示例:**
```json
{
  "success": true,
  "block_id": "block-789",
  "message": "Widget 'terminal' created successfully",
  "widget": {
    "block_id": "block-789", 
    "tab_id": "tab-1",
    "workspace_id": "workspace-123",
    "widget_type": "terminal",
    "title": "My Custom Terminal",
    "icon": "terminal",
    "meta": {
      "cwd": "/home/user/projects"
    },
    "created_at": 1642694400000
  }
}
```

## Widget类型说明

### 支持的Widget类型

1. **terminal** - 终端会话
   - `controller`: shell控制器类型 (默认: 'shell')
   - `cwd`: 工作目录
   - `env`: 环境变量

2. **web** - Web浏览器
   - `url`: 初始加载的URL (默认: 'https://www.waveterm.dev')

3. **files** - 文件浏览器
   - `file`: 初始浏览路径 (默认: '~')

4. **ai** - AI助手
   - 无特殊参数

5. **sysinfo** - 系统信息
   - 无特殊参数

6. **help** - 帮助文档
   - 无特殊参数

7. **tips** - 快速提示
   - 无特殊参数

### Position动作类型

- `replace`: 替换目标块
- `splitright`: 在右侧分割
- `splitdown`: 在下方分割
- `splitleft`: 在左侧分割
- `splitup`: 在上方分割

## 测试用例

### 测试1: 创建终端widget
```bash
curl -X POST http://localhost:8080/api/v1/widgets \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_id": "workspace-123",
    "widget_type": "terminal",
    "title": "Development Terminal",
    "meta": {
      "cwd": "/home/user/dev"
    }
  }'
```

### 测试2: 创建Web浏览器widget
```bash
curl -X POST http://localhost:8080/api/v1/widgets \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_id": "workspace-123", 
    "widget_type": "web",
    "title": "Documentation",
    "meta": {
      "url": "https://docs.waveterm.dev"
    }
  }'
```

### 测试3: 列出工作空间
```bash
curl -X GET http://localhost:8080/api/v1/widgets/workspaces
```

### 测试4: 获取widget类型
```bash
curl -X GET http://localhost:8080/api/v1/widgets
```

## 错误处理

API使用标准HTTP状态码：

- `200 OK`: 请求成功
- `201 Created`: Widget创建成功
- `400 Bad Request`: 请求参数错误
- `401 Unauthorized`: 认证失败
- `404 Not Found`: 资源不存在
- `500 Internal Server Error`: 服务器内部错误

**错误响应格式:**
```json
{
  "success": false,
  "error": "错误描述信息"
}
```

## 认证

API使用Wave Terminal内置的认证系统，需要有效的认证密钥才能访问。

## 实现细节

- API集成到现有的Wave Terminal web服务中
- 使用Go语言实现后端逻辑
- 支持CORS跨域请求
- 遵循RESTful API设计原则
- 与现有的widget系统无缝集成

## 注意事项

1. 创建widget前，工作空间和标签页必须存在
2. widget类型必须是支持的类型之一
3. 位置参数是可选的，如果不提供会使用默认位置
4. 自定义元数据会根据widget类型进行验证
5. API会自动发送更新事件通知前端刷新