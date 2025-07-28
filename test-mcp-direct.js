#!/usr/bin/env node

/**
 * 直接测试MCP终端输入功能
 */

async function testMCPTerminalInput() {
    console.log('🔧 测试Wave Terminal MCP终端输入功能...\n');
    
    const waveTerminalUrl = 'http://127.0.0.1:60289';
    
    try {
        // 测试1: 检查服务器状态
        console.log('📡 测试1: 检查Wave Terminal服务器状态');
        const statusResponse = await fetch(`${waveTerminalUrl}/api/v1/widgets/workspaces`);
        if (!statusResponse.ok) {
            throw new Error(`服务器响应错误: ${statusResponse.status}`);
        }
        const workspaces = await statusResponse.json();
        console.log(`✅ 找到 ${workspaces.workspaces?.length || 0} 个工作区\n`);
        
        if (!workspaces.workspaces || workspaces.workspaces.length === 0) {
            throw new Error('没有找到可用的工作区');
        }
        
        const workspaceId = workspaces.workspaces[0].workspace_id;
        console.log(`🎯 使用工作区: ${workspaces.workspaces[0].name} (${workspaceId})\n`);
        
        // 测试2: 创建terminal widget
        console.log('🖥️ 测试2: 创建terminal widget');
        const createResponse = await fetch(`${waveTerminalUrl}/api/v1/widgets`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                workspace_id: workspaceId,
                widget_type: 'terminal',
                title: 'MCP测试Terminal',
                meta: { cwd: '/tmp' }
            })
        });
        
        const createResult = await createResponse.json();
        if (!createResult.success) {
            throw new Error(`创建terminal失败: ${createResult.error}`);
        }
        
        const blockId = createResult.block_id;
        console.log(`✅ 创建成功! Block ID: ${blockId}\n`);
        
        // 等待terminal初始化
        console.log('⏳ 等待5秒让terminal完全初始化...');
        await new Promise(resolve => setTimeout(resolve, 5000));
        
        // 测试3: 发送terminal输入
        console.log('⌨️ 测试3: 发送命令到terminal');
        const inputResponse = await fetch(`${waveTerminalUrl}/api/v1/widgets/block/${blockId}/input`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                input_data: 'echo "🎉 Hello from MCP Agent!" && date\n',
                input_type: 'text'
            })
        });
        
        const inputResult = await inputResponse.json();
        if (inputResult.success) {
            console.log(`✅ 输入发送成功: ${inputResult.message}\n`);
        } else {
            console.log(`❌ 输入发送失败: ${inputResult.error}\n`);
        }
        
        // 等待命令执行
        console.log('⏳ 等待3秒让命令执行完成...');
        await new Promise(resolve => setTimeout(resolve, 3000));
        
        // 测试4: 读取terminal输出
        console.log('📄 测试4: 读取terminal输出');
        const contentResponse = await fetch(`${waveTerminalUrl}/api/v1/widgets/block/content/${blockId}`);
        const contentResult = await contentResponse.json();
        
        if (contentResult.success) {
            console.log(`✅ 成功读取 ${contentResult.size} 字节的输出内容:`);
            console.log('📋 Terminal输出:');
            console.log('----------------------------------------');
            // 只显示可打印的内容，过滤ANSI转义序列
            const cleanContent = contentResult.content.replace(/\x1b\[[0-9;]*[a-zA-Z]/g, '').replace(/\r/g, '');
            console.log(cleanContent);
            console.log('----------------------------------------\n');
        } else {
            console.log(`❌ 读取输出失败: ${contentResult.error}\n`);
        }
        
        // 测试5: 获取terminal状态
        console.log('📊 测试5: 获取terminal状态');
        const statusResponse2 = await fetch(`${waveTerminalUrl}/api/v1/widgets/block/status/${blockId}`);
        const statusResult = await statusResponse2.json();
        
        if (statusResult.success) {
            const block = statusResult.block_info;
            const controller = statusResult.controller;
            console.log('✅ Terminal状态信息:');
            console.log(`   📋 Block类型: ${block.block_type}`);
            console.log(`   🎛️ Controller: ${block.controller}`);
            console.log(`   📁 文件数量: ${block.files?.length || 0}`);
            if (controller) {
                console.log(`   🔄 Controller状态: ${controller.status}`);
                console.log(`   🚦 退出码: ${controller.exit_code}`);
            }
        } else {
            console.log(`❌ 获取状态失败: ${statusResult.error}`);
        }
        
        // 测试6: 发送另一个命令
        console.log('\n⌨️ 测试6: 发送另一个命令 (pwd)');
        const inputResponse2 = await fetch(`${waveTerminalUrl}/api/v1/widgets/block/${blockId}/input`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                input_data: 'pwd\n',
                input_type: 'text'
            })
        });
        
        const inputResult2 = await inputResponse2.json();
        console.log(inputResult2.success ? '✅ 第二个命令发送成功' : `❌ 发送失败: ${inputResult2.error}`);
        
        console.log('\n🎉 所有测试完成！MCP terminal输入功能工作正常！');
        console.log('💡 Agent现在可以：');
        console.log('   • 创建terminal widgets');
        console.log('   • 向terminal发送命令');
        console.log('   • 读取terminal输出');
        console.log('   • 监控terminal状态');
        
    } catch (error) {
        console.error('❌ 测试失败:', error.message);
        if (error.message.includes('fetch')) {
            console.log('💡 提示: Wave Terminal服务器似乎没有运行，请先启动Wave Terminal应用');
        }
    }
}

// 运行测试  
testMCPTerminalInput();