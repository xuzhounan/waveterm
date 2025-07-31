// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { globalStore, WOS } from "@/store/global";
import * as jotai from "jotai";
import * as React from "react";
import { RpcApi } from "@/app/store/wshclientapi";
import { TabRpcClient } from "@/app/store/wshrpcutil";
import { OverlayScrollbarsComponent, OverlayScrollbarsComponentRef } from "overlayscrollbars-react";
import clsx from "clsx";
import "./serverstatus.scss";

type ServerStatusData = {
    isRunning: boolean;
    pid?: number;
    webPort?: number;
    wsPort?: number;
    apiUrl?: string;
    authKey?: string;
    uptime?: number;
    lastUpdated: number;
    error?: string;
    bridgeEnabled?: boolean;
    bridgeRemoteUrls?: string[];
};

type PersistentServerStatusData = {
    isRunning: boolean;
    pid?: number;
    webPort?: number;
    wsPort?: number;
    apiUrl?: string;
    lastUpdated: number;
    error?: string;
};

class ServerStatusViewModel implements ViewModel {
    viewType: string;
    blockAtom: jotai.Atom<Block>;
    htmlElemFocusRef: React.RefObject<HTMLInputElement>;
    blockId: string;
    viewIcon: jotai.Atom<string>;
    viewText: jotai.Atom<string>;
    viewName: jotai.Atom<string>;
    statusDataAtom: jotai.PrimitiveAtom<ServerStatusData>;
    persistentStatusDataAtom: jotai.PrimitiveAtom<PersistentServerStatusData>;
    loadingAtom: jotai.PrimitiveAtom<boolean>;
    persistentLoadingAtom: jotai.PrimitiveAtom<boolean>;
    refreshIntervalRef: React.MutableRefObject<NodeJS.Timeout | null>;
    persistentIntervalRef: React.MutableRefObject<NodeJS.Timeout | null>;
    persistentFailureCount: number;

    constructor(blockId: string, viewType: string) {
        this.viewType = viewType;
        this.blockId = blockId;
        this.blockAtom = WOS.getWaveObjectAtom<Block>(`block:${blockId}`);
        this.refreshIntervalRef = { current: null };
        this.persistentIntervalRef = { current: null };
        
        this.statusDataAtom = jotai.atom<ServerStatusData>({
            isRunning: false,
            lastUpdated: Date.now(),
        });
        
        // 在一体化模式下，持久化服务器状态就是MCP服务器状态
        this.persistentStatusDataAtom = jotai.atom<PersistentServerStatusData>({
            isRunning: false,
            lastUpdated: Date.now(),
        });
        
        this.loadingAtom = jotai.atom(false);
        this.persistentLoadingAtom = jotai.atom(false);
        this.persistentFailureCount = 0;
        
        this.viewIcon = jotai.atom((get) => {
            const statusData = get(this.statusDataAtom);
            return statusData.isRunning ? "server" : "server-off"; 
        });
        
        this.viewName = jotai.atom((get) => {
            const statusData = get(this.statusDataAtom);
            return statusData.isRunning ? "Server Status (Running)" : "Server Status (Stopped)";
        });
        
        this.viewText = jotai.atom((get) => {
            const statusData = get(this.statusDataAtom);
            if (statusData.isRunning && statusData.apiUrl) {
                return `API: ${statusData.apiUrl}`;
            }
            return "Server Monitor";
        });

        // 开始定期检查服务器状态
        this.startStatusChecking();
        // 在一体化模式下，不需要单独检查持久化服务器
        // this.startPersistentStatusChecking();
    }

    get viewComponent(): ViewComponent {
        return ServerStatusView;
    }

    startStatusChecking() {
        this.checkServerStatus();
        this.refreshIntervalRef.current = setInterval(() => {
            this.checkServerStatus();
        }, 5000); // 每5秒检查一次
    }

    startPersistentStatusChecking() {
        this.checkPersistentServerStatus();
        this.schedulePersistentStatusCheck();
    }

    schedulePersistentStatusCheck() {
        // 根据失败次数调整检查间隔
        let interval = 10000; // 基础间隔10秒
        if (this.persistentFailureCount > 3) {
            interval = 60000; // 连续失败3次后，改为60秒检查一次
        } else if (this.persistentFailureCount > 1) {
            interval = 30000; // 连续失败1次后，改为30秒检查一次
        }

        this.persistentIntervalRef.current = setTimeout(() => {
            this.checkPersistentServerStatus();
            this.schedulePersistentStatusCheck();
        }, interval);
    }

    async checkServerStatus() {
        try {
            globalStore.set(this.loadingAtom, true);
            globalStore.set(this.persistentLoadingAtom, true);
            
            // 使用动态端点配置而不是硬编码端口
            const { getWebServerEndpoint } = await import("@/util/endpoints");
            const baseUrl = getWebServerEndpoint().replace('localhost', '127.0.0.1');
            
            const response = await fetch(`${baseUrl}/api/v1/widgets/mcp/status`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                },
                signal: AbortSignal.timeout(3000),
            });
            
            if (response.ok) {
                const responseData = await response.json();
                
                const isRunning = responseData.success && responseData.status?.running;
                
                // 从baseUrl中提取端口
                const urlMatch = baseUrl.match(/:(\d+)/);
                const webPort = urlMatch ? parseInt(urlMatch[1]) : 8090;
                
                const statusData: ServerStatusData = {
                    isRunning: isRunning,
                    webPort: responseData.status?.port || webPort,
                    wsPort: webPort + 1, // WebSocket端口通常是Web端口+1
                    apiUrl: baseUrl,
                    lastUpdated: Date.now(),
                    bridgeEnabled: responseData.status?.bridge_enabled || responseData.bridge?.enabled,
                    bridgeRemoteUrls: responseData.bridge?.remote_urls || [],
                };
                globalStore.set(this.statusDataAtom, statusData);
                
                // 在一体化模式下，持久化服务器状态与MCP服务器状态同步
                const persistentStatusData: PersistentServerStatusData = {
                    isRunning: isRunning,
                    webPort: responseData.status?.port || webPort,
                    wsPort: webPort + 1,
                    apiUrl: baseUrl,
                    lastUpdated: Date.now(),
                };
                globalStore.set(this.persistentStatusDataAtom, persistentStatusData);
                
                // 重置失败计数器
                this.persistentFailureCount = 0;
            } else {
                throw new Error(`HTTP ${response.status}`);
            }
        } catch (error) {
            // 服务器不可用
            const statusData: ServerStatusData = {
                isRunning: false,
                lastUpdated: Date.now(),
                error: error instanceof Error ? error.message : 'Connection failed',
            };
            globalStore.set(this.statusDataAtom, statusData);
            
            // 持久化服务器状态也同步更新
            const persistentStatusData: PersistentServerStatusData = {
                isRunning: false,
                lastUpdated: Date.now(),
                error: error instanceof Error ? error.message : 'Connection failed',
            };
            globalStore.set(this.persistentStatusDataAtom, persistentStatusData);
        } finally {
            globalStore.set(this.loadingAtom, false);
            globalStore.set(this.persistentLoadingAtom, false);
        }
    }

    // 在一体化模式下，不再需要单独的持久化服务器状态检查
    // 持久化服务器状态将通过 checkServerStatus 方法同步更新

    async startPersistentServer() {
        // 在一体化模式下，服务器已经随应用启动，不需要单独启动
        console.log('在一体化模式下，服务器已经随Wave Terminal启动');
        
        // 立即检查服务器状态以更新UI
        this.checkServerStatus();
    }

    async stopPersistentServer() {
        // 在一体化模式下，服务器与Wave Terminal应用生命周期绑定，不能单独停止
        console.log('在一体化模式下，服务器不能单独停止，它与Wave Terminal应用生命周期绑定');
        
        // 立即检查服务器状态以更新UI
        this.checkServerStatus();
    }

    getSettingsMenuItems(): ContextMenuItem[] {
        return [
            {
                label: "Refresh Status",
                click: () => {
                    this.checkServerStatus();
                },
            },
            { type: "separator" },
            {
                label: "Integrated Mode Info",
                type: "separator"
            },
            {
                label: "Server runs with Wave Terminal",
                click: () => {
                    console.log("在一体化模式下，服务器随Wave Terminal自动启动");
                },
            },
            { type: "separator" },
            {
                label: "View Logs",
                click: () => {
                    // 打开日志文件
                    console.log("Opening logs...");
                },
            },
        ];
    }

    dispose() {
        if (this.refreshIntervalRef.current) {
            clearInterval(this.refreshIntervalRef.current);
            this.refreshIntervalRef.current = null;
        }
        if (this.persistentIntervalRef.current) {
            clearTimeout(this.persistentIntervalRef.current);
            this.persistentIntervalRef.current = null;
        }
    }
}

type ServerStatusViewProps = {
    blockId: string;
    model: ServerStatusViewModel;
};

function ServerStatusView({ model, blockId }: ServerStatusViewProps) {
    const statusData = jotai.useAtomValue(model.statusDataAtom);
    const persistentStatusData = jotai.useAtomValue(model.persistentStatusDataAtom);
    const loading = jotai.useAtomValue(model.loadingAtom);
    const persistentLoading = jotai.useAtomValue(model.persistentLoadingAtom);
    const osRef = React.useRef<OverlayScrollbarsComponentRef>();

    // 清理定时器
    React.useEffect(() => {
        return () => {
            model.dispose();
        };
    }, [model]);

    const formatUptime = (uptime: number) => {
        const hours = Math.floor(uptime / 3600);
        const minutes = Math.floor((uptime % 3600) / 60);
        const seconds = uptime % 60;
        return `${hours}h ${minutes}m ${seconds}s`;
    };

    const formatLastUpdated = (timestamp: number) => {
        const date = new Date(timestamp);
        return date.toLocaleTimeString();
    };

    return (
        <OverlayScrollbarsComponent
            ref={osRef}
            className="serverstatus-view"
            options={{ scrollbars: { autoHide: "leave" } }}
        >
            <div className="serverstatus-content">
                {/* MCP服务器状态指示器 */}
                <div className="server-section">
                    <h3 className="section-title">MCP Server Status</h3>
                    <div className={clsx("status-indicator", { 
                        "running": statusData.isRunning,
                        "stopped": !statusData.isRunning,
                        "loading": loading
                    })}>
                        <div className="status-dot"></div>
                        <div className="status-text">
                            {loading ? "Checking..." : 
                             statusData.isRunning ? "MCP Server Running" : "MCP Server Stopped"}
                        </div>
                    </div>
                </div>

                {/* 持久化服务器状态指示器 */}
                <div className="server-section">
                    <h3 className="section-title">Persistent Server Status</h3>
                    <div className={clsx("status-indicator", { 
                        "running": persistentStatusData.isRunning,
                        "stopped": !persistentStatusData.isRunning,
                        "loading": persistentLoading
                    })}>
                        <div className="status-dot"></div>
                        <div className="status-text">
                            {persistentLoading ? "Checking..." : 
                             persistentStatusData.isRunning ? "Persistent Server Running" : "Persistent Server Stopped"}
                        </div>
                    </div>
                    
                    {/* 一体化模式说明 */}
                    <div className="integrated-mode-info">
                        <div className="info-message">
                            <span className="info-icon">ℹ️</span>
                            Server runs automatically with Wave Terminal. No manual control needed.
                        </div>
                    </div>
                </div>

                {/* 服务器信息 */}
                {statusData.isRunning && (
                    <div className="server-info">
                        <div className="info-grid">
                            <div className="info-item">
                                <div className="info-label">Web Port</div>
                                <div className="info-value">{statusData.webPort}</div>
                            </div>
                            <div className="info-item">
                                <div className="info-label">WebSocket Port</div>
                                <div className="info-value">{statusData.wsPort}</div>
                            </div>
                            <div className="info-item">
                                <div className="info-label">API URL</div>
                                <div className="info-value">
                                    <a href={statusData.apiUrl} target="_blank" rel="noopener noreferrer">
                                        {statusData.apiUrl}
                                    </a>
                                </div>
                            </div>
                            <div className="info-item">
                                <div className="info-label">Event Bridge</div>
                                <div className={clsx("info-value", "bridge-status", {
                                    "bridge-enabled": statusData.bridgeEnabled,
                                    "bridge-disabled": !statusData.bridgeEnabled
                                })}>
                                    <span className="bridge-indicator">
                                        {statusData.bridgeEnabled ? "🟢 Enabled" : "🔴 Disabled"}
                                    </span>
                                    {statusData.bridgeEnabled && statusData.bridgeRemoteUrls && statusData.bridgeRemoteUrls.length > 0 && (
                                        <span className="bridge-remotes">
                                            ({statusData.bridgeRemoteUrls.length} remote{statusData.bridgeRemoteUrls.length > 1 ? 's' : ''})
                                        </span>
                                    )}
                                </div>
                            </div>
                            {statusData.authKey && (
                                <div className="info-item">
                                    <div className="info-label">Auth Key</div>
                                    <div className="info-value auth-key">
                                        {statusData.authKey.substring(0, 12)}...
                                        <button 
                                            className="copy-btn"
                                            onClick={() => navigator.clipboard.writeText(statusData.authKey || '')}
                                            title="Copy full auth key"
                                        >
                                            📋
                                        </button>
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                )}

                {/* 错误信息 */}
                {!statusData.isRunning && statusData.error && (
                    <div className="error-info">
                        <div className="error-title">MCP Server Connection Error</div>
                        <div className="error-message">{statusData.error}</div>
                        <div className="error-suggestion">
                            Server runs automatically with Wave Terminal. Try restarting the application.
                        </div>
                    </div>
                )}

                {/* 持久化服务器信息 */}
                {persistentStatusData.isRunning && (
                    <div className="server-info">
                        <div className="info-title">Persistent Server Information</div>
                        <div className="info-grid">
                            <div className="info-item">
                                <div className="info-label">Process ID</div>
                                <div className="info-value">{persistentStatusData.pid}</div>
                            </div>
                            <div className="info-item">
                                <div className="info-label">Web Port</div>
                                <div className="info-value">{persistentStatusData.webPort}</div>
                            </div>
                            <div className="info-item">
                                <div className="info-label">WebSocket Port</div>
                                <div className="info-value">{persistentStatusData.wsPort}</div>
                            </div>
                            {persistentStatusData.apiUrl && (
                                <div className="info-item">
                                    <div className="info-label">API URL</div>
                                    <div className="info-value">
                                        <a href={persistentStatusData.apiUrl} target="_blank" rel="noopener noreferrer">
                                            {persistentStatusData.apiUrl}
                                        </a>
                                    </div>
                                </div>
                            )}
                        </div>
                    </div>
                )}

                {/* 持久化服务器错误信息 */}
                {!persistentStatusData.isRunning && persistentStatusData.error && (
                    <div className="error-info">
                        <div className="error-title">Server Connection Error</div>
                        <div className="error-message">{persistentStatusData.error}</div>
                        <div className="error-suggestion">
                            Server runs automatically with Wave Terminal. Check your connection or restart the application.
                        </div>
                    </div>
                )}

                {/* MCP API端点信息 */}
                {statusData.isRunning && (
                    <div className="api-endpoints">
                        <div className="endpoints-title">Available MCP API Endpoints</div>
                        <div className="endpoint-list">
                            <div className="endpoint-item">
                                <div className="endpoint-method">GET</div>
                                <div className="endpoint-path">/api/v1/widgets/mcp/status</div>
                                <div className="endpoint-desc">Check MCP server status</div>
                            </div>
                            <div className="endpoint-item">
                                <div className="endpoint-method">GET</div>
                                <div className="endpoint-path">/api/v1/widgets/workspaces</div>
                                <div className="endpoint-desc">List all workspaces</div>
                            </div>
                            <div className="endpoint-item">
                                <div className="endpoint-method">POST</div>
                                <div className="endpoint-path">/api/v1/widgets</div>
                                <div className="endpoint-desc">Create new widget</div>
                            </div>
                        </div>
                    </div>
                )}

                {/* 最后更新时间 */}
                <div className="last-updated">
                    Last updated: {formatLastUpdated(statusData.lastUpdated)}
                </div>
            </div>
        </OverlayScrollbarsComponent>
    );
}

export { ServerStatusViewModel };