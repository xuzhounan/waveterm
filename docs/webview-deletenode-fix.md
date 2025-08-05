# DeleteNode 布局错误修复说明

## 问题描述

删除 Web block（或任何 block）时出现错误：
```
Cannot apply eventbus layout action DeleteNode, could not find leaf node with blockId
```

## 根本原因

这是一个典型的竞态条件问题，删除流程如下：

1. 用户触发删除操作（点击删除按钮或快捷键）
2. 前端 `LayoutModel.closeNode()` 被调用
3. `closeNode` 执行以下操作：
   - 从布局树中删除节点（通过 `treeReducer` 执行 `DeleteNode` action）
   - 调用 `onNodeDelete` 回调，触发 `ObjectService.DeleteBlock`
4. 后端删除 block 并发送 `blockclose` 事件
5. 前端收到 `blockclose` 事件后，尝试再次通过 `DeleteNode` action 删除节点
6. 但此时节点已在步骤3中被删除，导致报错

## 解决方案

### 1. 降低错误日志级别

将找不到节点的 `console.error` 改为 `console.debug`，因为这是正常的情况：

```typescript
// frontend/layout/lib/layoutModel.ts - closeNode 方法
console.debug("closeNode: node not found in tree, may have been already deleted", nodeId);
```

### 2. 避免重复调用 onNodeDelete

在处理来自后端的 `DeleteNode` 事件时，不再调用 `onNodeDelete`，因为 block 已经在后端被删除：

```typescript
// frontend/layout/lib/layoutModel.ts - onTreeStateAtomUpdated 方法
case LayoutTreeActionType.DeleteNode: {
    const leaf = this?.getNodeByBlockId(action.blockid);
    if (leaf) {
        // 直接删除布局节点，不调用 onNodeDelete
        const deleteAction: LayoutTreeDeleteNodeAction = {
            type: LayoutTreeActionType.DeleteNode,
            nodeId: leaf.id,
        };
        this.treeReducer(deleteAction);
        // 注意：这里不调用 onNodeDelete，因为这个 action 来自后端
        // block 已经在后端被删除了
    } else {
        // 节点已被删除是正常情况
        console.debug(
            "Layout action DeleteNode: node already deleted, blockId:",
            action.blockid
        );
    }
    break;
}
```

## 测试验证

1. 创建一个 Web block
2. 删除该 block
3. 不应再看到 DeleteNode 错误
4. 控制台可能会有 debug 日志（如果启用了 debug 级别）

## 影响范围

- 此修复适用于所有 block 类型，不仅仅是 WebView
- 不会影响正常的删除功能
- 提高了系统的健壮性，避免了不必要的错误日志

## 相关文件

- `frontend/layout/lib/layoutModel.ts` - 布局模型，处理节点删除逻辑
- `frontend/app/tab/tabcontent.tsx` - 定义 `onNodeDelete` 回调
- `pkg/wcore/block.go` - 后端 block 删除逻辑，发送 `blockclose` 事件