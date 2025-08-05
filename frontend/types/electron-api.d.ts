// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// WebView API types
interface WebViewOptions {
    blockId: string;
    initialUrl?: string;
    partition?: string;
    zoomFactor?: number;
}

interface WebViewState {
    url: string;
    title: string;
    canGoBack: boolean;
    canGoForward: boolean;
    isLoading: boolean;
    zoomFactor: number;
}

// Extend the Window interface to include our API
declare global {
    interface Window {
        api: {
            // Existing API methods
            getAuthKey: () => string;
            getIsDev: () => boolean;
            getPlatform: () => string;
            getCursorPoint: () => { x: number; y: number };
            getUserName: () => string;
            getHostName: () => string;
            getDataDir: () => string;
            getConfigDir: () => string;
            getAboutModalDetails: () => any;
            getDocsiteUrl: () => string;
            getWebviewPreload: () => string;
            openNewWindow: () => void;
            showContextMenu: (workspaceId: string, menu: any) => void;
            onContextMenuClick: (callback: (id: string) => void) => void;
            downloadFile: (filePath: string) => void;
            openExternal: (url: string) => void;
            getEnv: (varName: string) => string | null;
            onFullScreenChange: (callback: (isFullScreen: boolean) => void) => void;
            onUpdaterStatusChange: (callback: (status: any) => void) => void;
            getUpdaterStatus: () => any;
            getUpdaterChannel: () => string;
            installAppUpdate: () => void;
            onMenuItemAbout: (callback: () => void) => void;
            updateWindowControlsOverlay: (rect: any) => void;
            onReinjectKey: (callback: (waveEvent: any) => void) => void;
            setWebviewFocus: (focused: number) => void;
            registerGlobalWebviewKeys: (keys: any) => void;
            onControlShiftStateUpdate: (callback: (state: any) => void) => void;
            createWorkspace: () => void;
            switchWorkspace: (workspaceId: string) => void;
            deleteWorkspace: (workspaceId: string) => void;
            setActiveTab: (tabId: string) => void;
            createTab: () => void;
            closeTab: (workspaceId: string, tabId: string) => void;
            setWindowInitStatus: (status: any) => void;
            onWaveInit: (callback: (initOpts: any) => void) => void;
            sendLog: (log: any) => void;
            onQuicklook: (filePath: string) => void;
            openNativePath: (filePath: string) => void;
            captureScreenshot: (rect: any) => Promise<any>;
            captureScreenshotSimple: (rect: any) => Promise<any>;
            saveScreenshotToTemp: (base64Data: string, filename: string, requestId?: string, eventScopes?: any[]) => Promise<any>;
            sendScreenshotResponse: (responseEvent: any) => void;
            onScreenshotResponseFromMain: (callback: (responseEvent: any) => void) => void;
            setKeyboardChordMode: () => void;
            
            // New WebView API methods
            webViewCreate: (options: WebViewOptions) => Promise<{ success: boolean; blockId?: string; error?: string }>;
            webViewNavigate: (blockId: string, url: string) => Promise<void>;
            webViewGoBack: (blockId: string) => Promise<void>;
            webViewGoForward: (blockId: string) => Promise<void>;
            webViewReload: (blockId: string) => Promise<void>;
            webViewStop: (blockId: string) => Promise<void>;
            webViewFindInPage: (blockId: string, text: string, options?: Electron.FindInPageOptions) => Promise<void>;
            webViewStopFindInPage: (blockId: string, action: "clearSelection" | "keepSelection" | "activateSelection") => Promise<void>;
            webViewSetZoomFactor: (blockId: string, factor: number) => Promise<void>;
            webViewGetZoomFactor: (blockId: string) => Promise<number>;
            webViewToggleDevTools: (blockId: string) => Promise<void>;
            webViewSetAudioMuted: (blockId: string, muted: boolean) => Promise<void>;
            webViewIsAudioMuted: (blockId: string) => Promise<boolean>;
            webViewExecuteJavaScript: (blockId: string, code: string) => Promise<any>;
            webViewGetState: (blockId: string) => Promise<WebViewState | undefined>;
            webViewDestroy: (blockId: string) => Promise<void>;
            onWebViewEvent: (blockId: string, event: string, callback: (data: any) => void) => () => void;
        };
    }
}

export {};