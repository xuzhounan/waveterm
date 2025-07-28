package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
	"github.com/wavetermdev/waveterm/pkg/wavebase"
	"github.com/wavetermdev/waveterm/pkg/wcore"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

func main() {
	log.Printf("测试Widget数据库存储...")

	// 设置Wave数据目录环境变量 - 使用与服务器相同的路径
	waveHome := "/Users/xzn/Library/Application Support/waveterm"
	os.Setenv("WAVETERM_HOME", waveHome)
	os.Setenv("WAVETERM_DATA_HOME", waveHome)

	// 初始化数据库
	err := wavebase.EnsureWaveDataDir()
	if err != nil {
		log.Fatalf("Failed to ensure data dir: %v", err)
	}

	// 创建context
	ctx := context.Background()
	ctx = waveobj.ContextWithUpdates(ctx)

	// 测试workspace信息
	log.Printf("检查workspace信息...")
	workspaceId := "39720a34-6d5b-477c-bc5f-4ac6f8eb1abf"
	workspace, err := wcore.GetWorkspace(ctx, workspaceId)
	if err != nil {
		log.Fatalf("Failed to get workspace: %v", err)
	}

	fmt.Printf("Workspace: %s\n", workspace.Name)
	fmt.Printf("TabIds: %v\n", workspace.TabIds)
	fmt.Printf("ActiveTabId: %s\n", workspace.ActiveTabId)

	// 检查tab信息
	if workspace.ActiveTabId != "" {
		tab, err := wstore.DBGet[*waveobj.Tab](ctx, workspace.ActiveTabId)
		if err != nil {
			log.Printf("Failed to get tab: %v", err)
		} else {
			fmt.Printf("Tab Name: %s\n", tab.Name)
			fmt.Printf("Tab BlockIds: %v\n", tab.BlockIds)
		}
	}

	// 测试创建widget
	log.Printf("测试创建widget...")
	req := widgetapiservice.CreateWidgetAPIRequest{
		WorkspaceId: workspaceId,
		WidgetType:  "terminal",
		Title:       "测试数据库终端",
		Meta: map[string]any{
			"cwd": "/tmp",
		},
	}

	response, err := widgetapiservice.WidgetAPIServiceInstance.CreateWidget(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create widget: %v", err)
	}

	if !response.Success {
		log.Fatalf("Widget creation failed: %s", response.Error)
	}

	fmt.Printf("Widget created: %s\n", response.BlockId)

	// 重新检查workspace信息
	log.Printf("重新检查workspace信息...")
	workspace, err = wcore.GetWorkspace(ctx, workspaceId)
	if err != nil {
		log.Fatalf("Failed to get workspace after widget creation: %v", err)
	}

	fmt.Printf("Updated Workspace TabIds: %v\n", workspace.TabIds)

	// 检查tab是否更新
	if workspace.ActiveTabId != "" {
		tab, err := wstore.DBGet[*waveobj.Tab](ctx, workspace.ActiveTabId)
		if err != nil {
			log.Printf("Failed to get updated tab: %v", err)
		} else {
			fmt.Printf("Updated Tab BlockIds: %v\n", tab.BlockIds)
		}
	}

	// 检查创建的block
	if response.BlockId != "" {
		block, err := wstore.DBGet[*waveobj.Block](ctx, response.BlockId)
		if err != nil {
			log.Printf("Failed to get created block: %v", err)
		} else {
			blockJson, _ := json.MarshalIndent(block, "", "  ")
			fmt.Printf("Created Block: %s\n", blockJson)
		}
	}
}