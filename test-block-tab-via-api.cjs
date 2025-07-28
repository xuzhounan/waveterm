#!/usr/bin/env node

const http = require('http');

// Configuration
const HOST = '127.0.0.1';
const PORT = 51362; // Wave Terminal port (from lsof output)

function makeRequest(path, method = 'GET', data = null) {
    return new Promise((resolve, reject) => {
        const options = {
            hostname: HOST,
            port: PORT,
            path: path,
            method: method,
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            }
        };

        const req = http.request(options, (res) => {
            let responseData = '';
            
            res.on('data', (chunk) => {
                responseData += chunk;
            });
            
            res.on('end', () => {
                try {
                    const parsed = JSON.parse(responseData);
                    resolve({ status: res.statusCode, data: parsed });
                } catch (e) {
                    resolve({ status: res.statusCode, data: responseData });
                }
            });
        });

        req.on('error', (err) => {
            reject(err);
        });

        if (data) {
            req.write(JSON.stringify(data));
        }
        
        req.end();
    });
}

async function main() {
    try {
        console.log('🧪 Testing Block Tab ID Fix via REST API');
        console.log('='.repeat(50));
        
        // 1. List workspaces
        console.log('\n1. 📋 Listing workspaces...');
        const workspacesResp = await makeRequest('/api/v1/widgets/workspaces');
        
        if (workspacesResp.status !== 200 || !workspacesResp.data.success) {
            console.log('❌ Failed to list workspaces:', workspacesResp.data);
            return;
        }
        
        const workspaces = workspacesResp.data.workspaces;
        if (workspaces.length === 0) {
            console.log('❌ No workspaces found');
            return;
        }
        
        const workspace = workspaces[0];
        console.log(`✅ Found workspace: ${workspace.name} (ID: ${workspace.workspace_id})`);
        console.log(`   Total tabs: ${workspace.total_tabs}, Total blocks: ${workspace.total_blocks}`);
        
        if (workspace.tabs_info.length === 0) {
            console.log('❌ No tabs found in workspace');
            return;
        }
        
        const tab = workspace.tabs_info[0];
        console.log(`✅ Using tab: ${tab.name} (ID: ${tab.tab_id})`);
        
        // 2. List blocks in the tab
        console.log('\n2. 📦 Listing blocks in tab...');
        const blocksResp = await makeRequest(`/api/v1/widgets/blocks?tab_id=${tab.tab_id}`);
        
        if (blocksResp.status !== 200 || !blocksResp.data.success) {
            console.log('❌ Failed to list blocks:', blocksResp.data);
            return;
        }
        
        const blocks = blocksResp.data.blocks;
        console.log(`✅ Found ${blocks.length} blocks in tab`);
        
        if (blocks.length === 0) {
            console.log('⚠️  No blocks found in tab');
            return;
        }
        
        // 3. Test each block
        console.log('\n3. 🔍 Testing block Tab ID associations...');
        let correctBlocks = 0;
        let totalBlocks = blocks.length;
        
        for (let i = 0; i < blocks.length; i++) {
            const block = blocks[i];
            console.log(`\n   Block ${i + 1}:`);
            console.log(`     Block ID: ${block.block_id}`);
            console.log(`     Tab ID: ${block.tab_id}`);
            console.log(`     Workspace ID: ${block.workspace_id}`);
            console.log(`     Block Type: ${block.block_type}`);
            console.log(`     View: ${block.view}`);
            console.log(`     Controller: ${block.controller}`);
            
            // Check Tab ID
            if (!block.tab_id) {
                console.log('     ❌ ERROR: Tab ID is empty!');
            } else if (block.tab_id === tab.tab_id) {
                console.log('     ✅ SUCCESS: Tab ID correctly set');
                correctBlocks++;
            } else {
                console.log(`     ⚠️  WARNING: Tab ID mismatch (expected: ${tab.tab_id}, got: ${block.tab_id})`);
            }
            
            // Check Workspace ID
            if (!block.workspace_id) {
                console.log('     ❌ ERROR: Workspace ID is empty!');
            } else if (block.workspace_id === workspace.workspace_id) {
                console.log('     ✅ SUCCESS: Workspace ID correctly set');
            } else {
                console.log(`     ⚠️  WARNING: Workspace ID mismatch (expected: ${workspace.workspace_id}, got: ${block.workspace_id})`);
            }
        }
        
        // 4. Test GetBlockStatus for first block
        if (blocks.length > 0) {
            console.log('\n4. 📊 Testing GetBlockStatus...');
            const testBlock = blocks[0];
            const statusResp = await makeRequest(`/api/v1/widgets/block/status/${testBlock.block_id}`);
            
            if (statusResp.status !== 200 || !statusResp.data.success) {
                console.log('❌ GetBlockStatus failed:', statusResp.data);
            } else {
                console.log('✅ GetBlockStatus succeeded');
                const blockInfo = statusResp.data.block_info;
                console.log(`   Block ID: ${blockInfo.block_id}`);
                console.log(`   Tab ID: ${blockInfo.tab_id}`);
                console.log(`   Workspace ID: ${blockInfo.workspace_id}`);
                
                if (blockInfo.tab_id === tab.tab_id) {
                    console.log('   ✅ Tab ID correct in GetBlockStatus');
                } else {
                    console.log('   ❌ Tab ID incorrect in GetBlockStatus');
                }
            }
        }
        
        // 5. Summary
        console.log('\n' + '='.repeat(50));
        console.log('📋 TEST SUMMARY');
        console.log('='.repeat(50));
        console.log(`Total blocks tested: ${totalBlocks}`);
        console.log(`Blocks with correct Tab ID: ${correctBlocks}`);
        
        if (correctBlocks === totalBlocks && totalBlocks > 0) {
            console.log('🎉 SUCCESS: All blocks have correct Tab ID associations!');
            console.log('✅ The fix is working correctly.');
        } else {
            console.log('❌ FAILURE: Some blocks still have missing or incorrect Tab ID associations');
            console.log('🔧 The fix may need more work.');
        }
        
    } catch (error) {
        console.error('❌ Error during testing:', error);
        console.log('\n💡 Make sure Wave Terminal server is running on port', PORT);
        console.log('   You can start it with: ./wavesrv-test');
    }
}

main();