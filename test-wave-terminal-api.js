#!/usr/bin/env node

/**
 * 直接测试Wave Terminal API的完整流程
 * 1. 列出工作区
 * 2. 创建terminal widget
 * 3. 发送echo命令
 * 4. 读取terminal输出
 */

async function testWaveTerminalAPI() {
    const baseUrl = 'http://127.0.0.1:60289';
    
    console.log('🔧 测试Wave Terminal API完整流程...\n');
    
    try {
        // 测试1: 列出工作区
        console.log('📋 测试1: 列出工作区');
        const workspacesResponse = await fetch(`${baseUrl}/api/v1/widgets/workspaces`);
        const workspacesData = await workspacesResponse.json();
        
        if (!workspacesData.success || !workspacesData.workspaces || workspacesData.workspaces.length === 0) {
            throw new Error('未找到工作区');
        }
        
        const workspace = workspacesData.workspaces[0];
        console.log(`✅ 找到工作区: ${workspace.name} (ID: ${workspace.workspace_id})\n`);
        
        // 测试2: 创建terminal widget
        console.log('🖥️ 测试2: 创建terminal widget');
        const createResponse = await fetch(`${baseUrl}/api/v1/widgets`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                workspace_id: workspace.workspace_id,
                widget_type: 'terminal',
                title: 'API测试Terminal',
                meta: {
                    cwd: '/tmp'
                }
            })
        });
        
        const createData = await createResponse.json();
        if (!createData.success) {
            throw new Error(`创建terminal失败: ${createData.error || '未知错误'}`);
        }
        
        const blockId = createData.block_id;
        console.log(`✅ 创建了terminal，Block ID: ${blockId}\n`);
        
        // 等待terminal初始化（重试机制）
        console.log('⏳ 等待terminal初始化...');
        
        let inputSuccess = false;
        let retries = 0;
        const maxRetries = 8;
        
        while (retries < maxRetries && !inputSuccess) {
            console.log(`尝试 ${retries + 1}/${maxRetries}: 等待 ${2 + retries} 秒...`);
            await new Promise(resolve => setTimeout(resolve, (2 + retries) * 1000));
            
            try {
                // 尝试发送简单测试命令
                const testResponse = await fetch(`${baseUrl}/api/v1/widgets/block/${blockId}/input`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        input_data: 'echo ready\n',
                        input_type: 'text'
                    })
                });
                
                const testData = await testResponse.json();
                if (testData.success) {
                    console.log('✅ Terminal已准备就绪\n');
                    inputSuccess = true;
                    // 等待测试命令执行完成
                    await new Promise(resolve => setTimeout(resolve, 1000));
                    break;
                }
            } catch (error) {
                console.log(`尝试 ${retries + 1} 失败: ${error.message}`);
            }
            
            retries++;
        }
        
        if (!inputSuccess) {
            throw new Error('Terminal初始化超时，无法发送命令');
        }
        
        // 测试3: 发送echo命令到terminal
        console.log('⌨️ 测试3: 发送echo命令到terminal');
        const inputResponse = await fetch(`${baseUrl}/api/v1/widgets/block/${blockId}/input`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                input_data: 'echo "Hello from Wave Terminal API test!"\n',
                input_type: 'text'
            })
        });
        
        const inputData = await inputResponse.json();
        if (!inputData.success) {
            throw new Error(`发送命令失败: ${inputData.error || '未知错误'}`);
        }
        
        console.log('✅ 命令发送成功\n');
        
        // 等待命令执行
        console.log('⏳ 等待命令执行...');
        await new Promise(resolve => setTimeout(resolve, 2000));
        
        // 测试4: 读取terminal输出
        console.log('📄 测试4: 读取terminal输出');
        const contentResponse = await fetch(`${baseUrl}/api/v1/widgets/block/${blockId}/content?file_name=term&size=0`);
        const contentData = await contentResponse.json();
        
        if (!contentData.success) {
            throw new Error(`读取输出失败: ${contentData.error || '未知错误'}`);
        }
        
        console.log('✅ Terminal输出:');
        console.log('---');
        console.log(contentData.content || '(无输出)');
        console.log('---\n');
        
        console.log('🎉 所有测试完成！Wave Terminal API功能工作正常！');
        
        return {
            success: true,
            workspace_id: workspace.workspace_id,
            workspace_name: workspace.name,
            block_id: blockId,
            output: contentData.content
        };
        
    } catch (error) {
        console.error('❌ 测试失败:', error.message);
        if (error.message.includes('fetch') || error.message.includes('ECONNREFUSED')) {
            console.log('💡 提示: Wave Terminal服务器似乎没有运行，请先启动服务器');
        }
        return { success: false, error: error.message };
    }
}

// 运行测试
testWaveTerminalAPI().then(result => {
    if (result.success) {
        console.log('\n📊 测试结果总结:');
        console.log(`- 工作区: ${result.workspace_name}`);
        console.log(`- 工作区ID: ${result.workspace_id}`);
        console.log(`- Terminal Block ID: ${result.block_id}`);
        console.log(`- 输出长度: ${result.output ? result.output.length : 0} 字符`);
    }
    process.exit(result.success ? 0 : 1);
});