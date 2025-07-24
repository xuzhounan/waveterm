// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// MCP (Model Context Protocol) API 接口
// 提供与MCP服务器交互的功能

export interface MCPTool {
    name: string;
    description: string;
    inputSchema: any; // JSON Schema
}

export interface MCPResource {
    uri: string;
    name: string;
    description?: string;
    mimeType?: string;
}

export interface MCPServer {
    id: string;
    name: string;
    url: string;
    status: "connected" | "disconnected" | "connecting" | "error";
    tools: MCPTool[];
    resources: MCPResource[];
    capabilities?: string[];
    error?: string;
}

export interface MCPToolCallRequest {
    server: string;
    tool: string;
    arguments?: Record<string, any>;
}

export interface MCPToolCallResponse {
    success: boolean;
    result?: any;
    error?: string;
    isText?: boolean;
    content?: Array<{
        type: "text" | "image" | "resource";
        text?: string;
        data?: string;
        annotations?: any;
    }>;
}

export interface MCPResourceRequest {
    server: string;
    uri: string;
}

export interface MCPResourceResponse {
    success: boolean;
    content?: string;
    mimeType?: string;
    error?: string;
}

class MCPAPIClient {
    private baseUrl: string;

    constructor(baseUrl: string = "http://localhost:61269") {
        this.baseUrl = baseUrl;
    }

    // 获取MCP服务器状态
    async getServerStatus(): Promise<{ servers: Record<string, MCPServer> }> {
        try {
            const response = await fetch(`${this.baseUrl}/api/v1/mcp/status`, {
                method: "GET",
                signal: AbortSignal.timeout(5000),
            });

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            return await response.json();
        } catch (error) {
            console.warn("MCP status API not available, using fallback");
            // 返回基于现有Widget API的fallback数据
            return {
                servers: {
                    "wave-terminal": {
                        id: "wave-terminal",
                        name: "Wave Terminal",
                        url: this.baseUrl,
                        status: "connected",
                        tools: [
                            {
                                name: "create_widget",
                                description: "Create a new widget in a workspace",
                                inputSchema: {
                                    type: "object",
                                    properties: {
                                        workspace_id: { type: "string" },
                                        widget_type: { type: "string" },
                                        title: { type: "string" },
                                        meta: { type: "object" }
                                    },
                                    required: ["workspace_id", "widget_type"]
                                }
                            },
                            {
                                name: "list_workspaces",
                                description: "List all available workspaces",
                                inputSchema: { type: "object", properties: {} }
                            },
                            {
                                name: "get_workspace",
                                description: "Get workspace information by name or ID",
                                inputSchema: {
                                    type: "object",
                                    properties: {
                                        name: { type: "string" },
                                        id: { type: "string" }
                                    }
                                }
                            }
                        ],
                        resources: [
                            {
                                uri: "workspaces://",
                                name: "Workspaces",
                                description: "Available Wave Terminal workspaces"
                            },
                            {
                                uri: "widgets://",
                                name: "Widgets",
                                description: "Available widget types and configurations"
                            }
                        ],
                        capabilities: ["tools", "resources", "sampling"]
                    }
                }
            };
        }
    }

    // 调用MCP工具
    async callTool(request: MCPToolCallRequest): Promise<MCPToolCallResponse> {
        try {
            // 首先尝试标准MCP API
            const response = await fetch(`${this.baseUrl}/api/v1/mcp/tools/call`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(request),
                signal: AbortSignal.timeout(10000),
            });

            if (response.ok) {
                return await response.json();
            }

            // 如果MCP API不可用，尝试使用现有Widget API
            return await this.callToolFallback(request);
        } catch (error) {
            console.warn("MCP tool call failed, trying fallback:", error);
            return await this.callToolFallback(request);
        }
    }

    // Fallback到现有Widget API
    private async callToolFallback(request: MCPToolCallRequest): Promise<MCPToolCallResponse> {
        try {
            switch (request.tool) {
                case "list_workspaces":
                    const workspacesResponse = await fetch(`${this.baseUrl}/api/v1/widgets/workspaces`);
                    if (workspacesResponse.ok) {
                        const data = await workspacesResponse.json();
                        return {
                            success: true,
                            isText: true,
                            content: [{
                                type: "text",
                                text: JSON.stringify(data, null, 2)
                            }]
                        };
                    }
                    break;

                case "get_workspace":
                    const { name, id } = request.arguments || {};
                    if (name) {
                        const workspaceResponse = await fetch(
                            `${this.baseUrl}/api/v1/widgets/workspace/name/${encodeURIComponent(name)}`
                        );
                        if (workspaceResponse.ok) {
                            const data = await workspaceResponse.json();
                            return {
                                success: true,
                                isText: true,
                                content: [{
                                    type: "text",
                                    text: JSON.stringify(data, null, 2)
                                }]
                            };
                        }
                    }
                    break;

                case "create_widget":
                    const createResponse = await fetch(`${this.baseUrl}/api/v1/widgets`, {
                        method: "POST",
                        headers: {
                            "Content-Type": "application/json",
                        },
                        body: JSON.stringify(request.arguments),
                    });
                    if (createResponse.ok) {
                        const data = await createResponse.json();
                        return {
                            success: true,
                            isText: true,
                            content: [{
                                type: "text",
                                text: JSON.stringify(data, null, 2)
                            }]
                        };
                    }
                    break;
            }

            return {
                success: false,
                error: `Tool '${request.tool}' not supported or failed to execute`
            };
        } catch (error) {
            return {
                success: false,
                error: `Tool execution failed: ${error.message}`
            };
        }
    }

    // 获取MCP资源
    async getResource(request: MCPResourceRequest): Promise<MCPResourceResponse> {
        try {
            const response = await fetch(`${this.baseUrl}/api/v1/mcp/resources/get`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(request),
                signal: AbortSignal.timeout(5000),
            });

            if (response.ok) {
                return await response.json();
            }

            // Fallback处理
            return await this.getResourceFallback(request);
        } catch (error) {
            console.warn("MCP resource request failed, trying fallback:", error);
            return await this.getResourceFallback(request);
        }
    }

    private async getResourceFallback(request: MCPResourceRequest): Promise<MCPResourceResponse> {
        try {
            if (request.uri.startsWith("workspaces://")) {
                const response = await fetch(`${this.baseUrl}/api/v1/widgets/workspaces`);
                if (response.ok) {
                    const data = await response.json();
                    return {
                        success: true,
                        content: JSON.stringify(data, null, 2),
                        mimeType: "application/json"
                    };
                }
            }

            return {
                success: false,
                error: `Resource '${request.uri}' not found or not supported`
            };
        } catch (error) {
            return {
                success: false,
                error: `Resource request failed: ${error.message}`
            };
        }
    }

    // 列出可用工具
    async listTools(serverId?: string): Promise<MCPTool[]> {
        const status = await this.getServerStatus();
        const tools: MCPTool[] = [];

        for (const server of Object.values(status.servers)) {
            if (!serverId || server.id === serverId) {
                tools.push(...server.tools);
            }
        }

        return tools;
    }

    // 列出可用资源
    async listResources(serverId?: string): Promise<MCPResource[]> {
        const status = await this.getServerStatus();
        const resources: MCPResource[] = [];

        for (const server of Object.values(status.servers)) {
            if (!serverId || server.id === serverId) {
                resources.push(...server.resources);
            }
        }

        return resources;
    }
}

// 全局MCP API客户端实例
export const mcpAPI = new MCPAPIClient();

// 便捷函数
export async function callMCPTool(tool: string, args?: Record<string, any>, server: string = "wave-terminal") {
    return mcpAPI.callTool({ server, tool, arguments: args });
}

export async function getMCPResource(uri: string, server: string = "wave-terminal") {
    return mcpAPI.getResource({ server, uri });
}

export async function getMCPStatus() {
    return mcpAPI.getServerStatus();
}