package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
	"github.com/wavetermdev/waveterm/pkg/wavebase"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

func main() {
	// 使用默认的Wave Terminal数据目录
	dataDir := os.Getenv("HOME") + "/Library/Application Support/Wave"
	configDir := dataDir
	
	// 设置环境变量
	os.Setenv("WAVETERM_DATA_HOME", dataDir)
	os.Setenv("WAVETERM_CONFIG_HOME", configDir)
	
	log.Printf("Using Wave data directory: %s", dataDir)
	
	// 缓存环境变量
	err := wavebase.CacheAndRemoveEnvVars()
	if err != nil {
		log.Fatalf("error caching env vars: %v", err)
	}
	
	// 初始化数据库
	err = wstore.InitWStore()
	if err != nil {
		log.Fatalf("error initializing wstore: %v", err)
	}
	
	log.Println("Connected to Wave Terminal database!")
	
	ctx := context.Background()
	service := &widgetapiservice.WidgetAPIService{}
	
	// 列出所有工作区
	fmt.Println("\n=== Current Workspaces ===")
	workspacesResp, err := service.ListWorkspaces(ctx)
	if err != nil {
		log.Printf("Error listing workspaces: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(workspacesResp, "", "  ")
		fmt.Printf("Response: %s\n", jsonData)
		
		if workspacesResp.Success && len(workspacesResp.Workspaces) > 0 {
			fmt.Printf("\n📋 Found %d workspace(s):\n", len(workspacesResp.Workspaces))
			for i, ws := range workspacesResp.Workspaces {
				fmt.Printf("   %d. %s (ID: %s)\n", i+1, ws.Name, ws.WorkspaceId)
			}
		}
	}
	
	// 尝试获取名为"waveterm"的工作区
	fmt.Println("\n=== Looking for 'waveterm' workspace ===")
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
		} else {
			fmt.Printf("\n❌ Workspace 'waveterm' not found: %s\n", resp.Error)
		}
	}
	
	fmt.Println("\n=== Test completed ===")
}