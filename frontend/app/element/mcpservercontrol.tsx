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

// 固定端口配置 - 与服务器端保持一致
const FIXED_MCP_PORT = 60289;

// 尝试多个地址以解决代理/DNS问题
// 优先使用127.0.0.1避免代理问题，localhost作为备选
const ENDPOINTS = [
    `http://127.0.0.1:${FIXED_MCP_PORT}`,
    `http://localhost:${FIXED_MCP_PORT}`
];

async function checkMCPServerStatus(): Promise<MCPServerStatus> {
    let lastError: Error | null = null;
    
    // 尝试不同的端点地址
    for (const endpoint of ENDPOINTS) {
        try {
            console.log(`尝试连接MCP服务器: ${endpoint}`);
            const response = await fetch(`${endpoint}/api/v1/widgets/mcp/status`, {
                method: 'GET',
                headers: {
                    'Cache-Control': 'no-cache',
                    'Accept': 'application/json',
                },
                signal: AbortSignal.timeout(5000), // 增加超时时间
            });
            
            if (response.ok) {
                const data = await response.json();
                if (data.success) {
                    console.log(`MCP服务器连接成功: ${endpoint}`);
                    return {
                        isRunning: data.status?.running || true,
                        port: FIXED_MCP_PORT,
                        lastCheck: Date.now(),
                    };
                } else {
                    console.warn(`MCP服务器响应失败: ${endpoint}`, data);
                }
            } else {
                console.warn(`MCP服务器HTTP错误: ${endpoint} - ${response.status} ${response.statusText}`);
            }
        } catch (error) {
            const errorMsg = error instanceof Error ? error.message : 'Unknown error';
            console.warn(`MCP服务器连接失败 ${endpoint}:`, errorMsg);
            lastError = error instanceof Error ? error : new Error('Unknown error');
            continue; // 尝试下一个端点
        }
    }
    
    return {
        isRunning: false,
        error: lastError ? `所有连接尝试失败: ${lastError.message}` : '连接失败',
        lastCheck: Date.now(),
    };
}

async function startMCPServer(): Promise<boolean> {
    // 尝试使用相同的端点列表启动服务器
    for (const endpoint of ENDPOINTS) {
        try {
            console.log(`尝试启动MCP服务器: ${endpoint}`);
            const response = await fetch(`${endpoint}/api/v1/widgets/mcp/restart`, {
                method: 'POST',
                headers: {
                    'Cache-Control': 'no-cache',
                    'Accept': 'application/json',
                    'Content-Type': 'application/json',
                },
                signal: AbortSignal.timeout(10000), // 增加启动超时时间
            });
            
            if (response.ok) {
                const data = await response.json();
                if (data.success) {
                    console.log(`MCP服务器启动请求成功: ${endpoint}`);
                    // Wait a bit for server to start
                    await new Promise(resolve => setTimeout(resolve, 3000));
                    const status = await checkMCPServerStatus();
                    return status.isRunning;
                } else {
                    console.warn(`MCP服务器启动响应失败: ${endpoint}`, data);
                }
            } else {
                console.warn(`MCP服务器启动HTTP错误: ${endpoint} - ${response.status} ${response.statusText}`);
            }
        } catch (error) {
            const errorMsg = error instanceof Error ? error.message : 'Unknown error';
            console.warn(`MCP服务器启动失败 ${endpoint}:`, errorMsg);
            continue; // 尝试下一个端点
        }
    }
    
    console.error('所有MCP服务器启动尝试都失败了');
    return false;
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