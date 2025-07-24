package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/wavetermdev/waveterm/pkg/filestore"
	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
	"github.com/wavetermdev/waveterm/pkg/wavebase"
	"github.com/wavetermdev/waveterm/pkg/wcore"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

func main() {
	log.Println("Initializing Wave Terminal database...")
	
	// Cache environment variables first
	err := wavebase.CacheAndRemoveEnvVars()
	if err != nil {
		log.Fatalf("error caching env vars: %v", err)
	}
	
	// Initialize WaveBase directories (required for database paths)
	err = wavebase.EnsureWaveDataDir()
	if err != nil {
		log.Fatalf("error ensuring wave data dir: %v", err)
	}
	
	err = wavebase.EnsureWaveDBDir()
	if err != nil {
		log.Fatalf("error ensuring wave db dir: %v", err)
	}
	
	// Initialize FileStore
	err = filestore.InitFilestore()
	if err != nil {
		log.Fatalf("error initializing filestore: %v", err)
	}
	
	// Initialize WStore (main database)
	err = wstore.InitWStore()
	if err != nil {
		log.Fatalf("error initializing wstore: %v", err)
	}
	
	log.Println("Database initialized successfully!")
	log.Println("Testing Widget API - Get Workspace by Name")
	
	ctx := context.Background()
	service := &widgetapiservice.WidgetAPIService{}
	
	// Create a test workspace first if none exist
	fmt.Println("\n=== Creating test workspace 'waveterm' ===")
	workspace, err := wcore.CreateWorkspace(ctx, "waveterm", "wave-logo", "#58C142", true, false)
	if err != nil {
		log.Printf("Error creating workspace (might already exist): %v", err)
	} else {
		fmt.Printf("Created workspace: %s (ID: %s)\n", workspace.Name, workspace.OID)
	}
	
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
		} else {
			fmt.Printf("\n❌ Case insensitive search failed: %s\n", resp2.Error)
		}
	}
	
	// Test 4: Try non-existent workspace
	fmt.Println("\n=== Test 4: Get workspace by name 'NonExistent' ===")
	resp3, err := service.GetWorkspaceByName(ctx, "NonExistent")
	if err != nil {
		log.Printf("Error getting workspace by name: %v", err)
	} else {
		jsonData, _ := json.MarshalIndent(resp3, "", "  ")
		fmt.Printf("Response: %s\n", jsonData)
		
		if !resp3.Success {
			fmt.Printf("\n✅ Correctly returned error for non-existent workspace\n")
		}
	}
	
	fmt.Println("\n=== All tests completed ===")
}