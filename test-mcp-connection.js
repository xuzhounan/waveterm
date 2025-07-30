#!/usr/bin/env node

// 测试 MCP bridge 连接
async function testConnection() {
    // 清除代理设置
    delete process.env.http_proxy;
    delete process.env.https_proxy;
    delete process.env.HTTP_PROXY;
    delete process.env.HTTPS_PROXY;
    
    console.log('Testing Wave Terminal API connection without proxy...');
    
    try {
        const response = await fetch('http://127.0.0.1:8090/api/v1/widgets/workspaces');
        if (response.ok) {
            const data = await response.json();
            console.log('✅ Connection successful!');
            console.log(`Found ${data.workspaces?.length || 0} workspaces`);
            return true;
        } else {
            console.log('❌ HTTP error:', response.status, response.statusText);
            return false;
        }
    } catch (error) {
        console.log('❌ Connection failed:', error.message);
        return false;
    }
}

// 测试创建 widget
async function testCreateWidget() {
    // 清除代理设置
    delete process.env.http_proxy;
    delete process.env.https_proxy;
    delete process.env.HTTP_PROXY;
    delete process.env.HTTPS_PROXY;
    
    console.log('Testing widget creation...');
    
    try {
        // 首先获取一个工作区
        const workspacesResponse = await fetch('http://127.0.0.1:8090/api/v1/widgets/workspaces');
        const workspacesData = await workspacesResponse.json();
        
        if (!workspacesData.success || !workspacesData.workspaces?.length) {
            console.log('❌ No workspaces found');
            return false;
        }
        
        const workspace = workspacesData.workspaces[0];
        console.log(`Using workspace: ${workspace.name} (${workspace.workspace_id})`);
        
        // 创建一个测试 widget
        const createResponse = await fetch('http://127.0.0.1:8090/api/v1/widgets', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                workspace_id: workspace.workspace_id,
                widget_type: 'web',
                url: 'https://github.com'
            })
        });
        
        if (createResponse.ok) {
            const result = await createResponse.json();
            console.log('✅ Widget creation successful!');
            console.log('Widget ID:', result.widget_id);
            return true;
        } else {
            const errorText = await createResponse.text();
            console.log('❌ Widget creation failed:', createResponse.status, errorText);
            return false;
        }
    } catch (error) {
        console.log('❌ Widget creation error:', error.message);
        return false;
    }
}

async function main() {
    console.log('=== MCP Connection Test ===');
    
    const connectionOk = await testConnection();
    if (connectionOk) {
        await testCreateWidget();
    }
    
    console.log('=== Test Complete ===');
}

main().catch(console.error);