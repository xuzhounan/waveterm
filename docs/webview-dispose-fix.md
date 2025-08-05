# WebView Dispose 错误修复说明

## 问题描述

删除 Web block 时出现错误，因为旧的 `WebViewModel` 类没有实现 `dispose` 方法，而 block 组件在清理时会尝试调用这个方法。

## 已完成的修复

### 1. 为旧的 WebViewModel 添加 dispose 方法

在 `frontend/app/view/webview/webview.tsx` 中添加了 dispose 方法：

```typescript
// Dispose method for cleanup when block is deleted
dispose() {
    // No special cleanup needed for legacy webview implementation
    // The webview tag will be automatically cleaned up by React
}
```

### 2. 新的 ImprovedWebViewModel 已有完整的 dispose 实现

在 `frontend/app/view/webview/webview-improved.tsx` 中：

```typescript
// 清理资源
async dispose() {
    await getApi().webViewDestroy(this.blockId);
}
```

### 3. ViewModel 接口定义

`dispose` 方法在 `frontend/types/custom.d.ts` 的 ViewModel 接口中已定义为可选方法：

```typescript
interface ViewModel {
    // ... 其他属性 ...
    
    // Cleans up resources when the block is disposed.
    dispose?: () => void;
}
```

### 4. 调用点使用了安全的可选链操作符

在 `frontend/app/block/block.tsx` 中，dispose 的调用使用了可选链：

```typescript
useEffect(() => {
    return () => {
        unregisterBlockComponentModel(props.nodeModel.blockId);
        viewModel?.dispose?.();  // 安全调用
    };
}, []);
```

## 测试验证

1. **使用旧实现**（`USE_IMPROVED_WEBVIEW = false`）：
   - 创建 Web block
   - 删除 block
   - 不应再出现错误

2. **使用新实现**（`USE_IMPROVED_WEBVIEW = true`）：
   - 创建 Web block
   - 删除 block
   - WebView 资源应被正确清理

## 注意事项

- 旧的 webview 标签实现不需要特殊的清理逻辑，React 会自动处理 DOM 元素的移除
- 新的 WebContentsView 实现需要显式调用主进程的清理方法
- 其他 ViewModel 实现（如 TermViewModel）也应该实现 dispose 方法来清理资源

## 后续建议

1. 确保所有 ViewModel 实现都有适当的 dispose 方法
2. 在开发新的 View 类型时，记得实现资源清理逻辑
3. 考虑在 ViewModel 基类中提供默认的空 dispose 实现