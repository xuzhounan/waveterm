# WebView 改进实现测试指南

## 概述

本文档说明如何在 Wave Terminal 中测试新的 WebContentsView 实现。

## 前提条件

1. 确保已经安装了开发依赖：
   ```bash
   yarn install
   ```

2. 确保开发环境正常运行：
   ```bash
   task dev
   ```

## 集成步骤

### 1. 启用新的 WebView 管理器

在 `emain/emain.ts` 中添加初始化代码：

```typescript
import { setupWebViewIPC } from "./emain-webview-manager";

// 在 app.whenReady() 回调中添加
setupWebViewIPC();
```

### 2. 添加功能开关

在设置中添加实验性功能开关：

```typescript
// pkg/wconfig/settingsconfig.go
"web:use-webcontentsview": {
    Type:        ConfigTypeBool,
    Description: "Use improved WebContentsView instead of webview tag (experimental)",
    Default:     false,
}
```

### 3. 修改 Block 创建逻辑

在前端根据功能开关选择实现：

```typescript
// frontend/app/block/block.tsx
import { WebViewModel } from "@/app/view/webview/webview";
import { ImprovedWebViewModel } from "@/app/view/webview/webview-improved";

function createViewModel(blockId: string, blockView: string, nodeModel: BlockNodeModel): ViewModel {
    if (blockView === "web") {
        const useImproved = globalStore.get(getSettingsKeyAtom("web:use-webcontentsview"));
        if (useImproved) {
            return new ImprovedWebViewModel(blockId, nodeModel);
        }
        return new WebViewModel(blockId, nodeModel);
    }
    // ... 其他视图类型
}
```

### 4. 添加 API 方法

在 `emain/preload.ts` 中添加新的 API 方法：

```typescript
// 添加 WebView API
webViewCreate: (options: any) => ipcRenderer.invoke("webview:create", options),
webViewNavigate: (blockId: string, url: string) => ipcRenderer.invoke("webview:navigate", blockId, url),
webViewGoBack: (blockId: string) => ipcRenderer.invoke("webview:go-back", blockId),
webViewGoForward: (blockId: string) => ipcRenderer.invoke("webview:go-forward", blockId),
webViewReload: (blockId: string) => ipcRenderer.invoke("webview:reload", blockId),
webViewStop: (blockId: string) => ipcRenderer.invoke("webview:stop", blockId),
webViewFindInPage: (blockId: string, text: string, options?: any) => 
    ipcRenderer.invoke("webview:find-in-page", blockId, text, options),
webViewStopFindInPage: (blockId: string, action: string) => 
    ipcRenderer.invoke("webview:stop-find-in-page", blockId, action),
webViewSetZoomFactor: (blockId: string, factor: number) => 
    ipcRenderer.invoke("webview:set-zoom-factor", blockId, factor),
webViewToggleDevTools: (blockId: string) => 
    ipcRenderer.invoke("webview:toggle-dev-tools", blockId),
webViewSetAudioMuted: (blockId: string, muted: boolean) => 
    ipcRenderer.invoke("webview:set-audio-muted", blockId, muted),
webViewIsAudioMuted: (blockId: string) => 
    ipcRenderer.invoke("webview:is-audio-muted", blockId),
webViewGetState: (blockId: string) => 
    ipcRenderer.invoke("webview:get-state", blockId),
webViewDestroy: (blockId: string) => 
    ipcRenderer.invoke("webview:destroy", blockId),

// 事件监听器
onWebViewEvent: (blockId: string, event: string, callback: (data: any) => void) => {
    const channel = `webview:${blockId}:${event}`;
    ipcRenderer.on(channel, (_, data) => callback(data));
    return () => ipcRenderer.removeAllListeners(channel);
},
```

## 测试场景

### 基础功能测试

1. **创建 Web 视图**
   - 启用功能开关：设置 > 实验性功能 > 使用 WebContentsView
   - 创建新的 Web block
   - 验证页面正常加载

2. **导航测试**
   - 测试 URL 输入和导航
   - 测试后退/前进按钮
   - 测试刷新功能
   - 测试主页按钮

3. **搜索功能**
   - 按 Cmd/Ctrl + F 打开搜索
   - 输入搜索词，验证高亮显示
   - 测试上一个/下一个按钮

4. **缩放控制**
   - 右键菜单 > 设置缩放因子
   - 测试不同缩放级别
   - 验证缩放状态保持

5. **媒体控制**
   - 访问包含音频/视频的网站
   - 验证播放状态检测
   - 测试静音控制

6. **开发者工具**
   - 右键菜单 > 切换开发者工具
   - 验证 DevTools 正常打开

### 性能测试

1. **内存使用**
   - 对比新旧实现的内存占用
   - 测试多个 Web 视图的内存使用

2. **加载速度**
   - 测试页面加载时间
   - 测试导航响应速度

3. **CPU 使用**
   - 监控空闲和活动状态的 CPU 使用

### 兼容性测试

1. **网站兼容性**
   - 测试常见网站（Google、GitHub、YouTube 等）
   - 测试 Web 应用（Gmail、Google Docs 等）
   - 测试本地开发服务器

2. **功能兼容性**
   - 验证所有原有功能正常工作
   - 测试书签功能
   - 测试自定义名称

## 调试方法

### 1. 主进程日志

在 `emain/emain-webview-manager.ts` 中添加日志：

```typescript
console.log("[WebViewManager]", "Creating WebView:", options);
```

### 2. 渲染进程日志

在 `webview-improved.tsx` 中添加日志：

```typescript
console.log("[ImprovedWebView]", "State updated:", state);
```

### 3. 使用 Chrome DevTools

1. 启动应用时添加调试参数：
   ```bash
   ELECTRON_ENABLE_LOGGING=1 task dev
   ```

2. 在主进程中启用远程调试：
   ```bash
   --remote-debugging-port=9222
   ```

### 4. 监控 IPC 通信

在 preload 脚本中添加 IPC 监控：

```typescript
const originalInvoke = ipcRenderer.invoke;
ipcRenderer.invoke = async (channel, ...args) => {
    console.log("[IPC] Invoke:", channel, args);
    const result = await originalInvoke.call(ipcRenderer, channel, ...args);
    console.log("[IPC] Result:", result);
    return result;
};
```

## 已知问题和解决方案

### 1. WebContentsView 定位问题

如果 WebContentsView 没有正确显示，需要在标签切换时更新边界：

```typescript
// 在 WaveTabView 中
webView.setBounds({ x: 0, y: 0, width: bounds.width, height: bounds.height });
```

### 2. 焦点管理

WebContentsView 的焦点需要手动管理：

```typescript
// 获得焦点
webView.webContents.focus();

// 失去焦点
parentWindow.webContents.focus();
```

### 3. 事件传递

某些事件可能需要手动传递：

```typescript
// 键盘事件传递
parentWindow.webContents.on('before-input-event', (event, input) => {
    if (webView.webContents.isFocused()) {
        webView.webContents.sendInputEvent(input);
    }
});
```

## 回滚方案

如果测试发现问题，可以通过以下步骤回滚：

1. 关闭功能开关
2. 重启应用
3. 所有 Web 视图将使用原有实现

## 反馈收集

请在测试过程中记录：

1. 功能是否正常工作
2. 性能表现（更好/相同/更差）
3. 遇到的任何问题
4. 改进建议

可以通过以下方式提交反馈：
- GitHub Issues
- 内部测试频道
- 直接反馈给开发团队