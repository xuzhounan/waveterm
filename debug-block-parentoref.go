package main

import (
	"context"
	"fmt"
	"log"

	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

func main() {
	// 初始化数据库
	err := wstore.InitWStore()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to initialize wstore: %w", err))
	}
	
	ctx := context.Background()
	
	// 获取所有blocks
	blocks, err := wstore.DBGetAllObjsByType[*waveobj.Block](ctx, "block")
	if err != nil {
		log.Fatal(fmt.Errorf("failed to get blocks: %w", err))
	}
	
	if len(blocks) == 0 {
		log.Fatal("no blocks found")
	}
	
	fmt.Printf("Found %d blocks in database\n", len(blocks))
	fmt.Printf("Testing first block:\n")
	
	block := blocks[0]
	
	fmt.Printf("Block Debug Information:\n")
	fmt.Printf("=========================\n")
	fmt.Printf("Block ID: %s\n", block.OID)
	fmt.Printf("ParentORef: '%s'\n", block.ParentORef)
	fmt.Printf("ParentORef length: %d\n", len(block.ParentORef))
	fmt.Printf("ParentORef is empty: %t\n", block.ParentORef == "")
	
	// 尝试解析ParentORef
	if block.ParentORef != "" {
		parentORef := waveobj.ParseORefNoErr(block.ParentORef)
		if parentORef != nil {
			fmt.Printf("Parsed ParentORef:\n")
			fmt.Printf("  Type: %s\n", parentORef.OType)
			fmt.Printf("  ID: %s\n", parentORef.OID)
		} else {
			fmt.Printf("Failed to parse ParentORef\n")
		}
	} else {
		fmt.Printf("❌ ParentORef is empty!\n")
	}
	
	// 检查Meta字段
	fmt.Printf("\nMeta contents:\n")
	for key, value := range block.Meta {
		fmt.Printf("  %s: %v\n", key, value)
	}
	
	// 测试我们的helper函数
	fmt.Printf("\nTesting getTabIdFromBlock helper:\n")
	
	// 由于getTabIdFromBlock在另一个package中，我们手动实现相同的逻辑
	getTabIdFromBlock := func(block *waveobj.Block) string {
		if block == nil || block.ParentORef == "" {
			return ""
		}
		
		parentORef := waveobj.ParseORefNoErr(block.ParentORef)
		if parentORef == nil {
			return ""
		}
		
		// Only return TabId if the parent is actually a Tab
		if parentORef.OType == waveobj.OType_Tab {
			return parentORef.OID
		}
		
		return ""
	}
	
	tabId := getTabIdFromBlock(block)
	fmt.Printf("Result from getTabIdFromBlock: '%s'\n", tabId)
	
	if tabId == "" {
		fmt.Printf("❌ getTabIdFromBlock returned empty string\n")
		
		if block.ParentORef == "" {
			fmt.Printf("   Reason: ParentORef is empty\n")
		} else {
			parentORef := waveobj.ParseORefNoErr(block.ParentORef)
			if parentORef == nil {
				fmt.Printf("   Reason: Failed to parse ParentORef\n")
			} else if parentORef.OType != waveobj.OType_Tab {
				fmt.Printf("   Reason: Parent is not a Tab (type: %s)\n", parentORef.OType)
			}
		}
	} else {
		fmt.Printf("✅ getTabIdFromBlock returned: %s\n", tabId)
	}
}