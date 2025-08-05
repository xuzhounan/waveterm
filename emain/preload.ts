// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { contextBridge, ipcRenderer, Rectangle, WebviewTag } from "electron";

contextBridge.exposeInMainWorld("api", {
    getAuthKey: () => ipcRenderer.sendSync("get-auth-key"),
    getIsDev: () => ipcRenderer.sendSync("get-is-dev"),
    getPlatform: () => ipcRenderer.sendSync("get-platform"),
    getCursorPoint: () => ipcRenderer.sendSync("get-cursor-point"),
    getUserName: () => ipcRenderer.sendSync("get-user-name"),
    getHostName: () => ipcRenderer.sendSync("get-host-name"),
    getDataDir: () => ipcRenderer.sendSync("get-data-dir"),
    getConfigDir: () => ipcRenderer.sendSync("get-config-dir"),
    getAboutModalDetails: () => ipcRenderer.sendSync("get-about-modal-details"),
    getDocsiteUrl: () => ipcRenderer.sendSync("get-docsite-url"),
    getWebviewPreload: () => ipcRenderer.sendSync("get-webview-preload"),
    openNewWindow: () => ipcRenderer.send("open-new-window"),
    showContextMenu: (workspaceId, menu) => ipcRenderer.send("contextmenu-show", workspaceId, menu),
    onContextMenuClick: (callback) => ipcRenderer.on("contextmenu-click", (_event, id) => callback(id)),
    downloadFile: (filePath) => ipcRenderer.send("download", { filePath }),
    openExternal: (url) => {
        if (url && typeof url === "string") {
            ipcRenderer.send("open-external", url);
        } else {
            console.error("Invalid URL passed to openExternal:", url);
        }
    },
    getEnv: (varName) => ipcRenderer.sendSync("get-env", varName),
    onFullScreenChange: (callback) =>
        ipcRenderer.on("fullscreen-change", (_event, isFullScreen) => callback(isFullScreen)),
    onUpdaterStatusChange: (callback) => ipcRenderer.on("app-update-status", (_event, status) => callback(status)),
    getUpdaterStatus: () => ipcRenderer.sendSync("get-app-update-status"),
    getUpdaterChannel: () => ipcRenderer.sendSync("get-updater-channel"),
    installAppUpdate: () => ipcRenderer.send("install-app-update"),
    onMenuItemAbout: (callback) => ipcRenderer.on("menu-item-about", callback),
    updateWindowControlsOverlay: (rect) => ipcRenderer.send("update-window-controls-overlay", rect),
    onReinjectKey: (callback) => ipcRenderer.on("reinject-key", (_event, waveEvent) => callback(waveEvent)),
    setWebviewFocus: (focused: number) => ipcRenderer.send("webview-focus", focused),
    registerGlobalWebviewKeys: (keys) => ipcRenderer.send("register-global-webview-keys", keys),
    onControlShiftStateUpdate: (callback) =>
        ipcRenderer.on("control-shift-state-update", (_event, state) => callback(state)),
    createWorkspace: () => ipcRenderer.send("create-workspace"),
    switchWorkspace: (workspaceId) => ipcRenderer.send("switch-workspace", workspaceId),
    deleteWorkspace: (workspaceId) => ipcRenderer.send("delete-workspace", workspaceId),
    setActiveTab: (tabId) => ipcRenderer.send("set-active-tab", tabId),
    createTab: () => ipcRenderer.send("create-tab"),
    closeTab: (workspaceId, tabId) => ipcRenderer.send("close-tab", workspaceId, tabId),
    setWindowInitStatus: (status) => ipcRenderer.send("set-window-init-status", status),
    onWaveInit: (callback) => ipcRenderer.on("wave-init", (_event, initOpts) => callback(initOpts)),
    sendLog: (log) => ipcRenderer.send("fe-log", log),
    onQuicklook: (filePath: string) => ipcRenderer.send("quicklook", filePath),
    openNativePath: (filePath: string) => ipcRenderer.send("open-native-path", filePath),
    captureScreenshot: (rect: Rectangle) => ipcRenderer.invoke("capture-screenshot", rect),
    captureScreenshotSimple: (rect: Rectangle) => ipcRenderer.invoke("capture-screenshot-simple", rect),
    saveScreenshotToTemp: (base64Data: string, filename: string, requestId?: string, eventScopes?: any[]) => ipcRenderer.invoke("save-screenshot-to-temp", base64Data, filename, requestId, eventScopes),
    sendScreenshotResponse: (responseEvent: any) => ipcRenderer.send("send-screenshot-response", responseEvent),
    onScreenshotResponseFromMain: (callback: (responseEvent: any) => void) => 
        ipcRenderer.on("screenshot-response-from-main", (_event, responseEvent) => callback(responseEvent)),
    setKeyboardChordMode: () => ipcRenderer.send("set-keyboard-chord-mode"),
    
    // WebView API
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
    webViewGetZoomFactor: (blockId: string) => 
        ipcRenderer.invoke("webview:get-zoom-factor", blockId),
    webViewToggleDevTools: (blockId: string) => 
        ipcRenderer.invoke("webview:toggle-dev-tools", blockId),
    webViewSetAudioMuted: (blockId: string, muted: boolean) => 
        ipcRenderer.invoke("webview:set-audio-muted", blockId, muted),
    webViewIsAudioMuted: (blockId: string) => 
        ipcRenderer.invoke("webview:is-audio-muted", blockId),
    webViewExecuteJavaScript: (blockId: string, code: string) => 
        ipcRenderer.invoke("webview:execute-javascript", blockId, code),
    webViewGetState: (blockId: string) => 
        ipcRenderer.invoke("webview:get-state", blockId),
    webViewDestroy: (blockId: string) => 
        ipcRenderer.invoke("webview:destroy", blockId),
    onWebViewEvent: (blockId: string, event: string, callback: (data: any) => void) => {
        const channel = `webview:${blockId}:${event}`;
        ipcRenderer.on(channel, (_, data) => callback(data));
        return () => ipcRenderer.removeAllListeners(channel);
    },
});

// Custom event for "new-window"
ipcRenderer.on("webview-new-window", (e, webContentsId, details) => {
    const event = new CustomEvent("new-window", { detail: details });
    document.getElementById("webview").dispatchEvent(event);
});

ipcRenderer.on("webcontentsid-from-blockid", (e, blockId, responseCh) => {
    const webviewElem: WebviewTag = document.querySelector("div[data-blockid='" + blockId + "'] webview");
    const wcId = webviewElem?.dataset?.webcontentsid;
    ipcRenderer.send(responseCh, wcId);
});
