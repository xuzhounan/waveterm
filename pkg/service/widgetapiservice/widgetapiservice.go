// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Widget API service for creating widgets in workspaces via REST API
package widgetapiservice

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wavetermdev/waveterm/pkg/blockcontroller"
	"github.com/wavetermdev/waveterm/pkg/filestore"
	"github.com/wavetermdev/waveterm/pkg/service/workspaceservice"
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

// InitScreenshotEventHandling sets up screenshot response event handling
func InitScreenshotEventHandling() {
	log.Printf("[Screenshot] Backend: Initializing screenshot event handling")
	
	// Note: The actual event handling is done in wshserver.EventPublishCommand
	// which calls the handler set by SetScreenshotResponseHandler in main-server.go
	
	log.Printf("[Screenshot] Backend: Screenshot event handling initialized")
}

// ScreenshotResponse represents the response from frontend screenshot capture
type ScreenshotResponse struct {
	RequestID      string                 `json:"request_id"`
	Success        bool                   `json:"success"`
	ScreenshotData string                 `json:"screenshot_data,omitempty"`
	Format         string                 `json:"format,omitempty"`
	Error          string                 `json:"error,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
}

// screenshotWaitingRequests maps request IDs to response channels
var screenshotWaitingRequests = make(map[string]chan ScreenshotResponse)
var screenshotMutex sync.RWMutex

// getScreenshotDirectory returns the configured screenshot directory or default
func getScreenshotDirectory() string {
	watcher := wconfig.GetWatcher()
	if watcher != nil {
		fullConfig := watcher.GetFullConfig()
		if fullConfig.Settings.ScreenshotDirectory != "" {
			return fullConfig.Settings.ScreenshotDirectory
		}
	}
	// Default directory - use user's Downloads folder
	return "/Users/xzn/Downloads"
}

// ensureScreenshotDirectoryConfig ensures screenshot directory is written to config
func ensureScreenshotDirectoryConfig() error {
	watcher := wconfig.GetWatcher()
	if watcher != nil {
		fullConfig := watcher.GetFullConfig()
		if fullConfig.Settings.ScreenshotDirectory == "" {
			// Set default directory in config
			defaultDir := "/Users/xzn/Downloads"
			configToMerge := waveobj.MetaMapType{
				"screenshot:directory": defaultDir,
			}
			err := wconfig.SetBaseConfigValue(configToMerge)
			if err != nil {
				log.Printf("[Screenshot] Failed to set default directory in config: %v", err)
				return err
			}
			log.Printf("[Screenshot] Set default screenshot directory in config: %s", defaultDir)
		}
	}
	return nil
}


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
	
	
	// Verify that the block was actually added to the tab
	tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
	if err != nil {
	} else {
		// Verify block was added to tab
		for _, bid := range tab.BlockIds {
			if bid == block.OID {
				break
			}
		}
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
	// Add widget to workspace configuration if not ephemeral
	if !req.Ephemeral && req.AddToWorkspace {
		widgetKey := fmt.Sprintf("mcp-%s-%d", req.WidgetType, time.Now().Unix())
		widgetConfig := wconfig.WidgetConfigType{
			Icon:  req.Icon,
			Label: req.Title,
			BlockDef: waveobj.BlockDef{
				Meta: blockDef.Meta,
			},
		}
		
		if widgetConfig.Icon == "" {
			widgetConfig.Icon = "square-terminal"
		}
		
		err = wconfig.SetWorkspaceWidgetConfig(req.WorkspaceId, widgetKey, widgetConfig)
		if err != nil {
			log.Printf("Warning: Failed to add widget to workspace config: %v", err)
		}
	}

	// Send database update events to notify frontend
	updates := waveobj.ContextGetUpdatesRtn(ctx)
	wps.Broker.SendUpdateEvents(updates)

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
		// Map cwd parameter to the correct meta key
		if cwd, exists := customMeta["cwd"]; exists {
			blockDef.Meta["cmd:cwd"] = cwd
			// Remove the raw cwd key to avoid confusion
			delete(blockDef.Meta, "cwd")
		}
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
// Delete Widget API Structures
// ================================

// DeleteWidgetAPIResponse represents the response after deleting a widget
type DeleteWidgetAPIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
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
	// log.Printf("WidgetAPIService.SendBlockInput called with block_id=%s, input_type=%s", req.BlockId, req.InputType)

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
		// Allow empty input_data for cases like sending newlines or empty commands
		inputUnion.InputData = []byte(req.InputData)
		if req.InputData == "" {
			inputDescription = "empty text input (newline/enter)"
		} else {
			inputDescription = fmt.Sprintf("text input (%d bytes)", len(req.InputData))
		}
		
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

	// log.Printf("Successfully sent %s to block %s", inputDescription, req.BlockId)

	return &SendBlockInputAPIResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully sent %s to terminal", inputDescription),
	}, nil
}

// ================================
// Delete Widget API Methods
// ================================

// DeleteWidget deletes a widget/block and optionally its parent containers if empty
func (ws *WidgetAPIService) DeleteWidget(ctx context.Context, blockId string, recursive bool) (*DeleteWidgetAPIResponse, error) {
	log.Printf("WidgetAPIService.DeleteWidget called with block_id=%s, recursive=%t", blockId, recursive)

	if blockId == "" {
		return &DeleteWidgetAPIResponse{
			Success: false,
			Error:   "block_id is required",
		}, nil
	}

	// Validate block exists before attempting deletion
	block, err := wstore.DBGet[*waveobj.Block](ctx, blockId)
	if err != nil || block == nil {
		return &DeleteWidgetAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("block not found: %s", blockId),
		}, nil
	}

	// 确保EventBridge启用以支持MCP实时更新
	if !wps.Bridge.IsEnabled() {
		wps.Bridge.SetEnabled(true)
		log.Printf("EventBridge auto-enabled for MCP widget deletion")
	}

	// Add updates context to collect database changes
	ctx = waveobj.ContextWithUpdates(ctx)

	// Get block info for the response message
	blockType := getBlockTypeFromMeta(block.Meta)
	blockTitle := getStringFromMeta(block.Meta, "title")
	if blockTitle == "" {
		blockTitle = blockType
	}

	// Delete the block using wcore.DeleteBlock which handles:
	// - Stopping controllers
	// - Cleaning up file storage
	// - Removing from parent tab
	// - Optionally deleting empty parent containers
	err = wcore.DeleteBlock(ctx, blockId, recursive)
	if err != nil {
		return &DeleteWidgetAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to delete block: %s", err.Error()),
		}, nil
	}

	// Send database update events to notify frontend
	updates := waveobj.ContextGetUpdatesRtn(ctx)
	wps.Broker.SendUpdateEvents(updates)

	// Clean up file storage for the block
	filestore.WFS.DeleteZone(ctx, blockId)

	log.Printf("Successfully deleted widget: block_id=%s, type=%s", blockId, blockType)

	return &DeleteWidgetAPIResponse{
		Success: true,
		Message: fmt.Sprintf("Widget '%s' (ID: %s) deleted successfully", blockTitle, blockId),
	}, nil
}

// ================================
// Screenshot API Structures
// ================================

// ScreenshotAPIResponse represents the response after capturing a screenshot
type ScreenshotAPIResponse struct {
	Success  bool   `json:"success"`
	Data     string `json:"data,omitempty"`     // Base64 image data with data URI prefix
	FilePath string `json:"file_path,omitempty"` // Path where screenshot was saved (if saved to file)
	FileSize int64  `json:"file_size,omitempty"` // Size of the saved file in bytes
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
}

// CaptureScreenshot captures a screenshot of the specified workspace, tab, or block
func (ws *WidgetAPIService) CaptureScreenshot(ctx context.Context, workspaceId string, tabId string, blockId string, rect map[string]interface{}, format string, savePath string) (*ScreenshotAPIResponse, error) {
	log.Printf("WidgetAPIService.CaptureScreenshot called with workspace_id=%s, tab_id=%s, block_id=%s, savePath=%s", workspaceId, tabId, blockId, savePath)

	// Ensure screenshot directory configuration exists
	err := ensureScreenshotDirectoryConfig()
	if err != nil {
		log.Printf("[Screenshot] Warning: Failed to ensure screenshot directory config: %v", err)
	}

	if workspaceId == "" {
		return &ScreenshotAPIResponse{
			Success: false,
			Error:   "workspace_id is required",
		}, nil
	}

	// Validate workspace exists
	_, err = wcore.GetWorkspace(ctx, workspaceId)
	if err != nil {
		return &ScreenshotAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("workspace not found: %s", err.Error()),
		}, nil
	}

	// Validate tab if specified
	if tabId != "" {
		_, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
		if err != nil {
			return &ScreenshotAPIResponse{
				Success: false,
				Error:   fmt.Sprintf("tab not found: %s", err.Error()),
			}, nil
		}
	}

	// Validate block if specified
	if blockId != "" {
		_, err := wstore.DBGet[*waveobj.Block](ctx, blockId)
		if err != nil {
			return &ScreenshotAPIResponse{
				Success: false,
				Error:   fmt.Sprintf("block not found: %s", err.Error()),
			}, nil
		}
	}

	// Set default format
	if format == "" {
		format = "png"
	}

	// Try to capture a real screenshot using the frontend
	imageDataOrPath, err := ws.captureRealScreenshot(ctx, workspaceId, tabId, blockId, rect, format)
	if err != nil {
		log.Printf("[Screenshot] Backend: Failed to capture real screenshot: %v", err)
		return &ScreenshotAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("Screenshot capture failed: %s", err.Error()),
		}, nil
	}
	
	// Check if the result is a file path (frontend saved directly)
	var imageData string
	var frontendSavedPath string
	var isFrontendSaved bool
	
	// If the result doesn't start with "data:", it's likely a file path
	if !strings.HasPrefix(imageDataOrPath, "data:") && strings.Contains(imageDataOrPath, "/") {
		// This is a file path from frontend
		frontendSavedPath = imageDataOrPath
		isFrontendSaved = true
		log.Printf("[Screenshot] Backend: Using frontend-saved file: %s", frontendSavedPath)
		// Don't load the file data, just use the path
		imageData = "" // Empty data since file is already saved
	} else {
		// This is actual image data
		imageData = imageDataOrPath
		isFrontendSaved = false
	}
	
	message := "Screenshot captured successfully"

	response := &ScreenshotAPIResponse{
		Success: true,
		Data:    imageData,
		Message: message,
	}

	// Handle file path based on whether frontend saved it
	if isFrontendSaved {
		// Frontend already saved the file, just use that path
		response.FilePath = frontendSavedPath
		// Get file size
		if fileInfo, err := os.Stat(frontendSavedPath); err == nil {
			response.FileSize = fileInfo.Size()
			log.Printf("[Screenshot] Frontend-saved file size: %d bytes", response.FileSize)
		} else {
			log.Printf("[Screenshot] Warning: Cannot stat frontend-saved file: %v", err)
		}
		response.Message = "Screenshot captured and saved by frontend"
	} else {
		// We have image data, need to save it
		var actualSavePath string
		log.Printf("[Screenshot] Provided savePath parameter: '%s'", savePath)
		if savePath != "" {
			actualSavePath = savePath
			log.Printf("[Screenshot] Using provided save path: %s", actualSavePath)
		} else {
			// Generate default filename in configured directory
			screenshotDir := getScreenshotDirectory()
			timestamp := time.Now().Format("2006-01-02_15-04-05")
			filename := fmt.Sprintf("waveterm-screenshot_%s.%s", timestamp, format)
			actualSavePath = filepath.Join(screenshotDir, filename)
			log.Printf("[Screenshot] Using default save path: %s", actualSavePath)
		}

		// Validate image data before saving
		log.Printf("[Screenshot] Validating image data: length=%d", len(imageData))
		if imageData == "" {
			log.Printf("[Screenshot] Error: Image data is empty")
			response.Success = false
			response.Error = "Screenshot capture failed: no image data received"
			return response, nil
		}
		
		// Save to file
		log.Printf("[Screenshot] Saving screenshot to: %s", actualSavePath)
		err = ws.saveScreenshotToFile(imageData, actualSavePath)
		if err != nil {
			log.Printf("[Screenshot] Failed to save screenshot to file %s: %v", actualSavePath, err)
			response.Success = false
			response.Error = fmt.Sprintf("Failed to save screenshot: %v", err)
			return response, nil
		}
		
		log.Printf("[Screenshot] Successfully saved screenshot to: %s", actualSavePath)
		response.FilePath = actualSavePath
		// Get file size
		if fileInfo, err := os.Stat(actualSavePath); err == nil {
			response.FileSize = fileInfo.Size()
			log.Printf("[Screenshot] Screenshot file size: %d bytes", response.FileSize)
		}
	}

	return response, nil
}

// generatePlaceholderScreenshot creates a simple placeholder image with workspace information
func (ws *WidgetAPIService) generatePlaceholderScreenshot(workspaceId string, tabId string, blockId string, format string) (string, error) {
	// For now, return a simple 200x100 green PNG
	// This is a valid 200x100 green PNG image in base64
	greenPngBase64 := "iVBORw0KGgoAAAANSUhEUgAAAMgAAABkCAYAAADDhn8LAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAAdgAAAHYBTnsmCAAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAKBSURBVHic7doxAQAACMOwgX+dw/7PShOC5qBVqzYAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAALxjvQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAALVeQSgAEJ5zdmAAAAASUVORK5CYII="
	
	// For a more detailed image, we'll create text-based information
	if format == "jpeg" {
		// For JPEG, we could create a different placeholder or convert
		return fmt.Sprintf("data:image/jpeg;base64,%s", greenPngBase64), nil
	}
	
	return fmt.Sprintf("data:image/png;base64,%s", greenPngBase64), nil
}

// saveScreenshotToFile saves the base64 image data to a file
func (ws *WidgetAPIService) saveScreenshotToFile(imageData string, savePath string) error {
	log.Printf("[Screenshot] === saveScreenshotToFile START ===")
	log.Printf("[Screenshot] savePath: %s", savePath)
	log.Printf("[Screenshot] imageData length: %d", len(imageData))
	
	// Special handling if imageData is empty or invalid
	if imageData == "" {
		log.Printf("[Screenshot] ERROR: imageData is empty")
		return fmt.Errorf("imageData is empty")
	}
	
	// Check if it's a proper data URI
	if !strings.HasPrefix(imageData, "data:") {
		log.Printf("[Screenshot] ERROR: imageData doesn't start with 'data:' - first 50 chars: %s", func() string {
			if len(imageData) > 50 {
				return imageData[:50]
			}
			return imageData
		}())
		return fmt.Errorf("invalid image data format - doesn't start with 'data:'")
	}
	
	// Parse the data URI to extract base64 data
	// Format: "data:image/png;base64,..."
	parts := strings.Split(imageData, ",")
	log.Printf("[Screenshot] Split imageData into %d parts", len(parts))
	if len(parts) != 2 {
		log.Printf("[Screenshot] ERROR: Invalid image data format - expected 2 parts, got %d. Full imageData: %s", len(parts), imageData)
		return fmt.Errorf("invalid image data format - expected 'data:image/format;base64,data' but got %d parts", len(parts))
	}
	
	log.Printf("[Screenshot] Image data header: %s", parts[0])
	log.Printf("[Screenshot] Base64 data length: %d", len(parts[1]))
	log.Printf("[Screenshot] Base64 data first 100 chars: %s", func() string {
		if len(parts[1]) > 100 {
			return parts[1][:100]
		}
		return parts[1]
	}())
	
	// Clean and decode base64 data with enhanced repair logic
	base64Data := ws.cleanBase64Data(parts[1])
	log.Printf("[Screenshot] Cleaned base64 data length: %d", len(base64Data))
	
	if base64Data == "" {
		log.Printf("[Screenshot] ERROR: base64Data is empty after cleaning")
		return fmt.Errorf("base64Data is empty after cleaning")
	}
	
	imageBytes, err := ws.decodeBase64WithFallback(base64Data)
	if err != nil {
		log.Printf("[Screenshot] ERROR: All base64 decoding methods failed: %v", err)
		log.Printf("[Screenshot] Raw base64 data (first 200 chars): %s", func() string {
			if len(base64Data) > 200 {
				return base64Data[:200]
			}
			return base64Data
		}())
		return fmt.Errorf("failed to decode base64 image data after cleanup: %w", err)
	}
	
	log.Printf("[Screenshot] Successfully decoded %d bytes of image data", len(imageBytes))
	
	// Basic sanity check for image data
	if len(imageBytes) < 100 {
		log.Printf("[Screenshot] WARNING: Image data seems too small (%d bytes) - might be corrupted", len(imageBytes))
	}
	
	// Ensure directory exists
	dir := filepath.Dir(savePath)
	log.Printf("[Screenshot] Target directory: %s", dir)
	if dir != "" && dir != "." {
		log.Printf("[Screenshot] Creating directory: %s", dir)
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.Printf("[Screenshot] ERROR: Failed to create directory %s: %v", dir, err)
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		log.Printf("[Screenshot] Directory created/verified successfully: %s", dir)
	}
	
	// Write file  
	log.Printf("[Screenshot] Writing %d bytes to file: %s", len(imageBytes), savePath)
	err = os.WriteFile(savePath, imageBytes, 0644)
	if err != nil {
		log.Printf("[Screenshot] ERROR: Failed to write file %s: %v", savePath, err)
		return fmt.Errorf("failed to write file %s: %w", savePath, err)
	}
	
	// Verify file was written
	if fileInfo, err := os.Stat(savePath); err == nil {
		log.Printf("[Screenshot] SUCCESS: File written successfully: %s (%d bytes)", savePath, fileInfo.Size())
	} else {
		log.Printf("[Screenshot] ERROR: File verification failed: %v", err)
		return fmt.Errorf("file verification failed: %w", err)
	}
	
	log.Printf("[Screenshot] === saveScreenshotToFile END ===")
	return nil
}

// cleanBase64Data removes invalid characters and fixes common base64 issues
func (ws *WidgetAPIService) cleanBase64Data(data string) string {
	// Remove all whitespace characters (spaces, tabs, newlines, etc.)
	data = strings.ReplaceAll(data, " ", "")
	data = strings.ReplaceAll(data, "\t", "")
	data = strings.ReplaceAll(data, "\n", "")
	data = strings.ReplaceAll(data, "\r", "")
	
	// Remove any URL encoding artifacts
	data = strings.ReplaceAll(data, "%3D", "=")
	data = strings.ReplaceAll(data, "%2B", "+")
	data = strings.ReplaceAll(data, "%2F", "/")
	
	// Remove any non-base64 characters (keep only A-Z, a-z, 0-9, +, /, =)
	var cleaned strings.Builder
	for _, char := range data {
		if (char >= 'A' && char <= 'Z') || 
		   (char >= 'a' && char <= 'z') || 
		   (char >= '0' && char <= '9') || 
		   char == '+' || char == '/' || char == '=' {
			cleaned.WriteRune(char)
		}
	}
	
	result := cleaned.String()
	
	// Ensure proper padding
	missing := len(result) % 4
	if missing != 0 {
		result += strings.Repeat("=", 4-missing)
	}
	
	log.Printf("[Screenshot] Base64 cleanup: original length=%d, cleaned length=%d", len(data), len(result))
	return result
}

// decodeBase64WithFallback attempts multiple base64 decoding methods
func (ws *WidgetAPIService) decodeBase64WithFallback(data string) ([]byte, error) {
	var lastErr error
	
	// Method 1: Standard base64 decoding
	imageBytes, err := base64.StdEncoding.DecodeString(data)
	if err == nil {
		log.Printf("[Screenshot] Standard base64 decoding succeeded")
		return imageBytes, nil
	}
	lastErr = err
	log.Printf("[Screenshot] Standard base64 decoding failed: %v", err)
	
	// Method 2: Raw standard encoding (no padding)
	imageBytes, err = base64.RawStdEncoding.DecodeString(data)
	if err == nil {
		log.Printf("[Screenshot] Raw standard base64 decoding succeeded")
		return imageBytes, nil
	}
	log.Printf("[Screenshot] Raw standard base64 decoding failed: %v", err)
	
	// Method 3: URL encoding
	imageBytes, err = base64.URLEncoding.DecodeString(data)
	if err == nil {
		log.Printf("[Screenshot] URL base64 decoding succeeded")
		return imageBytes, nil
	}
	log.Printf("[Screenshot] URL base64 decoding failed: %v", err)
	
	// Method 4: Raw URL encoding
	imageBytes, err = base64.RawURLEncoding.DecodeString(data)
	if err == nil {
		log.Printf("[Screenshot] Raw URL base64 decoding succeeded")
		return imageBytes, nil
	}
	log.Printf("[Screenshot] Raw URL base64 decoding failed: %v", err)
	
	// Method 5: Try removing last few characters in case of corruption
	if len(data) > 10 {
		for i := 1; i <= 5; i++ {
			truncated := data[:len(data)-i]
			// Ensure proper padding for truncated data
			missing := len(truncated) % 4
			if missing != 0 {
				truncated += strings.Repeat("=", 4-missing)
			}
			
			imageBytes, err = base64.StdEncoding.DecodeString(truncated)
			if err == nil {
				log.Printf("[Screenshot] Truncated base64 decoding succeeded (removed %d chars)", i)
				return imageBytes, nil
			}
		}
	}
	
	return nil, fmt.Errorf("all decoding methods failed, last error: %w", lastErr)
}

// HandleScreenshotResponse processes screenshot response events from frontend
func (ws *WidgetAPIService) HandleScreenshotResponse(responseData map[string]interface{}) error {
	log.Printf("[Screenshot] Backend: Received screenshot response: %+v", responseData)
	
	requestID, ok := responseData["request_id"].(string)
	if !ok {
		log.Printf("[Screenshot] Backend: Invalid or missing request_id in response: %+v", responseData)
		return fmt.Errorf("invalid or missing request_id in response")
	}

	log.Printf("[Screenshot] Backend: Processing response for request: %s", requestID)

	screenshotMutex.RLock()
	responseChan, exists := screenshotWaitingRequests[requestID]
	screenshotMutex.RUnlock()

	if !exists {
		log.Printf("[Screenshot] Backend: No waiting request found for ID: %s (possibly expired or already processed)", requestID)
		return fmt.Errorf("no waiting request found for ID: %s", requestID)
	}

	// Build response
	response := ScreenshotResponse{
		RequestID: requestID,
	}

	if success, ok := responseData["success"].(bool); ok {
		response.Success = success
	}

	if screenshotData, ok := responseData["screenshot_data"].(string); ok {
		response.ScreenshotData = screenshotData
	}

	if format, ok := responseData["format"].(string); ok {
		response.Format = format
	}

	if errorMsg, ok := responseData["error"].(string); ok {
		response.Error = errorMsg
	}

	// Handle additional data fields
	if response.Data == nil {
		response.Data = make(map[string]interface{})
	}
	
	// Copy all response data for processing
	for key, value := range responseData {
		response.Data[key] = value
	}

	log.Printf("[Screenshot] Backend: Sending response to waiting channel for request: %s", requestID)
	log.Printf("[Screenshot] Backend: Response contains frontend_saved: %v, file_path: %v", 
		response.Data["frontend_saved"], response.Data["file_path"])
	
	// Send response to waiting goroutine
	select {
	case responseChan <- response:
		log.Printf("[Screenshot] Backend: Response sent successfully")
	default:
		log.Printf("[Screenshot] Backend: Response channel is full or closed")
	}
	return nil
}

// captureRealScreenshot captures a real screenshot using the frontend
func (ws *WidgetAPIService) captureRealScreenshot(ctx context.Context, workspaceId string, tabId string, blockId string, rect map[string]interface{}, format string) (string, error) {
	requestID := uuid.New().String()
	
	// Create response channel
	responseChan := make(chan ScreenshotResponse, 1)
	
	// Register waiting request
	screenshotMutex.Lock()
	screenshotWaitingRequests[requestID] = responseChan
	screenshotMutex.Unlock()
	
	// Cleanup function
	defer func() {
		screenshotMutex.Lock()
		delete(screenshotWaitingRequests, requestID)
		screenshotMutex.Unlock()
		close(responseChan)
	}()

	// Create screenshot request event with enhanced configuration
	screenshotEvent := wps.WaveEvent{
		Event: "screenshot:request",
		Scopes: []string{
			fmt.Sprintf("workspace:%s", workspaceId),
		},
		Data: map[string]interface{}{
			"workspace_id": workspaceId,
			"tab_id":       tabId,
			"block_id":     blockId,
			"rect":         rect,
			"format":       format,
			"request_id":   requestID,
			"capture_mode": "window", // Hint to frontend to capture the main window
			"ensure_visible": true,   // Ensure the target is visible before capture
		},
	}

	log.Printf("[Screenshot] Backend: Sending enhanced screenshot request event with ID: %s (mode: window)", requestID)
	
	// Publish the event to frontend
	wps.Broker.Publish(screenshotEvent)
	
	// Wait for response with extended timeout for window capture and large image data transfer
	timeout := 60 * time.Second
	log.Printf("[Screenshot] Backend: Waiting for response (timeout: %v)", timeout)
	select {
	case response := <-responseChan:
		log.Printf("[Screenshot] Backend: Received response for request: %s, success: %v", requestID, response.Success)
		log.Printf("[Screenshot] Backend: Response data keys: %+v", response.Data)
		if response.Success {
			// Check if frontend saved the file directly
			if frontendSaved, ok := response.Data["frontend_saved"].(bool); ok && frontendSaved {
				log.Printf("[Screenshot] Backend: Frontend saved screenshot directly")
				if filePath, ok := response.Data["file_path"].(string); ok && filePath != "" {
					log.Printf("[Screenshot] Backend: Frontend saved to: %s", filePath)
					// Store the file path for later use in the response
					// We'll handle this specially in CaptureScreenshot method
					return filePath, nil
				}
			}
			
			// Validate that we received meaningful image data
			if len(response.ScreenshotData) < 100 {
				log.Printf("[Screenshot] Backend: Warning - received very small image data (%d bytes)", len(response.ScreenshotData))
				return "", fmt.Errorf("screenshot capture produced empty or invalid image data")
			}
			return response.ScreenshotData, nil
		} else {
			return "", fmt.Errorf("screenshot capture failed: %s", response.Error)
		}
	case <-time.After(timeout):
		log.Printf("[Screenshot] Backend: Timeout (%v) waiting for screenshot response for request: %s", timeout, requestID)
		return "", fmt.Errorf("screenshot capture timeout after %v", timeout)
	case <-ctx.Done():
		log.Printf("[Screenshot] Backend: Context cancelled while waiting for screenshot response for request: %s", requestID)
		return "", fmt.Errorf("screenshot capture cancelled: %v", ctx.Err())
	}
}
