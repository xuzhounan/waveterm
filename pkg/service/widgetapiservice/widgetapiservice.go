// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Widget API service for creating widgets in workspaces via REST API
package widgetapiservice

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/wavetermdev/waveterm/pkg/blockcontroller"
	"github.com/wavetermdev/waveterm/pkg/filestore"
	"github.com/wavetermdev/waveterm/pkg/service/workspaceservice"
	"github.com/wavetermdev/waveterm/pkg/util/utilfn"
	"github.com/wavetermdev/waveterm/pkg/wavebase"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wconfig"
	"github.com/wavetermdev/waveterm/pkg/wcore"
	"github.com/wavetermdev/waveterm/pkg/wps"
	"github.com/wavetermdev/waveterm/pkg/wshrpc"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

type WidgetAPIService struct{}

var WidgetAPIServiceInstance = &WidgetAPIService{}

// CreateWidgetAPIRequest represents the REST API request for creating a widget
type CreateWidgetAPIRequest struct {
	WorkspaceId     string          `json:"workspace_id"`
	TabId           string          `json:"tab_id,omitempty"`          // If empty, will use active tab
	WidgetType      string          `json:"widget_type"`               // terminal, web, files, ai, sysinfo, or custom
	Title           string          `json:"title,omitempty"`           // Optional custom title
	Icon            string          `json:"icon,omitempty"`            // Optional custom icon
	Meta            map[string]any  `json:"meta,omitempty"`            // Additional metadata for the widget
	Position        *WidgetPosition `json:"position,omitempty"`        // Where to place the widget
	Magnified       bool            `json:"magnified,omitempty"`       // Whether widget should be magnified
	Ephemeral       bool            `json:"ephemeral,omitempty"`       // Whether widget is temporary
	AddToWorkspace  bool            `json:"add_to_workspace,omitempty"` // Whether to add widget to workspace widget bar (default: true)
}

// WidgetPosition specifies where to place the new widget
type WidgetPosition struct {
	TargetBlockId string `json:"target_block_id,omitempty"` // ID of block to position relative to
	Action        string `json:"action,omitempty"`          // replace, splitright, splitdown, splitleft, splitup
}

// CreateWidgetAPIResponse represents the API response after creating a widget
type CreateWidgetAPIResponse struct {
	Success bool        `json:"success"`
	BlockId string      `json:"block_id,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Widget  *WidgetInfo `json:"widget,omitempty"`
}

// WidgetInfo contains information about the created widget
type WidgetInfo struct {
	BlockId     string         `json:"block_id"`
	TabId       string         `json:"tab_id"`
	WorkspaceId string         `json:"workspace_id"`
	WidgetType  string         `json:"widget_type"`
	Title       string         `json:"title"`
	Icon        string         `json:"icon"`
	Meta        map[string]any `json:"meta"`
	CreatedAt   int64          `json:"created_at"`
}

// GetWorkspaceWidgetsAPIResponse represents available widgets in a workspace
type GetWorkspaceWidgetsAPIResponse struct {
	Success bool                                 `json:"success"`
	Widgets map[string]*wconfig.WidgetConfigType `json:"widgets,omitempty"`
	Error   string                               `json:"error,omitempty"`
}

// ListWorkspacesAPIResponse represents the list of available workspaces
type ListWorkspacesAPIResponse struct {
	Success    bool                 `json:"success"`
	Workspaces []WorkspaceBasicInfo `json:"workspaces,omitempty"`
	Error      string               `json:"error,omitempty"`
}

// WorkspaceBasicInfo contains basic workspace information
type WorkspaceBasicInfo struct {
	WorkspaceId  string    `json:"workspace_id"`
	Name         string    `json:"name"`
	Icon         string    `json:"icon,omitempty"`
	Color        string    `json:"color,omitempty"`
	TabIds       []string  `json:"tab_ids"`
	PinnedTabIds []string  `json:"pinned_tab_ids,omitempty"`
	ActiveTabId  string    `json:"active_tab_id,omitempty"`
	TabsInfo     []TabInfo `json:"tabs_info,omitempty"`
	TotalTabs    int       `json:"total_tabs"`
	TotalBlocks  int       `json:"total_blocks"`
}

// GetWorkspaceByNameAPIResponse represents the response for getting workspace by name
type GetWorkspaceByNameAPIResponse struct {
	Success   bool                `json:"success"`
	Workspace *WorkspaceBasicInfo `json:"workspace,omitempty"`
	Error     string              `json:"error,omitempty"`
}

// GetWorkspaceInfoAPIResponse represents the response for getting detailed workspace info
type GetWorkspaceInfoAPIResponse struct {
	Success   bool                `json:"success"`
	Workspace *WorkspaceBasicInfo `json:"workspace,omitempty"`
	Error     string              `json:"error,omitempty"`
}

// CreateTabAPIRequest represents the request for creating a new tab
type CreateTabAPIRequest struct {
	WorkspaceId string `json:"workspace_id"`
	TabName     string `json:"tab_name,omitempty"` // Optional custom tab name
	Pinned      bool   `json:"pinned,omitempty"`   // Whether tab should be pinned
	Activate    bool   `json:"activate,omitempty"` // Whether to activate the new tab
}

// CreateTabAPIResponse represents the response after creating a tab
type CreateTabAPIResponse struct {
	Success bool     `json:"success"`
	TabId   string   `json:"tab_id,omitempty"`
	Message string   `json:"message,omitempty"`
	Error   string   `json:"error,omitempty"`
	Tab     *TabInfo `json:"tab,omitempty"`
}

// TabInfo contains information about a tab
type TabInfo struct {
	TabId       string   `json:"tab_id"`
	WorkspaceId string   `json:"workspace_id"`
	Name        string   `json:"name"`
	Pinned      bool     `json:"pinned"`
	BlockIds    []string `json:"block_ids"`
	IsActive    bool     `json:"is_active"`
}

// ListTabsAPIResponse represents the response for listing tabs in a workspace
type ListTabsAPIResponse struct {
	Success bool      `json:"success"`
	Tabs    []TabInfo `json:"tabs,omitempty"`
	Error   string    `json:"error,omitempty"`
}

// SetActiveTabAPIRequest represents the request for setting active tab
type SetActiveTabAPIRequest struct {
	WorkspaceId string `json:"workspace_id"`
	TabId       string `json:"tab_id"`
}

// SetActiveTabAPIResponse represents the response after setting active tab
type SetActiveTabAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CreateWidget creates a new widget in the specified workspace
func (ws *WidgetAPIService) CreateWidget(ctx context.Context, req CreateWidgetAPIRequest) (*CreateWidgetAPIResponse, error) {
	log.Printf("WidgetAPIService.CreateWidget called with workspace_id=%s, widget_type=%s", req.WorkspaceId, req.WidgetType)

	// 确保EventBridge启用以支持MCP实时更新
	if !wps.Bridge.IsEnabled() {
		wps.Bridge.SetEnabled(true)
		log.Printf("EventBridge auto-enabled for MCP widget creation")
	}

	// Add updates context to collect database changes
	ctx = waveobj.ContextWithUpdates(ctx)

	// Validate workspace exists
	workspace, err := wcore.GetWorkspace(ctx, req.WorkspaceId)
	if err != nil {
		return &CreateWidgetAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("workspace not found: %s", err.Error()),
		}, nil
	}

	// Get target tab
	tabId := req.TabId
	if tabId == "" {
		// Use active tab if not specified
		tabId = workspace.ActiveTabId
		if tabId == "" && len(workspace.TabIds) > 0 {
			tabId = workspace.TabIds[0]
		}
	}

	if tabId == "" {
		return &CreateWidgetAPIResponse{
			Success: false,
			Error:   "no tab available in workspace",
		}, nil
	}

	// Create block definition based on widget type
	blockDef, err := ws.createBlockDefFromWidgetType(req.WidgetType, req.Meta)
	if err != nil {
		return &CreateWidgetAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create block definition: %s", err.Error()),
		}, nil
	}

	// Add custom title and icon if provided
	if req.Title != "" {
		if blockDef.Meta == nil {
			blockDef.Meta = make(map[string]any)
		}
		blockDef.Meta["title"] = req.Title
	}
	if req.Icon != "" {
		if blockDef.Meta == nil {
			blockDef.Meta = make(map[string]any)
		}
		blockDef.Meta["icon"] = req.Icon
	}

	// Create the block
	createData := wshrpc.CommandCreateBlockData{
		TabId:     tabId,
		BlockDef:  blockDef,
		Magnified: req.Magnified,
		Ephemeral: req.Ephemeral,
	}

	// Set position if specified
	if req.Position != nil {
		createData.TargetBlockId = req.Position.TargetBlockId
		createData.TargetAction = req.Position.Action
	}

	// Create runtime options if not provided
	rtOpts := &waveobj.RuntimeOpts{}

	block, err := wcore.CreateBlock(ctx, tabId, createData.BlockDef, rtOpts)
	if err != nil {
		return &CreateWidgetAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create block: %s", err.Error()),
		}, nil
	}

	// Create layout action to make the block visible in the UI
	layoutAction := &waveobj.LayoutActionData{
		ActionType: wcore.LayoutActionDataType_Insert,
		BlockId:    block.OID,
		Magnified:  req.Magnified,
		Ephemeral:  req.Ephemeral,
		Focused:    true, // Focus the new widget
	}

	// Queue the layout action so the frontend knows how to display the block
	err = wcore.QueueLayoutActionForTab(ctx, tabId, *layoutAction)
	if err != nil {
		return &CreateWidgetAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to queue layout action: %s", err.Error()),
		}, nil
	}

	// Start the block controller if it's a terminal/shell block
	controllerType := getStringFromMeta(block.Meta, "controller")
	if controllerType == "shell" || controllerType == "cmd" {
		log.Printf("Starting controller for block %s (type: %s)", block.OID, controllerType)
		err = blockcontroller.ResyncController(ctx, tabId, block.OID, nil, false)
		if err != nil {
			log.Printf("Warning: Failed to start controller for block %s: %v", block.OID, err)
			// Don't fail the widget creation, just log the warning
		}
	}

	// Add widget to workspace widget configuration to make it appear in widget bar
	// Default to true if not explicitly set to false and not ephemeral
	addToWorkspace := req.AddToWorkspace
	log.Printf("[DEBUG] WidgetAPI: initial addToWorkspace=%v, Ephemeral=%v", addToWorkspace, req.Ephemeral)
	
	// For non-ephemeral widgets, default to adding to workspace if not explicitly set
	if !req.Ephemeral && !addToWorkspace {
		// Check if AddToWorkspace was explicitly set to false in the request
		// If it wasn't provided in JSON, it defaults to false, so we set it to true
		addToWorkspace = true
		log.Printf("[DEBUG] WidgetAPI: setting addToWorkspace=true because not ephemeral and not explicitly disabled")
	}
	
	log.Printf("[DEBUG] WidgetAPI: final addToWorkspace=%v", addToWorkspace)
	if addToWorkspace {
		log.Printf("[DEBUG] WidgetAPI: proceeding to add widget to workspace configuration")
		widgetKey := fmt.Sprintf("mcp-%s-%d", req.WidgetType, time.Now().Unix())
		widgetConfig := wconfig.WidgetConfigType{
			Icon:  req.Icon,
			Label: req.Title,
			BlockDef: waveobj.BlockDef{
				Meta: blockDef.Meta,
			},
		}
		
		// Set default icon if not provided
		if widgetConfig.Icon == "" {
			widgetConfig.Icon = "square-terminal" // Default icon
		}
		
		// Convert to map to set display order
		widgetConfigMap := make(map[string]any)
		utilfn.ReUnmarshal(&widgetConfigMap, widgetConfig)
		widgetConfigMap["display:order"] = 0
		
		// Convert back to WidgetConfigType
		var finalWidgetConfig wconfig.WidgetConfigType
		utilfn.ReUnmarshal(&finalWidgetConfig, widgetConfigMap)
		
		err = wconfig.SetWorkspaceWidgetConfig(req.WorkspaceId, widgetKey, finalWidgetConfig)
		if err != nil {
			log.Printf("Warning: Failed to add widget to workspace config: %v", err)
			// Don't fail the widget creation, just log the warning
		} else {
			log.Printf("Added widget %s to workspace %s configuration", widgetKey, req.WorkspaceId)
		}
	} else {
		log.Printf("[DEBUG] WidgetAPI: skipping workspace configuration because addToWorkspace=false")
	}

	// Send database update events to notify frontend
	// Use the standard method for sending updates to ensure compatibility
	updates := waveobj.ContextGetUpdatesRtn(ctx)
	log.Printf("[DEBUG] WidgetAPI: sending %d update events to WPS broker", len(updates))
	wps.Broker.SendUpdateEvents(updates)
	log.Printf("[DEBUG] WidgetAPI: update events sent successfully")

	// Prepare response
	widgetInfo := &WidgetInfo{
		BlockId:     block.OID,
		TabId:       tabId,
		WorkspaceId: req.WorkspaceId,
		WidgetType:  req.WidgetType,
		Title:       req.Title,
		Icon:        req.Icon,
		Meta:        req.Meta,
		CreatedAt:   time.Now().UnixMilli(),
	}

	return &CreateWidgetAPIResponse{
		Success: true,
		BlockId: block.OID,
		Message: fmt.Sprintf("Widget '%s' created successfully", req.WidgetType),
		Widget:  widgetInfo,
	}, nil
}

// GetWorkspaceWidgets returns the available widget configurations for a workspace
func (ws *WidgetAPIService) GetWorkspaceWidgets(ctx context.Context, workspaceId string) (*GetWorkspaceWidgetsAPIResponse, error) {
	log.Printf("WidgetAPIService.GetWorkspaceWidgets called with workspace_id=%s", workspaceId)

	// Validate workspace exists
	_, err := wcore.GetWorkspace(ctx, workspaceId)
	if err != nil {
		return &GetWorkspaceWidgetsAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("workspace not found: %s", err.Error()),
		}, nil
	}

	// Get widget configuration for the workspace
	// This will return both default widgets and workspace-specific widgets
	widgetConfig, err := wconfig.GetWorkspaceWidgetConfig(ctx, workspaceId)
	if err != nil {
		return &GetWorkspaceWidgetsAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to get widget config: %s", err.Error()),
		}, nil
	}

	// 转换 map[string]wconfig.WidgetConfigType 为 map[string]*wconfig.WidgetConfigType
	widgetConfigPtr := make(map[string]*wconfig.WidgetConfigType)
	for key, config := range widgetConfig {
		configCopy := config
		widgetConfigPtr[key] = &configCopy
	}

	return &GetWorkspaceWidgetsAPIResponse{
		Success: true,
		Widgets: widgetConfigPtr,
	}, nil
}

// ListWorkspaces returns a list of all available workspaces
func (ws *WidgetAPIService) ListWorkspaces(ctx context.Context) (*ListWorkspacesAPIResponse, error) {
	log.Printf("WidgetAPIService.ListWorkspaces called")

	// 使用正确的workspace服务实例
	workspaceService := &workspaceservice.WorkspaceService{}
	workspaceInfos, err := workspaceService.ListWorkspaces()
	if err != nil {
		return &ListWorkspacesAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to list workspaces: %s", err.Error()),
		}, nil
	}

	workspaces := make([]WorkspaceBasicInfo, 0, len(workspaceInfos))
	for _, info := range workspaceInfos {
		// 获取完整的workspace信息
		workspace, err := wcore.GetWorkspace(ctx, info.WorkspaceId)
		if err != nil {
			log.Printf("Failed to get workspace %s: %v", info.WorkspaceId, err)
			continue
		}

		// 收集标签页详细信息
		var tabsInfo []TabInfo
		totalBlocks := 0

		// 处理普通标签页
		for _, tabId := range workspace.TabIds {
			tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
			if err != nil || tab == nil {
				log.Printf("Failed to get tab %s: %v", tabId, err)
				continue
			}

			tabInfo := TabInfo{
				TabId:       tab.OID,
				WorkspaceId: workspace.OID,
				Name:        tab.Name,
				Pinned:      false,
				BlockIds:    tab.BlockIds,
				IsActive:    workspace.ActiveTabId == tab.OID,
			}
			tabsInfo = append(tabsInfo, tabInfo)
			totalBlocks += len(tab.BlockIds)
		}

		// 处理固定标签页
		for _, tabId := range workspace.PinnedTabIds {
			tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
			if err != nil || tab == nil {
				log.Printf("Failed to get pinned tab %s: %v", tabId, err)
				continue
			}

			tabInfo := TabInfo{
				TabId:       tab.OID,
				WorkspaceId: workspace.OID,
				Name:        tab.Name,
				Pinned:      true,
				BlockIds:    tab.BlockIds,
				IsActive:    workspace.ActiveTabId == tab.OID,
			}
			tabsInfo = append(tabsInfo, tabInfo)
			totalBlocks += len(tab.BlockIds)
		}

		workspaces = append(workspaces, WorkspaceBasicInfo{
			WorkspaceId:  workspace.OID,
			Name:         workspace.Name,
			Icon:         workspace.Icon,
			Color:        workspace.Color,
			TabIds:       workspace.TabIds,
			PinnedTabIds: workspace.PinnedTabIds,
			ActiveTabId:  workspace.ActiveTabId,
			TabsInfo:     tabsInfo,
			TotalTabs:    len(workspace.TabIds) + len(workspace.PinnedTabIds),
			TotalBlocks:  totalBlocks,
		})
	}

	return &ListWorkspacesAPIResponse{
		Success:    true,
		Workspaces: workspaces,
	}, nil
}

// GetWorkspaceByName returns workspace information by name
func (ws *WidgetAPIService) GetWorkspaceByName(ctx context.Context, workspaceName string) (*GetWorkspaceByNameAPIResponse, error) {
	log.Printf("WidgetAPIService.GetWorkspaceByName called with workspace_name=%s", workspaceName)

	if workspaceName == "" {
		return &GetWorkspaceByNameAPIResponse{
			Success: false,
			Error:   "workspace name is required",
		}, nil
	}

	// Get all workspaces and find the one with matching name
	workspaceService := &workspaceservice.WorkspaceService{}
	workspaceInfos, err := workspaceService.ListWorkspaces()
	if err != nil {
		return &GetWorkspaceByNameAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to list workspaces: %s", err.Error()),
		}, nil
	}

	// Search for workspace with matching name
	for _, info := range workspaceInfos {
		workspace, err := wcore.GetWorkspace(ctx, info.WorkspaceId)
		if err != nil {
			log.Printf("Failed to get workspace %s: %v", info.WorkspaceId, err)
			continue
		}

		// Case-insensitive name comparison
		if strings.EqualFold(workspace.Name, workspaceName) {
			// 收集标签页详细信息
			var tabsInfo []TabInfo
			totalBlocks := 0

			// 处理普通标签页
			for _, tabId := range workspace.TabIds {
				tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
				if err != nil || tab == nil {
					log.Printf("Failed to get tab %s: %v", tabId, err)
					continue
				}

				tabInfo := TabInfo{
					TabId:       tab.OID,
					WorkspaceId: workspace.OID,
					Name:        tab.Name,
					Pinned:      false,
					BlockIds:    tab.BlockIds,
					IsActive:    workspace.ActiveTabId == tab.OID,
				}
				tabsInfo = append(tabsInfo, tabInfo)
				totalBlocks += len(tab.BlockIds)
			}

			// 处理固定标签页
			for _, tabId := range workspace.PinnedTabIds {
				tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
				if err != nil || tab == nil {
					log.Printf("Failed to get pinned tab %s: %v", tabId, err)
					continue
				}

				tabInfo := TabInfo{
					TabId:       tab.OID,
					WorkspaceId: workspace.OID,
					Name:        tab.Name,
					Pinned:      true,
					BlockIds:    tab.BlockIds,
					IsActive:    workspace.ActiveTabId == tab.OID,
				}
				tabsInfo = append(tabsInfo, tabInfo)
				totalBlocks += len(tab.BlockIds)
			}

			workspaceInfo := &WorkspaceBasicInfo{
				WorkspaceId:  workspace.OID,
				Name:         workspace.Name,
				Icon:         workspace.Icon,
				Color:        workspace.Color,
				TabIds:       workspace.TabIds,
				PinnedTabIds: workspace.PinnedTabIds,
				ActiveTabId:  workspace.ActiveTabId,
				TabsInfo:     tabsInfo,
				TotalTabs:    len(workspace.TabIds) + len(workspace.PinnedTabIds),
				TotalBlocks:  totalBlocks,
			}

			return &GetWorkspaceByNameAPIResponse{
				Success:   true,
				Workspace: workspaceInfo,
			}, nil
		}
	}

	// Workspace not found
	return &GetWorkspaceByNameAPIResponse{
		Success: false,
		Error:   fmt.Sprintf("workspace with name '%s' not found", workspaceName),
	}, nil
}

// GetWorkspaceInfo returns detailed information about a specific workspace by ID
func (ws *WidgetAPIService) GetWorkspaceInfo(ctx context.Context, workspaceId string) (*GetWorkspaceInfoAPIResponse, error) {
	log.Printf("WidgetAPIService.GetWorkspaceInfo called with workspace_id=%s", workspaceId)

	if workspaceId == "" {
		return &GetWorkspaceInfoAPIResponse{
			Success: false,
			Error:   "workspace_id is required",
		}, nil
	}

	// Get the workspace directly by ID
	workspace, err := wcore.GetWorkspace(ctx, workspaceId)
	if err != nil {
		return &GetWorkspaceInfoAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to get workspace: %s", err.Error()),
		}, nil
	}

	if workspace == nil {
		return &GetWorkspaceInfoAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("workspace with ID '%s' not found", workspaceId),
		}, nil
	}

	// Build detailed tab information
	var tabsInfo []TabInfo
	totalBlocks := 0

	// 处理普通标签页
	for _, tabId := range workspace.TabIds {
		tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
		if err != nil || tab == nil {
			log.Printf("Failed to get tab %s: %v", tabId, err)
			continue
		}

		tabInfo := TabInfo{
			TabId:       tab.OID,
			WorkspaceId: workspace.OID,
			Name:        tab.Name,
			Pinned:      false,
			BlockIds:    tab.BlockIds,
			IsActive:    workspace.ActiveTabId == tab.OID,
		}
		tabsInfo = append(tabsInfo, tabInfo)
		totalBlocks += len(tab.BlockIds)
	}

	// 处理固定标签页
	for _, tabId := range workspace.PinnedTabIds {
		tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
		if err != nil || tab == nil {
			log.Printf("Failed to get pinned tab %s: %v", tabId, err)
			continue
		}

		tabInfo := TabInfo{
			TabId:       tab.OID,
			WorkspaceId: workspace.OID,
			Name:        tab.Name,
			Pinned:      true,
			BlockIds:    tab.BlockIds,
			IsActive:    workspace.ActiveTabId == tab.OID,
		}
		tabsInfo = append(tabsInfo, tabInfo)
		totalBlocks += len(tab.BlockIds)
	}

	workspaceInfo := &WorkspaceBasicInfo{
		WorkspaceId: workspace.OID,
		Name:        workspace.Name,
		Icon:        workspace.Icon,
		Color:       workspace.Color,
		TabIds:      append(workspace.TabIds, workspace.PinnedTabIds...),
		ActiveTabId: workspace.ActiveTabId,
		TabsInfo:    tabsInfo,
		TotalTabs:   len(tabsInfo),
		TotalBlocks: totalBlocks,
	}

	return &GetWorkspaceInfoAPIResponse{
		Success:   true,
		Workspace: workspaceInfo,
	}, nil
}

// createBlockDefFromWidgetType creates a BlockDef based on the widget type
func (ws *WidgetAPIService) createBlockDefFromWidgetType(widgetType string, customMeta map[string]any) (*waveobj.BlockDef, error) {
	blockDef := &waveobj.BlockDef{
		Meta: make(map[string]any),
	}

	// Apply custom metadata first
	if customMeta != nil {
		for k, v := range customMeta {
			blockDef.Meta[k] = v
		}
	}

	// Set view and controller based on widget type
	switch widgetType {
	case "terminal":
		blockDef.Meta["view"] = "term"
		blockDef.Meta["controller"] = "shell"
	case "web":
		blockDef.Meta["view"] = "web"
		// Add default URL if not provided
		if _, exists := blockDef.Meta["url"]; !exists {
			blockDef.Meta["url"] = "https://www.waveterm.dev"
		}
	case "files":
		blockDef.Meta["view"] = "preview"
		// Add default file path if not provided
		if _, exists := blockDef.Meta["file"]; !exists {
			blockDef.Meta["file"] = "~"
		}
	case "ai":
		blockDef.Meta["view"] = "waveai"
	case "sysinfo":
		blockDef.Meta["view"] = "sysinfo"
	case "help":
		blockDef.Meta["view"] = "help"
	case "tips":
		blockDef.Meta["view"] = "tips"
	case "serverstatus":
		blockDef.Meta["view"] = "serverstatus"
	default:
		// For custom widget types, assume they provided the view in meta
		if _, exists := blockDef.Meta["view"]; !exists {
			return nil, fmt.Errorf("unknown widget type '%s' and no view specified in meta", widgetType)
		}
	}

	return blockDef, nil
}

// CreateTab creates a new tab in the specified workspace
func (ws *WidgetAPIService) CreateTab(ctx context.Context, req CreateTabAPIRequest) (*CreateTabAPIResponse, error) {
	log.Printf("WidgetAPIService.CreateTab called with workspace_id=%s, tab_name=%s", req.WorkspaceId, req.TabName)

	// Add updates context to collect database changes
	ctx = waveobj.ContextWithUpdates(ctx)

	// Validate workspace exists
	workspace, err := wcore.GetWorkspace(ctx, req.WorkspaceId)
	if err != nil {
		return &CreateTabAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("workspace not found: %s", err.Error()),
		}, nil
	}

	// Create the tab
	tabId, err := wcore.CreateTab(ctx, req.WorkspaceId, req.TabName, req.Activate, req.Pinned, false)
	if err != nil {
		return &CreateTabAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create tab: %s", err.Error()),
		}, nil
	}

	// Get the created tab for response
	tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
	if err != nil {
		return &CreateTabAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to get created tab: %s", err.Error()),
		}, nil
	}

	// Send database update events to notify frontend
	updates := waveobj.ContextGetUpdatesRtn(ctx)
	wps.Broker.SendUpdateEvents(updates)

	// Prepare tab info for response
	tabInfo := &TabInfo{
		TabId:       tab.OID,
		WorkspaceId: req.WorkspaceId,
		Name:        tab.Name,
		Pinned:      req.Pinned,
		BlockIds:    tab.BlockIds,
		IsActive:    workspace.ActiveTabId == tab.OID,
	}

	return &CreateTabAPIResponse{
		Success: true,
		TabId:   tabId,
		Message: fmt.Sprintf("Tab '%s' created successfully", tab.Name),
		Tab:     tabInfo,
	}, nil
}

// ListTabs returns all tabs in the specified workspace
func (ws *WidgetAPIService) ListTabs(ctx context.Context, workspaceId string) (*ListTabsAPIResponse, error) {
	log.Printf("WidgetAPIService.ListTabs called with workspace_id=%s", workspaceId)

	// Validate workspace exists
	workspace, err := wcore.GetWorkspace(ctx, workspaceId)
	if err != nil {
		return &ListTabsAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("workspace not found: %s", err.Error()),
		}, nil
	}

	var tabs []TabInfo

	// Process regular tabs
	for _, tabId := range workspace.TabIds {
		tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
		if err != nil {
			log.Printf("Failed to get tab %s: %v", tabId, err)
			continue
		}
		if tab == nil {
			continue
		}

		tabs = append(tabs, TabInfo{
			TabId:       tab.OID,
			WorkspaceId: workspaceId,
			Name:        tab.Name,
			Pinned:      false,
			BlockIds:    tab.BlockIds,
			IsActive:    workspace.ActiveTabId == tab.OID,
		})
	}

	// Process pinned tabs
	for _, tabId := range workspace.PinnedTabIds {
		tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
		if err != nil {
			log.Printf("Failed to get pinned tab %s: %v", tabId, err)
			continue
		}
		if tab == nil {
			continue
		}

		tabs = append(tabs, TabInfo{
			TabId:       tab.OID,
			WorkspaceId: workspaceId,
			Name:        tab.Name,
			Pinned:      true,
			BlockIds:    tab.BlockIds,
			IsActive:    workspace.ActiveTabId == tab.OID,
		})
	}

	return &ListTabsAPIResponse{
		Success: true,
		Tabs:    tabs,
	}, nil
}

// SetActiveTab sets the active tab in the specified workspace
func (ws *WidgetAPIService) SetActiveTab(ctx context.Context, req SetActiveTabAPIRequest) (*SetActiveTabAPIResponse, error) {
	log.Printf("WidgetAPIService.SetActiveTab called with workspace_id=%s, tab_id=%s", req.WorkspaceId, req.TabId)

	// Add updates context to collect database changes
	ctx = waveobj.ContextWithUpdates(ctx)

	// Validate workspace exists
	_, err := wcore.GetWorkspace(ctx, req.WorkspaceId)
	if err != nil {
		return &SetActiveTabAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("workspace not found: %s", err.Error()),
		}, nil
	}

	// Validate tab exists
	tab, err := wstore.DBGet[*waveobj.Tab](ctx, req.TabId)
	if err != nil || tab == nil {
		return &SetActiveTabAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("tab not found: %s", req.TabId),
		}, nil
	}

	// Set the active tab
	err = wcore.SetActiveTab(ctx, req.WorkspaceId, req.TabId)
	if err != nil {
		return &SetActiveTabAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to set active tab: %s", err.Error()),
		}, nil
	}

	// Send database update events to notify frontend
	updates := waveobj.ContextGetUpdatesRtn(ctx)
	wps.Broker.SendUpdateEvents(updates)

	// Send active tab update event
	wcore.SendActiveTabUpdate(ctx, req.WorkspaceId, req.TabId)

	return &SetActiveTabAPIResponse{
		Success: true,
		Message: fmt.Sprintf("Tab '%s' set as active", tab.Name),
	}, nil
}

// ================================
// Block Content API Structures
// ================================

// GetBlockContentAPIRequest represents the request for getting block content
type GetBlockContentAPIRequest struct {
	BlockId  string `json:"block_id"`
	FileName string `json:"file_name,omitempty"` // Default: "term"
	Offset   int64  `json:"offset,omitempty"`    // Starting position
	Size     int64  `json:"size,omitempty"`      // Max bytes to read, 0 means all
}

// GetBlockContentAPIResponse represents the response for block content
type GetBlockContentAPIResponse struct {
	Success   bool        `json:"success"`
	Content   string      `json:"content,omitempty"`
	Size      int64       `json:"size"`
	FileSize  int64       `json:"file_size"`
	BlockInfo *BlockInfo  `json:"block_info,omitempty"`
	Error     string      `json:"error,omitempty"`
}

// GetBlockStatusAPIResponse represents the response for block status
type GetBlockStatusAPIResponse struct {
	Success     bool              `json:"success"`
	BlockInfo   *BlockInfo        `json:"block_info,omitempty"`
	Controller  *ControllerStatus `json:"controller,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// ListBlocksAPIRequest represents the request for listing blocks
type ListBlocksAPIRequest struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
	TabId       string `json:"tab_id,omitempty"`
	BlockType   string `json:"block_type,omitempty"` // Filter by block type: "terminal", "web", etc.
}

// ListBlocksAPIResponse represents the response for listing blocks
type ListBlocksAPIResponse struct {
	Success bool        `json:"success"`
	Blocks  []BlockInfo `json:"blocks,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// BlockInfo contains detailed information about a block
type BlockInfo struct {
	BlockId     string            `json:"block_id"`
	TabId       string            `json:"tab_id"`
	WorkspaceId string            `json:"workspace_id"`
	BlockType   string            `json:"block_type"`    // "terminal", "web", "files", etc.
	View        string            `json:"view"`          // "term", "web", "preview", etc.
	Controller  string            `json:"controller"`    // "shell", "cmd", etc.
	Meta        map[string]any    `json:"meta"`
	CreatedTs   int64             `json:"created_ts"`
	Files       []BlockFileInfo   `json:"files,omitempty"`
	Status      *ControllerStatus `json:"status,omitempty"`
}

// BlockFileInfo contains information about files in a block
type BlockFileInfo struct {
	FileName  string `json:"file_name"`
	Size      int64  `json:"size"`
	ModTs     int64  `json:"mod_ts"`
	Circular  bool   `json:"circular"`
	MaxSize   int64  `json:"max_size,omitempty"`
}

// ControllerStatus contains controller runtime status
type ControllerStatus struct {
	ControllerType string `json:"controller_type"`
	Status         string `json:"status"`         // "init", "running", "done"
	ExitCode       int    `json:"exit_code"`
	PID            int    `json:"pid,omitempty"`
	StartTs        int64  `json:"start_ts,omitempty"`
	EndTs          int64  `json:"end_ts,omitempty"`
}

// ================================
// Block Input API Structures
// ================================

// SendBlockInputAPIRequest represents the request for sending input to a block
type SendBlockInputAPIRequest struct {
	BlockId    string                `json:"block_id"`
	InputData  string                `json:"input_data,omitempty"`  // Text input to send
	SigName    string                `json:"sig_name,omitempty"`    // Signal name (e.g., "SIGINT", "SIGTERM")
	TermSize   *waveobj.TermSize     `json:"term_size,omitempty"`   // Terminal size for resize
	InputType  string                `json:"input_type,omitempty"` // "text", "signal", "resize"
}

// SendBlockInputAPIResponse represents the response after sending input
type SendBlockInputAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ================================
// Block Content API Methods  
// ================================

// GetBlockContent reads content from a block file (typically terminal output)
func (ws *WidgetAPIService) GetBlockContent(ctx context.Context, blockId string, fileName string, offset int64, size int64) (*GetBlockContentAPIResponse, error) {
	log.Printf("WidgetAPIService.GetBlockContent called with block_id=%s, file=%s, offset=%d, size=%d", blockId, fileName, offset, size)

	if blockId == "" {
		return &GetBlockContentAPIResponse{
			Success: false,
			Error:   "block_id is required",
		}, nil
	}

	// Default to terminal output file
	if fileName == "" {
		fileName = wavebase.BlockFile_Term
	}

	// Get block info first
	block, err := wstore.DBGet[*waveobj.Block](ctx, blockId)
	if err != nil || block == nil {
		return &GetBlockContentAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("block not found: %s", blockId),
		}, nil
	}

	// Read file content
	var content []byte
	var fileSize int64

	if size == 0 {
		// Read entire file
		_, content, err = filestore.WFS.ReadFile(ctx, blockId, fileName)
		if err != nil {
			return &GetBlockContentAPIResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to read file: %s", err.Error()),
			}, nil
		}
		fileSize = int64(len(content))
	} else {
		// Read specific range
		_, content, err = filestore.WFS.ReadAt(ctx, blockId, fileName, offset, size)
		if err != nil {
			return &GetBlockContentAPIResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to read file at offset: %s", err.Error()),
			}, nil
		}
		
		// Get file size
		stat, err := filestore.WFS.Stat(ctx, blockId, fileName)
		if err == nil {
			fileSize = stat.Size
		}
	}

	// Build block info
	blockInfo := &BlockInfo{
		BlockId:    block.OID,
		TabId:      getTabIdFromBlock(block),
		BlockType:  getBlockTypeFromMeta(block.Meta),
		View:       getStringFromMeta(block.Meta, "view"),
		Controller: getStringFromMeta(block.Meta, "controller"),
		Meta:       block.Meta,
		CreatedTs:  time.Now().UnixMilli(), // Use current time as fallback
	}

	// Get workspace ID by finding which workspace contains this tab
	if blockInfo.TabId != "" {
		workspaceId, err := wstore.DBFindWorkspaceForTabId(ctx, blockInfo.TabId)
		if err == nil {
			blockInfo.WorkspaceId = workspaceId
		}
	}

	return &GetBlockContentAPIResponse{
		Success:   true,
		Content:   string(content),
		Size:      int64(len(content)),
		FileSize:  fileSize,
		BlockInfo: blockInfo,
	}, nil
}

// GetBlockStatus gets the status and metadata of a block
func (ws *WidgetAPIService) GetBlockStatus(ctx context.Context, blockId string) (*GetBlockStatusAPIResponse, error) {
	log.Printf("WidgetAPIService.GetBlockStatus called with block_id=%s", blockId)

	if blockId == "" {
		return &GetBlockStatusAPIResponse{
			Success: false,
			Error:   "block_id is required",
		}, nil
	}

	// Get block from database
	block, err := wstore.DBGet[*waveobj.Block](ctx, blockId)
	if err != nil || block == nil {
		return &GetBlockStatusAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("block not found: %s", blockId),
		}, nil
	}

	// Build block info
	blockInfo := &BlockInfo{
		BlockId:    block.OID,
		TabId:      getTabIdFromBlock(block),
		BlockType:  getBlockTypeFromMeta(block.Meta),
		View:       getStringFromMeta(block.Meta, "view"),
		Controller: getStringFromMeta(block.Meta, "controller"),
		Meta:       block.Meta,
		CreatedTs:  time.Now().UnixMilli(), // Use current time as fallback
	}

	// Get workspace ID by finding which workspace contains this tab
	if blockInfo.TabId != "" {
		workspaceId, err := wstore.DBFindWorkspaceForTabId(ctx, blockInfo.TabId)
		if err == nil {
			blockInfo.WorkspaceId = workspaceId
		}
	}

	// Get block files info
	files := []BlockFileInfo{}
	
	// Check common block files
	commonFiles := []string{wavebase.BlockFile_Term, wavebase.BlockFile_VDom, "env", "history"}
	for _, fileName := range commonFiles {
		stat, err := filestore.WFS.Stat(ctx, blockId, fileName)
		if err == nil {
			files = append(files, BlockFileInfo{
				FileName: fileName,
				Size:     stat.Size,
				ModTs:    stat.ModTs,
				Circular: stat.Opts.Circular,
				MaxSize:  stat.Opts.MaxSize,
			})
		}
	}
	blockInfo.Files = files

	// Get controller status if it's a shell/terminal block
	var controllerStatus *ControllerStatus
	if blockInfo.Controller == "shell" || blockInfo.Controller == "cmd" {
		bc := blockcontroller.GetBlockController(blockId)
		if bc != nil {
			status := bc.GetRuntimeStatus()
			controllerStatus = &ControllerStatus{
				ControllerType: bc.ControllerType,
				Status:         status.ShellProcStatus,
				ExitCode:       status.ShellProcExitCode,
				StartTs:        time.Now().UnixMilli(), // Use current time as fallback
			}
		}
	}

	return &GetBlockStatusAPIResponse{
		Success:    true,
		BlockInfo:  blockInfo,
		Controller: controllerStatus,
	}, nil
}

// ListBlocks lists blocks in a workspace or tab with optional filtering
func (ws *WidgetAPIService) ListBlocks(ctx context.Context, req ListBlocksAPIRequest) (*ListBlocksAPIResponse, error) {
	log.Printf("WidgetAPIService.ListBlocks called with workspace_id=%s, tab_id=%s, block_type=%s", req.WorkspaceId, req.TabId, req.BlockType)

	var blocks []BlockInfo
	var err error

	if req.TabId != "" {
		// List blocks in specific tab
		blocks, err = ws.listBlocksInTab(ctx, req.TabId, req.BlockType)
	} else if req.WorkspaceId != "" {
		// List blocks in all tabs of workspace
		blocks, err = ws.listBlocksInWorkspace(ctx, req.WorkspaceId, req.BlockType)
	} else {
		return &ListBlocksAPIResponse{
			Success: false,
			Error:   "either workspace_id or tab_id is required",
		}, nil
	}

	if err != nil {
		return &ListBlocksAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to list blocks: %s", err.Error()),
		}, nil
	}

	return &ListBlocksAPIResponse{
		Success: true,
		Blocks:  blocks,
	}, nil
}

// Helper methods

func (ws *WidgetAPIService) listBlocksInTab(ctx context.Context, tabId string, blockType string) ([]BlockInfo, error) {
	tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
	if err != nil || tab == nil {
		return nil, fmt.Errorf("tab not found: %s", tabId)
	}

	var blocks []BlockInfo
	for _, blockId := range tab.BlockIds {
		block, err := wstore.DBGet[*waveobj.Block](ctx, blockId)
		if err != nil || block == nil {
			continue
		}

		blockInfo := ws.buildBlockInfo(ctx, block)
		
		// Filter by block type if specified
		if blockType != "" && blockInfo.BlockType != blockType {
			continue
		}

		blocks = append(blocks, blockInfo)
	}

	return blocks, nil
}

func (ws *WidgetAPIService) listBlocksInWorkspace(ctx context.Context, workspaceId string, blockType string) ([]BlockInfo, error) {
	workspace, err := wcore.GetWorkspace(ctx, workspaceId)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %s", workspaceId)
	}

	var blocks []BlockInfo
	
	// Process all tabs (regular + pinned)
	allTabIds := append(workspace.TabIds, workspace.PinnedTabIds...)
	
	for _, tabId := range allTabIds {
		tabBlocks, err := ws.listBlocksInTab(ctx, tabId, blockType)
		if err != nil {
			log.Printf("Failed to list blocks in tab %s: %v", tabId, err)
			continue
		}
		blocks = append(blocks, tabBlocks...)
	}

	return blocks, nil
}

func (ws *WidgetAPIService) buildBlockInfo(ctx context.Context, block *waveobj.Block) BlockInfo {
	blockInfo := BlockInfo{
		BlockId:    block.OID,
		TabId:      getTabIdFromBlock(block),
		BlockType:  getBlockTypeFromMeta(block.Meta),
		View:       getStringFromMeta(block.Meta, "view"),
		Controller: getStringFromMeta(block.Meta, "controller"),
		Meta:       block.Meta,
		CreatedTs:  time.Now().UnixMilli(), // Use current time as fallback
	}

	// Get workspace ID by finding which workspace contains this tab
	if blockInfo.TabId != "" {
		workspaceId, err := wstore.DBFindWorkspaceForTabId(ctx, blockInfo.TabId)
		if err == nil {
			blockInfo.WorkspaceId = workspaceId
		}
	}

	return blockInfo
}

func getBlockTypeFromMeta(meta map[string]any) string {
	view := getStringFromMeta(meta, "view")
	switch view {
	case "term":
		return "terminal"
	case "web":
		return "web"
	case "preview":
		return "files"
	case "waveai":
		return "ai"
	case "sysinfo":
		return "sysinfo"
	default:
		return view
	}
}

func getStringFromMeta(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	if val, ok := meta[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getTabIdFromBlock extracts the TabId from a block's ParentORef
func getTabIdFromBlock(block *waveobj.Block) string {
	if block == nil || block.ParentORef == "" {
		return ""
	}
	
	parentORef := waveobj.ParseORefNoErr(block.ParentORef)
	if parentORef == nil {
		return ""
	}
	
	// Only return TabId if the parent is actually a Tab
	if parentORef.OType == waveobj.OType_Tab {
		return parentORef.OID
	}
	
	return ""
}

// ================================
// Block Input API Methods
// ================================

// SendBlockInput sends input (text, signals, or terminal resize) to a block
func (ws *WidgetAPIService) SendBlockInput(ctx context.Context, req SendBlockInputAPIRequest) (*SendBlockInputAPIResponse, error) {
	log.Printf("WidgetAPIService.SendBlockInput called with block_id=%s, input_type=%s", req.BlockId, req.InputType)

	if req.BlockId == "" {
		return &SendBlockInputAPIResponse{
			Success: false,
			Error:   "block_id is required",
		}, nil
	}

	// Validate block exists
	block, err := wstore.DBGet[*waveobj.Block](ctx, req.BlockId)
	if err != nil || block == nil {
		return &SendBlockInputAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("block not found: %s", req.BlockId),
		}, nil
	}

	// Validate that this is a terminal block
	blockType := getBlockTypeFromMeta(block.Meta)
	controller := getStringFromMeta(block.Meta, "controller")
	if blockType != "terminal" || (controller != "shell" && controller != "cmd") {
		return &SendBlockInputAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("block %s is not a terminal (type: %s, controller: %s)", req.BlockId, blockType, controller),
		}, nil
	}

	// Get the block controller
	bc := blockcontroller.GetBlockController(req.BlockId)
	if bc == nil {
		return &SendBlockInputAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("no controller found for block %s", req.BlockId),
		}, nil
	}

	// Build input union based on request type
	inputUnion := &blockcontroller.BlockInputUnion{}
	var inputDescription string

	// Default input type to "text" if not specified but input_data is provided
	if req.InputType == "" && req.InputData != "" {
		req.InputType = "text"
	}

	switch req.InputType {
	case "text", "":
		if req.InputData == "" {
			return &SendBlockInputAPIResponse{
				Success: false,
				Error:   "input_data is required for text input",
			}, nil
		}
		inputUnion.InputData = []byte(req.InputData)
		inputDescription = fmt.Sprintf("text input (%d bytes)", len(req.InputData))
		
	case "signal":
		if req.SigName == "" {
			return &SendBlockInputAPIResponse{
				Success: false,
				Error:   "sig_name is required for signal input",
			}, nil
		}
		inputUnion.SigName = req.SigName
		inputDescription = fmt.Sprintf("signal %s", req.SigName)
		
	case "resize":
		if req.TermSize == nil {
			return &SendBlockInputAPIResponse{
				Success: false,
				Error:   "term_size is required for resize input",
			}, nil
		}
		inputUnion.TermSize = req.TermSize
		inputDescription = fmt.Sprintf("terminal resize to %dx%d", req.TermSize.Cols, req.TermSize.Rows)
		
	default:
		return &SendBlockInputAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("invalid input_type: %s (must be 'text', 'signal', or 'resize')", req.InputType),
		}, nil
	}

	// Send input to the block controller
	err = bc.SendInput(inputUnion)
	if err != nil {
		return &SendBlockInputAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to send input to block controller: %s", err.Error()),
		}, nil
	}

	log.Printf("Successfully sent %s to block %s", inputDescription, req.BlockId)

	return &SendBlockInputAPIResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully sent %s to terminal", inputDescription),
	}, nil
}
