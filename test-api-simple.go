package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
)

func main() {
	log.Println("Testing Widget API - Get Workspace by Name")
	
	ctx := context.Background()
	service := &widgetapiservice.WidgetAPIService{}
	
	// Test 1: List all workspaces
	fmt.Println("\n=== Test 1: List all workspaces ===")
	workspacesResp, err := service.ListWorkspaces(ctx)
	if err != nil {
		log.Printf("Error listing workspaces: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(workspacesResp, "", "  ")
		fmt.Printf("Response: %s\n", jsonData)
	}
	
	// Test 2: Try to find workspace by name "waveterm"
	fmt.Println("\n=== Test 2: Get workspace by name 'waveterm' ===")
	resp, err := service.GetWorkspaceByName(ctx, "waveterm")
	if err != nil {
		log.Printf("Error getting workspace by name: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Printf("Response: %s\n", jsonData)
		
		if resp.Success && resp.Workspace != nil {
			fmt.Printf("\n✅ Found workspace 'waveterm'!\n")
			fmt.Printf("   Workspace ID: %s\n", resp.Workspace.WorkspaceId)
			fmt.Printf("   Name: %s\n", resp.Workspace.Name)
			fmt.Printf("   Tab IDs: %v\n", resp.Workspace.TabIds)
			fmt.Printf("   Active Tab ID: %s\n", resp.Workspace.ActiveTabId)
		} else {
			fmt.Printf("\n❌ Workspace 'waveterm' not found: %s\n", resp.Error)
		}
	}
	
	// Test 3: Try case insensitive search
	fmt.Println("\n=== Test 3: Get workspace by name 'WAVETERM' (case insensitive) ===")
	resp2, err := service.GetWorkspaceByName(ctx, "WAVETERM")
	if err != nil {
		log.Printf("Error getting workspace by name: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(resp2, "", "  ")
		fmt.Printf("Response: %s\n", jsonData)
		
		if resp2.Success && resp2.Workspace != nil {
			fmt.Printf("\n✅ Case insensitive search works!\n")
			fmt.Printf("   Workspace ID: %s\n", resp2.Workspace.WorkspaceId)
		}
	}
	
	fmt.Println("\n=== Test completed ===")
}