# 终端日志清理

## 问题描述

后端产生大量的终端输入输出相关的调试日志，包括：

1. `📥 收到命令 (BlockId: xxx): "command"`
2. `📄 收到输出 (BlockId: xxx, n bytes): "output"`
3. `🖥️ 写入命令到 pty (BlockId: xxx): "command"`
4. `📤 发送命令到 shell channel (BlockId: xxx): "command"`
5. `SETVIEW: blockId | "view"`
6. `tab.BlockIds[xxx]: [...]`
7. `WidgetAPIService.SendBlockInput called with block_id=xxx, input_type=xxx`
8. `Successfully sent xxx to block xxx`

这些日志会记录每一个终端输入和输出，导致：
- 日志文件快速增长
- 可能泄露敏感信息（如密码）
- 影响系统性能

## 解决方案

注释掉了所有这些调试日志：

### 1. wshserver.go
- `📥 收到命令` 日志
- `SETVIEW` 日志

### 2. blockcontroller.go
- `📄 收到输出` 日志
- `🖥️ 写入命令到 pty` 日志
- `📤 发送命令到 shell channel` 日志

### 3. windowservice.go
- `tab.BlockIds` 日志

### 4. widgetapiservice.go
- `SendBlockInput called` 日志
- `Successfully sent` 日志

## 影响

1. **性能提升**：减少了高频的 I/O 操作
2. **安全性提升**：避免在日志中泄露敏感信息
3. **日志可读性**：日志不再被终端输入输出淹没
4. **存储优化**：大幅减少日志文件大小

## 建议

1. **开发环境**：可以通过环境变量或配置文件控制是否启用这些调试日志
2. **生产环境**：这些日志应该始终关闭
3. **调试模式**：需要调试时可以临时启用特定的日志

## 相关文件

- `pkg/wshrpc/wshserver/wshserver.go`
- `pkg/blockcontroller/blockcontroller.go`
- `pkg/service/windowservice/windowservice.go`
- `pkg/service/widgetapiservice/widgetapiservice.go`