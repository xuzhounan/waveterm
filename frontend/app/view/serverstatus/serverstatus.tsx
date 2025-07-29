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

    constructor(blockId: string, viewType: string) {
        this.viewType = viewType;
        this.blockId = blockId;
        this.blockAtom = WOS.getWaveObjectAtom<Block>(`block:${blockId}`);
        this.refreshIntervalRef = { current: null };
        
        this.statusDataAtom = jotai.atom<ServerStatusData>({
            isRunning: false,
            lastUpdated: Date.now(),
        });
        
        this.persistentStatusDataAtom = jotai.atom<PersistentServerStatusData>({
            isRunning: false,
            lastUpdated: Date.now(),
        });
        
        this.loadingAtom = jotai.atom(false);
        this.persistentLoadingAtom = jotai.atom(false);
        
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
        this.startPersistentStatusChecking();
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
        // 持久化服务器状态检查使用稍低的频率
        setInterval(() => {
            this.checkPersistentServerStatus();
        }, 10000); // 每10秒检查一次
    }

    async checkServerStatus() {
        try {
            globalStore.set(this.loadingAtom, true);
            
            // 使用动态端点配置而不是硬编码端口
            const { getWebServerEndpoint } = await import("@/util/endpoints");
            const baseUrl = getWebServerEndpoint().replace('localhost', '127.0.0.1');
            
            console.log(`检查MCP服务器状态: ${baseUrl}`);
            const response = await fetch(`${baseUrl}/api/v1/widgets/mcp/status`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                },
                signal: AbortSignal.timeout(3000),
            });
            
            if (response.ok) {
                const responseData = await response.json();
                console.log(`MCP服务器连接成功`, responseData);
                
                const isRunning = responseData.success && responseData.status?.running;
                
                // 从baseUrl中提取端口
                const urlMatch = baseUrl.match(/:(\d+)/);
                const webPort = urlMatch ? parseInt(urlMatch[1]) : 60289;
                
                const statusData: ServerStatusData = {
                    isRunning: isRunning,
                    webPort: responseData.status?.port || webPort,
                    wsPort: webPort + 1, // WebSocket端口通常是Web端口+1
                    apiUrl: baseUrl,
                    lastUpdated: Date.now(),
                };
                globalStore.set(this.statusDataAtom, statusData);
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
        } finally {
            globalStore.set(this.loadingAtom, false);
        }
    }

    async checkPersistentServerStatus() {
        try {
            globalStore.set(this.persistentLoadingAtom, true);
            
            // 使用动态端点配置
            const { getWebServerEndpoint } = await import("@/util/endpoints");
            const baseUrl = getWebServerEndpoint().replace('localhost', '127.0.0.1');
            
            console.log(`检查持久化服务器状态: ${baseUrl}`);
            const response = await fetch(`${baseUrl}/api/v1/widgets/persistent-server/status`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                },
                signal: AbortSignal.timeout(3000),
            });
            
            if (response.ok) {
                const responseData = await response.json();
                console.log(`持久化服务器状态`, responseData);
                
                const isRunning = responseData.success && responseData.status?.running;
                
                const statusData: PersistentServerStatusData = {
                    isRunning: isRunning,
                    pid: responseData.status?.pid,
                    webPort: responseData.status?.web_port,
                    wsPort: responseData.status?.ws_port,
                    apiUrl: responseData.api_url,
                    lastUpdated: Date.now(),
                };
                globalStore.set(this.persistentStatusDataAtom, statusData);
            } else {
                throw new Error(`HTTP ${response.status}`);
            }
        } catch (error) {
            // 持久化服务器不可用
            const statusData: PersistentServerStatusData = {
                isRunning: false,
                lastUpdated: Date.now(),
                error: error instanceof Error ? error.message : 'Connection failed',
            };
            globalStore.set(this.persistentStatusDataAtom, statusData);
        } finally {
            globalStore.set(this.persistentLoadingAtom, false);
        }
    }

    async startPersistentServer() {
        try {
            globalStore.set(this.persistentLoadingAtom, true);
            
            const { getWebServerEndpoint } = await import("@/util/endpoints");
            const baseUrl = getWebServerEndpoint().replace('localhost', '127.0.0.1');
            
            console.log(`启动持久化服务器: ${baseUrl}`);
            const response = await fetch(`${baseUrl}/api/v1/widgets/persistent-server/start`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                signal: AbortSignal.timeout(30000), // 启动可能需要更长时间
            });
            
            const responseData = await response.json();
            console.log(`持久化服务器启动响应`, responseData);
            
            if (responseData.success) {
                // 延迟检查状态，等待服务器完全启动
                setTimeout(() => {
                    this.checkPersistentServerStatus();
                }, 3000);
            }
            
            return responseData;
        } catch (error) {
            console.error('启动持久化服务器失败:', error);
            throw error;
        } finally {
            globalStore.set(this.persistentLoadingAtom, false);
        }
    }

    async stopPersistentServer() {
        try {
            globalStore.set(this.persistentLoadingAtom, true);
            
            const { getWebServerEndpoint } = await import("@/util/endpoints");
            const baseUrl = getWebServerEndpoint().replace('localhost', '127.0.0.1');
            
            console.log(`停止持久化服务器: ${baseUrl}`);
            const response = await fetch(`${baseUrl}/api/v1/widgets/persistent-server/stop`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                signal: AbortSignal.timeout(10000),
            });
            
            const responseData = await response.json();
            console.log(`持久化服务器停止响应`, responseData);
            
            if (responseData.success) {
                // 立即检查状态
                setTimeout(() => {
                    this.checkPersistentServerStatus();
                }, 1000);
            }
            
            return responseData;
        } catch (error) {
            console.error('停止持久化服务器失败:', error);
            throw error;
        } finally {
            globalStore.set(this.persistentLoadingAtom, false);
        }
    }

    getSettingsMenuItems(): ContextMenuItem[] {
        const persistentStatus = globalStore.get(this.persistentStatusDataAtom);
        return [
            {
                label: "Refresh Status",
                click: () => {
                    this.checkServerStatus();
                    this.checkPersistentServerStatus();
                },
            },
            { type: "separator" },
            {
                label: "Persistent Server",
                type: "separator"
            },
            {
                label: persistentStatus.isRunning ? "Stop Persistent Server" : "Start Persistent Server",
                click: async () => {
                    if (persistentStatus.isRunning) {
                        try {
                            await this.stopPersistentServer();
                            console.log("Persistent server stopped");
                        } catch (error) {
                            console.error("Failed to stop persistent server:", error);
                        }
                    } else {
                        try {
                            await this.startPersistentServer();
                            console.log("Persistent server started");
                        } catch (error) {
                            console.error("Failed to start persistent server:", error);
                        }
                    }
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
                    
                    {/* 持久化服务器控制按钮 */}
                    <div className="control-buttons">
                        <button 
                            className={clsx("control-btn", "start-btn", {
                                "disabled": persistentStatusData.isRunning || persistentLoading
                            })}
                            onClick={async () => {
                                try {
                                    await model.startPersistentServer();
                                } catch (error) {
                                    console.error('Failed to start server:', error);
                                }
                            }}
                            disabled={persistentStatusData.isRunning || persistentLoading}
                        >
                            {persistentLoading ? "Starting..." : "Start Server"}
                        </button>
                        <button 
                            className={clsx("control-btn", "stop-btn", {
                                "disabled": !persistentStatusData.isRunning || persistentLoading
                            })}
                            onClick={async () => {
                                try {
                                    await model.stopPersistentServer();
                                } catch (error) {
                                    console.error('Failed to stop server:', error);
                                }
                            }}
                            disabled={!persistentStatusData.isRunning || persistentLoading}
                        >
                            {persistentLoading ? "Stopping..." : "Stop Server"}
                        </button>
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
                        <div className="error-title">Connection Error</div>
                        <div className="error-message">{statusData.error}</div>
                        <div className="error-suggestion">
                            Try running: <code>./persistent-server.sh start</code>
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
                        <div className="error-title">Persistent Server Connection Error</div>
                        <div className="error-message">{persistentStatusData.error}</div>
                        <div className="error-suggestion">
                            Use the "Start Server" button above or run: <code>./persistent-server.sh start</code>
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