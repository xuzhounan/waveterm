// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { BlockNodeModel } from "@/app/block/blocktypes";
import { Search, useSearch } from "@/app/element/search";
import { createBlock, getApi, getBlockMetaKeyAtom, getSettingsKeyAtom } from "@/app/store/global";
import { RpcApi } from "@/app/store/wshclientapi";
import { TabRpcClient } from "@/app/store/wshrpcutil";
import { editBlockCustomName } from "@/app/util/blockutil";
import { WOS, globalStore } from "@/store/global";
import { fireAndForget } from "@/util/util";
import { Atom, PrimitiveAtom, atom, useAtomValue, useSetAtom } from "jotai";
import { memo, useCallback, useEffect, useRef, useState } from "react";
import clsx from "clsx";

interface WebViewState {
    url: string;
    title: string;
    canGoBack: boolean;
    canGoForward: boolean;
    isLoading: boolean;
    zoomFactor: number;
}

/**
 * 改进的 WebViewModel，直接使用 WebContentsView 而不是 webview 标签
 */
export class ImprovedWebViewModel implements ViewModel {
    viewType: string;
    blockId: string;
    noPadding?: Atom<boolean>;
    blockAtom: Atom<Block>;
    viewIcon: Atom<string | IconButtonDecl>;
    viewName: Atom<string>;
    viewText: Atom<HeaderElem[]>;
    url: PrimitiveAtom<string>;
    homepageUrl: Atom<string>;
    urlInputFocused: PrimitiveAtom<boolean>;
    isLoading: PrimitiveAtom<boolean>;
    urlWrapperClassName: PrimitiveAtom<string>;
    refreshIcon: PrimitiveAtom<string>;
    urlInputRef: React.RefObject<HTMLInputElement>;
    nodeModel: BlockNodeModel;
    endIconButtons?: Atom<IconButtonDecl[]>;
    mediaPlaying: PrimitiveAtom<boolean>;
    mediaMuted: PrimitiveAtom<boolean>;
    hideNav: Atom<boolean>;
    searchAtoms?: SearchAtoms;
    typeaheadOpen: PrimitiveAtom<boolean>;
    
    // 新增：WebContentsView 状态
    private webViewState: PrimitiveAtom<WebViewState>;
    private isInitialized: PrimitiveAtom<boolean>;

    constructor(blockId: string, nodeModel: BlockNodeModel) {
        this.nodeModel = nodeModel;
        this.viewType = "web";
        this.blockId = blockId;
        this.noPadding = atom(true);
        this.blockAtom = WOS.getWaveObjectAtom<Block>(`block:${blockId}`);
        this.url = atom("");
        
        const defaultUrlAtom = getSettingsKeyAtom("web:defaulturl");
        this.homepageUrl = atom((get) => {
            const defaultUrl = get(defaultUrlAtom);
            const pinnedUrl = get(this.blockAtom).meta.pinnedurl;
            return pinnedUrl ?? defaultUrl;
        });
        
        this.urlWrapperClassName = atom("");
        this.urlInputFocused = atom(false);
        this.isLoading = atom(false);
        this.refreshIcon = atom("rotate-right");
        this.viewIcon = atom("globe");
        this.viewName = atom((get) => {
            const blockData = get(this.blockAtom);
            const customName = blockData?.meta?.name || blockData?.meta?.title;
            if (customName && typeof customName === "string" && customName.trim()) {
                return customName.trim();
            }
            const state = get(this.webViewState);
            return state?.title || "Web";
        });
        
        this.urlInputRef = createRef<HTMLInputElement>();
        this.hideNav = getBlockMetaKeyAtom(blockId, "web:hidenav");
        this.typeaheadOpen = atom(false);
        this.mediaPlaying = atom(false);
        this.mediaMuted = atom(false);
        
        // 初始化 WebContentsView 状态
        this.webViewState = atom<WebViewState>({
            url: "",
            title: "",
            canGoBack: false,
            canGoForward: false,
            isLoading: false,
            zoomFactor: 1
        });
        this.isInitialized = atom(false);
        
        // 初始化 WebContentsView
        this.initializeWebView();
        
        // 构建视图文本（导航栏）
        this.viewText = atom((get) => {
            const homepageUrl = get(this.homepageUrl);
            const metaUrl = get(this.blockAtom)?.meta?.url;
            const state = get(this.webViewState);
            const currUrl = get(this.url) || state.url;
            const urlWrapperClassName = get(this.urlWrapperClassName);
            const refreshIcon = get(this.refreshIcon);
            const mediaPlaying = get(this.mediaPlaying);
            const mediaMuted = get(this.mediaMuted);
            const url = currUrl ?? metaUrl ?? homepageUrl;
            const rtn: HeaderElem[] = [];
            
            if (get(this.hideNav)) {
                return rtn;
            }

            rtn.push({
                elemtype: "iconbutton",
                icon: "chevron-left",
                click: this.handleBack.bind(this),
                disabled: !state.canGoBack,
            });
            rtn.push({
                elemtype: "iconbutton",
                icon: "chevron-right",
                click: this.handleForward.bind(this),
                disabled: !state.canGoForward,
            });
            rtn.push({
                elemtype: "iconbutton",
                icon: "house",
                click: this.handleHome.bind(this),
                disabled: url === homepageUrl,
            });
            
            const divChildren: HeaderElem[] = [];
            divChildren.push({
                elemtype: "input",
                value: url,
                ref: this.urlInputRef,
                className: "url-input",
                onChange: this.handleUrlChange.bind(this),
                onKeyDown: this.handleKeyDown.bind(this),
                onFocus: this.handleFocus.bind(this),
                onBlur: this.handleBlur.bind(this),
            });
            
            if (mediaPlaying) {
                divChildren.push({
                    elemtype: "iconbutton",
                    icon: mediaMuted ? "volume-slash" : "volume",
                    click: this.handleMuteChange.bind(this),
                });
            }
            
            divChildren.push({
                elemtype: "iconbutton",
                icon: refreshIcon,
                click: this.handleRefresh.bind(this),
            });
            
            rtn.push({
                elemtype: "div",
                className: clsx("block-frame-div-url", urlWrapperClassName),
                onMouseOver: this.handleUrlWrapperMouseOver.bind(this),
                onMouseOut: this.handleUrlWrapperMouseOut.bind(this),
                children: divChildren,
            });
            
            return rtn;
        });

        this.endIconButtons = atom((get) => {
            if (get(this.hideNav)) {
                return null;
            }
            const url = get(this.url) || get(this.webViewState).url;
            return [
                {
                    elemtype: "iconbutton",
                    icon: "pencil",
                    title: "Edit Web View Name",
                    click: () => {
                        this.editBlockName();
                    },
                },
                {
                    elemtype: "iconbutton",
                    icon: "arrow-up-right-from-square",
                    title: "Open in External Browser",
                    click: () => {
                        if (url != null && url != "") {
                            return getApi().openExternal(url);
                        }
                    },
                },
            ];
        });
    }

    get viewComponent(): ViewComponent {
        return ImprovedWebView;
    }

    /**
     * 初始化 WebContentsView
     */
    private async initializeWebView() {
        const blockData = globalStore.get(this.blockAtom);
        const defaultUrl = globalStore.get(this.homepageUrl);
        const initialUrl = blockData?.meta?.url || defaultUrl;
        const zoomFactor = blockData?.meta?.["web:zoom"] || 1;
        const partition = blockData?.meta?.["web:partition"];
        
        try {
            // 通过 IPC 在主进程创建 WebContentsView
            const result = await getApi().webViewCreate({
                blockId: this.blockId,
                initialUrl: this.ensureUrlScheme(initialUrl),
                partition,
                zoomFactor
            });
            
            if (result.success) {
                globalStore.set(this.isInitialized, true);
                this.setupEventListeners();
                
                // 获取初始状态
                const state = await getApi().webViewGetState(this.blockId);
                if (state) {
                    globalStore.set(this.webViewState, state);
                    globalStore.set(this.url, state.url);
                }
            }
        } catch (error) {
            console.error("Failed to initialize WebView:", error);
        }
    }

    /**
     * 设置事件监听器
     */
    private setupEventListeners() {
        // 监听来自主进程的事件
        const api = getApi();
        
        // 状态更新
        api.onWebViewEvent(this.blockId, "state-updated", (state: WebViewState) => {
            globalStore.set(this.webViewState, state);
            globalStore.set(this.url, state.url);
            globalStore.set(this.isLoading, state.isLoading);
            globalStore.set(this.refreshIcon, state.isLoading ? "xmark-large" : "rotate-right");
        });
        
        // 导航事件
        api.onWebViewEvent(this.blockId, "did-navigate", ({ url }) => {
            this.handleNavigate(url);
        });
        
        api.onWebViewEvent(this.blockId, "did-navigate-in-page", ({ url }) => {
            this.handleNavigate(url);
        });
        
        // 加载事件
        api.onWebViewEvent(this.blockId, "did-fail-load", ({ errorCode, errorDescription, validatedURL }) => {
            console.error(`Failed to load ${validatedURL}: ${errorDescription}`);
        });
        
        // 新窗口
        api.onWebViewEvent(this.blockId, "new-window", ({ url }) => {
            fireAndForget(() => openLink(url, true));
        });
        
        // 媒体事件
        api.onWebViewEvent(this.blockId, "media-started-playing", () => {
            globalStore.set(this.mediaPlaying, true);
        });
        
        api.onWebViewEvent(this.blockId, "media-paused", () => {
            globalStore.set(this.mediaPlaying, false);
        });
        
        // 搜索结果
        api.onWebViewEvent(this.blockId, "found-in-page", (result) => {
            if (this.searchAtoms) {
                globalStore.set(this.searchAtoms.resultsCount, result.matches || 0);
                globalStore.set(this.searchAtoms.resultsIndex, result.activeMatchOrdinal - 1);
            }
        });
    }

    handleNavigate(url: string) {
        fireAndForget(() => ObjectService.UpdateObjectMeta(WOS.makeORef("block", this.blockId), { url }));
        globalStore.set(this.url, url);
        if (this.searchAtoms) {
            globalStore.set(this.searchAtoms.isOpen, false);
        }
    }

    ensureUrlScheme(url: string, searchTemplate?: string): string {
        if (!url) return "";
        
        if (/^(http|https|file):/.test(url)) {
            return url;
        }
        
        const isLocal = /^(localhost|(\d{1,3}\.){3}\d{1,3})(:\d+)?$/.test(url.split("/")[0]);
        if (isLocal) {
            return `http://${url}`;
        }
        
        const domainRegex = /^[a-z0-9.-]+\.[a-z]{2,}$/i;
        const isDomain = domainRegex.test(url.split("/")[0]);
        if (isDomain) {
            return `https://${url}`;
        }
        
        const template = searchTemplate || globalStore.get(getSettingsKeyAtom("web:defaultsearch"));
        if (!template) {
            return `https://www.google.com/search?q=${encodeURIComponent(url)}`;
        }
        return template.replace("{query}", encodeURIComponent(url));
    }

    async loadUrl(url: string) {
        const finalUrl = this.ensureUrlScheme(url);
        await getApi().webViewNavigate(this.blockId, finalUrl);
    }

    // 导航控制方法
    async handleBack() {
        await getApi().webViewGoBack(this.blockId);
    }

    async handleForward() {
        await getApi().webViewGoForward(this.blockId);
    }

    async handleHome() {
        const homepageUrl = globalStore.get(this.homepageUrl);
        await this.loadUrl(homepageUrl);
    }

    async handleRefresh() {
        const isLoading = globalStore.get(this.isLoading);
        if (isLoading) {
            await getApi().webViewStop(this.blockId);
        } else {
            await getApi().webViewReload(this.blockId);
        }
    }

    handleUrlChange(event: React.ChangeEvent<HTMLInputElement>) {
        globalStore.set(this.url, event.target.value);
    }

    handleKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
        if (event.key === "Enter") {
            const url = globalStore.get(this.url);
            this.loadUrl(url);
            this.urlInputRef.current?.blur();
        } else if (event.key === "Escape") {
            this.urlInputRef.current?.blur();
        }
    }

    handleFocus() {
        globalStore.set(this.urlWrapperClassName, "focused");
        globalStore.set(this.urlInputFocused, true);
        this.urlInputRef.current?.select();
    }

    handleBlur() {
        globalStore.set(this.urlWrapperClassName, "");
        globalStore.set(this.urlInputFocused, false);
    }

    handleUrlWrapperMouseOver() {
        if (!globalStore.get(this.urlInputFocused)) {
            globalStore.set(this.urlWrapperClassName, "hovered");
        }
    }

    handleUrlWrapperMouseOut() {
        if (!globalStore.get(this.urlInputFocused)) {
            globalStore.set(this.urlWrapperClassName, "");
        }
    }

    async handleMuteChange() {
        const isMuted = await getApi().webViewIsAudioMuted(this.blockId);
        await getApi().webViewSetAudioMuted(this.blockId, !isMuted);
        globalStore.set(this.mediaMuted, !isMuted);
    }

    async setZoomFactor(factor: number | null) {
        if (factor != null) {
            factor = Math.max(0.1, Math.min(5, factor));
        }
        await getApi().webViewSetZoomFactor(this.blockId, factor || 1);
        await RpcApi.SetMetaCommand(TabRpcClient, {
            oref: WOS.makeORef("block", this.blockId),
            meta: { "web:zoom": factor },
        });
    }

    getSettingsMenuItems(): ContextMenuItem[] {
        const state = globalStore.get(this.webViewState);
        const curZoom = state?.zoomFactor || 1;
        const model = this;
        
        function makeZoomFactorMenuItem(label: string, factor: number): ContextMenuItem {
            return {
                label: label,
                type: "checkbox",
                click: () => {
                    model.setZoomFactor(factor);
                },
                checked: Math.abs(curZoom - factor) < 0.01,
            };
        }
        
        const zoomSubMenu: ContextMenuItem[] = [
            {
                label: "Reset",
                click: () => {
                    model.setZoomFactor(null);
                },
            },
            makeZoomFactorMenuItem("25%", 0.25),
            makeZoomFactorMenuItem("50%", 0.5),
            makeZoomFactorMenuItem("70%", 0.7),
            makeZoomFactorMenuItem("80%", 0.8),
            makeZoomFactorMenuItem("90%", 0.9),
            makeZoomFactorMenuItem("100%", 1),
            makeZoomFactorMenuItem("110%", 1.1),
            makeZoomFactorMenuItem("120%", 1.2),
            makeZoomFactorMenuItem("130%", 1.3),
            makeZoomFactorMenuItem("150%", 1.5),
            makeZoomFactorMenuItem("175%", 1.75),
            makeZoomFactorMenuItem("200%", 2),
        ];

        const isNavHidden = globalStore.get(this.hideNav);
        
        return [
            {
                label: "Copy URL to Clipboard",
                click: () => {
                    const url = globalStore.get(this.webViewState).url;
                    if (url) {
                        fireAndForget(() => navigator.clipboard.writeText(url));
                    }
                },
            },
            {
                label: "Set Block Homepage",
                click: () => fireAndForget(() => this.setHomepageUrl(globalStore.get(this.webViewState).url, "block")),
            },
            {
                label: "Set Default Homepage",
                click: () => fireAndForget(() => this.setHomepageUrl(globalStore.get(this.webViewState).url, "global")),
            },
            {
                type: "separator",
            },
            {
                label: isNavHidden ? "Un-Hide Navigation" : "Hide Navigation",
                click: () =>
                    fireAndForget(() => {
                        return RpcApi.SetMetaCommand(TabRpcClient, {
                            oref: WOS.makeORef("block", this.blockId),
                            meta: { "web:hidenav": !isNavHidden },
                        });
                    }),
            },
            {
                label: "Set Zoom Factor",
                submenu: zoomSubMenu,
            },
            {
                label: "Toggle DevTools",
                click: () => {
                    fireAndForget(() => getApi().webViewToggleDevTools(this.blockId));
                },
            },
        ];
    }

    async setHomepageUrl(url: string, scope: "global" | "block") {
        if (url != null && url != "") {
            switch (scope) {
                case "block":
                    await RpcApi.SetMetaCommand(TabRpcClient, {
                        oref: WOS.makeORef("block", this.blockId),
                        meta: { pinnedurl: url },
                    });
                    break;
                case "global":
                    await RpcApi.SetMetaCommand(TabRpcClient, {
                        oref: WOS.makeORef("block", this.blockId),
                        meta: { pinnedurl: "" },
                    });
                    await RpcApi.SetConfigCommand(TabRpcClient, { "web:defaulturl": url });
                    break;
            }
        }
    }

    editBlockName() {
        const currentName = globalStore.get(this.blockAtom)?.meta?.name || 
                           globalStore.get(this.blockAtom)?.meta?.title || 
                           "";
        
        editBlockCustomName(this.blockId, "web", currentName);
    }

    giveFocus(): boolean {
        // WebContentsView 的焦点由主进程管理
        return true;
    }

    keyDownHandler(e: WaveKeyboardEvent): boolean {
        if (e.metaKey && e.code === "KeyL") {
            this.urlInputRef?.current?.focus();
            this.urlInputRef?.current?.select();
            return true;
        }
        if (e.metaKey && e.code === "KeyR") {
            this.handleRefresh();
            return true;
        }
        if (e.metaKey && e.code === "ArrowLeft") {
            this.handleBack();
            return true;
        }
        if (e.metaKey && e.code === "ArrowRight") {
            this.handleForward();
            return true;
        }
        return false;
    }

    // 清理资源
    async dispose() {
        await getApi().webViewDestroy(this.blockId);
    }
}

/**
 * 改进的 WebView 组件，使用 WebContentsView 而不是 webview 标签
 */
const ImprovedWebView = memo(({ model, blockRef }: { model: ImprovedWebViewModel; blockRef: React.RefObject<HTMLDivElement> }) => {
    const isInitialized = useAtomValue(model.isInitialized);
    const webViewState = useAtomValue(model.webViewState);
    const isLoading = useAtomValue(model.isLoading);
    
    // 搜索功能
    const searchProps = useSearch({ 
        viewModel: model,
        onSearch: useCallback((searchText: string) => {
            if (searchText) {
                getApi().webViewFindInPage(model.blockId, searchText, { findNext: true });
            } else {
                getApi().webViewStopFindInPage(model.blockId, "clearSelection");
            }
        }, [model.blockId]),
        onNext: useCallback(() => {
            const searchValue = globalStore.get(searchProps.searchValue);
            getApi().webViewFindInPage(model.blockId, searchValue, { forward: true });
        }, [model.blockId, searchProps.searchValue]),
        onPrev: useCallback(() => {
            const searchValue = globalStore.get(searchProps.searchValue);
            getApi().webViewFindInPage(model.blockId, searchValue, { forward: false });
        }, [model.blockId, searchProps.searchValue])
    });
    
    // 组件卸载时清理
    useEffect(() => {
        return () => {
            model.dispose();
        };
    }, []);

    return (
        <div className="webview-container">
            {!isInitialized && (
                <div className="webview-loading">
                    <div className="loading-spinner">Loading...</div>
                </div>
            )}
            {isInitialized && webViewState.url && isLoading && (
                <div className="webview-progress">
                    <div className="progress-bar" />
                </div>
            )}
            <Search {...searchProps} />
        </div>
    );
});

export { ImprovedWebView };