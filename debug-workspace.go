package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/wavetermdev/waveterm/pkg/wavebase"
	"github.com/wavetermdev/waveterm/pkg/wcore"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

func main() {
	// 设置Wave数据目录环境变量
	waveHome := "/Users/xzn/Library/Application Support/waveterm"
	os.Setenv("WAVETERM_HOME", waveHome)
	os.Setenv("WAVETERM_DATA_HOME", waveHome)

	// 初始化数据库
	err := wavebase.EnsureWaveDataDir()
	if err != nil {
		log.Fatalf("Failed to ensure data dir: %v", err)
	}

	ctx := context.Background()
	
	// 检查waveterm workspace
	workspaceId := "39720a34-6d5b-477c-bc5f-4ac6f8eb1abf"
	workspace, err := wcore.GetWorkspace(ctx, workspaceId)
	if err != nil {
		log.Fatalf("Failed to get workspace: %v", err)
	}

	fmt.Printf("Workspace: %s\n", workspace.Name)
	fmt.Printf("TabIds: %v (count: %d)\n", workspace.TabIds, len(workspace.TabIds))
	fmt.Printf("ActiveTabId: %s\n", workspace.ActiveTabId)

	// 检查活跃tab
	if workspace.ActiveTabId != "" {
		tab, err := wstore.DBGet[*waveobj.Tab](ctx, workspace.ActiveTabId)
		if err != nil {
			log.Printf("Failed to get tab: %v", err)
		} else {
			fmt.Printf("\nTab: %s\n", tab.Name)
			fmt.Printf("BlockIds: %v (count: %d)\n", tab.BlockIds, len(tab.BlockIds))
			
			// 检查每个block
			for i, blockId := range tab.BlockIds {
				block, err := wstore.DBGet[*waveobj.Block](ctx, blockId)
				if err != nil {
					log.Printf("Failed to get block %s: %v", blockId, err)
					continue
				}
				
				fmt.Printf("Block %d: %s\n", i+1, blockId)
				if block.Meta != nil {
					fmt.Printf("  View: %s\n", block.Meta.GetString("view", ""))
					fmt.Printf("  Title: %s\n", block.Meta.GetString("title", ""))
				}
			}
		}
	}

	// 列出所有最近的blocks
	fmt.Printf("\n=== 最近创建的blocks ===\n")
	allBlocks, err := wstore.DBGetAllObjsByType[*waveobj.Block](ctx, waveobj.OType_Block)
	if err != nil {
		log.Printf("Failed to get all blocks: %v", err)
	} else {
		for _, block := range allBlocks {
			if block.Meta != nil && block.Meta.GetString("title", "") != "" {
				blockJson, _ := json.MarshalIndent(map[string]any{
					"id": block.OID,
					"parent": block.ParentORef,
					"title": block.Meta.GetString("title", ""),
					"view": block.Meta.GetString("view", ""),
				}, "", "  ")
				fmt.Printf("%s\n", blockJson)
			}
		}
	}
}