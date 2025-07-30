#!/usr/bin/env node

// 测试直接 MCP 调用
async function testDirectCall() {
    console.log('Testing direct Wave Terminal API call...');
    
    // 保存原始代理设置
    const originalProxy = {
        http_proxy: process.env.http_proxy,
        https_proxy: process.env.https_proxy,
        HTTP_PROXY: process.env.HTTP_PROXY,
        HTTPS_PROXY: process.env.HTTPS_PROXY
    };
    
    console.log('Original proxy settings:', originalProxy);
    
    // 临时清除代理设置
    delete process.env.http_proxy;
    delete process.env.https_proxy;
    delete process.env.HTTP_PROXY;
    delete process.env.HTTPS_PROXY;
    
    console.log('Proxy cleared. Testing API call...');
    
    try {
        const response = await fetch('http://127.0.0.1:8090/api/v1/widgets/workspaces');
        const result = await response.json();
        console.log('Success! Response:', JSON.stringify(result, null, 2));
    } catch (error) {
        console.error('Error:', error.message);
    } finally {
        // 恢复代理设置
        if (originalProxy.http_proxy) process.env.http_proxy = originalProxy.http_proxy;
        if (originalProxy.https_proxy) process.env.https_proxy = originalProxy.https_proxy;
        if (originalProxy.HTTP_PROXY) process.env.HTTP_PROXY = originalProxy.HTTP_PROXY;
        if (originalProxy.HTTPS_PROXY) process.env.HTTPS_PROXY = originalProxy.HTTPS_PROXY;
        
        console.log('Proxy settings restored.');
    }
}

testDirectCall().catch(console.error);