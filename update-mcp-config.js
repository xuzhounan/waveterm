#!/usr/bin/env node

/**
 * 动态更新Claude Code的MCP配置
 * 自动检测Wave Terminal服务器端口并更新配置文件
 */

import { detectWaveConfig } from './detect-wave-config.js';
import fs from 'fs';
import path from 'path';

const MCP_CONFIG_PATHS = [
    path.join(process.env.HOME, 'Library/Application Support/Claude/claude_desktop_config.json'),
    path.join(process.env.HOME, '.config/claude-desktop/claude_desktop_config.json')
];

async function updateMCPConfig() {
    console.log('🔄 Updating Claude Code MCP configuration...');
    
    // 检测Wave配置
    const waveConfig = await detectWaveConfig();
    
    if (!waveConfig) {
        console.log('❌ Could not detect Wave Terminal server');
        console.log('💡 Try starting Wave Terminal development server with:');
        console.log('   WCLOUD_ENDPOINT="https://wcloud.commandline.com" WCLOUD_WS_ENDPOINT="wss://wcloud.commandline.com/ws" npm run dev');
        return false;
    }
    
    console.log(`✅ Detected Wave Terminal at ${waveConfig.url}`);
    
    // 准备MCP配置
    const mcpConfig = {
        "wave-terminal": {
            "command": "node",
            "args": ["/Users/xzn/Desktop/code-project/waveterm/mcp-bridge.cjs"],
            "env": {
                "WAVE_TERMINAL_URL": waveConfig.url,
                ...(waveConfig.authKey && { "WAVE_TERMINAL_AUTH_KEY": waveConfig.authKey })
            }
        }
    };
    
    let updatedConfigs = 0;
    
    // 更新每个配置文件
    for (const configPath of MCP_CONFIG_PATHS) {
        try {
            if (!fs.existsSync(configPath)) {
                // 创建配置文件
                const dir = path.dirname(configPath);
                if (!fs.existsSync(dir)) {
                    fs.mkdirSync(dir, { recursive: true });
                }
                
                const newConfig = {
                    mcpServers: mcpConfig
                };
                
                fs.writeFileSync(configPath, JSON.stringify(newConfig, null, 2), 'utf8');
                console.log(`✅ Created new config: ${configPath}`);
                updatedConfigs++;
            } else {
                // 更新现有配置文件
                const existingConfig = JSON.parse(fs.readFileSync(configPath, 'utf8'));
                
                if (!existingConfig.mcpServers) {
                    existingConfig.mcpServers = {};
                }
                
                // 更新wave-terminal配置
                existingConfig.mcpServers['wave-terminal'] = mcpConfig['wave-terminal'];
                
                fs.writeFileSync(configPath, JSON.stringify(existingConfig, null, 2), 'utf8');
                console.log(`✅ Updated config: ${configPath}`);
                updatedConfigs++;
            }
        } catch (error) {
            console.error(`❌ Failed to update ${configPath}:`, error.message);
        }
    }
    
    if (updatedConfigs > 0) {
        console.log(`\n🎯 Successfully updated ${updatedConfigs} MCP configuration file(s)`);
        console.log('💡 Restart Claude Code to apply the changes');
        console.log('\n📋 Wave Terminal MCP Configuration:');
        console.log(JSON.stringify(mcpConfig, null, 2));
        return true;
    } else {
        console.log('❌ No configuration files were updated');
        return false;
    }
}

async function main() {
    try {
        const success = await updateMCPConfig();
        process.exit(success ? 0 : 1);
    } catch (error) {
        console.error('Error:', error.message);
        process.exit(1);
    }
}

if (import.meta.url === `file://${process.argv[1]}`) {
    main();
}

export { updateMCPConfig };