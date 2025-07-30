#!/usr/bin/env node

// 测试无代理环境下的 MCP bridge
process.env.http_proxy = '';
process.env.https_proxy = '';
process.env.HTTP_PROXY = '';
process.env.HTTPS_PROXY = '';
process.env.WAVE_TERMINAL_URL = 'http://127.0.0.1:8090';

console.log('Testing MCP bridge with no proxy...');
console.log('Environment variables:');
console.log('- http_proxy:', process.env.http_proxy);
console.log('- https_proxy:', process.env.https_proxy);
console.log('- WAVE_TERMINAL_URL:', process.env.WAVE_TERMINAL_URL);

// 导入 MCP bridge
try {
    require('./mcp-bridge.cjs');
} catch (error) {
    console.error('Error loading MCP bridge:', error);
}