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
	"github.com/wavetermdev/waveterm/pkg/wcore"
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
	
	// 创建一个测试工作区
	fmt.Println("\n=== Creating test workspace 'waveterm' ===")
	workspace, err := wcore.CreateWorkspace(ctx, "waveterm", "wave-logo", "#58C142", true, false)
	if err != nil {
		log.Printf("Error creating workspace (might already exist): %v", err)
	} else {
		fmt.Printf("Created workspace: %s (ID: %s)\n", workspace.Name, workspace.OID)
	}
	
	// 测试API
	fmt.Println("\n=== Testing Widget API ===")
	
	// Test 1: List workspaces
	fmt.Println("\n1. Listing all workspaces:")
	workspacesResp, err := service.ListWorkspaces(ctx)
	if err != nil {
		log.Printf("Error listing workspaces: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(workspacesResp, "", "  ")
		fmt.Printf("Response: %s\n", jsonData)
	}
	
	// Test 2: Get workspace by name
	fmt.Println("\n2. Getting workspace by name 'waveterm':")
	resp, err := service.GetWorkspaceByName(ctx, "waveterm")
	if err != nil {
		log.Printf("Error getting workspace by name: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Printf("Response: %s\n", jsonData)
		
		if resp.Success && resp.Workspace != nil {
			fmt.Printf("\n✅ Found workspace 'waveterm'!\n")
			fmt.Printf("   🆔 Workspace ID: %s\n", resp.Workspace.WorkspaceId)
			fmt.Printf("   📝 Name: %s\n", resp.Workspace.Name)
			fmt.Printf("   📑 Tab IDs: %v\n", resp.Workspace.TabIds)
			fmt.Printf("   🎯 Active Tab ID: %s\n", resp.Workspace.ActiveTabId)
		}
	}
	
	fmt.Println("\n=== API test completed ===")
}