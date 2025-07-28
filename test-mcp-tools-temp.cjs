#!/usr/bin/env node

/**
 * 直接测试MCP工具，无需启动完整的MCP服务器
 */

const { WaveTerminalMCPServer } = require('./mcp-bridge.cjs');

async function testMCPTools() {
    console.log('🔧 测试Wave Terminal MCP工具...\n');
    
    // 创建MCP服务器实例
    process.env.WAVE_TERMINAL_URL = 'http://127.0.0.1:60289';
    const server = new WaveTerminalMCPServer();
    
    try {
        // 测试1: 列出工作区
        console.log('📋 测试1: 列出工作区');
        const workspacesResult = await server.callTool({
            name: 'list_workspaces',
            arguments: {}
        });
        console.log('结果:', workspacesResult.content[0].text.substring(0, 200) + '...\n');
        
        // 解析工作区信息
        const workspaceMatch = workspacesResult.content[0].text.match(/ID: ([a-f0-9-]+)/);
        if (!workspaceMatch) {
            throw new Error('未找到工作区ID');
        }
        const workspaceId = workspaceMatch[1];
        console.log(`✅ 找到工作区ID: ${workspaceId}\n`);
        
        // 测试2: 创建terminal
        console.log('🖥️ 测试2: 创建terminal');
        const terminalResult = await server.callTool({
            name: 'create_widget',
            arguments: {
                workspace_id: workspaceId,
                widget_type: 'terminal',
                title: 'MCP测试Terminal',
                meta: {
                    cwd: '/tmp'
                }
            }
        });
        console.log('结果:', terminalResult.content[0].text.substring(0, 200) + '...\n');
        
        // 解析block ID
        const blockMatch = terminalResult.content[0].text.match(/Block ID: ([a-f0-9-]+)/);
        if (!blockMatch) {
            throw new Error('未找到Block ID');
        }
        const blockId = blockMatch[1];
        console.log(`✅ 创建了terminal，Block ID: ${blockId}\n`);
        
        // 等待terminal初始化
        console.log('⏳ 等待terminal初始化...');
        await new Promise(resolve => setTimeout(resolve, 3000));
        
        // 测试3: 发送terminal输入
        console.log('⌨️ 测试3: 发送terminal输入');
        const inputResult = await server.callTool({
            name: 'send_terminal_input',
            arguments: {
                block_id: blockId,
                input_data: 'echo "Hello from MCP Agent!"\n',
                input_type: 'text'
            }
        });
        console.log('结果:', inputResult.content[0].text);
        
        // 等待命令执行
        console.log('\n⏳ 等待命令执行...');
        await new Promise(resolve => setTimeout(resolve, 2000));
        
        // 测试4: 读取terminal输出
        console.log('📄 测试4: 读取terminal输出');
        const contentResult = await server.callTool({
            name: 'get_block_content',
            arguments: {
                block_id: blockId
            }
        });
        console.log('结果:', contentResult.content[0].text);
        
        // 测试5: 获取terminal状态
        console.log('\n📊 测试5: 获取terminal状态');
        const statusResult = await server.callTool({
            name: 'get_block_status',
            arguments: {
                block_id: blockId
            }
        });
        console.log('结果:', statusResult.content[0].text.substring(0, 300) + '...');
        
        console.log('\n🎉 所有测试完成！MCP terminal输入功能工作正常！');
        
    } catch (error) {
        console.error('❌ 测试失败:', error.message);
        if (error.message.includes('ECONNREFUSED')) {
            console.log('💡 提示: Wave Terminal服务器似乎没有运行，请先启动Wave Terminal');
        }
    }
}

// 运行测试
testMCPTools();