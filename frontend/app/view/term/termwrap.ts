// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { getFileSubject } from "@/app/store/wps";
import { sendWSCommand } from "@/app/store/ws";
import { RpcApi } from "@/app/store/wshclientapi";
import { TabRpcClient } from "@/app/store/wshrpcutil";
import { WOS, atoms, fetchWaveFile, getSettingsKeyAtom, globalStore, openLink } from "@/store/global";
import * as services from "@/store/services";
import { PLATFORM, PlatformMacOS } from "@/util/platformutil";
import { base64ToArray, fireAndForget } from "@/util/util";
import { SearchAddon } from "@xterm/addon-search";
import { SerializeAddon } from "@xterm/addon-serialize";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { WebglAddon } from "@xterm/addon-webgl";
import * as TermTypes from "@xterm/xterm";
import { Terminal } from "@xterm/xterm";
import debug from "debug";
import { debounce } from "throttle-debounce";
import { FitAddon } from "./fitaddon";
import { EnhancedIMEHandler } from "./enhanced-ime";
import { EnhancedKeybindingsHandler } from "./enhanced-keybindings";

const dlog = debug("wave:termwrap");

const TermFileName = "term";
const TermCacheFileName = "cache:term:full";
const MinDataProcessedForCache = 100 * 1024;

// detect webgl support
function detectWebGLSupport(): boolean {
    try {
        const canvas = document.createElement("canvas");
        const ctx = canvas.getContext("webgl");
        return !!ctx;
    } catch (e) {
        return false;
    }
}

const WebGLSupported = detectWebGLSupport();
let loggedWebGL = false;

type TermWrapOptions = {
    keydownHandler?: (e: KeyboardEvent) => boolean;
    useWebGl?: boolean;
    sendDataHandler?: (data: string) => void;
};

function handleOscWaveCommand(data: string, blockId: string, loaded: boolean): boolean {
    if (!loaded) {
        return false;
    }
    if (!data || data.length === 0) {
        console.log("Invalid Wave OSC command received (empty)");
        return false;
    }

    // Expected formats:
    // "setmeta;{JSONDATA}"
    // "setmeta;[wave-id];{JSONDATA}"
    const parts = data.split(";");
    if (parts[0] !== "setmeta") {
        console.log("Invalid Wave OSC command received (bad command)", data);
        return false;
    }
    let jsonPayload: string;
    let waveId: string | undefined;
    if (parts.length === 2) {
        jsonPayload = parts[1];
    } else if (parts.length >= 3) {
        waveId = parts[1];
        jsonPayload = parts.slice(2).join(";");
    } else {
        console.log("Invalid Wave OSC command received (1 part)", data);
        return false;
    }

    let meta: any;
    try {
        meta = JSON.parse(jsonPayload);
    } catch (e) {
        console.error("Invalid JSON in Wave OSC command:", e);
        return false;
    }

    if (waveId) {
        // Resolve the wave id to an ORef using our ResolveIdsCommand.
        fireAndForget(() => {
            return RpcApi.ResolveIdsCommand(TabRpcClient, { blockid: blockId, ids: [waveId] })
                .then((response: { resolvedids: { [key: string]: any } }) => {
                    const oref = response.resolvedids[waveId];
                    if (!oref) {
                        console.error("Failed to resolve wave id:", waveId);
                        return;
                    }
                    services.ObjectService.UpdateObjectMeta(oref, meta);
                })
                .catch((err: any) => {
                    console.error("Error resolving wave id", waveId, err);
                });
        });
    } else {
        // No wave id provided; update using the current block id.
        fireAndForget(() => {
            return services.ObjectService.UpdateObjectMeta(WOS.makeORef("block", blockId), meta);
        });
    }
    return true;
}

function handleOsc7Command(data: string, blockId: string, loaded: boolean): boolean {
    if (!loaded) {
        return false;
    }
    if (data == null || data.length == 0) {
        console.log("Invalid OSC 7 command received (empty)");
        return false;
    }
    if (data.startsWith("file://")) {
        data = data.substring(7);
        const nextSlashIdx = data.indexOf("/");
        if (nextSlashIdx == -1) {
            console.log("Invalid OSC 7 command received (bad path)", data);
            return false;
        }
        data = data.substring(nextSlashIdx);
    }
    setTimeout(() => {
        fireAndForget(() =>
            services.ObjectService.UpdateObjectMeta(WOS.makeORef("block", blockId), {
                "cmd:cwd": data,
            })
        );
    }, 0);
    return true;
}

export class TermWrap {
    blockId: string;
    ptyOffset: number;
    dataBytesProcessed: number;
    terminal: Terminal;
    connectElem: HTMLDivElement;
    fitAddon: FitAddon;
    searchAddon: SearchAddon;
    serializeAddon: SerializeAddon;
    mainFileSubject: SubjectWithRef<WSFileEventData>;
    loaded: boolean;
    heldData: Uint8Array[];
    handleResize_debounced: () => void;
    hasResized: boolean;
    multiInputCallback: (data: string) => void;
    sendDataHandler: (data: string) => void;
    onSearchResultsDidChange?: (result: { resultIndex: number; resultCount: number }) => void;
    private toDispose: TermTypes.IDisposable[] = [];
    pasteActive: boolean = false;

    // 增强功能
    private enhancedIME: EnhancedIMEHandler | null = null;
    private enhancedKeybindings: EnhancedKeybindingsHandler | null = null;
    
    // 输入过滤器 - 防止 caps 键切换时的重复输入
    private lastInputData = "";
    private lastInputTime = 0;
    private inputFilterEnabled = true;

    constructor(
        blockId: string,
        connectElem: HTMLDivElement,
        options: TermTypes.ITerminalOptions & TermTypes.ITerminalInitOnlyOptions,
        waveOptions: TermWrapOptions
    ) {
        this.loaded = false;
        this.blockId = blockId;
        this.sendDataHandler = waveOptions.sendDataHandler;
        this.ptyOffset = 0;
        this.dataBytesProcessed = 0;
        this.hasResized = false;
        this.terminal = new Terminal(options);
        this.fitAddon = new FitAddon();
        this.fitAddon.noScrollbar = PLATFORM === PlatformMacOS;
        this.serializeAddon = new SerializeAddon();
        this.searchAddon = new SearchAddon();
        this.terminal.loadAddon(this.searchAddon);
        this.terminal.loadAddon(this.fitAddon);
        this.terminal.loadAddon(this.serializeAddon);
        this.terminal.loadAddon(
            new WebLinksAddon((e, uri) => {
                e.preventDefault();
                switch (PLATFORM) {
                    case PlatformMacOS:
                        if (e.metaKey) {
                            fireAndForget(() => openLink(uri));
                        }
                        break;
                    default:
                        if (e.ctrlKey) {
                            fireAndForget(() => openLink(uri));
                        }
                        break;
                }
            })
        );
        if (WebGLSupported && waveOptions.useWebGl) {
            const webglAddon = new WebglAddon();
            this.toDispose.push(
                webglAddon.onContextLoss(() => {
                    webglAddon.dispose();
                })
            );
            this.terminal.loadAddon(webglAddon);
            if (!loggedWebGL) {
                console.log("loaded webgl!");
                loggedWebGL = true;
            }
        }
        // Register OSC 9283 handler
        this.terminal.parser.registerOscHandler(9283, (data: string) => {
            return handleOscWaveCommand(data, this.blockId, this.loaded);
        });
        this.terminal.parser.registerOscHandler(7, (data: string) => {
            return handleOsc7Command(data, this.blockId, this.loaded);
        });
        // 创建增强的键盘事件处理器
        const enhancedKeyHandler = (event: KeyboardEvent): boolean => {
            // 首先让增强功能处理
            if (this.handleEnhancedKeydown && this.handleEnhancedKeydown(event)) {
                return false; // 阻止进一步处理
            }
            
            // 然后让原始处理器处理
            if (waveOptions.keydownHandler) {
                return waveOptions.keydownHandler(event);
            }
            
            return true; // 允许默认处理
        };
        
        this.terminal.attachCustomKeyEventHandler(enhancedKeyHandler);
        this.connectElem = connectElem;
        this.mainFileSubject = null;
        this.heldData = [];
        this.handleResize_debounced = debounce(50, this.handleResize.bind(this));
        this.terminal.open(this.connectElem);
        this.handleResize();
        
        // 初始化增强功能
        console.log('🔧 Initializing terminal enhancements for blockId:', this.blockId);
        this.initializeEnhancements();
        let pasteEventHandler = () => {
            this.pasteActive = true;
            setTimeout(() => {
                this.pasteActive = false;
            }, 30);
        };
        pasteEventHandler = pasteEventHandler.bind(this);
        this.connectElem.addEventListener("paste", pasteEventHandler, true);
        this.toDispose.push({
            dispose: () => {
                this.connectElem.removeEventListener("paste", pasteEventHandler, true);
            },
        });
    }

    async initTerminal() {
        const copyOnSelectAtom = getSettingsKeyAtom("term:copyonselect");
        this.toDispose.push(this.terminal.onData(this.handleTermData.bind(this)));
        this.toDispose.push(this.terminal.onKey(this.onKeyHandler.bind(this)));
        this.toDispose.push(
            this.terminal.onSelectionChange(
                debounce(50, () => {
                    if (!globalStore.get(copyOnSelectAtom)) {
                        return;
                    }
                    const selectedText = this.terminal.getSelection();
                    if (selectedText.length > 0) {
                        navigator.clipboard.writeText(selectedText);
                    }
                })
            )
        );
        if (this.onSearchResultsDidChange != null) {
            this.toDispose.push(this.searchAddon.onDidChangeResults(this.onSearchResultsDidChange.bind(this)));
        }
        this.mainFileSubject = getFileSubject(this.blockId, TermFileName);
        this.mainFileSubject.subscribe(this.handleNewFileSubjectData.bind(this));
        try {
            await this.loadInitialTerminalData();
        } finally {
            this.loaded = true;
        }
        this.runProcessIdleTimeout();
    }

    dispose() {
        this.terminal.dispose();
        this.toDispose.forEach((d) => {
            try {
                d.dispose();
            } catch (_) {}
        });
        this.mainFileSubject.release();
    }

    handleTermData(data: string) {
        if (!this.loaded) {
            return;
        }

        // 应用输入过滤，防止 caps 键切换时的重复输入
        if (this.inputFilterEnabled && this.shouldFilterInput(data)) {
            console.log('🎯 Filtered duplicate input:', data, 'blockId:', this.blockId);
            return;
        }
        
        console.log('🎯 Processing terminal input:', data, 'blockId:', this.blockId);

        if (this.pasteActive) {
            this.pasteActive = false;
            if (this.multiInputCallback) {
                this.multiInputCallback(data);
            }
        }
        this.sendDataHandler?.(data);
    }

    onKeyHandler(data: { key: string; domEvent: KeyboardEvent }) {
        if (this.multiInputCallback) {
            this.multiInputCallback(data.key);
        }
    }

    addFocusListener(focusFn: () => void) {
        this.terminal.textarea.addEventListener("focus", focusFn);
    }

    handleNewFileSubjectData(msg: WSFileEventData) {
        if (msg.fileop == "truncate") {
            this.terminal.clear();
            this.heldData = [];
        } else if (msg.fileop == "append") {
            const decodedData = base64ToArray(msg.data64);
            if (this.loaded) {
                this.doTerminalWrite(decodedData, null);
            } else {
                this.heldData.push(decodedData);
            }
        } else {
            console.log("bad fileop for terminal", msg);
            return;
        }
    }

    doTerminalWrite(data: string | Uint8Array, setPtyOffset?: number): Promise<void> {
        let resolve: () => void = null;
        let prtn = new Promise<void>((presolve, _) => {
            resolve = presolve;
        });
        this.terminal.write(data, () => {
            if (setPtyOffset != null) {
                this.ptyOffset = setPtyOffset;
            } else {
                this.ptyOffset += data.length;
                this.dataBytesProcessed += data.length;
            }
            resolve();
        });
        return prtn;
    }

    async loadInitialTerminalData(): Promise<void> {
        let startTs = Date.now();
        const { data: cacheData, fileInfo: cacheFile } = await fetchWaveFile(this.blockId, TermCacheFileName);
        let ptyOffset = 0;
        if (cacheFile != null) {
            ptyOffset = cacheFile.meta["ptyoffset"] ?? 0;
            if (cacheData.byteLength > 0) {
                const curTermSize: TermSize = { rows: this.terminal.rows, cols: this.terminal.cols };
                const fileTermSize: TermSize = cacheFile.meta["termsize"];
                let didResize = false;
                if (
                    fileTermSize != null &&
                    (fileTermSize.rows != curTermSize.rows || fileTermSize.cols != curTermSize.cols)
                ) {
                    console.log("terminal restore size mismatch, temp resize", fileTermSize, curTermSize);
                    this.terminal.resize(fileTermSize.cols, fileTermSize.rows);
                    didResize = true;
                }
                this.doTerminalWrite(cacheData, ptyOffset);
                if (didResize) {
                    this.terminal.resize(curTermSize.cols, curTermSize.rows);
                }
            }
        }
        const { data: mainData, fileInfo: mainFile } = await fetchWaveFile(this.blockId, TermFileName, ptyOffset);
        console.log(
            `terminal loaded cachefile:${cacheData?.byteLength ?? 0} main:${mainData?.byteLength ?? 0} bytes, ${Date.now() - startTs}ms`
        );
        if (mainFile != null) {
            await this.doTerminalWrite(mainData, null);
        }
    }

    async resyncController(reason: string) {
        dlog("resync controller", this.blockId, reason);
        const tabId = globalStore.get(atoms.staticTabId);
        const rtOpts: RuntimeOpts = { termsize: { rows: this.terminal.rows, cols: this.terminal.cols } };
        try {
            await RpcApi.ControllerResyncCommand(TabRpcClient, {
                tabid: tabId,
                blockid: this.blockId,
                rtopts: rtOpts,
            });
        } catch (e) {
            console.log(`error controller resync (${reason})`, this.blockId, e);
        }
    }

    handleResize() {
        const oldRows = this.terminal.rows;
        const oldCols = this.terminal.cols;
        this.fitAddon.fit();
        if (oldRows !== this.terminal.rows || oldCols !== this.terminal.cols) {
            const termSize: TermSize = { rows: this.terminal.rows, cols: this.terminal.cols };
            const wsCommand: SetBlockTermSizeWSCommand = {
                wscommand: "setblocktermsize",
                blockid: this.blockId,
                termsize: termSize,
            };
            sendWSCommand(wsCommand);
        }
        dlog("resize", `${this.terminal.rows}x${this.terminal.cols}`, `${oldRows}x${oldCols}`, this.hasResized);
        if (!this.hasResized) {
            this.hasResized = true;
            this.resyncController("initial resize");
        }
    }

    processAndCacheData() {
        if (this.dataBytesProcessed < MinDataProcessedForCache) {
            return;
        }
        const serializedOutput = this.serializeAddon.serialize();
        const termSize: TermSize = { rows: this.terminal.rows, cols: this.terminal.cols };
        console.log("idle timeout term", this.dataBytesProcessed, serializedOutput.length, termSize);
        fireAndForget(() =>
            services.BlockService.SaveTerminalState(this.blockId, serializedOutput, "full", this.ptyOffset, termSize)
        );
        this.dataBytesProcessed = 0;
    }

    runProcessIdleTimeout() {
        setTimeout(() => {
            window.requestIdleCallback(() => {
                this.processAndCacheData();
                this.runProcessIdleTimeout();
            });
        }, 5000);
    }

    /**
     * 初始化增强功能
     */
    private initializeEnhancements() {
        try {
            // 初始化增强的 IME 处理，传入输入过滤器重置回调
            this.enhancedIME = new EnhancedIMEHandler(
                this.terminal, 
                this.connectElem,
                () => this.resetInputFilter() // 当 IME 状态变化时重置输入过滤器
            );

            // 初始化增强的快捷键绑定
            this.enhancedKeybindings = new EnhancedKeybindingsHandler(this.terminal, (data: string) => {
                if (this.sendDataHandler) {
                    this.sendDataHandler(data);
                }
            });

            console.log("✅ Enhanced terminal features initialized for blockId:", this.blockId);
            console.log("🎯 IME Handler:", !!this.enhancedIME);
            console.log("🎯 Keybindings Handler:", !!this.enhancedKeybindings);
        } catch (error) {
            console.warn("⚠️ Failed to initialize enhanced terminal features:", error);
        }
    }

    /**
     * 增强的键盘处理（在原有处理之前调用）
     */
    public handleEnhancedKeydown(event: KeyboardEvent): boolean {
        console.log('🎯 handleEnhancedKeydown called for key:', event.key, 'blockId:', this.blockId);
        
        if (!this.enhancedKeybindings) {
            console.log('🎯 No enhanced keybindings available');
            return false;
        }

        const result = this.enhancedKeybindings.handleKeydown(event);
        if (result) {
            console.log('🎯 Enhanced keybinding handled key:', event.key);
        }
        return result;
    }

    /**
     * 获取 IME 状态
     */
    public getIMEState() {
        return this.enhancedIME?.getCompositionState() || { isComposing: false, compositionText: "" };
    }

    /**
     * 输入过滤器 - 防止重复输入
     */
    private shouldFilterInput(data: string): boolean {
        const now = Date.now();
        const timeDiff = now - this.lastInputTime;
        
        // 检查是否是重复的单字符输入（在很短时间内）
        if (data === this.lastInputData && timeDiff < 50 && data.length === 1) {
            return true;
        }
        
        // 更激进的过滤：检查是否是输入法切换后的重复字符
        if (data === this.lastInputData && timeDiff < 500 && /^[a-zA-Z]$/.test(data)) {
            // 检查 IME 状态
            const imeState = this.getIMEState();
            if (!imeState.isComposing) {
                console.log('🎯 Aggressive filter triggered for:', data, 'timeDiff:', timeDiff);
                return true;
            }
        }
        
        // 特殊处理：如果是连续的相同字符且频率很高
        if (data === this.lastInputData && data.length === 1) {
            if (timeDiff < 100) {
                return true;
            }
        }
        
        // 更新状态
        this.lastInputData = data;
        this.lastInputTime = now;
        
        return false;
    }

    /**
     * 重置输入过滤器（在输入法状态变化时调用）
     */
    public resetInputFilter() {
        this.lastInputData = "";
        this.lastInputTime = 0;
        console.log('🎯 Input filter reset');
    }

    /**
     * 启用/禁用输入过滤
     */
    public setInputFilterEnabled(enabled: boolean) {
        this.inputFilterEnabled = enabled;
        if (enabled) {
            this.resetInputFilter();
        }
    }

    /**
     * 清理增强功能
     */
    public disposeEnhancements() {
        if (this.enhancedIME) {
            this.enhancedIME.dispose();
            this.enhancedIME = null;
        }
        this.enhancedKeybindings = null;
    }
}
