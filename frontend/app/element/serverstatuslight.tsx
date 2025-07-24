// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import * as React from "react";
import * as jotai from "jotai";
import { globalStore } from "@/store/global";
import clsx from "clsx";
import "./serverstatuslight.scss";

type ServerStatus = "running" | "stopped" | "checking" | "error";

interface ServerStatusData {
    status: ServerStatus;
    message?: string;
    lastChecked: number;
}

class ServerStatusLightModel {
    statusAtom: jotai.PrimitiveAtom<ServerStatusData>;
    intervalRef: React.MutableRefObject<NodeJS.Timeout | null>;

    constructor() {
        this.statusAtom = jotai.atom<ServerStatusData>({
            status: "checking",
            lastChecked: Date.now(),
        });
        
        this.intervalRef = { current: null };
        this.startStatusChecking();
    }

    startStatusChecking() {
        // 立即检查一次
        this.checkServerStatus();
        
        // 每5秒检查一次
        this.intervalRef.current = setInterval(() => {
            this.checkServerStatus();
        }, 5000);
    }

    async checkServerStatus() {
        try {
            const response = await fetch('http://localhost:61269/api/v1/widgets', {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                },
                // 设置较短的超时时间
                signal: AbortSignal.timeout(3000),
            });
            
            if (response.ok) {
                globalStore.set(this.statusAtom, {
                    status: "running",
                    message: "MCP Server is running",
                    lastChecked: Date.now(),
                });
            } else {
                globalStore.set(this.statusAtom, {
                    status: "error",
                    message: `Server responded with ${response.status}`,
                    lastChecked: Date.now(),
                });
            }
        } catch (error) {
            globalStore.set(this.statusAtom, {
                status: "stopped",
                message: "MCP Server is not responding",
                lastChecked: Date.now(),
            });
        }
    }

    dispose() {
        if (this.intervalRef.current) {
            clearInterval(this.intervalRef.current);
            this.intervalRef.current = null;
        }
    }
}

// 全局单例
const serverStatusModel = new ServerStatusLightModel();

interface ServerStatusLightProps {
    className?: string;
    size?: "small" | "medium" | "large";
    showTooltip?: boolean;
}

export const ServerStatusLight: React.FC<ServerStatusLightProps> = ({ 
    className, 
    size = "medium", 
    showTooltip = true 
}) => {
    const statusData = jotai.useAtomValue(serverStatusModel.statusAtom);
    const [showDetails, setShowDetails] = React.useState(false);

    // 清理函数
    React.useEffect(() => {
        return () => {
            // 注意：不要在这里dispose，因为这是全局单例
        };
    }, []);

    const getStatusColor = (status: ServerStatus): string => {
        switch (status) {
            case "running":
                return "#10b981"; // emerald-500
            case "stopped":
                return "#ef4444"; // red-500
            case "checking":
                return "#f59e0b"; // amber-500
            case "error":
                return "#f97316"; // orange-500
            default:
                return "#6b7280"; // gray-500
        }
    };

    const getStatusText = (status: ServerStatus): string => {
        switch (status) {
            case "running":
                return "Server Running";
            case "stopped":
                return "Server Stopped";
            case "checking":
                return "Checking...";
            case "error":
                return "Server Error";
            default:
                return "Unknown";
        }
    };

    const handleClick = () => {
        if (showTooltip) {
            setShowDetails(!showDetails);
        }
    };

    const formatLastChecked = (timestamp: number): string => {
        const now = Date.now();
        const diff = now - timestamp;
        if (diff < 60000) { // 小于1分钟
            return "just now";
        } else if (diff < 3600000) { // 小于1小时
            return `${Math.floor(diff / 60000)}m ago`;
        } else {
            return `${Math.floor(diff / 3600000)}h ago`;
        }
    };

    return (
        <div 
            className={clsx("server-status-light", className, size)}
            onClick={handleClick}
            onMouseEnter={() => showTooltip && setShowDetails(true)}
            onMouseLeave={() => showTooltip && setShowDetails(false)}
        >
            {/* 主呼吸灯 */}
            <div className="status-orb-container">
                <div 
                    className={clsx("status-orb", statusData.status)}
                    style={{
                        backgroundColor: getStatusColor(statusData.status),
                        boxShadow: `0 0 20px ${getStatusColor(statusData.status)}40`,
                    }}
                >
                    {/* 呼吸效果的外圈 */}
                    <div 
                        className="breathing-ring"
                        style={{
                            borderColor: getStatusColor(statusData.status),
                        }}
                    />
                    
                    {/* 中心点 */}
                    <div 
                        className="center-dot"
                        style={{
                            backgroundColor: getStatusColor(statusData.status),
                        }}
                    />
                    
                    {/* 服务器图标 */}
                    <i className="fa fa-server status-icon" />
                </div>
            </div>

            {/* 详细信息提示 */}
            {showDetails && showTooltip && (
                <div className="status-tooltip">
                    <div className="tooltip-content">
                        <div className="status-header">
                            <div 
                                className="status-indicator-small"
                                style={{ backgroundColor: getStatusColor(statusData.status) }}
                            />
                            <span className="status-text">{getStatusText(statusData.status)}</span>
                        </div>
                        
                        {statusData.message && (
                            <div className="status-message">{statusData.message}</div>
                        )}
                        
                        <div className="status-footer">
                            <span className="last-checked">
                                Last checked: {formatLastChecked(statusData.lastChecked)}
                            </span>
                        </div>
                        
                        <div className="tooltip-actions">
                            <button 
                                className="action-btn"
                                onClick={(e) => {
                                    e.stopPropagation();
                                    serverStatusModel.checkServerStatus();
                                }}
                            >
                                <i className="fa fa-refresh" /> Refresh
                            </button>
                            
                            {statusData.status === "running" && (
                                <button 
                                    className="action-btn"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        window.open('http://localhost:61269/api/v1/widgets', '_blank');
                                    }}
                                >
                                    <i className="fa fa-external-link" /> Open API
                                </button>
                            )}
                        </div>
                    </div>
                    
                    {/* 箭头指示器 */}
                    <div className="tooltip-arrow" />
                </div>
            )}
        </div>
    );
};

// 导出模型以供其他组件使用
export { serverStatusModel };