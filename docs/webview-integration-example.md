# WebView 集成示例

## 快速开始

以下是集成新 WebView 实现的最小示例。

### 1. 修改 emain/emain.ts

在文件开头添加导入：

```typescript
import { setupWebViewIPC } from "./emain-webview-manager";
```

在 `app.whenReady()` 回调中（约第 575 行附近）添加：

```typescript
app.whenReady().then(async () => {
    // ... 现有代码 ...
    
    // 初始化 WebView IPC
    setupWebViewIPC();
    
    // ... 继续现有代码 ...
});
```

### 2. 修改 emain/preload.ts

添加 WebView API 方法到 `contextBridge.exposeInMainWorld` 的 api 对象中：

```typescript
// WebView API
webViewCreate: (options: any) => ipcRenderer.invoke("webview:create", options),
webViewNavigate: (blockId: string, url: string) => ipcRenderer.invoke("webview:navigate", blockId, url),
webViewGoBack: (blockId: string) => ipcRenderer.invoke("webview:go-back", blockId),
webViewGoForward: (blockId: string) => ipcRenderer.invoke("webview:go-forward", blockId),
webViewReload: (blockId: string) => ipcRenderer.invoke("webview:reload", blockId),
webViewStop: (blockId: string) => ipcRenderer.invoke("webview:stop", blockId),
webViewGetState: (blockId: string) => ipcRenderer.invoke("webview:get-state", blockId),
webViewDestroy: (blockId: string) => ipcRenderer.invoke("webview:destroy", blockId),
onWebViewEvent: (blockId: string, event: string, callback: (data: any) => void) => {
    const channel = `webview:${blockId}:${event}`;
    ipcRenderer.on(channel, (_, data) => callback(data));
    return () => ipcRenderer.removeAllListeners(channel);
},
```

### 3. 创建功能切换

创建一个临时的功能切换来测试：

```typescript
// frontend/app/view/webview/index.tsx
import { WebViewModel } from "./webview";
import { ImprovedWebViewModel } from "./webview-improved";

export function createWebViewModel(blockId: string, nodeModel: BlockNodeModel): ViewModel {
    // 临时硬编码为 true 来测试新实现
    const USE_IMPROVED_WEBVIEW = true;
    
    if (USE_IMPROVED_WEBVIEW) {
        console.log("[WebView] Using improved WebContentsView implementation");
        return new ImprovedWebViewModel(blockId, nodeModel);
    } else {
        console.log("[WebView] Using legacy webview tag implementation");
        return new WebViewModel(blockId, nodeModel);
    }
}
```

### 4. 修改 block.tsx

在 `frontend/app/block/block.tsx` 中修改 web 视图的创建：

```typescript
import { createWebViewModel } from "@/app/view/webview";

// 在 getViewModel 函数中
case "web":
    return createWebViewModel(blockId, blockData, nodeModel);
```

## 测试步骤

1. **构建并运行**
   ```bash
   task dev
   ```

2. **创建 Web 视图**
   - 点击 "+" 创建新 block
   - 选择 "Web" 类型
   - 应该在控制台看到 "Using improved WebContentsView implementation"

3. **验证基本功能**
   - 输入 URL 并回车
   - 测试导航按钮
   - 测试刷新功能

4. **检查日志**
   - 主进程日志会显示 WebView 创建和事件
   - 渲染进程日志会显示状态更新

## 故障排除

### WebView 不显示

1. 检查控制台是否有错误
2. 确认 `setupWebViewIPC()` 已调用
3. 验证 IPC 通道是否正确注册

### 导航不工作

1. 检查 URL 格式是否正确
2. 查看主进程日志中的导航事件
3. 确认 WebContentsView 已正确创建

### 开发工具

在控制台运行以下命令查看 WebView 状态：

```javascript
// 获取当前 block 的 WebView 状态
const blockId = "your-block-id";
const state = await window.api.webViewGetState(blockId);
console.log("WebView State:", state);
```

## 回滚到原实现

只需将 `USE_IMPROVED_WEBVIEW` 设置为 `false`：

```typescript
const USE_IMPROVED_WEBVIEW = false; // 使用原有实现
```