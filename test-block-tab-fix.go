package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

func main() {
	// 初始化数据库 - 使用现有数据库
	err := wstore.InitWStore()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to initialize wstore: %w", err))
	}
	
	// 添加更新上下文
	ctx := waveobj.ContextWithUpdates(context.Background())
	
	// 获取所有workspace
	workspaceService := &widgetapiservice.WidgetAPIService{}
	workspacesResp, err := workspaceService.ListWorkspaces(ctx)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to list workspaces: %w", err))
	}
	
	if !workspacesResp.Success || len(workspacesResp.Workspaces) == 0 {
		log.Fatal("No workspaces available")
	}
	
	// 使用第一个workspace
	workspace := workspacesResp.Workspaces[0]
	fmt.Printf("Testing with workspace: %s (name: %s)\n", workspace.WorkspaceId, workspace.Name)
	
	// 获取第一个tab
	if len(workspace.TabsInfo) == 0 {
		log.Fatal("No tabs available in workspace")
	}
	
	tab := workspace.TabsInfo[0]
	fmt.Printf("Testing with tab: %s (name: %s)\n", tab.TabId, tab.Name)
	
	// 列出该tab中的所有blocks
	listReq := widgetapiservice.ListBlocksAPIRequest{
		TabId: tab.TabId,
	}
	
	blocksResp, err := workspaceService.ListBlocks(ctx, listReq)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to list blocks: %w", err))
	}
	
	if !blocksResp.Success {
		log.Fatal(fmt.Errorf("failed to list blocks: %s", blocksResp.Error))
	}
	
	fmt.Printf("\nFound %d blocks in tab %s:\n", len(blocksResp.Blocks), tab.Name)
	
	// 检查每个block的tab关联
	for i, block := range blocksResp.Blocks {
		fmt.Printf("\nBlock %d:\n", i+1)
		fmt.Printf("  Block ID: %s\n", block.BlockId)
		fmt.Printf("  Tab ID: %s\n", block.TabId)
		fmt.Printf("  Workspace ID: %s\n", block.WorkspaceId)
		fmt.Printf("  Block Type: %s\n", block.BlockType)
		fmt.Printf("  View: %s\n", block.View)
		fmt.Printf("  Controller: %s\n", block.Controller)
		
		// 验证Tab ID是否正确设置
		if block.TabId == "" {
			fmt.Printf("  ❌ ERROR: Tab ID is empty!\n")
		} else if block.TabId == tab.TabId {
			fmt.Printf("  ✅ SUCCESS: Tab ID correctly set\n")
		} else {
			fmt.Printf("  ⚠️  WARNING: Tab ID mismatch (expected: %s, got: %s)\n", tab.TabId, block.TabId)
		}
		
		// 验证Workspace ID是否正确设置
		if block.WorkspaceId == "" {
			fmt.Printf("  ❌ ERROR: Workspace ID is empty!\n")
		} else if block.WorkspaceId == workspace.WorkspaceId {
			fmt.Printf("  ✅ SUCCESS: Workspace ID correctly set\n")
		} else {
			fmt.Printf("  ⚠️  WARNING: Workspace ID mismatch (expected: %s, got: %s)\n", workspace.WorkspaceId, block.WorkspaceId)
		}

		// 获取原始block对象以检查ParentORef
		rawBlock, err := wstore.DBGet[*waveobj.Block](ctx, block.BlockId)
		if err == nil && rawBlock != nil {
			fmt.Printf("  Raw ParentORef: %s\n", rawBlock.ParentORef)
			
			// 解析ParentORef
			parentORef := waveobj.ParseORefNoErr(rawBlock.ParentORef)
			if parentORef != nil {
				fmt.Printf("  Parsed Parent: Type=%s, ID=%s\n", parentORef.OType, parentORef.OID)
				if parentORef.OType == waveobj.OType_Tab && parentORef.OID == tab.TabId {
					fmt.Printf("  ✅ ParentORef correctly points to tab\n")
				} else {
					fmt.Printf("  ❌ ParentORef mismatch\n")
				}
			} else {
				fmt.Printf("  ❌ Failed to parse ParentORef\n")
			}
		}
	}
	
	// 如果有blocks，测试获取block状态
	if len(blocksResp.Blocks) > 0 {
		testBlock := blocksResp.Blocks[0]
		fmt.Printf("\n=== Testing GetBlockStatus for Block: %s ===\n", testBlock.BlockId)
		
		statusResp, err := workspaceService.GetBlockStatus(ctx, testBlock.BlockId)
		if err != nil {
			fmt.Printf("❌ Error getting block status: %v\n", err)
		} else if !statusResp.Success {
			fmt.Printf("❌ GetBlockStatus failed: %s\n", statusResp.Error)
		} else {
			fmt.Printf("✅ GetBlockStatus succeeded\n")
			statusJson, _ := json.MarshalIndent(statusResp.BlockInfo, "  ", "  ")
			fmt.Printf("Block Info: %s\n", string(statusJson))
		}
	}
	
	fmt.Printf("\n=== Test Summary ===\n")
	fmt.Printf("Total blocks tested: %d\n", len(blocksResp.Blocks))
	
	allCorrect := true
	for _, block := range blocksResp.Blocks {
		if block.TabId == "" || block.WorkspaceId == "" {
			allCorrect = false
			break
		}
	}
	
	if allCorrect {
		fmt.Printf("✅ All blocks have correct Tab ID and Workspace ID associations!\n")
	} else {
		fmt.Printf("❌ Some blocks still have missing Tab ID or Workspace ID associations\n")
	}
}