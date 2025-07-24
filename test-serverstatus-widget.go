package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/wavetermdev/waveterm/pkg/filestore"
	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
	"github.com/wavetermdev/waveterm/pkg/wavebase"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

func main() {
	// 设置环境变量
	dataDir := "/tmp/waveterm-mcp"
	os.Setenv("WAVETERM_DATA_HOME", dataDir)
	os.Setenv("WAVETERM_CONFIG_HOME", dataDir)
	
	log.Printf("Using Wave data directory: %s", dataDir)
	
	// 缓存环境变量
	err := wavebase.CacheAndRemoveEnvVars()
	if err != nil {
		log.Fatalf("error caching env vars: %v", err)
	}
	
	// 初始化数据库
	err = filestore.InitFilestore()
	if err != nil {
		log.Fatalf("error initializing filestore: %v", err)
	}
	
	err = wstore.InitWStore()
	if err != nil {
		log.Fatalf("error initializing wstore: %v", err)
	}
	
	log.Println("Connected to Wave Terminal database!")
	
	ctx := context.Background()
	service := &widgetapiservice.WidgetAPIService{}
	
	// 测试创建一个Server Status widget
	fmt.Println("\n=== Testing Server Status Widget Creation ===")
	
	// 首先获取一个工作区
	workspacesResp, err := service.ListWorkspaces(ctx)
	if err != nil {
		log.Printf("Error listing workspaces: %v", err)
		return
	}
	
	if !workspacesResp.Success || len(workspacesResp.Workspaces) == 0 {
		log.Printf("No workspaces available")
		return
	}
	
	// 使用第一个工作区
	workspace := workspacesResp.Workspaces[0]
	fmt.Printf("Using workspace: %s (ID: %s)\n", workspace.Name, workspace.WorkspaceId)
	
	// 创建Server Status widget
	createReq := &widgetapiservice.CreateWidgetAPIRequest{
		WorkspaceId: workspace.WorkspaceId,
		WidgetType:  "serverstatus",
		Title:       "MCP Server Monitor",
		Meta: map[string]any{
			"view": "serverstatus",
		},
	}
	
	createResp, err := service.CreateWidget(ctx, *createReq)
	if err != nil {
		log.Printf("Error creating Server Status widget: %v", err)
		return
	}
	
	jsonData, _ := json.MarshalIndent(createResp, "", "  ")
	fmt.Printf("Created Server Status widget: %s\n", jsonData)
	
	if createResp.Success {
		fmt.Printf("\n✅ Successfully created Server Status widget!\n")
		fmt.Printf("   🆔 Block ID: %s\n", createResp.BlockId)
		fmt.Printf("   📝 Title: %s\n", createResp.Widget.Title)
		fmt.Printf("   🔧 Widget Type: %s\n", createResp.Widget.WidgetType)
		fmt.Printf("   📍 Workspace: %s\n", createResp.Widget.WorkspaceId)
		fmt.Printf("\nYou can now see the Server Status widget in your Wave Terminal interface!\n")
	}
	
	fmt.Println("\n=== Widget creation test completed ===")
}