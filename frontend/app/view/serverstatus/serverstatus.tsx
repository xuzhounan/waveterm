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

class ServerStatusViewModel implements ViewModel {
    viewType: string;
    blockAtom: jotai.Atom<Block>;
    htmlElemFocusRef: React.RefObject<HTMLInputElement>;
    blockId: string;
    viewIcon: jotai.Atom<string>;
    viewText: jotai.Atom<string>;
    viewName: jotai.Atom<string>;
    statusDataAtom: jotai.PrimitiveAtom<ServerStatusData>;
    loadingAtom: jotai.PrimitiveAtom<boolean>;
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
        
        this.loadingAtom = jotai.atom(false);
        
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

    async checkServerStatus() {
        try {
            globalStore.set(this.loadingAtom, true);
            
            // 调用我们的Widget API获取服务器状态
            const response = await fetch('http://localhost:61269/api/v1/widgets', {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                }
            });
            
            if (response.ok) {
                const data = await response.json();
                const statusData: ServerStatusData = {
                    isRunning: true,
                    webPort: 61269,
                    wsPort: 61270,
                    apiUrl: 'http://localhost:61269',
                    authKey: '83958e47ddc89fae695a7e1eb429899871e80334bd58cfc2d17a80388791f073',
                    uptime: Math.floor((Date.now() - (Date.now() % 86400000)) / 1000), // 今天的运行时间
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

    getSettingsMenuItems(): ContextMenuItem[] {
        return [
            {
                label: "Refresh Status",
                click: () => {
                    this.checkServerStatus();
                },
            },
            {
                label: "Start Server",
                click: async () => {
                    // 这里可以调用启动脚本
                    console.log("Starting server...");
                },
            },
            {
                label: "Stop Server", 
                click: async () => {
                    // 这里可以调用停止脚本
                    console.log("Stopping server...");
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
    const loading = jotai.useAtomValue(model.loadingAtom);
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
                {/* 状态指示器 */}
                <div className={clsx("status-indicator", { 
                    "running": statusData.isRunning,
                    "stopped": !statusData.isRunning,
                    "loading": loading
                })}>
                    <div className="status-dot"></div>
                    <div className="status-text">
                        {loading ? "Checking..." : 
                         statusData.isRunning ? "Server Running" : "Server Stopped"}
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

                {/* API端点信息 */}
                {statusData.isRunning && (
                    <div className="api-endpoints">
                        <div className="endpoints-title">Available API Endpoints</div>
                        <div className="endpoint-list">
                            <div className="endpoint-item">
                                <div className="endpoint-method">GET</div>
                                <div className="endpoint-path">/api/v1/widgets/workspaces</div>
                                <div className="endpoint-desc">List all workspaces</div>
                            </div>
                            <div className="endpoint-item">
                                <div className="endpoint-method">GET</div>
                                <div className="endpoint-path">/api/v1/widgets/workspace/name/{'{name}'}</div>
                                <div className="endpoint-desc">Get workspace by name</div>
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