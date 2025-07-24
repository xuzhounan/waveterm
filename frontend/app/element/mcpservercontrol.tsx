// Copyright 2024, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// import { Button } from "@/app/element/button"; // 使用原生button元素
import { atom, useAtomValue } from "jotai";
import { useEffect, useState } from "react";
import "./mcpservercontrol.scss";

interface MCPServerStatus {
    isRunning: boolean;
    port?: number;
    authKey?: string;
    error?: string;
    lastCheck: number;
}

// Atom for MCP server status
const mcpServerStatusAtom = atom<MCPServerStatus>({
    isRunning: false,
    lastCheck: 0,
});

async function checkMCPServerStatus(): Promise<MCPServerStatus> {
    try {
        // Use the current Wave Terminal server to check MCP status
        const response = await fetch('/api/v1/widgets/mcp/status', {
            method: 'GET',
        });
        
        if (response.ok) {
            const data = await response.json();
            if (data.success && data.status) {
                return {
                    isRunning: data.status.running,
                    port: data.status.port,
                    lastCheck: Date.now(),
                };
            }
        }
        
        return {
            isRunning: false,
            lastCheck: Date.now(),
        };
    } catch (error) {
        return {
            isRunning: false,
            error: error instanceof Error ? error.message : 'Unknown error',
            lastCheck: Date.now(),
        };
    }
}

async function startMCPServer(): Promise<boolean> {
    try {
        // Use the restart API endpoint
        const response = await fetch('/api/v1/widgets/mcp/restart', {
            method: 'POST',
        });
        
        if (response.ok) {
            const data = await response.json();
            if (data.success) {
                // Wait a bit for server to start
                await new Promise(resolve => setTimeout(resolve, 3000));
                const status = await checkMCPServerStatus();
                return status.isRunning;
            }
        }
        
        return false;
    } catch (error) {
        console.error('Failed to start MCP server:', error);
        return false;
    }
}

export function MCPServerControl({ className }: { className?: string }) {
    const [status, setStatus] = useState<MCPServerStatus>({
        isRunning: false,
        lastCheck: 0,
    });
    const [isStarting, setIsStarting] = useState(false);
    const [showTooltip, setShowTooltip] = useState(false);

    // Check server status periodically
    useEffect(() => {
        const checkStatus = async () => {
            const newStatus = await checkMCPServerStatus();
            setStatus(newStatus);
        };

        // Check immediately
        checkStatus();

        // Then check every 10 seconds
        const interval = setInterval(checkStatus, 10000);

        return () => clearInterval(interval);
    }, []);

    const handleStartServer = async () => {
        setIsStarting(true);
        try {
            const success = await startMCPServer();
            if (success) {
                // Refresh status
                const newStatus = await checkMCPServerStatus();
                setStatus(newStatus);
            }
        } finally {
            setIsStarting(false);
        }
    };

    const getStatusColor = (): string => {
        if (isStarting) return "yellow";
        return status.isRunning ? "green" : "red";
    };

    const getStatusText = (): string => {
        if (isStarting) return "正在启动...";
        if (status.isRunning) return `MCP服务器运行中 (端口 ${status.port})`;
        return "MCP服务器未运行";
    };

    const getTooltipContent = (): string => {
        const lines = [getStatusText()];
        
        if (status.error) {
            lines.push(`错误: ${status.error}`);
        }
        
        if (status.lastCheck) {
            const lastCheckTime = new Date(status.lastCheck).toLocaleTimeString();
            lines.push(`最后检查: ${lastCheckTime}`);
        }
        
        if (!status.isRunning) {
            lines.push("点击启动MCP服务器");
        }
        
        return lines.join('\n');
    };

    return (
        <div 
            className={`mcp-server-control ${className || ""}`}
            onMouseEnter={() => setShowTooltip(true)}
            onMouseLeave={() => setShowTooltip(false)}
        >
            <div className="mcp-server-indicator">
                <div 
                    className={`status-dot ${getStatusColor()}`}
                    data-status={status.isRunning ? "running" : "stopped"}
                />
                
                {!status.isRunning && (
                    <button
                        className="start-server-button"
                        onClick={(e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            console.log("MCP Server start button clicked");
                            handleStartServer();
                        }}
                        disabled={isStarting}
                        title="启动MCP服务器"
                        type="button"
                    >
                        {isStarting ? "⏳" : "▶️"}
                    </button>
                )}
            </div>

            {showTooltip && (
                <div className="mcp-server-tooltip">
                    <div className="tooltip-content">
                        {getTooltipContent().split('\n').map((line, index) => (
                            <div key={index} className="tooltip-line">
                                {line}
                            </div>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}