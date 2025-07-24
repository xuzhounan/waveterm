#!/usr/bin/env node

/**
 * 检测Wave Terminal开发服务器的配置
 * 用于更新MCP配置
 */

import net from 'net';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// 可能的端口范围
const PORTS_TO_SCAN = [
    57023, 57024, // 当前检测到的端口
    61269, // 默认生产端口
    51920, 50531, 57029, // 常见动态端口
    50396, 50397, 50398, 50399, // 开发版本常用端口范围
    52020, 52021, 52022, 52023, // 新的开发端口范围
    53075, 53076, 53077, 53078, // 最新开发端口范围
    65224, 65225, 65226, 65227, // 另一个开发端口范围
    8080, 3000, // 开发服务器端口
];

async function checkPort(port) {
    return new Promise((resolve) => {
        const socket = new net.Socket();
        
        socket.setTimeout(1000);
        socket.on('connect', () => {
            socket.destroy();
            resolve(true);
        });
        
        socket.on('timeout', () => {
            socket.destroy();
            resolve(false);
        });
        
        socket.on('error', () => {
            resolve(false);
        });
        
        socket.connect(port, 'localhost');
    });
}

async function testWaveAPI(port, authKey = null) {
    try {
        // 临时禁用代理
        const originalProxy = process.env.http_proxy;
        const originalHttpsProxy = process.env.https_proxy;
        const originalAllProxy = process.env.all_proxy;
        
        delete process.env.http_proxy;
        delete process.env.https_proxy;
        delete process.env.all_proxy;
        delete process.env.HTTP_PROXY;
        delete process.env.HTTPS_PROXY;
        delete process.env.ALL_PROXY;
        
        const fetch = global.fetch || (await import('node-fetch')).default;
        const headers = {
            'Content-Type': 'application/json'
        };
        
        if (authKey) {
            headers['X-AuthKey'] = authKey;
        }
        
        const response = await fetch(`http://localhost:${port}/api/v1/widgets/workspaces`, {
            method: 'GET',
            headers,
            timeout: 3000
        });
        
        if (response.ok) {
            const data = await response.json();
            
            // 恢复代理设置
            if (originalProxy) process.env.http_proxy = originalProxy;
            if (originalHttpsProxy) process.env.https_proxy = originalHttpsProxy;
            if (originalAllProxy) process.env.all_proxy = originalAllProxy;
            
            return { success: true, data };
        } else {
            // 恢复代理设置
            if (originalProxy) process.env.http_proxy = originalProxy;
            if (originalHttpsProxy) process.env.https_proxy = originalHttpsProxy;
            if (originalAllProxy) process.env.all_proxy = originalAllProxy;
            
            return { success: false, status: response.status };
        }
    } catch (error) {
        // 恢复代理设置
        if (originalProxy) process.env.http_proxy = originalProxy;
        if (originalHttpsProxy) process.env.https_proxy = originalHttpsProxy;
        if (originalAllProxy) process.env.all_proxy = originalAllProxy;
        
        return { success: false, error: error.message };
    }
}

async function readAuthKey() {
    try {
        // 尝试从开发环境的配置读取
        const devDataDir = path.join(process.env.HOME, 'Library/Application Support/waveterm-dev');
        const lockFile = path.join(devDataDir, 'wave.lock');
        
        if (fs.existsSync(lockFile)) {
            // 可以尝试从lock文件或其他配置中读取
            console.log(`Found dev data dir: ${devDataDir}`);
        }
        
        // 也可以尝试从生产环境读取
        const prodDataDir = path.join(process.env.HOME, 'Library/Application Support/waveterm');
        const prodLockFile = path.join(prodDataDir, 'wave.lock');
        
        if (fs.existsSync(prodLockFile)) {
            console.log(`Found prod data dir: ${prodDataDir}`);
        }
        
    } catch (error) {
        console.error('Error reading auth key:', error.message);
    }
    
    return null;
}

async function detectWaveConfig() {
    console.log('🔍 Detecting Wave Terminal server configuration...');
    
    const authKey = await readAuthKey();
    
    for (const port of PORTS_TO_SCAN) {
        console.log(`  Checking port ${port}...`);
        
        const isPortOpen = await checkPort(port);
        if (isPortOpen) {
            console.log(`  ✅ Port ${port} is open`);
            
            // 测试API
            const apiResult = await testWaveAPI(port, authKey);
            if (apiResult.success) {
                console.log(`  ✅ Wave API responding on port ${port}`);
                console.log(`  📊 Found ${apiResult.data?.length || 0} workspaces`);
                
                return {
                    port,
                    url: `http://localhost:${port}`,
                    authKey,
                    workspaces: apiResult.data
                };
            } else {
                console.log(`  ❌ Wave API not responding: ${apiResult.error || apiResult.status}`);
            }
        }
    }
    
    console.log('❌ No Wave Terminal server found');
    return null;
}

async function main() {
    try {
        const config = await detectWaveConfig();
        
        if (config) {
            console.log('\n🎯 Wave Terminal Configuration:');
            console.log(`  URL: ${config.url}`);
            console.log(`  Auth Key: ${config.authKey ? 'Present' : 'None'}`);
            console.log(`  Workspaces: ${config.workspaces?.length || 0}`);
            
            // 输出MCP配置格式
            console.log('\n📋 MCP Configuration:');
            console.log(JSON.stringify({
                "wave-terminal": {
                    "command": "node",
                    "args": ["/Users/xzn/Desktop/code-project/waveterm/mcp-bridge.cjs"],
                    "env": {
                        "WAVE_TERMINAL_URL": config.url,
                        ...(config.authKey && { "WAVE_TERMINAL_AUTH_KEY": config.authKey })
                    }
                }
            }, null, 2));
            
        } else {
            console.log('\n❌ Could not detect Wave Terminal server');
            console.log('Make sure Wave Terminal is running and try again.');
        }
        
    } catch (error) {
        console.error('Error:', error.message);
        process.exit(1);
    }
}

if (import.meta.url === `file://${process.argv[1]}`) {
    main();
}

export { detectWaveConfig };