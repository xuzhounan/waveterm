# Wave Terminal Widget API 开发指南

## 概述

Wave Terminal Widget API 是一个REST API系统，允许通过HTTP请求在工作区中创建和管理widgets。该API集成到Wave Terminal的现有架构中，提供了编程式创建widgets的能力。

## 架构设计

### 核心组件

1. **WidgetAPIService** (`pkg/service/widgetapiservice/`)
   - 业务逻辑层，处理widget创建、查询和管理
   - 与现有的wcore、wconfig系统集成
   - 提供类型安全的API接口

2. **REST处理器** (`pkg/web/widgetapi.go`)
   - HTTP请求处理和路由
   - JSON序列化/反序列化
   - 错误处理和响应格式化

3. **配置扩展** (`pkg/wconfig/settingsconfig.go`)
   - GetWorkspaceWidgetConfig函数
   - 工作空间级别的widget配置管理

### 数据流

```
HTTP Request → REST Handler → WidgetAPIService → wcore.CreateBlock → Database
     ↓
Response ← JSON Formatter ← Service Response ← Block Creation ← Event Publishing
```

## API端点设计

### 1. 获取Widget类型列表
```http
GET /api/v1/widgets
```

**功能**: 返回所有支持的widget类型及其配置选项

**响应结构**:
```typescript
{
  success: boolean;
  widget_types: {
    [key: string]: {
      name: string;
      description: string;
      icon: string;
      meta_fields: { [key: string]: string };
    }
  };
  examples: { [key: string]: CreateWidgetAPIRequest };
}
```

### 2. 列出工作空间
```http
GET /api/v1/widgets/workspaces
```

**功能**: 返回所有可用的工作空间及其基本信息

**响应结构**:
```typescript
{
  success: boolean;
  workspaces: WorkspaceBasicInfo[];
  error?: string;
}

type WorkspaceBasicInfo = {
  workspace_id: string;
  name: string;
  tab_ids: string[];
  active_tab_id?: string;
}
```

### 3. 获取工作空间Widget配置
```http
GET /api/v1/widgets/workspace/{workspace_id}
```

**功能**: 返回指定工作空间的widget配置（包括默认和自定义配置）

**响应结构**:
```typescript
{
  success: boolean;
  widgets: { [key: string]: WidgetConfigType };
  error?: string;
}
```

### 4. 创建Widget
```http
POST /api/v1/widgets
```

**功能**: 在指定工作空间创建新的widget

**请求结构**:
```typescript
type CreateWidgetAPIRequest = {
  workspace_id: string;
  tab_id?: string;              // 可选，默认使用活动标签页
  widget_type: string;          // 必需，widget类型
  title?: string;               // 可选，自定义标题
  icon?: string;                // 可选，自定义图标
  meta?: { [key: string]: any }; // 可选，额外元数据
  position?: WidgetPosition;     // 可选，位置设置
  magnified?: boolean;          // 可选，是否放大显示
  ephemeral?: boolean;          // 可选，是否临时widget
}

type WidgetPosition = {
  target_block_id?: string;     // 目标块ID
  action?: string;              // 位置动作：replace, splitright, splitdown, splitleft, splitup
}
```

**响应结构**:
```typescript
{
  success: boolean;
  block_id?: string;
  message?: string;
  error?: string;
  widget?: WidgetInfo;
}

type WidgetInfo = {
  block_id: string;
  tab_id: string;
  workspace_id: string;
  widget_type: string;
  title: string;
  icon: string;
  meta: { [key: string]: any };
  created_at: number;
}
```

## 支持的Widget类型

### 1. Terminal (`terminal`)
创建交互式终端会话

**元数据选项**:
- `controller`: 控制器类型 (默认: "shell")
- `cwd`: 工作目录
- `env`: 环境变量映射

**示例**:
```json
{
  "workspace_id": "ws-123",
  "widget_type": "terminal",
  "title": "Development Terminal",
  "meta": {
    "cwd": "/home/user/project",
    "env": {
      "NODE_ENV": "development"
    }
  }
}
```

### 2. Web Browser (`web`)
创建web浏览器widget

**元数据选项**:
- `url`: 初始URL (默认: "https://www.waveterm.dev")

**示例**:
```json
{
  "workspace_id": "ws-123",
  "widget_type": "web",
  "title": "Documentation",
  "meta": {
    "url": "https://docs.waveterm.dev"
  }
}
```

### 3. File Browser (`files`)
创建文件浏览器widget

**元数据选项**:
- `file`: 初始路径 (默认: "~")

**示例**:
```json
{
  "workspace_id": "ws-123",
  "widget_type": "files",
  "title": "Project Files",
  "meta": {
    "file": "/home/user/project"
  }
}
```

### 4. AI Assistant (`ai`)
创建WaveAI聊天助手

**元数据选项**: 无特殊选项

### 5. System Information (`sysinfo`)
创建系统监控和信息显示widget

**元数据选项**: 无特殊选项

### 6. Help (`help`)
创建帮助文档widget

**元数据选项**: 无特殊选项

### 7. Quick Tips (`tips`)
创建快速提示widget

**元数据选项**: 无特殊选项

## 代码实现模式

### 1. 服务层实现
```go
type WidgetAPIService struct{}

func (ws *WidgetAPIService) CreateWidget(ctx context.Context, req CreateWidgetAPIRequest) (*CreateWidgetAPIResponse, error) {
    // 1. 验证工作空间存在
    workspace, err := wcore.GetWorkspace(ctx, req.WorkspaceId)
    
    // 2. 确定目标标签页
    tabId := req.TabId
    if tabId == "" {
        tabId = workspace.ActiveTabId
    }
    
    // 3. 创建块定义
    blockDef, err := ws.createBlockDefFromWidgetType(req.WidgetType, req.Meta)
    
    // 4. 创建块
    blockRef, err := wcore.CreateBlock(ctx, tabId, blockDef, nil)
    
    // 5. 发布事件
    wps.Broker.Publish(wps.WaveEvent{...})
    
    // 6. 返回响应
    return &CreateWidgetAPIResponse{...}, nil
}
```

### 2. REST处理器实现
```go
func handleCreateWidget(w http.ResponseWriter, r *http.Request, ctx context.Context) {
    // 1. 解析请求
    var req widgetapiservice.CreateWidgetAPIRequest
    decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&req)
    
    // 2. 验证参数
    if req.WorkspaceId == "" {
        writeErrorResponse(w, "workspace_id is required", http.StatusBadRequest)
        return
    }
    
    // 3. 调用服务
    response, err := widgetapiservice.WidgetAPIServiceInstance.CreateWidget(ctx, req)
    
    // 4. 返回响应
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(response)
}
```

### 3. 路由注册
```go
func RunWebServer(listener net.Listener) {
    gr := mux.NewRouter()
    
    // 现有路由...
    
    // Widget API endpoints
    gr.PathPrefix("/api/v1/widgets").HandlerFunc(handleWidgetAPI)
    
    // 其他路由...
}
```

## 错误处理

### HTTP状态码
- `200 OK`: 请求成功
- `201 Created`: Widget创建成功
- `400 Bad Request`: 请求参数错误
- `401 Unauthorized`: 认证失败
- `404 Not Found`: 资源不存在
- `500 Internal Server Error`: 服务器内部错误

### 错误响应格式
```json
{
  "success": false,
  "error": "详细错误信息"
}
```

### 常见错误场景
1. **工作空间不存在**: 返回404和相应错误信息
2. **无效widget类型**: 返回400和支持的类型列表
3. **认证失败**: 返回401
4. **内部服务错误**: 返回500和日志记录

## 安全考虑

### 认证
API使用Wave Terminal内置的认证系统：
```go
if !authkey.ValidateIncomingRequest(r) {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
```

### CORS支持
```go
w.Header().Set("Access-Control-Allow-Origin", "*")
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
```

## 扩展指南

### 添加新的Widget类型
1. 在`createBlockDefFromWidgetType`方法中添加新的case
2. 更新`handleListWidgetTypes`中的类型定义
3. 添加相应的元数据验证逻辑
4. 更新文档和测试

### 添加新的API端点
1. 在`handleWidgetAPI`中添加新的路由逻辑
2. 实现对应的处理函数
3. 在服务层添加业务逻辑方法
4. 更新API文档

## 测试策略

### 单元测试
- 服务层方法的单元测试
- 错误处理逻辑测试
- 参数验证测试

### 集成测试
- 完整的API请求/响应测试
- 数据库集成测试
- 事件发布测试

### 示例测试脚本
```bash
# 创建终端widget
curl -X POST http://localhost:8080/api/v1/widgets \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_id": "ws-123",
    "widget_type": "terminal",
    "title": "Test Terminal"
  }'

# 列出工作空间
curl -X GET http://localhost:8080/api/v1/widgets/workspaces

# 获取widget类型
curl -X GET http://localhost:8080/api/v1/widgets
```

## 最佳实践

### 1. 错误处理
- 始终返回结构化的错误响应
- 记录详细的错误日志
- 提供有用的错误信息给客户端

### 2. 参数验证
- 验证所有必需参数
- 检查参数格式和范围
- 提供清晰的验证错误信息

### 3. 响应格式
- 保持一致的响应结构
- 使用适当的HTTP状态码
- 包含足够的信息用于调试

### 4. 性能优化
- 避免不必要的数据库查询
- 使用适当的缓存策略
- 实现合理的超时机制

## 故障排除

### 常见问题
1. **Widget创建失败**
   - 检查工作空间ID是否存在
   - 验证widget类型是否支持
   - 确认认证是否通过

2. **API响应超时**
   - 检查数据库连接
   - 验证网络配置
   - 查看服务器日志

3. **权限错误**
   - 确认认证密钥是否有效
   - 检查CORS配置
   - 验证请求头设置

### 调试工具
- 使用浏览器开发者工具查看网络请求
- 检查Wave Terminal服务器日志
- 使用curl或Postman测试API端点

通过这个API系统，开发者可以轻松地通过编程方式管理Wave Terminal中的widgets，为自动化和集成提供了强大的工具。