package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/wavetermdev/waveterm/pkg/wavebase"
	"github.com/wavetermdev/waveterm/pkg/wcore"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

func main() {
	// 设置环境变量
	waveHome := "/Users/xzn/Library/Application Support/waveterm"
	os.Setenv("WAVETERM_HOME", waveHome)
	os.Setenv("WAVETERM_DATA_HOME", waveHome)

	// 初始化数据库
	err := wavebase.EnsureWaveDataDir()
	if err != nil {
		log.Fatalf("Failed to ensure data dir: %v", err)
	}

	ctx := context.Background()
	ctx = waveobj.ContextWithUpdates(ctx)
	
	// 修复waveterm workspace
	workspaceId := "39720a34-6d5b-477c-bc5f-4ac6f8eb1abf"
	activeTabId := "3c1f7d5e-f971-4812-a688-4e1b2310411f"
	
	fmt.Printf("修复workspace数据不一致问题...\n")
	
	// 检查active tab是否真的存在
	tab, err := wstore.DBGet[*waveobj.Tab](ctx, activeTabId)
	if err != nil {
		log.Fatalf("Failed to get active tab: %v", err)
	}
	if tab == nil {
		log.Fatalf("Active tab not found: %s", activeTabId)
	}
	
	fmt.Printf("Active tab exists: %s\n", tab.Name)
	fmt.Printf("Tab has %d blocks: %v\n", len(tab.BlockIds), tab.BlockIds)
	
	// 获取workspace
	workspace, err := wcore.GetWorkspace(ctx, workspaceId)
	if err != nil {
		log.Fatalf("Failed to get workspace: %v", err)
	}
	
	fmt.Printf("Workspace tab_ids before fix: %v\n", workspace.TabIds)
	
	// 检查active_tab_id是否在tab_ids中
	found := false
	for _, tabId := range workspace.TabIds {
		if tabId == activeTabId {
			found = true
			break
		}
	}
	
	if !found {
		fmt.Printf("修复：将active tab添加到workspace tab_ids中\n")
		// 添加active tab到tab_ids中
		workspace.TabIds = append(workspace.TabIds, activeTabId)
		
		// 更新workspace
		err = wstore.WithTx(ctx, func(tx *wstore.TxWrap) error {
			return wstore.DBUpdate(tx.Context(), workspace)
		})
		if err != nil {
			log.Fatalf("Failed to update workspace: %v", err)
		}
		
		fmt.Printf("Workspace tab_ids after fix: %v\n", workspace.TabIds)
		
		// 发送更新事件
		updates := waveobj.ContextGetUpdatesRtn(ctx)
		fmt.Printf("Sending %d updates to frontend\n", len(updates))
		
	} else {
		fmt.Printf("Active tab already in tab_ids, no fix needed\n")
	}
}