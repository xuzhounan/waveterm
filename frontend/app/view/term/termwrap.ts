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
    
    // 输入过滤器 - 防止多重事件处理器导致的重复输入
    private lastInputData = "";
    private lastInputTime = 0;
    private inputFilterEnabled = true;
    private inputProcessingLock = false;
    private pendingInputs = new Map<string, number>();

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

        // 防止 IME composition 中间状态被发送
        if (this.enhancedIME) {
            const imeState = this.enhancedIME.getCompositionState();
            if (imeState.isComposing) {
                return; // 忽略 composition 过程中的中间输入
            }
        }

        // 防止重复输入
        const now = Date.now();
        const timeDiff = now - this.lastInputTime;
        
        // 完全相同的快速重复输入
        if (data === this.lastInputData && timeDiff < 50) {
            return;
        }
        
        // 检查 IME 相关的重复（如 "a s d" 和 "asd"）
        if (this.lastInputData && timeDiff < 100) {
            const lastWithoutSpaces = this.lastInputData.replace(/\s/g, '');
            const currentWithoutSpaces = data.replace(/\s/g, '');
            
            if (lastWithoutSpaces === currentWithoutSpaces && lastWithoutSpaces.length > 0) {
                if (this.lastInputData.includes(' ') && !data.includes(' ')) {
                    // 当前输入是最终结果，继续处理
                } else {
                    // 其他情况认为是重复，忽略
                    return;
                }
            }
        }

        // 更新输入记录
        this.lastInputData = data;
        this.lastInputTime = now;

        // 处理输入
        if (this.pasteActive) {
            this.pasteActive = false;
            if (this.multiInputCallback) {
                this.multiInputCallback(data);
            }
        }
        this.sendDataHandler?.(data);
    }

    onKeyHandler(data: { key: string; domEvent: KeyboardEvent }) {
        // onKey 事件已由 onData 统一处理，避免重复处理
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
            // 初始化增强的 IME 处理
            this.enhancedIME = new EnhancedIMEHandler(
                this.terminal, 
                this.connectElem,
                () => this.resetInputFilter()
            );

            // 初始化增强的快捷键绑定
            this.enhancedKeybindings = new EnhancedKeybindingsHandler(this.terminal, (data: string) => {
                if (this.sendDataHandler) {
                    this.sendDataHandler(data);
                }
            });
        } catch (error) {
            console.warn("⚠️ Failed to initialize enhanced terminal features:", error);
        }
    }

    /**
     * 增强的键盘处理
     */
    public handleEnhancedKeydown(event: KeyboardEvent): boolean {
        if (!this.enhancedKeybindings) {
            return false;
        }
        return this.enhancedKeybindings.handleKeydown(event);
    }

    /**
     * 获取 IME 状态
     */
    public getIMEState() {
        return this.enhancedIME?.getCompositionState() || { isComposing: false, compositionText: "" };
    }

    /**
     * 激进的输入过滤器 - 防止多重事件处理器导致的重复输入
     */
    private shouldFilterInputAggressive(data: string): boolean {
        const now = Date.now();
        const timeDiff = now - this.lastInputTime;
        
        // 1. 检查 pending 输入映射（防止多个事件处理器同时处理同一输入）
        const pendingTime = this.pendingInputs.get(data);
        if (pendingTime && (now - pendingTime) < 300) {
            console.log('🎯 Filtered pending duplicate input:', data, 'pendingAge:', now - pendingTime, 'ms');
            return true;
        }
        
        // 2. 检查是否是相同数据在短时间内重复
        if (data === this.lastInputData) {
            // 根据输入类型使用不同的时间窗口
            let timeThreshold = 200; // 默认 200ms
            
            // 单字符（特别是字母和数字）使用更严格的时间窗口
            if (data.length === 1 && /^[a-zA-Z0-9]$/.test(data)) {
                timeThreshold = 500; // 单字符使用 500ms
            }
            
            if (timeDiff < timeThreshold) {
                console.log('🎯 Filtered duplicate input:', data, 'timeDiff:', timeDiff, 'ms', 'threshold:', timeThreshold);
                return true;
            }
        }
        
        // 3. 更新状态
        this.lastInputData = data;
        this.lastInputTime = now;
        this.pendingInputs.set(data, now);
        
        // 4. 清理过期的 pending 输入
        this.cleanupPendingInputs(now);
        
        return false;
    }
    
    /**
     * 清理过期的 pending 输入
     */
    private cleanupPendingInputs(now: number) {
        for (const [key, time] of this.pendingInputs.entries()) {
            if (now - time > 1000) { // 1秒后清理
                this.pendingInputs.delete(key);
            }
        }
    }

    /**
     * 检测可能的重复输入 - 特别针对输入法切换场景
     */
    private isLikelyDuplicateInput(data: string): boolean {
        if (!data || data.length !== 1) {
            return false;
        }

        const now = Date.now();
        const timeDiff = now - this.lastInputTime;
        
        // 对于单字符输入，如果在很短的时间内重复出现，很可能是重复
        if (data === this.lastInputData && timeDiff < 100) {
            console.log('🎯 Very fast duplicate detected:', data, 'timeDiff:', timeDiff, 'ms');
            return true;
        }

        // 检查是否是输入法切换后的快速重复
        // 如果 IME 不在 composing 状态，但有相同的字符在短时间内出现
        if (this.enhancedIME) {
            const imeState = this.enhancedIME.getCompositionState();
            if (!imeState.isComposing && data === this.lastInputData && timeDiff < 150) {
                console.log('🎯 Non-IME duplicate detected:', data, 'timeDiff:', timeDiff, 'ms');
                return true;
            }
        }

        return false;
    }

    /**
     * 重置输入过滤器（在输入法状态变化时调用）
     */
    public resetInputFilter() {
        this.lastInputData = "";
        this.lastInputTime = 0;
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
