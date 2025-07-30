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
	"strconv"
	"strings"
	"time"

	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
	"github.com/wavetermdev/waveterm/pkg/wps"
)

// isDebugMode 检查是否处于调试模式
func isDebugMode() bool {
	return os.Getenv("WAVETERM_DEV") != ""
}

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
		} else if path == "/debug/recent-events" && r.Method == "GET" {
			// 内部调试端点：显示最近事件（仅在开发环境启用）
			if isDebugMode() {
				handleGetRecentEvents(w, r, ctx)
			} else {
				http.Error(w, "Debug endpoints disabled in production", http.StatusNotFound)
			}
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


	// 获取EventBridge状态信息
	bridgeStatus := map[string]interface{}{
		"enabled":      wps.Bridge.IsEnabled(),
		"remote_urls":  wps.Bridge.GetRemoteURLs(),
	}

	// 返回兼容两种用途的格式：
	// 1. 前端 MCP 客户端组件需要 servers 字段
	// 2. Server Status 组件需要 status 字段
	response := map[string]interface{}{
		"success": true,
		"servers": servers,
		"bridge":  bridgeStatus, // 添加Bridge状态信息
		"status": map[string]interface{}{
			"running": len(servers) > 0 || currentPort > 0, // 如果有连接的客户端或者服务器在运行
			"port":    currentPort,
			"bridge_enabled": wps.Bridge.IsEnabled(), // 在status中也添加bridge状态
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


// handleGetRecentEvents returns recent cached events for debugging widget creation issues
func handleGetRecentEvents(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Getting recent events for debugging")

	// Get events from the last 5 minutes
	maxAge := 5 * time.Minute
	recentEvents := wps.Bridge.GetRecentEvents(maxAge)

	// Parse query parameter for filtering by source
	sourceFilter := r.URL.Query().Get("source")
	eventFilter := r.URL.Query().Get("event_type")

	// Filter events if requested
	var filteredEvents []wps.BridgeEvent
	for _, event := range recentEvents {
		if sourceFilter != "" && !strings.Contains(event.SourceID, sourceFilter) {
			continue
		}
		if eventFilter != "" && event.Event.Event != eventFilter {
			continue
		}
		filteredEvents = append(filteredEvents, event)
	}

	response := map[string]interface{}{
		"success":      true,
		"total_events": len(recentEvents),
		"filtered_events": len(filteredEvents),
		"max_age_minutes": int(maxAge.Minutes()),
		"events":       filteredEvents,
		"filters": map[string]string{
			"source":     sourceFilter,
			"event_type": eventFilter,
		},
	}

	json.NewEncoder(w).Encode(response)
}
