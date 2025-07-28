# CORS修复验证指南

## 修复内容

已成功修复MCP API的CORS跨域访问问题，解决了以下问题：

1. **CORS policy错误** - 从 `http://localhost:5173` 访问 `http://127.0.0.1:60289` 被阻止
2. **缺少CORS头** - Access-Control-Allow-Origin等头部缺失
3. **端口配置不一致** - 消除了错误端口53098的引用
4. **预检请求处理** - 完整支持OPTIONS预检请求

## 修改的文件

### 后端修改
- **`pkg/web/cors.go`** (新增) - 专用CORS处理器，支持开发和生产环境
- **`pkg/web/web.go`** - 增强CORS中间件配置
- **`pkg/web/widgetapi.go`** - 简化CORS头设置，避免重复

### 前端修改  
- **`frontend/app/store/mcpapi.ts`** - 改进baseURL获取逻辑
- **`frontend/app/view/serverstatus/serverstatus.tsx`** - 使用动态端点配置

### 工具和配置
- **`test-cors.sh`** (新增) - CORS测试脚本
- **`.gitignore`** - 排除运行时文件

## 验证步骤

### 1. 启动服务器
```bash
./persistent-server.sh start
```

### 2. 运行CORS测试
```bash
# 完整测试
./test-cors.sh

# 快速测试
./test-cors.sh quick

# 只测试预检请求
./test-cors.sh preflight
```

### 3. 手动验证

在浏览器开发者工具中，应该看到：

**成功的预检请求 (OPTIONS):**
```
Access-Control-Allow-Origin: http://localhost:5173
Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS, HEAD
Access-Control-Allow-Headers: Accept, Authorization, Content-Type, X-CSRF-Token, X-AuthKey, X-Requested-With, Cache-Control
Access-Control-Allow-Credentials: true
```

**成功的API请求 (GET/POST):**
```
Access-Control-Allow-Origin: http://localhost:5173
Access-Control-Expose-Headers: Content-Type, Content-Length, X-ZoneFileInfo, Cache-Control, Last-Modified
```

### 4. 前端测试

在前端开发环境中，以下请求应该正常工作：

```javascript
// 检查服务器状态
fetch('http://127.0.0.1:60289/api/v1/widgets/mcp/status')

// 列出工作区
fetch('http://127.0.0.1:60289/api/v1/widgets/workspaces')

// 创建widget
fetch('http://127.0.0.1:60289/api/v1/widgets', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ workspace_id: 'test', widget_type: 'terminal' })
})
```

## CORS配置特性

### 支持的源 (Origins)
- `http://localhost:5173` (Vite开发服务器)
- `http://127.0.0.1:5173` (IP地址访问)
- `http://localhost:3000` (其他常见开发端口)
- 开发模式下的任意localhost/127.0.0.1端口

### 支持的方法 (Methods)
- GET, POST, PUT, DELETE, OPTIONS, HEAD

### 支持的头部 (Headers)
- Accept, Authorization, Content-Type
- X-CSRF-Token, X-AuthKey, X-Requested-With
- Cache-Control

### 暴露的头部 (Exposed Headers)
- Content-Type, Content-Length
- X-ZoneFileInfo, Cache-Control, Last-Modified

## 故障排除

### 如果CORS测试失败

1. **检查服务器状态**
   ```bash
   ./persistent-server.sh status
   ```

2. **查看服务器日志**
   ```bash
   ./persistent-server.sh logs
   ```

3. **重启服务器**
   ```bash
   ./persistent-server.sh restart
   ```

4. **检查端口占用**
   ```bash
   lsof -i :60289
   lsof -i :60290
   ```

### 如果前端仍有CORS错误

1. 确保使用正确的URL格式 (127.0.0.1而不是localhost)
2. 检查浏览器开发者工具的网络标签
3. 确认服务器在开发模式下运行 (WAVETERM_DEV环境变量)

## 开发vs生产环境

- **开发环境**: 更宽松的CORS策略，支持localhost和127.0.0.1的各种端口
- **生产环境**: 更严格的CORS策略，需要配置具体的允许源

当前配置主要为开发环境优化，生产环境部署时可能需要进一步调整CORS源白名单。