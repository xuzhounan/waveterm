package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
	"github.com/wavetermdev/waveterm/pkg/service/workspaceservice"
	"github.com/wavetermdev/waveterm/pkg/wcore"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

func main() {
	ctx := context.Background()
	
	log.Println("=== Testing workspace API chain ===")
	
	// 1. Test direct ListWorkspaces from wcore
	log.Println("\n1. Testing wcore.ListWorkspaces...")
	wcoreWorkspaces, err := wcore.ListWorkspaces(ctx)
	if err != nil {
		log.Printf("Error from wcore.ListWorkspaces: %v", err)
	} else {
		log.Printf("wcore.ListWorkspaces returned %d workspaces", len(wcoreWorkspaces))
		for i, ws := range wcoreWorkspaces {
			log.Printf("  [%d] WorkspaceId: %s, WindowId: %s", i, ws.WorkspaceId, ws.WindowId)
		}
	}
	
	// 2. Test WorkspaceService.ListWorkspaces
	log.Println("\n2. Testing WorkspaceService.ListWorkspaces...")
	workspaceService := &workspaceservice.WorkspaceService{}
	wsServiceResult, err := workspaceService.ListWorkspaces()
	if err != nil {
		log.Printf("Error from WorkspaceService.ListWorkspaces: %v", err)
	} else {
		log.Printf("WorkspaceService.ListWorkspaces returned %d workspaces", len(wsServiceResult))
		for i, ws := range wsServiceResult {
			log.Printf("  [%d] WorkspaceId: %s, WindowId: %s", i, ws.WorkspaceId, ws.WindowId)
		}
	}
	
	// 3. Test WidgetAPIService.ListWorkspaces
	log.Println("\n3. Testing WidgetAPIService.ListWorkspaces...")
	widgetApiService := &widgetapiservice.WidgetAPIService{}
	apiResult, err := widgetApiService.ListWorkspaces(ctx)
	if err != nil {
		log.Printf("Error from WidgetAPIService.ListWorkspaces: %v", err)
	} else {
		log.Printf("WidgetAPIService.ListWorkspaces returned success=%v with %d workspaces", 
			apiResult.Success, len(apiResult.Workspaces))
		for i, ws := range apiResult.Workspaces {
			log.Printf("  [%d] WorkspaceId: %s, Name: %s, TabIds: %v, ActiveTabId: %s", 
				i, ws.WorkspaceId, ws.Name, ws.TabIds, ws.ActiveTabId)
		}
		
		// Print JSON representation
		jsonData, _ := json.MarshalIndent(apiResult, "", "  ")
		log.Printf("JSON output:\n%s", string(jsonData))
	}
	
	// 4. Get raw workspace data from database
	log.Println("\n4. Testing direct database access...")
	rawWorkspaces, err := wstore.DBGetAllObjsByType[*waveobj.Workspace](ctx, waveobj.OType_Workspace)
	if err != nil {
		log.Printf("Error from direct DB access: %v", err)
	} else {
		log.Printf("Direct DB access returned %d workspaces", len(rawWorkspaces))
		for i, ws := range rawWorkspaces {
			log.Printf("  [%d] OID: %s, Name: %s, Icon: %s, Color: %s, TabIds: %v", 
				i, ws.OID, ws.Name, ws.Icon, ws.Color, ws.TabIds)
		}
	}
}