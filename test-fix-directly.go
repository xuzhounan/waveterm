package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

// Test our fix directly by setting WAVETERM_HOME to use the server's database
func main() {
	// Set the wave home directory to the server's directory
	os.Setenv("WAVETERM_HOME", "/Users/xzn/Library/Application Support/waveterm")
	
	// Initialize database
	err := wstore.InitWStore()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to initialize wstore: %w", err))
	}
	
	// 添加更新上下文
	ctx := waveobj.ContextWithUpdates(context.Background())
	
	// Create service instance
	service := &widgetapiservice.WidgetAPIService{}
	
	// Test the specific block that was showing empty Tab ID
	blockId := "e0eb22cf-a08f-4df2-928c-43f23af8e99c"
	
	fmt.Printf("🧪 Testing Block Tab ID Fix\n")
	fmt.Printf("===========================\n")
	fmt.Printf("Block ID: %s\n", blockId)
	
	// 1. Get the raw block from database
	fmt.Printf("\n1. 📦 Getting raw block from database...\n")
	block, err := wstore.DBGet[*waveobj.Block](ctx, blockId)
	if err != nil || block == nil {
		log.Fatal(fmt.Errorf("failed to get block: %w", err))
	}
	
	fmt.Printf("   Raw ParentORef: '%s'\n", block.ParentORef)
	
	// Parse the ParentORef manually
	parentORef := waveobj.ParseORefNoErr(block.ParentORef)
	if parentORef != nil {
		fmt.Printf("   Parsed ParentORef: Type=%s, ID=%s\n", parentORef.OType, parentORef.OID)
	} else {
		fmt.Printf("   ❌ Failed to parse ParentORef\n")
		return
	}
	
	// 2. Test GetBlockStatus with our fix
	fmt.Printf("\n2. 🔍 Testing GetBlockStatus with our fix...\n")
	statusResp, err := service.GetBlockStatus(ctx, blockId)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to get block status: %w", err))
	}
	
	if !statusResp.Success {
		fmt.Printf("❌ GetBlockStatus failed: %s\n", statusResp.Error)
		return
	}
	
	blockInfo := statusResp.BlockInfo
	fmt.Printf("   Block ID: %s\n", blockInfo.BlockId)
	fmt.Printf("   Tab ID from fix: '%s'\n", blockInfo.TabId)
	fmt.Printf("   Workspace ID from fix: '%s'\n", blockInfo.WorkspaceId)
	
	// 3. Verify the fix worked
	fmt.Printf("\n3. ✅ Verification:\n")
	expectedTabId := parentORef.OID
	if blockInfo.TabId == expectedTabId {
		fmt.Printf("   ✅ SUCCESS: Tab ID correctly extracted from ParentORef\n")
		fmt.Printf("   Expected: %s, Got: %s\n", expectedTabId, blockInfo.TabId)
	} else {
		fmt.Printf("   ❌ FAILURE: Tab ID not correctly extracted\n")
		fmt.Printf("   Expected: %s, Got: %s\n", expectedTabId, blockInfo.TabId)
	}
	
	if blockInfo.WorkspaceId != "" {
		fmt.Printf("   ✅ SUCCESS: Workspace ID correctly populated\n")
		fmt.Printf("   Workspace ID: %s\n", blockInfo.WorkspaceId)
	} else {
		fmt.Printf("   ❌ FAILURE: Workspace ID still empty\n")
	}
	
	// 4. Test with multiple blocks
	fmt.Printf("\n4. 🔍 Testing ListBlocks API...\n")
	
	// First get the workspace for the tab
	workspaceId, err := wstore.DBFindWorkspaceForTabId(ctx, expectedTabId)
	if err != nil {
		fmt.Printf("❌ Failed to find workspace for tab: %v\n", err)
		return
	}
	
	fmt.Printf("   Found workspace: %s\n", workspaceId)
	
	// List all blocks in the tab
	listReq := widgetapiservice.ListBlocksAPIRequest{
		TabId: expectedTabId,
	}
	
	blocksResp, err := service.ListBlocks(ctx, listReq)
	if err != nil {
		fmt.Printf("❌ Failed to list blocks: %v\n", err)
		return
	}
	
	if !blocksResp.Success {
		fmt.Printf("❌ ListBlocks failed: %s\n", blocksResp.Error)
		return
	}
	
	fmt.Printf("   Found %d blocks in tab\n", len(blocksResp.Blocks))
	
	// Count blocks with correct Tab ID
	correctBlocks := 0
	for _, block := range blocksResp.Blocks {
		if block.TabId == expectedTabId {
			correctBlocks++
		}
	}
	
	fmt.Printf("   Blocks with correct Tab ID: %d/%d\n", correctBlocks, len(blocksResp.Blocks))
	
	if correctBlocks == len(blocksResp.Blocks) && len(blocksResp.Blocks) > 0 {
		fmt.Printf("   ✅ SUCCESS: All blocks have correct Tab ID!\n")
	} else {
		fmt.Printf("   ❌ PARTIAL/FAILURE: Not all blocks have correct Tab ID\n")
	}
	
	// 5. Output JSON for debugging
	fmt.Printf("\n5. 📄 Sample Block Info (JSON):\n")
	if len(blocksResp.Blocks) > 0 {
		sampleJson, _ := json.MarshalIndent(blocksResp.Blocks[0], "   ", "  ")
		fmt.Printf("   %s\n", string(sampleJson))
	}
	
	fmt.Printf("\n🎉 Test completed!\n")
	
	// Final summary
	if blockInfo.TabId == expectedTabId && correctBlocks == len(blocksResp.Blocks) {
		fmt.Printf("🎊 OVERALL SUCCESS: The fix is working correctly!\n")
	} else {
		fmt.Printf("⚠️  MIXED RESULTS: The fix may need more work or debugging.\n")
	}
}