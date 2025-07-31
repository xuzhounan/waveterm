# MCP vs Widget API 概念区分指南

## 🎯 核心概念对比

### **MCP (Model Context Protocol)**
- **定义**: Anthropic开发的AI模型上下文协议
- **用途**: 为AI助手提供工具和资源访问能力
- **场景**: AI代理需要访问外部系统、文件、API等
- **协议层**: 应用层协议，类似于LSP (Language Server Protocol)

### **Widget API** 
- **定义**: Wave Terminal的REST API系统
- **用途**: 程序化创建和管理终端界面组件
- **场景**: 外部程序需要在Wave Terminal中创建widget
- **协议层**: HTTP REST API

## 🏗️ 架构关系

```
Wave Terminal 应用
├── MCP 集成
│   ├── AI 助手功能
│   ├── 工具调用 (mcp__wave-terminal__*)
│   └── 资源访问
└── Widget API 系统
    ├── HTTP REST 端点
    ├── Widget 创建管理
    └── 工作空间操作
```

## 🔧 技术实现差异

### MCP 实现
```typescript
// MCP 工具调用示例
mcp__wave-terminal__create_widget({
  workspace_id: "ws-123",
  widget_type: "terminal"
})
```

### Widget API 实现
```bash
# HTTP REST API 调用示例
curl -X POST http://localhost:8090/api/v1/widgets \
  -H "Content-Type: application/json" \
  -d '{"workspace_id": "ws-123", "widget_type": "terminal"}'
```

## 📊 功能对比表

| 特性 | MCP | Widget API |
|------|-----|------------|
| **协议类型** | Anthropic MCP协议 | HTTP REST API |
| **主要用户** | AI助手/代理 | 外部应用程序 |
| **认证方式** | MCP认证 | AuthKey认证 |
| **数据格式** | MCP标准格式 | JSON |
| **调用方式** | 工具调用 | HTTP请求 |
| **状态管理** | MCP服务器 | Wave Terminal内部 |

## 🎭 使用场景区分

### **MCP适用场景**
- ✅ AI助手需要创建terminal
- ✅ Claude Code等AI工具集成
- ✅ 自动化AI工作流
- ✅ 智能代码助手功能

### **Widget API适用场景**
- ✅ 外部IDE集成
- ✅ 自动化脚本调用
- ✅ 第三方应用集成
- ✅ 批量widget管理

## ⚠️ 常见混淆点

### **错误理解**
- ❌ MCP就是Widget API
- ❌ MCP服务器管理widget创建
- ❌ Widget API是MCP的实现

### **正确理解**
- ✅ MCP和Widget API是两个独立系统
- ✅ 都可以创建widget，但协议不同
- ✅ MCP专为AI助手设计，Widget API面向通用程序

## 🔍 实际项目中的体现

### **MCP相关文件**
```
emain/emain-wavesrv.ts       # MCP服务器启动逻辑
mcp-bridge.cjs               # MCP桥接服务
frontend/app/store/mcpapi.ts # MCP API集成 (如果存在)
```

### **Widget API相关文件**
```
pkg/web/widgetapi.go           # REST API处理器
pkg/service/widgetapiservice/  # 业务逻辑服务
ai_docs/widget-api.md         # API文档
```

## 💡 开发建议

### **选择MCP当**
- 开发AI助手功能
- 需要与Claude等模型集成
- 构建智能自动化工具

### **选择Widget API当**
- 开发传统应用集成
- 需要HTTP API访问
- 构建管理工具或脚本

## 🚀 协同工作

MCP和Widget API可以协同工作：
- MCP调用可以间接使用Widget API逻辑
- 两个系统共享底层的widget创建机制
- 都使用相同的工作空间和标签页概念

## 📚 相关文档

- [MCP开发文档](./mcp-development.md) (待创建)
- [Widget API开发指南](./widget-api.md)
- [系统架构概览](../README.md)

---

> 💡 **记住**: MCP是为AI设计的协议，Widget API是为程序设计的接口。选择合适的工具来解决合适的问题。