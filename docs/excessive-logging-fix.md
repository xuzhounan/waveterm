# 后端日志过多问题修复

## 问题描述

后端产生大量重复的日志输出：
```
[wavesrv] 2025/08/05 12:18:02.873061 [Screenshot] Backend: EventPublishCommand received event: 'sysinfo' (looking for 'screenshot:response')
```

这些日志每秒都在输出，导致日志文件快速增长并影响性能。

## 根本原因

在 `pkg/wshrpc/wshserver/wshserver.go` 的 `EventPublishCommand` 方法中，有一行调试日志会对**每个**接收到的事件都打印输出：

```go
log.Printf("[Screenshot] Backend: EventPublishCommand received event: '%s' (looking for 'screenshot:response')", data.Event)
```

由于系统会每秒发送一次 `sysinfo` 事件（在 `pkg/wshrpc/wshremote/sysinfo.go` 中定义），这导致了大量的日志输出。

## 解决方案

删除了对所有事件的日志记录，只在真正处理 screenshot 响应时才打印日志：

```go
// 之前：对每个事件都打印日志
log.Printf("[Screenshot] Backend: EventPublishCommand received event: '%s' (looking for 'screenshot:response')", data.Event)
if data.Event == "screenshot:response" {
    // ...
}

// 修复后：只在处理 screenshot 事件时打印
if data.Event == "screenshot:response" {
    log.Printf("[Screenshot] Backend: Received screenshot response event from frontend")
    // ...
}
```

## 影响

1. **性能提升**：减少了不必要的日志 I/O 操作
2. **日志清晰**：日志文件不再被重复信息淹没
3. **存储优化**：避免日志文件快速增长

## 其他发现

### Screenshot 相关日志

Screenshot 功能有较多的调试日志，这些在开发阶段是有用的，但在生产环境中可能需要调整日志级别。建议：

1. 将详细的调试信息改为 Debug 级别
2. 只保留关键操作的 Info 级别日志
3. 错误情况使用 Error 级别

### SysInfo 循环

系统信息每秒更新一次，这个频率对于大多数用例来说是合理的。如果需要优化，可以考虑：

1. 增加更新间隔（如 5 秒）
2. 实现智能更新（只在数据变化时发送）
3. 为不同类型的信息使用不同的更新频率

## 相关文件

- `pkg/wshrpc/wshserver/wshserver.go` - 事件发布处理
- `pkg/wshrpc/wshremote/sysinfo.go` - 系统信息循环
- `pkg/service/widgetapiservice/widgetapiservice.go` - Screenshot 服务