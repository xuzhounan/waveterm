# Wave Terminal 进程冲突解决方案使用指南

## 概述

本指南提供了一套完整的工具来解决Wave Terminal开发环境和MCP服务器之间的进程冲突问题。通过环境变量隔离和独立的进程管理，确保两个环境可以同时运行而不产生冲突。

## 问题背景

原始问题是环境变量作用域污染导致的进程冲突：
- `persistent-server.sh` 设置的环境变量会影响后续启动的进程
- 多个Wave进程争夺同一个锁文件
- 数据目录冲突导致服务启动失败

## 解决方案组件

### 1. 修复的 persistent-server.sh
**目的**: MCP服务器启动脚本，环境变量完全隔离

**修复内容**:
- 使用子shell启动服务器，环境变量不会污染当前shell
- 移除了全局环境变量设置
- 使用固定数据目录 `/tmp/waveterm-mcp`

**使用方法**:
```bash
./persistent-server.sh start    # 启动MCP服务器
./persistent-server.sh status   # 查看状态
./persistent-server.sh stop     # 停止服务器
```

### 2. dev-isolated.sh
**目的**: 独立的开发环境启动脚本

**特性**:
- 完全隔离的环境变量设置
- 使用独立数据目录 `$HOME/.waveterm`
- 自动检测环境冲突
- 支持与MCP服务器同时运行

**使用方法**:
```bash
./dev-isolated.sh start      # 启动开发环境
./dev-isolated.sh status     # 查看状态
./dev-isolated.sh stop       # 停止开发环境
./dev-isolated.sh restart    # 重启
./dev-isolated.sh clean      # 清理环境
```

### 3. wave-process-manager.sh
**目的**: 统一的进程管理工具

**功能**:
- 显示所有Wave相关进程状态
- 检查PID文件、锁文件、端口占用
- 智能清理所有进程
- 环境变量状态检查

**使用方法**:
```bash
./wave-process-manager.sh status        # 显示进程状态
./wave-process-manager.sh smart-cleanup # 智能清理
./wave-process-manager.sh clean-locks   # 仅清理锁文件
./wave-process-manager.sh kill-all      # 强制终止所有进程
```

### 4. env-conflict-detector.sh
**目的**: 环境冲突检测和修复工具

**功能**:
- 检测环境变量污染
- 分析进程间冲突
- 检查数据目录冲突
- 交互式修复助手
- 生成详细报告

**使用方法**:
```bash
./env-conflict-detector.sh detect      # 全面检测
./env-conflict-detector.sh interactive # 交互式修复
./env-conflict-detector.sh report      # 生成报告
```

## 常见使用场景

### 场景1: 首次设置环境
```bash
# 1. 检测当前环境
./env-conflict-detector.sh detect

# 2. 如果检测到冲突，执行清理
./wave-process-manager.sh smart-cleanup

# 3. 清理环境变量 (在当前shell中执行)
unset WAVETERM_DATA_HOME WAVETERM_CONFIG_HOME WAVETERM_AUTH_KEY
unset WAVETERM_WEB_PORT WAVETERM_WS_PORT

# 4. 启动MCP服务器
./persistent-server.sh start

# 5. 在新的shell中启动开发环境
./dev-isolated.sh start
```

### 场景2: 日常开发工作流
```bash
# 启动MCP服务器 (一次性设置)
./persistent-server.sh start

# 在新shell中启动开发环境
./dev-isolated.sh start

# 查看所有进程状态
./wave-process-manager.sh status

# 停止开发环境
./dev-isolated.sh stop

# MCP服务器保持运行，可根据需要停止
./persistent-server.sh stop
```

### 场景3: 故障排除
```bash
# 快速诊断
./env-conflict-detector.sh detect

# 交互式修复
./env-conflict-detector.sh interactive

# 强制清理所有进程
./wave-process-manager.sh smart-cleanup

# 生成详细报告
./env-conflict-detector.sh report
```

## 关键特性

### 环境隔离
- **MCP服务器**: 使用 `/tmp/waveterm-mcp` 数据目录
- **开发环境**: 使用 `$HOME/.waveterm` 数据目录
- **环境变量**: 通过子shell完全隔离

### 进程管理
- **智能检测**: 自动检测冲突和孤立进程
- **优雅停止**: 先尝试正常终止，再强制清理
- **状态监控**: 实时显示所有相关进程状态

### 冲突预防
- **启动前检查**: 检测可能的环境冲突
- **数据目录隔离**: 不同服务使用不同目录
- **端口管理**: 自动选择可用端口

## 故障排除指南

### 问题: 进程启动失败，提示锁文件冲突
```bash
# 解决方案
./wave-process-manager.sh smart-cleanup
# 等待几秒后重新启动
```

### 问题: 环境变量污染
```bash
# 检测污染
./env-conflict-detector.sh detect

# 清理环境变量
unset WAVETERM_DATA_HOME WAVETERM_CONFIG_HOME WAVETERM_AUTH_KEY
unset WAVETERM_WEB_PORT WAVETERM_WS_PORT
```

### 问题: 多个服务争夺资源
```bash
# 检查进程状态
./wave-process-manager.sh status

# 确保使用不同的启动脚本
# MCP服务器: ./persistent-server.sh
# 开发环境: ./dev-isolated.sh
```

### 问题: 端口冲突
```bash
# 检查端口占用
./wave-process-manager.sh status

# 修改配置文件中的端口设置
# 或者使用自动端口选择
```

## 最佳实践

1. **环境隔离**: 始终在新shell中启动开发环境
2. **状态检查**: 定期使用状态检查工具
3. **清理习惯**: 开发完成后及时清理进程
4. **配置管理**: 使用不同的数据目录配置
5. **日志监控**: 定期查看日志文件

## 文件结构

```
waveterm/
├── persistent-server.sh          # MCP服务器启动脚本
├── dev-isolated.sh              # 独立开发环境脚本
├── wave-process-manager.sh      # 进程管理工具
├── env-conflict-detector.sh     # 环境冲突检测工具
├── PROCESS_MANAGEMENT_GUIDE.md  # 本使用指南
├── waveterm-server.log          # MCP服务器日志
├── dev-waveterm.log            # 开发环境日志
├── waveterm-server.pid         # MCP服务器PID
└── dev-waveterm.pid            # 开发环境PID
```

## 注意事项

1. **权限**: 确保所有脚本具有执行权限
2. **依赖**: 需要安装 yarn、go 等开发工具
3. **端口**: 确保所需端口未被其他服务占用
4. **磁盘空间**: 确保临时目录有足够空间
5. **网络**: 某些功能可能需要网络访问

## 支持和维护

如果遇到问题：
1. 运行 `./env-conflict-detector.sh interactive` 进行诊断
2. 生成报告: `./env-conflict-detector.sh report`
3. 查看日志文件获取详细错误信息
4. 使用 `./wave-process-manager.sh smart-cleanup` 重置环境

此解决方案确保了Wave Terminal开发环境和MCP服务器的完全隔离，消除了进程冲突问题。