// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { WebContentsView, ipcMain, session, BrowserWindow } from "electron";
import path from "path";
import { getElectronAppBasePath } from "./platform";
import { fireAndForget } from "@/util/util";

interface WebViewState {
    url: string;
    title: string;
    canGoBack: boolean;
    canGoForward: boolean;
    isLoading: boolean;
    zoomFactor: number;
}

interface WebViewOptions {
    blockId: string;
    initialUrl?: string;
    partition?: string;
    zoomFactor?: number;
}

/**
 * 管理 WebContentsView 实例，提供类似原生浏览器的功能
 */
export class WebViewManager {
    private webViews: Map<string, WebContentsView> = new Map();
    private webViewStates: Map<string, WebViewState> = new Map();
    private parentWindows: Map<string, BrowserWindow> = new Map();

    /**
     * 创建新的 WebContentsView
     */
    createWebView(options: WebViewOptions, parentWindow: BrowserWindow): WebContentsView {
        const { blockId, initialUrl, partition, zoomFactor = 1 } = options;

        // 创建 WebContentsView
        const webView = new WebContentsView({
            webPreferences: {
                contextIsolation: true,
                nodeIntegration: false,
                sandbox: true,
                webSecurity: true,
                partition: partition,
                // 自定义预加载脚本，提供与 webview 标签类似的功能
                preload: path.join(getElectronAppBasePath(), "preload", "webview-preload.js"),
                // 启用 webview 功能所需的特性
                plugins: true,
                javascript: true,
                webviewTag: false, // 不再需要 webview 标签
                images: true,
                // 安全相关
                allowRunningInsecureContent: false,
                experimentalFeatures: false,
            }
        });

        // 保存引用
        this.webViews.set(blockId, webView);
        this.parentWindows.set(blockId, parentWindow);

        // 初始化状态
        this.webViewStates.set(blockId, {
            url: initialUrl || "",
            title: "",
            canGoBack: false,
            canGoForward: false,
            isLoading: false,
            zoomFactor: zoomFactor
        });

        // 设置事件处理器
        this.setupEventHandlers(webView, blockId);

        // 配置 WebContents
        this.configureWebContents(webView, blockId);

        // 加载初始 URL
        if (initialUrl) {
            webView.webContents.loadURL(initialUrl);
        }

        return webView;
    }

    /**
     * 设置事件处理器
     */
    private setupEventHandlers(webView: WebContentsView, blockId: string) {
        const webContents = webView.webContents;

        // 导航事件
        webContents.on("will-navigate", (event, url) => {
            this.sendToRenderer(blockId, "webview:will-navigate", { url });
        });

        webContents.on("did-navigate", (event, url) => {
            this.updateState(blockId, { url });
            this.sendToRenderer(blockId, "webview:did-navigate", { url });
        });

        webContents.on("did-navigate-in-page", (event, url, isMainFrame) => {
            if (isMainFrame) {
                this.updateState(blockId, { url });
                this.sendToRenderer(blockId, "webview:did-navigate-in-page", { url });
            }
        });

        // 加载事件
        webContents.on("did-start-loading", () => {
            this.updateState(blockId, { isLoading: true });
            this.sendToRenderer(blockId, "webview:did-start-loading", {});
        });

        webContents.on("did-stop-loading", () => {
            this.updateState(blockId, { 
                isLoading: false,
                canGoBack: webContents.canGoBack(),
                canGoForward: webContents.canGoForward()
            });
            this.sendToRenderer(blockId, "webview:did-stop-loading", {});
        });

        webContents.on("did-fail-load", (event, errorCode, errorDescription, validatedURL) => {
            if (errorCode !== -3) { // 忽略 ERR_ABORTED
                this.sendToRenderer(blockId, "webview:did-fail-load", {
                    errorCode,
                    errorDescription,
                    validatedURL
                });
            }
        });

        // 页面元数据
        webContents.on("page-title-updated", (event, title) => {
            this.updateState(blockId, { title });
            this.sendToRenderer(blockId, "webview:page-title-updated", { title });
        });

        webContents.on("page-favicon-updated", (event, favicons) => {
            this.sendToRenderer(blockId, "webview:page-favicon-updated", { favicons });
        });

        // 新窗口处理
        webContents.setWindowOpenHandler(({ url, frameName, features }) => {
            // 发送到渲染进程处理
            this.sendToRenderer(blockId, "webview:new-window", { url, frameName, features });
            return { action: "deny" }; // 默认阻止，由渲染进程决定如何处理
        });

        // 媒体事件
        webContents.on("media-started-playing", () => {
            this.sendToRenderer(blockId, "webview:media-started-playing", {});
        });

        webContents.on("media-paused", () => {
            this.sendToRenderer(blockId, "webview:media-paused", {});
        });

        // 搜索事件
        webContents.on("found-in-page", (event, result) => {
            this.sendToRenderer(blockId, "webview:found-in-page", result);
        });

        // 上下文菜单
        webContents.on("context-menu", (event, params) => {
            this.sendToRenderer(blockId, "webview:context-menu", params);
        });
    }

    /**
     * 配置 WebContents
     */
    private configureWebContents(webView: WebContentsView, blockId: string) {
        const webContents = webView.webContents;
        const state = this.webViewStates.get(blockId);

        // 设置缩放
        if (state?.zoomFactor) {
            webContents.setZoomFactor(state.zoomFactor);
        }

        // 权限处理
        webContents.session.setPermissionRequestHandler((webContents, permission, callback) => {
            // 发送到渲染进程，让用户决定
            this.sendToRenderer(blockId, "webview:permission-request", { permission });
            
            // 默认拒绝，等待用户响应
            callback(false);
        });

        // 下载处理
        webContents.session.on("will-download", (event, item, webContents) => {
            this.sendToRenderer(blockId, "webview:will-download", {
                url: item.getURL(),
                filename: item.getFilename(),
                totalBytes: item.getTotalBytes()
            });
        });

        // 启用开发者工具
        webContents.on("devtools-opened", () => {
            this.sendToRenderer(blockId, "webview:devtools-opened", {});
        });

        webContents.on("devtools-closed", () => {
            this.sendToRenderer(blockId, "webview:devtools-closed", {});
        });
    }

    /**
     * 导航控制
     */
    async navigate(blockId: string, url: string): Promise<void> {
        const webView = this.webViews.get(blockId);
        if (webView) {
            await webView.webContents.loadURL(url);
        }
    }

    goBack(blockId: string): void {
        const webView = this.webViews.get(blockId);
        if (webView?.webContents.canGoBack()) {
            webView.webContents.goBack();
        }
    }

    goForward(blockId: string): void {
        const webView = this.webViews.get(blockId);
        if (webView?.webContents.canGoForward()) {
            webView.webContents.goForward();
        }
    }

    reload(blockId: string): void {
        const webView = this.webViews.get(blockId);
        webView?.webContents.reload();
    }

    stop(blockId: string): void {
        const webView = this.webViews.get(blockId);
        webView?.webContents.stop();
    }

    /**
     * 搜索功能
     */
    findInPage(blockId: string, text: string, options?: Electron.FindInPageOptions): void {
        const webView = this.webViews.get(blockId);
        if (webView) {
            webView.webContents.findInPage(text, options);
        }
    }

    stopFindInPage(blockId: string, action: "clearSelection" | "keepSelection" | "activateSelection"): void {
        const webView = this.webViews.get(blockId);
        if (webView) {
            webView.webContents.stopFindInPage(action);
        }
    }

    /**
     * 缩放控制
     */
    setZoomFactor(blockId: string, factor: number): void {
        const webView = this.webViews.get(blockId);
        if (webView) {
            webView.webContents.setZoomFactor(factor);
            this.updateState(blockId, { zoomFactor: factor });
        }
    }

    getZoomFactor(blockId: string): number {
        const webView = this.webViews.get(blockId);
        return webView?.webContents.getZoomFactor() || 1;
    }

    /**
     * 开发者工具
     */
    toggleDevTools(blockId: string): void {
        const webView = this.webViews.get(blockId);
        if (webView) {
            if (webView.webContents.isDevToolsOpened()) {
                webView.webContents.closeDevTools();
            } else {
                webView.webContents.openDevTools();
            }
        }
    }

    /**
     * 音频控制
     */
    setAudioMuted(blockId: string, muted: boolean): void {
        const webView = this.webViews.get(blockId);
        if (webView) {
            webView.webContents.setAudioMuted(muted);
        }
    }

    isAudioMuted(blockId: string): boolean {
        const webView = this.webViews.get(blockId);
        return webView?.webContents.isAudioMuted() || false;
    }

    /**
     * 执行 JavaScript
     */
    async executeJavaScript(blockId: string, code: string): Promise<any> {
        const webView = this.webViews.get(blockId);
        if (webView) {
            return await webView.webContents.executeJavaScript(code);
        }
    }

    /**
     * 获取状态
     */
    getState(blockId: string): WebViewState | undefined {
        return this.webViewStates.get(blockId);
    }

    /**
     * 更新状态
     */
    private updateState(blockId: string, updates: Partial<WebViewState>): void {
        const currentState = this.webViewStates.get(blockId);
        if (currentState) {
            const newState = { ...currentState, ...updates };
            this.webViewStates.set(blockId, newState);
            this.sendToRenderer(blockId, "webview:state-updated", newState);
        }
    }

    /**
     * 发送消息到渲染进程
     */
    private sendToRenderer(blockId: string, channel: string, data: any): void {
        const parentWindow = this.parentWindows.get(blockId);
        if (parentWindow && !parentWindow.isDestroyed()) {
            parentWindow.webContents.send(`webview:${blockId}:${channel}`, data);
        }
    }

    /**
     * 销毁 WebView
     */
    destroyWebView(blockId: string): void {
        const webView = this.webViews.get(blockId);
        if (webView) {
            // 关闭开发者工具
            if (webView.webContents.isDevToolsOpened()) {
                webView.webContents.closeDevTools();
            }

            // 销毁 WebContents
            webView.webContents.destroy();

            // 清理引用
            this.webViews.delete(blockId);
            this.webViewStates.delete(blockId);
            this.parentWindows.delete(blockId);
        }
    }

    /**
     * 获取 WebView 实例
     */
    getWebView(blockId: string): WebContentsView | undefined {
        return this.webViews.get(blockId);
    }
}

// 单例实例
export const webViewManager = new WebViewManager();

/**
 * 设置 IPC 处理器
 */
export function setupWebViewIPC(): void {
    // 创建 WebView
    ipcMain.handle("webview:create", async (event, options: WebViewOptions) => {
        const window = BrowserWindow.fromWebContents(event.sender);
        if (window) {
            const webView = webViewManager.createWebView(options, window);
            return { success: true, blockId: options.blockId };
        }
        return { success: false, error: "No parent window found" };
    });

    // 导航
    ipcMain.handle("webview:navigate", async (event, blockId: string, url: string) => {
        await webViewManager.navigate(blockId, url);
    });

    ipcMain.handle("webview:go-back", (event, blockId: string) => {
        webViewManager.goBack(blockId);
    });

    ipcMain.handle("webview:go-forward", (event, blockId: string) => {
        webViewManager.goForward(blockId);
    });

    ipcMain.handle("webview:reload", (event, blockId: string) => {
        webViewManager.reload(blockId);
    });

    ipcMain.handle("webview:stop", (event, blockId: string) => {
        webViewManager.stop(blockId);
    });

    // 搜索
    ipcMain.handle("webview:find-in-page", (event, blockId: string, text: string, options?: Electron.FindInPageOptions) => {
        webViewManager.findInPage(blockId, text, options);
    });

    ipcMain.handle("webview:stop-find-in-page", (event, blockId: string, action: "clearSelection" | "keepSelection" | "activateSelection") => {
        webViewManager.stopFindInPage(blockId, action);
    });

    // 缩放
    ipcMain.handle("webview:set-zoom-factor", (event, blockId: string, factor: number) => {
        webViewManager.setZoomFactor(blockId, factor);
    });

    ipcMain.handle("webview:get-zoom-factor", (event, blockId: string) => {
        return webViewManager.getZoomFactor(blockId);
    });

    // 开发者工具
    ipcMain.handle("webview:toggle-dev-tools", (event, blockId: string) => {
        webViewManager.toggleDevTools(blockId);
    });

    // 音频
    ipcMain.handle("webview:set-audio-muted", (event, blockId: string, muted: boolean) => {
        webViewManager.setAudioMuted(blockId, muted);
    });

    ipcMain.handle("webview:is-audio-muted", (event, blockId: string) => {
        return webViewManager.isAudioMuted(blockId);
    });

    // 执行 JavaScript
    ipcMain.handle("webview:execute-javascript", async (event, blockId: string, code: string) => {
        return await webViewManager.executeJavaScript(blockId, code);
    });

    // 获取状态
    ipcMain.handle("webview:get-state", (event, blockId: string) => {
        return webViewManager.getState(blockId);
    });

    // 销毁
    ipcMain.handle("webview:destroy", (event, blockId: string) => {
        webViewManager.destroyWebView(blockId);
    });
}