# Wave Terminal AI Documentation

本目录包含Wave Terminal项目的AI文档，用于帮助AI助手更好地理解和操作Wave Terminal的各个组件。

## 🎯 核心概念指南

### 系统架构文档
- **[mcp-vs-widget-api.md](mcp-vs-widget-api.md)** - 📋 MCP与Widget API概念区分指南 ⭐
- **[mcp-development.md](mcp-development.md)** - 🔧 MCP系统开发完整指南 ⭐
- **[widget-api.md](widget-api.md)** - 🚀 Widget API开发指南和参考文档

### 功能开发文档
- [contextmenu.md](../aiprompts/contextmenu.md) - 上下文菜单快速参考
- [getsetconfigvar.md](../aiprompts/getsetconfigvar.md) - 配置变量设置和读取
- [view-prompt.md](../aiprompts/view-prompt.md) - ViewModel开发指南

## 🚀 快速入门

### 对于AI助手开发者
```markdown
1. 📖 首先阅读 [mcp-vs-widget-api.md](mcp-vs-widget-api.md) 了解核心概念区别
2. 🔧 查看 [mcp-development.md](mcp-development.md) 了解MCP工具使用
3. 🚀 参考 [widget-api.md](widget-api.md) 进行Widget开发
```

### 重要概念区分

| 概念 | 用途 | 协议 | 适用场景 |
|------|------|------|----------|
| **MCP** | AI助手工具集成 | Anthropic MCP协议 | Claude Code、AI自动化 |
| **Widget API** | 程序化界面管理 | HTTP REST API | 外部应用、脚本集成 |

## 📚 文档结构说明

### 📋 概念澄清文档
这些文档专门用于澄清容易混淆的概念：
- `mcp-vs-widget-api.md` - 解决MCP与Widget API的概念混淆

### 🔧 技术实现文档
详细的开发指南和API参考：
- `mcp-development.md` - MCP系统完整开发指南
- `widget-api.md` - Widget API详细参考

### 🎯 功能特性文档
具体功能的使用指南：
- `contextmenu.md` - 上下文菜单开发
- `getsetconfigvar.md` - 配置管理
- `view-prompt.md` - 视图模型

## 🎯 使用说明

这些文档主要面向AI助手，用于：

### ✅ 核心目标
1. **概念理解** - 正确区分MCP和Widget API
2. **架构理解** - 理解Wave Terminal的设计模式
3. **代码生成** - 快速生成符合项目规范的代码
4. **问题解决** - 解决开发中的常见问题

### 📖 阅读建议
- **首次阅读**: 从概念区分文档开始
- **开发MCP工具**: 重点关注`mcp-development.md`
- **开发Widget**: 重点关注`widget-api.md`
- **故障排除**: 查看各文档的故障排除章节

## ⚠️ 重要提醒

### 避免常见错误
- ❌ 不要将MCP和Widget API混为一谈
- ❌ 不要使用错误的文件路径引用
- ❌ 不要忽略认证和权限要求

### 最佳实践
- ✅ 根据使用场景选择合适的API
- ✅ 遵循文档中的代码示例
- ✅ 注意错误处理和安全考虑

## 📝 贡献指南

添加新的AI文档时，请遵循以下原则：

### 文档标准
- **清晰概述** - 提供功能的明确定义和使用场景
- **完整示例** - 包含可运行的代码示例
- **概念解释** - 解释关键概念和设计决策
- **时效维护** - 保持文档与代码同步更新

### 命名规范
- 使用描述性文件名
- 概念区分文档使用 `xxx-vs-yyy.md` 格式
- 开发指南使用 `xxx-development.md` 格式
- API参考使用 `xxx-api.md` 格式

---

> 💡 **提示**: 如果你是AI助手，建议先阅读概念区分文档，确保正确理解Wave Terminal的架构体系。