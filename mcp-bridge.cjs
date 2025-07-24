#!/usr/bin/env node

/**
 * Wave Terminal MCP Bridge
 * 
 * 这个脚本作为MCP服务器，将Claude Code与Wave Terminal连接起来
 * 支持所有Wave Terminal的Widget API功能
 */

const { Server } = require('@modelcontextprotocol/sdk/server/index.js');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio.js');
const { 
    ListToolsRequestSchema, 
    CallToolRequestSchema,
    ListResourcesRequestSchema,
    ReadResourceRequestSchema,
    ListPromptsRequestSchema,
    GetPromptRequestSchema
} = require('@modelcontextprotocol/sdk/types.js');

class WaveTerminalMCPServer extends Server {
    constructor() {
        super({
            name: "wave-terminal",
            version: "1.0.0",
            description: "Wave Terminal MCP integration with breathing light and status monitoring"
        }, {
            capabilities: {
                tools: {},
                resources: {},
                prompts: {}
            }
        });
        
        this.waveTerminalUrl = process.env.WAVE_TERMINAL_URL || "http://localhost:61269";
        this.authKey = process.env.WAVE_TERMINAL_AUTH_KEY;
        
        console.error(`[MCP] Wave Terminal MCP Server starting...`);
        console.error(`[MCP] URL: ${this.waveTerminalUrl}`);
        console.error(`[MCP] Auth: ${this.authKey ? 'Enabled' : 'Disabled'}`);
        
        this.setupHandlers();
    }

    setupHandlers() {
        // 工具处理
        this.setRequestHandler(ListToolsRequestSchema, this.listTools.bind(this));
        this.setRequestHandler(CallToolRequestSchema, this.callTool.bind(this));
        
        // 资源处理
        this.setRequestHandler(ListResourcesRequestSchema, this.listResources.bind(this));
        this.setRequestHandler(ReadResourceRequestSchema, this.readResource.bind(this));
        
        // 提示处理
        this.setRequestHandler(ListPromptsRequestSchema, this.listPrompts.bind(this));
        this.setRequestHandler(GetPromptRequestSchema, this.getPrompt.bind(this));
    }

    async listTools() {
        return {
            tools: [
                {
                    name: "create_widget",
                    description: "在Wave Terminal工作区中创建新的widget (终端、浏览器等)",
                    inputSchema: {
                        type: "object",
                        properties: {
                            workspace_id: { 
                                type: "string", 
                                description: "工作区ID" 
                            },
                            widget_type: { 
                                type: "string", 
                                description: "Widget类型 (terminal, web, etc.)",
                                enum: ["terminal", "web"]
                            },
                            title: { 
                                type: "string", 
                                description: "Widget标题" 
                            },
                            meta: { 
                                type: "object", 
                                description: "Widget元数据 (如工作目录、URL等)",
                                properties: {
                                    cwd: { type: "string", description: "工作目录" },
                                    url: { type: "string", description: "网页URL" },
                                    env: { type: "object", description: "环境变量" }
                                }
                            }
                        },
                        required: ["workspace_id", "widget_type"]
                    }
                },
                {
                    name: "list_workspaces",
                    description: "列出所有可用的Wave Terminal工作区",
                    inputSchema: {
                        type: "object",
                        properties: {}
                    }
                },
                {
                    name: "get_workspace_by_name",
                    description: "根据名称获取特定工作区的信息",
                    inputSchema: {
                        type: "object",
                        properties: {
                            name: { 
                                type: "string", 
                                description: "工作区名称" 
                            }
                        },
                        required: ["name"]
                    }
                },
                {
                    name: "get_widget_types",
                    description: "获取所有可用的widget类型和它们的配置选项",
                    inputSchema: {
                        type: "object",
                        properties: {}
                    }
                },
                {
                    name: "check_server_status",
                    description: "检查Wave Terminal服务器状态和呼吸灯信息",
                    inputSchema: {
                        type: "object",
                        properties: {}
                    }
                }
            ]
        };
    }

    async callTool(request) {
        const { name, arguments: args } = request.params || request;
        
        try {
            console.error(`[MCP] Calling tool: ${name} with args:`, args);
            
            const headers = {
                'Content-Type': 'application/json'
            };
            
            if (this.authKey) {
                headers['X-AuthKey'] = this.authKey;
            }

            let response;
            let result;

            switch (name) {
                case "create_widget":
                    response = await fetch(`${this.waveTerminalUrl}/api/v1/widgets`, {
                        method: "POST",
                        headers,
                        body: JSON.stringify(args)
                    });
                    result = await response.json();
                    
                    if (response.ok) {
                        return {
                            content: [{
                                type: "text",
                                text: `✅ Widget创建成功!\n\n` +
                                      `类型: ${args.widget_type}\n` +
                                      `标题: ${args.title || 'Untitled'}\n` +
                                      `Block ID: ${result.block_id}\n` +
                                      `工作区: ${args.workspace_id}\n\n` +
                                      `详细信息:\n${JSON.stringify(result, null, 2)}`
                            }]
                        };
                    } else {
                        throw new Error(`API错误: ${response.status} - ${JSON.stringify(result)}`);
                    }

                case "list_workspaces":
                    response = await fetch(`${this.waveTerminalUrl}/api/v1/widgets/workspaces`, {
                        headers
                    });
                    result = await response.json();
                    
                    if (response.ok && result.success) {
                        const workspaceList = result.workspaces.map(ws => 
                            `• ${ws.name} (ID: ${ws.workspace_id})\n  活跃标签: ${ws.active_tab_id}\n  标签数量: ${ws.tab_ids.length}`
                        ).join('\n\n');
                        
                        return {
                            content: [{
                                type: "text",
                                text: `📋 可用工作区 (${result.workspaces.length}个):\n\n${workspaceList}\n\n` +
                                      `💡 使用create_widget工具在这些工作区中创建新的终端或浏览器widget。`
                            }]
                        };
                    } else {
                        throw new Error(`获取工作区列表失败: ${JSON.stringify(result)}`);
                    }

                case "get_workspace_by_name":
                    const encodedName = encodeURIComponent(args.name);
                    response = await fetch(`${this.waveTerminalUrl}/api/v1/widgets/workspace/name/${encodedName}`, {
                        headers
                    });
                    result = await response.json();
                    
                    if (response.ok && result.success) {
                        const ws = result.workspace;
                        return {
                            content: [{
                                type: "text",
                                text: `🔍 工作区详细信息:\n\n` +
                                      `名称: ${ws.name}\n` +
                                      `ID: ${ws.workspace_id}\n` +
                                      `活跃标签: ${ws.active_tab_id}\n` +
                                      `标签列表: ${ws.tab_ids.join(', ')}\n\n` +
                                      `💡 可以使用此工作区ID创建新的widget。`
                            }]
                        };
                    } else {
                        return {
                            content: [{
                                type: "text",
                                text: `❌ 未找到名为 "${args.name}" 的工作区。\n\n使用 list_workspaces 工具查看所有可用工作区。`
                            }]
                        };
                    }

                case "get_widget_types":
                    response = await fetch(`${this.waveTerminalUrl}/api/v1/widgets`, {
                        headers
                    });
                    result = await response.json();
                    
                    if (response.ok && result.success) {
                        const typesList = Object.entries(result.widget_types).map(([type, info]) =>
                            `• **${type}** (${info.name})\n  ${info.description}\n  图标: ${info.icon}\n  配置字段: ${Object.keys(info.meta_fields || {}).join(', ')}`
                        ).join('\n\n');
                        
                        return {
                            content: [{
                                type: "text",
                                text: `🛠️ 可用Widget类型:\n\n${typesList}\n\n` +
                                      `📋 可用API端点:\n${Object.entries(result.endpoints).map(([name, info]) =>
                                          `• ${name}: ${info.method} ${info.path}\n  ${info.description}`
                                      ).join('\n')}`
                            }]
                        };
                    } else {
                        throw new Error(`获取Widget类型失败: ${JSON.stringify(result)}`);
                    }

                case "check_server_status":
                    // 检查基本API是否可用
                    response = await fetch(`${this.waveTerminalUrl}/api/v1/widgets/workspaces`, {
                        headers
                    });
                    
                    const isApiWorking = response.ok;
                    const timestamp = new Date().toLocaleString();
                    
                    return {
                        content: [{
                            type: "text",
                            text: `🔋 Wave Terminal服务器状态报告\n\n` +
                                  `服务器地址: ${this.waveTerminalUrl}\n` +
                                  `API状态: ${isApiWorking ? '✅ 正常' : '❌ 异常'}\n` +
                                  `认证: ${this.authKey ? '🔒 已启用' : '🔓 未启用'}\n` +
                                  `检查时间: ${timestamp}\n\n` +
                                  `💡 状态工具栏中的呼吸灯显示实时服务器状态：\n` +
                                  `• 绿色呼吸 = 服务器运行正常\n` +
                                  `• 黄色闪烁 = 正在检查状态\n` +
                                  `• 红色警告 = 服务器异常\n\n` +
                                  `🤖 MCP客户端显示已连接的服务器数量和状态。`
                        }]
                    };

                default:
                    throw new Error(`未知工具: ${name}`);
            }
        } catch (error) {
            console.error(`[MCP] Tool call error:`, error);
            return {
                content: [{
                    type: "text",
                    text: `❌ 工具调用失败: ${error.message}\n\n请检查Wave Terminal服务器是否正在运行。`
                }],
                isError: true
            };
        }
    }

    async listResources() {
        return {
            resources: [
                {
                    uri: "workspaces://all",
                    name: "所有工作区",
                    description: "Wave Terminal中所有可用工作区的列表",
                    mimeType: "application/json"
                },
                {
                    uri: "widgets://types",
                    name: "Widget类型",
                    description: "所有可用的widget类型及其配置",
                    mimeType: "application/json"
                },
                {
                    uri: "status://server",
                    name: "服务器状态",
                    description: "Wave Terminal服务器当前状态",
                    mimeType: "application/json"
                }
            ]
        };
    }

    async readResource(request) {
        const { uri } = request;
        
        try {
            const headers = {};
            if (this.authKey) {
                headers['X-AuthKey'] = this.authKey;
            }

            let response;
            let result;

            switch (uri) {
                case "workspaces://all":
                    response = await fetch(`${this.waveTerminalUrl}/api/v1/widgets/workspaces`, {
                        headers
                    });
                    result = await response.json();
                    return {
                        contents: [{
                            uri,
                            mimeType: "application/json",
                            text: JSON.stringify(result, null, 2)
                        }]
                    };

                case "widgets://types":
                    response = await fetch(`${this.waveTerminalUrl}/api/v1/widgets`, {
                        headers
                    });
                    result = await response.json();
                    return {
                        contents: [{
                            uri,
                            mimeType: "application/json",
                            text: JSON.stringify(result, null, 2)
                        }]
                    };

                case "status://server":
                    response = await fetch(`${this.waveTerminalUrl}/api/v1/widgets/workspaces`, {
                        headers
                    });
                    const status = {
                        url: this.waveTerminalUrl,
                        isOnline: response.ok,
                        timestamp: new Date().toISOString(),
                        authEnabled: !!this.authKey
                    };
                    return {
                        contents: [{
                            uri,
                            mimeType: "application/json",
                            text: JSON.stringify(status, null, 2)
                        }]
                    };

                default:
                    throw new Error(`未知资源: ${uri}`);
            }
        } catch (error) {
            console.error(`[MCP] Resource read error:`, error);
            return {
                contents: [{
                    uri,
                    mimeType: "text/plain",
                    text: `错误: ${error.message}`
                }]
            };
        }
    }

    async listPrompts() {
        return {
            prompts: [
                {
                    name: "create_terminal_workspace",
                    description: "在指定工作区创建一个新的终端widget",
                    arguments: [
                        {
                            name: "workspace_name",
                            description: "工作区名称",
                            required: true
                        },
                        {
                            name: "title",
                            description: "终端标题",
                            required: false
                        },
                        {
                            name: "directory",
                            description: "工作目录",
                            required: false
                        }
                    ]
                }
            ]
        };
    }

    async getPrompt(request) {
        const { name, arguments: args } = request;
        
        if (name === "create_terminal_workspace") {
            const workspaceName = args?.workspace_name || "default";
            const title = args?.title || "Terminal";
            const directory = args?.directory || "/";
            
            return {
                description: `在工作区 "${workspaceName}" 中创建终端`,
                messages: [
                    {
                        role: "user",
                        content: {
                            type: "text",
                            text: `请在名为 "${workspaceName}" 的工作区中创建一个标题为 "${title}" 的终端widget，工作目录设置为 "${directory}"。`
                        }
                    }
                ]
            };
        }
        
        throw new Error(`未知提示: ${name}`);
    }
}

// 启动MCP服务器
async function main() {
    try {
        const server = new WaveTerminalMCPServer();
        const transport = new StdioServerTransport();
        
        console.error(`[MCP] Connecting transport...`);
        await server.connect(transport);
        console.error(`[MCP] Wave Terminal MCP Server is running!`);
        
        // 优雅关闭处理
        process.on('SIGINT', async () => {
            console.error(`[MCP] Shutting down gracefully...`);
            await server.close();
            process.exit(0);
        });
        
    } catch (error) {
        console.error(`[MCP] Failed to start server:`, error);
        process.exit(1);
    }
}

if (require.main === module) {
    main();
}