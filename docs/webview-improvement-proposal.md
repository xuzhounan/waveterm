# Wave Terminal Web 视图改进方案

## 概述

本文档提出将 Wave Terminal 当前的 webview 标签实现升级为直接使用 WebContentsView 的方案，以提供更接近原生浏览器的体验。

## 当前架构分析

### 现有实现
- **双层结构**：WaveTabView (WebContentsView) → webview 标签
- **webview 标签限制**：
  - 性能开销大（每个 webview 是独立进程）
  - 某些浏览器 API 受限
  - IPC 通信复杂
  - Electron 官方已不推荐使用

### 需要保留的核心功能
1. **导航控制**：后退、前进、刷新、主页
2. **URL 管理**：智能 URL 解析、搜索引擎集成
3. **安全隔离**：进程隔离、上下文隔离
4. **缩放控制**：页面缩放（0.1x - 5x）
5. **搜索功能**：页内搜索
6. **开发者工具**：DevTools 集成
7. **媒体控制**：音频播放检测和控制
8. **自定义预加载脚本**

## 改进方案

### 架构调整

```
当前: WaveTabView (WebContentsView) → webview 标签 → 网页内容
改进: WaveTabView (WebContentsView) → 网页内容（直接加载）
```

### 实现步骤

#### 1. 创建新的 WebViewModel

```typescript
// frontend/app/view/webview/webview-improved.tsx
export class ImprovedWebViewModel implements ViewModel {
    webContentsView: WebContentsView;
    
    constructor(blockId: string, nodeModel: BlockNodeModel) {
        // 直接使用 WebContentsView 而不是 webview 标签
        this.initializeWebContents();
    }
    
    private initializeWebContents() {
        // WebContentsView 在主进程中创建和管理
        // 前端通过 IPC 与主进程通信控制
    }
}
```

#### 2. 主进程 WebContentsView 管理

```typescript
// emain/emain-webview.ts
export class WebViewManager {
    private webViews: Map<string, WebContentsView> = new Map();
    
    createWebView(blockId: string, options: WebViewOptions): WebContentsView {
        const webView = new WebContentsView({
            webPreferences: {
                contextIsolation: true,
                nodeIntegration: false,
                sandbox: true,
                webSecurity: true,
                // 支持自定义预加载脚本
                preload: path.join(__dirname, 'preload-web.js')
            }
        });
        
        this.webViews.set(blockId, webView);
        this.setupWebViewHandlers(webView, blockId);
        
        return webView;
    }
    
    private setupWebViewHandlers(webView: WebContentsView, blockId: string) {
        const webContents = webView.webContents;
        
        // 导航事件
        webContents.on('will-navigate', (event, url) => {
            // 处理导航
        });
        
        // 新窗口处理
        webContents.setWindowOpenHandler(({ url }) => {
            // 处理新窗口请求
            return { action: 'deny' };
        });
        
        // 权限请求
        webContents.session.setPermissionRequestHandler((webContents, permission, callback) => {
            // 处理权限请求
        });
    }
}
```

#### 3. IPC 通信层

```typescript
// emain/emain-webview-ipc.ts
export function setupWebViewIPC() {
    // 导航控制
    ipcMain.handle('webview:navigate', async (event, blockId: string, url: string) => {
        const webView = webViewManager.getWebView(blockId);
        await webView.webContents.loadURL(url);
    });
    
    // 搜索功能
    ipcMain.handle('webview:findInPage', async (event, blockId: string, text: string) => {
        const webView = webViewManager.getWebView(blockId);
        webView.webContents.findInPage(text);
    });
    
    // 缩放控制
    ipcMain.handle('webview:setZoomFactor', async (event, blockId: string, factor: number) => {
        const webView = webViewManager.getWebView(blockId);
        webView.webContents.setZoomFactor(factor);
    });
}
```

#### 4. 前端集成

```typescript
// frontend/app/view/webview/webview-renderer.tsx
const ImprovedWebView = memo(({ model }: WebViewProps) => {
    const [isLoading, setIsLoading] = useState(false);
    const [canGoBack, setCanGoBack] = useState(false);
    const [canGoForward, setCanGoForward] = useState(false);
    
    // 通过 IPC 控制 WebContentsView
    const navigate = useCallback((url: string) => {
        getApi().webViewNavigate(model.blockId, url);
    }, [model.blockId]);
    
    const goBack = useCallback(() => {
        getApi().webViewGoBack(model.blockId);
    }, [model.blockId]);
    
    // 监听状态更新
    useEffect(() => {
        const unsubscribe = getApi().onWebViewStateChange(model.blockId, (state) => {
            setIsLoading(state.isLoading);
            setCanGoBack(state.canGoBack);
            setCanGoForward(state.canGoForward);
        });
        
        return unsubscribe;
    }, [model.blockId]);
    
    // 渲染控制界面（不再包含 webview 标签）
    return (
        <div className="web-view-container">
            <NavigationBar 
                onNavigate={navigate}
                onBack={goBack}
                canGoBack={canGoBack}
                // ...
            />
            {/* WebContentsView 在主进程中渲染 */}
        </div>
    );
});
```

### 优势

1. **性能提升**：移除 webview 标签层，减少进程开销
2. **功能完整**：直接访问所有 Chromium API
3. **更好的控制**：主进程直接管理 WebContents
4. **简化通信**：减少 IPC 层级

### 迁移计划

1. **第一阶段**：创建新的 ImprovedWebView 组件，与现有 WebView 并存
2. **第二阶段**：添加功能开关，允许用户选择使用新实现
3. **第三阶段**：收集反馈，修复问题
4. **第四阶段**：完全迁移到新实现

### 风险与挑战

1. **架构变更**：需要重构前端和主进程的通信
2. **功能兼容**：确保所有现有功能正常工作
3. **安全考虑**：需要仔细处理权限和安全策略

## 总结

通过直接使用 WebContentsView，Wave Terminal 可以提供更接近原生浏览器的体验，同时保持良好的性能和安全性。这种改进将使 Wave Terminal 的 Web 视图功能更加强大和灵活。