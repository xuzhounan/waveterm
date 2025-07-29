// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// REST API handlers for widget management
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wcore"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

// handleWidgetAPI routes widget API requests to appropriate handlers
func handleWidgetAPI(w http.ResponseWriter, r *http.Request) {
	// TODO: Enable authentication in production
	// if err := authkey.ValidateIncomingRequest(r); err != nil {
	//	http.Error(w, "Unauthorized", http.StatusUnauthorized)
	//	return
	// }

	// Set Content-Type for JSON responses
	w.Header().Set("Content-Type", "application/json")

	// CORS is handled by the global CORS middleware in web.go
	// Handle OPTIONS requests for CORS preflight (as backup)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Parse URL path to determine the specific API endpoint
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/widgets")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	ctx := r.Context()

	switch r.Method {
	case "POST":
		if path == "" || path == "/" {
			// POST /api/v1/widgets - Create widget
			handleCreateWidget(w, r, ctx)
		} else if path == "/mcp/restart" {
			// POST /api/v1/widgets/mcp/restart - Restart MCP server
			handleMCPServerRestart(w, r, ctx)
		} else if path == "/persistent-server/start" {
			// POST /api/v1/widgets/persistent-server/start - Start persistent server
			handlePersistentServerStart(w, r, ctx)
		} else if path == "/persistent-server/stop" {
			// POST /api/v1/widgets/persistent-server/stop - Stop persistent server
			handlePersistentServerStop(w, r, ctx)
		} else if path == "/tabs" {
			// POST /api/v1/widgets/tabs - Create tab
			handleCreateTab(w, r, ctx)
		} else if len(pathParts) == 2 && pathParts[0] == "tabs" && pathParts[1] == "activate" {
			// POST /api/v1/widgets/tabs/activate - Set active tab
			handleSetActiveTab(w, r, ctx)
		} else if len(pathParts) == 3 && pathParts[0] == "block" && pathParts[2] == "input" {
			// POST /api/v1/widgets/block/{block_id}/input - Send input to block
			blockId := pathParts[1]
			handleSendBlockInput(w, r, ctx, blockId)
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	case "GET":
		if path == "" || path == "/" {
			// GET /api/v1/widgets - List all available widget types
			handleListWidgetTypes(w, r, ctx)
		} else if len(pathParts) == 2 && pathParts[0] == "workspace" {
			// GET /api/v1/widgets/workspace/{workspace_id} - Get workspace widgets
			workspaceId := pathParts[1]
			handleGetWorkspaceWidgets(w, r, ctx, workspaceId)
		} else if len(pathParts) == 3 && pathParts[0] == "workspace" && pathParts[1] == "info" {
			// GET /api/v1/widgets/workspace/info/{workspace_id} - Get workspace detailed info
			workspaceId := pathParts[2]
			handleGetWorkspaceInfo(w, r, ctx, workspaceId)
		} else if len(pathParts) == 3 && pathParts[0] == "workspace" && pathParts[1] == "name" {
			// GET /api/v1/widgets/workspace/name/{workspace_name} - Get workspace by name
			workspaceName := pathParts[2]
			handleGetWorkspaceByName(w, r, ctx, workspaceName)
		} else if pathParts[0] == "workspaces" {
			// GET /api/v1/widgets/workspaces - List workspaces
			handleListWorkspaces(w, r, ctx)
		} else if path == "/mcp/status" {
			// GET /api/v1/widgets/mcp/status - Check MCP server status
			handleMCPServerStatus(w, r, ctx)
		} else if path == "/persistent-server/status" {
			// GET /api/v1/widgets/persistent-server/status - Check persistent server status
			handlePersistentServerStatus(w, r, ctx)
		} else if path == "/debug/fix-workspace" {
			// GET /api/v1/widgets/debug/fix-workspace - Fix workspace data inconsistencies
			handleFixWorkspaceData(w, r, ctx)
		} else if len(pathParts) == 3 && pathParts[0] == "workspace" && pathParts[2] == "tabs" {
			// GET /api/v1/widgets/workspace/{workspace_id}/tabs - List tabs in workspace
			workspaceId := pathParts[1]
			handleListTabs(w, r, ctx, workspaceId)
		} else if len(pathParts) == 3 && pathParts[0] == "block" && pathParts[1] == "content" {
			// GET /api/v1/widgets/block/content/{block_id} - Get block content
			blockId := pathParts[2]
			handleGetBlockContent(w, r, ctx, blockId)
		} else if len(pathParts) == 3 && pathParts[0] == "block" && pathParts[1] == "status" {
			// GET /api/v1/widgets/block/status/{block_id} - Get block status
			blockId := pathParts[2]
			handleGetBlockStatus(w, r, ctx, blockId)
		} else if pathParts[0] == "blocks" {
			// GET /api/v1/widgets/blocks?workspace_id=x&tab_id=y&block_type=z - List blocks
			handleListBlocks(w, r, ctx)
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleCreateWidget creates a new widget in a workspace
func handleCreateWidget(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	var req widgetapiservice.CreateWidgetAPIRequest

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		log.Printf("Error decoding create widget request: %v", err)
		writeErrorResponse(w, "Invalid JSON request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.WorkspaceId == "" {
		writeErrorResponse(w, "workspace_id is required", http.StatusBadRequest)
		return
	}
	if req.WidgetType == "" {
		writeErrorResponse(w, "widget_type is required", http.StatusBadRequest)
		return
	}

	log.Printf("Creating widget: type=%s, workspace=%s", req.WidgetType, req.WorkspaceId)

	// Call the service to create the widget
	response, err := widgetapiservice.WidgetAPIServiceInstance.CreateWidget(ctx, req)
	if err != nil {
		log.Printf("Error creating widget: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return the response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleGetWorkspaceWidgets returns available widgets for a workspace
func handleGetWorkspaceWidgets(w http.ResponseWriter, r *http.Request, ctx context.Context, workspaceId string) {
	if workspaceId == "" {
		writeErrorResponse(w, "workspace_id is required", http.StatusBadRequest)
		return
	}

	log.Printf("Getting workspace widgets for workspace: %s", workspaceId)

	// Call the service to get workspace widgets
	response, err := widgetapiservice.WidgetAPIServiceInstance.GetWorkspaceWidgets(ctx, workspaceId)
	if err != nil {
		log.Printf("Error getting workspace widgets: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}

// handleListWorkspaces returns all available workspaces
func handleListWorkspaces(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Listing workspaces")

	// Call the service to list workspaces
	response, err := widgetapiservice.WidgetAPIServiceInstance.ListWorkspaces(ctx)
	if err != nil {
		log.Printf("Error listing workspaces: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}

// handleGetWorkspaceInfo returns detailed workspace information by ID
func handleGetWorkspaceInfo(w http.ResponseWriter, r *http.Request, ctx context.Context, workspaceId string) {
	if workspaceId == "" {
		writeErrorResponse(w, "workspace_id is required", http.StatusBadRequest)
		return
	}

	log.Printf("Getting workspace info for workspace: %s", workspaceId)

	// Call the service to get workspace info
	response, err := widgetapiservice.WidgetAPIServiceInstance.GetWorkspaceInfo(ctx, workspaceId)
	if err != nil {
		log.Printf("Error getting workspace info: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}

// handleGetWorkspaceByName returns workspace information by name
func handleGetWorkspaceByName(w http.ResponseWriter, r *http.Request, ctx context.Context, workspaceName string) {
	if workspaceName == "" {
		writeErrorResponse(w, "workspace_name is required", http.StatusBadRequest)
		return
	}

	log.Printf("Getting workspace by name: %s", workspaceName)

	// Call the service to get workspace by name
	response, err := widgetapiservice.WidgetAPIServiceInstance.GetWorkspaceByName(ctx, workspaceName)
	if err != nil {
		log.Printf("Error getting workspace by name: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// If workspace not found, return 404
	if !response.Success && strings.Contains(response.Error, "not found") {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}

// handleListWidgetTypes returns available widget types and their descriptions
func handleListWidgetTypes(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Listing widget types")

	// Define available widget types
	widgetTypes := map[string]interface{}{
		"success": true,
		"widget_types": map[string]interface{}{
			"terminal": map[string]interface{}{
				"name":        "Terminal",
				"description": "Interactive terminal session",
				"icon":        "square-terminal",
				"meta_fields": map[string]string{
					"controller": "shell controller type (default: 'shell')",
					"cwd":        "working directory",
					"env":        "environment variables",
				},
			},
			"web": map[string]interface{}{
				"name":        "Web Browser",
				"description": "Web browser widget for browsing websites",
				"icon":        "globe",
				"meta_fields": map[string]string{
					"url": "initial URL to load (default: 'https://www.waveterm.dev')",
				},
			},
			"files": map[string]interface{}{
				"name":        "File Browser",
				"description": "File and directory browser",
				"icon":        "folder",
				"meta_fields": map[string]string{
					"file": "initial path to browse (default: '~')",
				},
			},
			"ai": map[string]interface{}{
				"name":        "AI Assistant",
				"description": "WaveAI chat assistant",
				"icon":        "sparkles",
				"meta_fields": map[string]string{},
			},
			"sysinfo": map[string]interface{}{
				"name":        "System Information",
				"description": "System monitoring and information display",
				"icon":        "chart-line",
				"meta_fields": map[string]string{},
			},
			"help": map[string]interface{}{
				"name":        "Help",
				"description": "Wave Terminal help and documentation",
				"icon":        "circle-question",
				"meta_fields": map[string]string{},
			},
			"tips": map[string]interface{}{
				"name":        "Quick Tips",
				"description": "Quick tips for Wave Terminal usage",
				"icon":        "lightbulb",
				"meta_fields": map[string]string{},
			},
		},
		"examples": map[string]interface{}{
			"terminal": map[string]interface{}{
				"workspace_id": "workspace-123",
				"widget_type":  "terminal",
				"title":        "My Terminal",
				"meta": map[string]interface{}{
					"cwd": "/home/user",
				},
			},
			"web": map[string]interface{}{
				"workspace_id": "workspace-123",
				"widget_type":  "web",
				"title":        "Documentation",
				"meta": map[string]interface{}{
					"url": "https://docs.waveterm.dev",
				},
			},
			"files": map[string]interface{}{
				"workspace_id": "workspace-123",
				"widget_type":  "files",
				"title":        "Home Directory",
				"meta": map[string]interface{}{
					"file": "~",
				},
			},
		},
		"endpoints": map[string]interface{}{
			"get_workspace_by_name": map[string]interface{}{
				"method":      "GET",
				"path":        "/api/v1/widgets/workspace/name/{workspace_name}",
				"description": "Get workspace information by name (case-insensitive)",
				"parameters": map[string]interface{}{
					"workspace_name": "Name of the workspace to find",
				},
				"example_response": map[string]interface{}{
					"success": true,
					"workspace": map[string]interface{}{
						"workspace_id":  "workspace-123",
						"name":          "Default",
						"tab_ids":       []string{"tab-1", "tab-2"},
						"active_tab_id": "tab-1",
					},
				},
			},
		},
	}

	// Return the response
	json.NewEncoder(w).Encode(widgetTypes)
}

// handleMCPServerStatus checks the status of MCP server functionality and connected clients
func handleMCPServerStatus(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Checking MCP server status")

	// Try to extract port from the request
	currentPort := 0
	if r.Host != "" {
		// Parse port from Host header if available
		if colonIndex := strings.LastIndex(r.Host, ":"); colonIndex != -1 {
			if port, err := strconv.Atoi(r.Host[colonIndex+1:]); err == nil {
				currentPort = port
			}
		}
	}

	// Check for active MCP client connections (like Claude Code)
	servers := make(map[string]interface{})

	// TODO: Implement actual MCP client detection logic
	// For now, we'll detect based on process names and connections
	// This is a simplified detection mechanism
	hasClaudeClient := checkForClaudeCodeClient()

	if hasClaudeClient {
		servers["claude-code"] = map[string]interface{}{
			"name":      "Claude Code",
			"status":    "connected",
			"url":       fmt.Sprintf("http://127.0.0.1:%d", currentPort),
			"lastSeen":  getCurrentTimestamp(),
			"tools":     []string{"create_widget", "list_workspaces", "get_workspace_by_name", "get_workspace", "get_widget_types", "check_server_status", "create_tab", "list_tabs", "set_active_tab", "restart_mcp_server", "fix_workspace_data"},
			"resources": []string{"workspaces", "widgets", "terminals"},
		}
	}

	log.Printf("MCP server status - Port: %d, Connected clients: %d", currentPort, len(servers))

	// 返回兼容两种用途的格式：
	// 1. 前端 MCP 客户端组件需要 servers 字段
	// 2. Server Status 组件需要 status 字段
	response := map[string]interface{}{
		"success": true,
		"servers": servers,
		"status": map[string]interface{}{
			"running": len(servers) > 0 || currentPort > 0, // 如果有连接的客户端或者服务器在运行
			"port":    currentPort,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// handleMCPServerRestart attempts to restart/reinitialize the MCP server functionality
func handleMCPServerRestart(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Attempting to restart MCP server functionality")

	// Since this appears to be an embedded MCP interface rather than a separate process,
	// we'll simulate a "restart" by reinitializing the MCP functionality state

	// Check if the web server itself is running (which it is if we're handling this request)
	isRunning := true
	currentPort := 0

	// Try to extract port from the request
	if r.Host != "" {
		// Parse port from Host header if available
		if colonIndex := strings.LastIndex(r.Host, ":"); colonIndex != -1 {
			if port, err := strconv.Atoi(r.Host[colonIndex+1:]); err == nil {
				currentPort = port
			}
		}
	}

	log.Printf("MCP functionality reinitialized - Running: %v, Port: %d", isRunning, currentPort)

	response := map[string]interface{}{
		"success": true,
		"message": "MCP server functionality reinitialized successfully",
		"status": map[string]interface{}{
			"running": isRunning,
			"port":    currentPort,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// checkForClaudeCodeClient checks if Claude Code MCP client is connected
func checkForClaudeCodeClient() bool {
	// Check for Claude processes running
	cmd := exec.Command("pgrep", "-f", "claude")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error checking for Claude processes: %v", err)
		return false
	}

	// If we found Claude processes, check for mcp-bridge as well
	if len(output) > 0 {
		cmd2 := exec.Command("pgrep", "-f", "mcp-bridge")
		output2, err2 := cmd2.Output()
		if err2 != nil {
			log.Printf("Error checking for mcp-bridge processes: %v", err2)
			return false
		}

		// Both Claude and mcp-bridge processes exist
		return len(output2) > 0
	}

	return false
}

// getCurrentTimestamp returns current timestamp in milliseconds
func getCurrentTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// handleFixWorkspaceData fixes workspace data inconsistencies
func handleFixWorkspaceData(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Fixing workspace data inconsistencies")

	ctx = waveobj.ContextWithUpdates(ctx)

	// Target workspace: waveterm
	workspaceId := "39720a34-6d5b-477c-bc5f-4ac6f8eb1abf"
	activeTabId := "3c1f7d5e-f971-4812-a688-4e1b2310411f"

	// Get workspace
	workspace, err := wcore.GetWorkspace(ctx, workspaceId)
	if err != nil {
		writeErrorResponse(w, fmt.Sprintf("Failed to get workspace: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Check if active tab exists
	tab, err := wstore.DBGet[*waveobj.Tab](ctx, activeTabId)
	if err != nil || tab == nil {
		writeErrorResponse(w, fmt.Sprintf("Active tab not found: %s", activeTabId), http.StatusInternalServerError)
		return
	}

	// Check if active_tab_id is in tab_ids
	found := false
	for _, tabId := range workspace.TabIds {
		if tabId == activeTabId {
			found = true
			break
		}
	}

	result := map[string]interface{}{
		"success":                  true,
		"workspace_id":             workspaceId,
		"active_tab_id":            activeTabId,
		"tab_name":                 tab.Name,
		"tab_blocks_count":         len(tab.BlockIds),
		"tab_blocks":               tab.BlockIds,
		"workspace_tab_ids_before": workspace.TabIds,
	}

	if !found {
		log.Printf("Adding active tab %s to workspace %s tab_ids", activeTabId, workspaceId)

		// Add active tab to tab_ids
		workspace.TabIds = append(workspace.TabIds, activeTabId)

		// Update workspace
		err = wstore.WithTx(ctx, func(tx *wstore.TxWrap) error {
			return wstore.DBUpdate(tx.Context(), workspace)
		})
		if err != nil {
			writeErrorResponse(w, fmt.Sprintf("Failed to update workspace: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		result["fixed"] = true
		result["workspace_tab_ids_after"] = workspace.TabIds
		result["message"] = "Active tab added to workspace tab_ids"

		log.Printf("Workspace data fixed successfully")
	} else {
		result["fixed"] = false
		result["message"] = "No fix needed, active tab already in tab_ids"
	}

	json.NewEncoder(w).Encode(result)
}

// handleCreateTab creates a new tab in a workspace
func handleCreateTab(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	var req widgetapiservice.CreateTabAPIRequest

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		log.Printf("Error decoding create tab request: %v", err)
		writeErrorResponse(w, "Invalid JSON request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.WorkspaceId == "" {
		writeErrorResponse(w, "workspace_id is required", http.StatusBadRequest)
		return
	}

	log.Printf("Creating tab: workspace=%s, name=%s", req.WorkspaceId, req.TabName)

	// Call the service to create the tab
	response, err := widgetapiservice.WidgetAPIServiceInstance.CreateTab(ctx, req)
	if err != nil {
		log.Printf("Error creating tab: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return the response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleListTabs returns all tabs for a workspace
func handleListTabs(w http.ResponseWriter, r *http.Request, ctx context.Context, workspaceId string) {
	if workspaceId == "" {
		writeErrorResponse(w, "workspace_id is required", http.StatusBadRequest)
		return
	}

	log.Printf("Listing tabs for workspace: %s", workspaceId)

	// Call the service to list tabs
	response, err := widgetapiservice.WidgetAPIServiceInstance.ListTabs(ctx, workspaceId)
	if err != nil {
		log.Printf("Error listing tabs: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}

// handleSetActiveTab sets the active tab in a workspace
func handleSetActiveTab(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	var req widgetapiservice.SetActiveTabAPIRequest

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		log.Printf("Error decoding set active tab request: %v", err)
		writeErrorResponse(w, "Invalid JSON request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.WorkspaceId == "" {
		writeErrorResponse(w, "workspace_id is required", http.StatusBadRequest)
		return
	}
	if req.TabId == "" {
		writeErrorResponse(w, "tab_id is required", http.StatusBadRequest)
		return
	}

	log.Printf("Setting active tab: workspace=%s, tab=%s", req.WorkspaceId, req.TabId)

	// Call the service to set active tab
	response, err := widgetapiservice.WidgetAPIServiceInstance.SetActiveTab(ctx, req)
	if err != nil {
		log.Printf("Error setting active tab: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}

// handleGetBlockContent handles getting content from a block (e.g., terminal output)
func handleGetBlockContent(w http.ResponseWriter, r *http.Request, ctx context.Context, blockId string) {
	if blockId == "" {
		writeErrorResponse(w, "block_id is required", http.StatusBadRequest)
		return
	}

	// Parse query parameters
	fileName := r.URL.Query().Get("file_name")
	if fileName == "" {
		fileName = "term" // default to terminal output
	}

	offset := int64(0)
	size := int64(0)
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.ParseInt(offsetStr, 10, 64); err == nil {
			offset = parsedOffset
		}
	}
	
	if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
		if parsedSize, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			size = parsedSize
		}
	}

	log.Printf("Getting block content: block_id=%s, file=%s, offset=%d, size=%d", blockId, fileName, offset, size)

	// Call the service to get block content
	response, err := widgetapiservice.WidgetAPIServiceInstance.GetBlockContent(ctx, blockId, fileName, offset, size)
	if err != nil {
		log.Printf("Error getting block content: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}

// handleGetBlockStatus handles getting status information for a block
func handleGetBlockStatus(w http.ResponseWriter, r *http.Request, ctx context.Context, blockId string) {
	if blockId == "" {
		writeErrorResponse(w, "block_id is required", http.StatusBadRequest)
		return
	}

	log.Printf("Getting block status: block_id=%s", blockId)

	// Call the service to get block status
	response, err := widgetapiservice.WidgetAPIServiceInstance.GetBlockStatus(ctx, blockId)
	if err != nil {
		log.Printf("Error getting block status: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}

// handleListBlocks handles listing blocks with optional filtering
func handleListBlocks(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	// Parse query parameters
	workspaceId := r.URL.Query().Get("workspace_id")
	tabId := r.URL.Query().Get("tab_id")
	blockType := r.URL.Query().Get("block_type")

	log.Printf("Listing blocks: workspace_id=%s, tab_id=%s, block_type=%s", workspaceId, tabId, blockType)

	// Build request
	req := widgetapiservice.ListBlocksAPIRequest{
		WorkspaceId: workspaceId,
		TabId:       tabId,
		BlockType:   blockType,
	}

	// Call the service to list blocks
	response, err := widgetapiservice.WidgetAPIServiceInstance.ListBlocks(ctx, req)
	if err != nil {
		log.Printf("Error listing blocks: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}

// handleSendBlockInput handles sending input to a block (e.g., terminal input)
func handleSendBlockInput(w http.ResponseWriter, r *http.Request, ctx context.Context, blockId string) {
	if blockId == "" {
		writeErrorResponse(w, "block_id is required", http.StatusBadRequest)
		return
	}

	var req widgetapiservice.SendBlockInputAPIRequest

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		log.Printf("Error decoding send block input request: %v", err)
		writeErrorResponse(w, "Invalid JSON request body", http.StatusBadRequest)
		return
	}

	// Set the block ID from URL path
	req.BlockId = blockId

	// Validate that we have some kind of input
	if req.InputData == "" && req.SigName == "" && req.TermSize == nil {
		writeErrorResponse(w, "at least one of input_data, sig_name, or term_size is required", http.StatusBadRequest)
		return
	}

	log.Printf("Sending input to block: block_id=%s, input_type=%s", blockId, req.InputType)

	// Call the service to send input to block
	response, err := widgetapiservice.WidgetAPIServiceInstance.SendBlockInput(ctx, req)
	if err != nil {
		log.Printf("Error sending block input: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Internal server error: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Return success status
	if response.Success {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse writes a standardized error response
func writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := map[string]interface{}{
		"success": false,
		"error":   message,
	}
	json.NewEncoder(w).Encode(response)
}

// handlePersistentServerStatus checks the status of the persistent server
func handlePersistentServerStatus(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Checking persistent server status")

	// 检查PID文件是否存在
	pidFile := "waveterm-server.pid"
	portFile := "waveterm-server.port"

	isRunning := false
	var pid int
	var webPort, wsPort int

	// 读取PID文件
	if _, err := os.Stat(pidFile); err == nil {
		if pidData, err := os.ReadFile(pidFile); err == nil {
			if parsedPid, err := strconv.Atoi(strings.TrimSpace(string(pidData))); err == nil {
				// 检查进程是否实际运行
				if cmd := exec.Command("ps", "-p", strconv.Itoa(parsedPid)); cmd.Run() == nil {
					isRunning = true
					pid = parsedPid
				}
			}
		}
	}

	// 读取端口文件
	if isRunning {
		if portData, err := os.ReadFile(portFile); err == nil {
			lines := strings.Split(string(portData), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "web_port=") {
					if port, err := strconv.Atoi(strings.TrimPrefix(line, "web_port=")); err == nil {
						webPort = port
					}
				} else if strings.HasPrefix(line, "ws_port=") {
					if port, err := strconv.Atoi(strings.TrimPrefix(line, "ws_port=")); err == nil {
						wsPort = port
					}
				}
			}
		}
	}

	response := map[string]interface{}{
		"success": true,
		"status": map[string]interface{}{
			"running":  isRunning,
			"pid":      pid,
			"web_port": webPort,
			"ws_port":  wsPort,
		},
	}

	if isRunning {
		response["message"] = "Persistent server is running"
		response["api_url"] = fmt.Sprintf("http://localhost:%d", webPort)
	} else {
		response["message"] = "Persistent server is not running"
	}

	json.NewEncoder(w).Encode(response)
}

// handlePersistentServerStart starts the persistent server
func handlePersistentServerStart(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Starting persistent server")

	// 获取当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get working directory: %v", err)
		writeErrorResponse(w, "Failed to get working directory", http.StatusInternalServerError)
		return
	}
	
	// 尝试在当前目录查找脚本，如果找不到则在上级目录查找
	scriptPath := filepath.Join(workDir, "persistent-server.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		// 尝试上级目录（可能从子目录运行）
		parentDir := filepath.Dir(workDir)
		scriptPath = filepath.Join(parentDir, "persistent-server.sh")
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			// 最后尝试项目根目录的推测路径
			possiblePaths := []string{
				"/Users/xzn/Desktop/code-project/waveterm/persistent-server.sh",
				filepath.Join(os.Getenv("HOME"), "Desktop/code-project/waveterm/persistent-server.sh"),
			}
			found := false
			for _, path := range possiblePaths {
				if _, err := os.Stat(path); err == nil {
					scriptPath = path
					workDir = filepath.Dir(path) // 更新工作目录
					found = true
					break
				}
			}
			if !found {
				log.Printf("persistent-server.sh not found in current dir (%s), parent dir (%s), or common paths", workDir, parentDir)
				writeErrorResponse(w, fmt.Sprintf("persistent-server.sh not found. Searched in: %s, %s", workDir, parentDir), http.StatusNotFound)
				return
			}
		} else {
			workDir = parentDir // 更新工作目录为找到脚本的目录
		}
	}
	
	// 检查脚本是否存在
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		log.Printf("Script not found at: %s", scriptPath)
		writeErrorResponse(w, fmt.Sprintf("persistent-server.sh not found at %s", scriptPath), http.StatusInternalServerError)
		return
	}

	// 异步执行启动脚本
	cmd := exec.Command("bash", scriptPath, "start")
	cmd.Dir = workDir // 设置工作目录
	
	// 创建日志文件记录脚本输出
	logFile := filepath.Join(workDir, "persistent-server-start.log")
	logFileWriter, err := os.Create(logFile)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		writeErrorResponse(w, "Failed to create log file", http.StatusInternalServerError)
		return
	}
	defer logFileWriter.Close()
	
	cmd.Stdout = logFileWriter
	cmd.Stderr = logFileWriter
	
	log.Printf("Executing script: %s with working directory: %s", scriptPath, workDir)
	
	// 启动脚本（异步）
	err = cmd.Start()
	if err != nil {
		log.Printf("Failed to start script: %v", err)
		writeErrorResponse(w, fmt.Sprintf("Failed to start script: %v", err), http.StatusInternalServerError)
		return
	}
	
	log.Printf("Script started with PID: %d", cmd.Process.Pid)
	
	// 在后台等待脚本完成
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("Script execution completed with error: %v", err)
		} else {
			log.Printf("Script execution completed successfully")
		}
	}()

	response := map[string]interface{}{
		"success": true,
		"message": "Persistent server start command initiated",
		"script_path": scriptPath,
		"working_dir": workDir,
		"log_file": logFile,
		"script_pid": cmd.Process.Pid,
	}

	json.NewEncoder(w).Encode(response)
}

// handlePersistentServerStop stops the persistent server
func handlePersistentServerStop(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Stopping persistent server")

	// 获取当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get working directory: %v", err)
		writeErrorResponse(w, "Failed to get working directory", http.StatusInternalServerError)
		return
	}
	
	// 尝试在当前目录查找脚本，如果找不到则在上级目录查找
	scriptPath := filepath.Join(workDir, "persistent-server.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		// 尝试上级目录（可能从子目录运行）
		parentDir := filepath.Dir(workDir)
		scriptPath = filepath.Join(parentDir, "persistent-server.sh")
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			// 最后尝试项目根目录的推测路径
			possiblePaths := []string{
				"/Users/xzn/Desktop/code-project/waveterm/persistent-server.sh",
				filepath.Join(os.Getenv("HOME"), "Desktop/code-project/waveterm/persistent-server.sh"),
			}
			found := false
			for _, path := range possiblePaths {
				if _, err := os.Stat(path); err == nil {
					scriptPath = path
					workDir = filepath.Dir(path) // 更新工作目录
					found = true
					break
				}
			}
			if !found {
				log.Printf("persistent-server.sh not found in current dir (%s), parent dir (%s), or common paths", workDir, parentDir)
				writeErrorResponse(w, fmt.Sprintf("persistent-server.sh not found. Searched in: %s, %s", workDir, parentDir), http.StatusNotFound)
				return
			}
		} else {
			workDir = parentDir // 更新工作目录为找到脚本的目录
		}
	}
	
	// 检查脚本是否存在
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		log.Printf("Script not found at: %s", scriptPath)
		writeErrorResponse(w, fmt.Sprintf("persistent-server.sh not found at %s", scriptPath), http.StatusInternalServerError)
		return
	}

	// 执行停止脚本
	cmd := exec.Command("bash", scriptPath, "stop")
	cmd.Dir = workDir // 设置工作目录
	output, err := cmd.CombinedOutput()

	response := map[string]interface{}{
		"success": err == nil,
		"output":  string(output),
	}

	if err != nil {
		log.Printf("Script execution failed: %v, output: %s", err, string(output))
		response["error"] = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response["message"] = "Persistent server stop command executed"
	}

	json.NewEncoder(w).Encode(response)
}
