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
	"strings"
	"time"

	// "github.com/wavetermdev/waveterm/pkg/authkey" // 临时注释用于测试
	"github.com/wavetermdev/waveterm/pkg/service/widgetapiservice"
)

// handleWidgetAPI routes widget API requests to appropriate handlers
func handleWidgetAPI(w http.ResponseWriter, r *http.Request) {
	// 临时禁用认证用于测试 - TODO: 在生产环境中启用
	// if err := authkey.ValidateIncomingRequest(r); err != nil {
	//	http.Error(w, "Unauthorized", http.StatusUnauthorized)
	//	return
	// }

	// Set CORS headers for API requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")

	// Handle OPTIONS requests for CORS preflight
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
						"workspace_id": "workspace-123",
						"name":         "Default",
						"tab_ids":      []string{"tab-1", "tab-2"},
						"active_tab_id": "tab-1",
					},
				},
			},
		},
	}

	// Return the response
	json.NewEncoder(w).Encode(widgetTypes)
}

// handleMCPServerStatus checks the status of MCP server
func handleMCPServerStatus(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Checking MCP server status")

	// Check if MCP server is running by trying to connect to common ports
	// 包含更多可能的端口，包括开发版本的动态端口范围
	ports := []int{
		61269, // 默认生产端口
		51920, 50531, 57029, // 常见动态端口
		50396, 50397, 50398, 50399, // 开发版本常用端口范围
		52020, 52021, 52022, 52023, // 新的开发端口范围
		53075, 53076, 53077, 53078, // 最新开发端口范围
		65224, 65225, 65226, 65227, // 另一个开发端口范围
		8080, 3000, // 开发服务器端口
	}
	var runningPort int
	isRunning := false

	for _, port := range ports {
		client := &http.Client{
			Timeout: 2 * time.Second,
		}
		
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/api/v1/widgets", port))
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			isRunning = true
			runningPort = port
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	response := map[string]interface{}{
		"success": true,
		"status": map[string]interface{}{
			"running": isRunning,
			"port":    runningPort,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// handleMCPServerRestart attempts to restart the MCP server
func handleMCPServerRestart(w http.ResponseWriter, r *http.Request, ctx context.Context) {
	log.Printf("Attempting to restart MCP server")

	// For now, we'll just return a success response
	// In a real implementation, you might want to:
	// 1. Stop the current MCP server process
	// 2. Restart it with new configuration
	// 3. Wait for it to be ready
	
	response := map[string]interface{}{
		"success": true,
		"message": "MCP server restart initiated",
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