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

	"github.com/wavetermdev/waveterm/pkg/service/workspaceservice"
	"github.com/wavetermdev/waveterm/pkg/wcore"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wconfig"
	"github.com/wavetermdev/waveterm/pkg/wps"
	"github.com/wavetermdev/waveterm/pkg/wshrpc"
	"github.com/wavetermdev/waveterm/pkg/wstore"
)

type WidgetAPIService struct{}

var WidgetAPIServiceInstance = &WidgetAPIService{}

// CreateWidgetAPIRequest represents the REST API request for creating a widget
type CreateWidgetAPIRequest struct {
	WorkspaceId   string            `json:"workspace_id"`
	TabId         string            `json:"tab_id,omitempty"`         // If empty, will use active tab
	WidgetType    string            `json:"widget_type"`              // terminal, web, files, ai, sysinfo, or custom
	Title         string            `json:"title,omitempty"`          // Optional custom title
	Icon          string            `json:"icon,omitempty"`           // Optional custom icon
	Meta          map[string]any    `json:"meta,omitempty"`           // Additional metadata for the widget
	Position      *WidgetPosition   `json:"position,omitempty"`       // Where to place the widget
	Magnified     bool              `json:"magnified,omitempty"`      // Whether widget should be magnified
	Ephemeral     bool              `json:"ephemeral,omitempty"`      // Whether widget is temporary
}

// WidgetPosition specifies where to place the new widget
type WidgetPosition struct {
	TargetBlockId string `json:"target_block_id,omitempty"` // ID of block to position relative to
	Action        string `json:"action,omitempty"`          // replace, splitright, splitdown, splitleft, splitup
}

// CreateWidgetAPIResponse represents the API response after creating a widget
type CreateWidgetAPIResponse struct {
	Success   bool                `json:"success"`
	BlockId   string              `json:"block_id,omitempty"`
	Message   string              `json:"message,omitempty"`
	Error     string              `json:"error,omitempty"`
	Widget    *WidgetInfo         `json:"widget,omitempty"`
}

// WidgetInfo contains information about the created widget
type WidgetInfo struct {
	BlockId     string            `json:"block_id"`
	TabId       string            `json:"tab_id"`
	WorkspaceId string            `json:"workspace_id"`
	WidgetType  string            `json:"widget_type"`
	Title       string            `json:"title"`
	Icon        string            `json:"icon"`
	Meta        map[string]any    `json:"meta"`
	CreatedAt   int64             `json:"created_at"`
}

// GetWorkspaceWidgetsAPIResponse represents available widgets in a workspace
type GetWorkspaceWidgetsAPIResponse struct {
	Success bool                             `json:"success"`
	Widgets map[string]*wconfig.WidgetConfigType `json:"widgets,omitempty"`
	Error   string                           `json:"error,omitempty"`
}

// ListWorkspacesAPIResponse represents the list of available workspaces
type ListWorkspacesAPIResponse struct {
	Success    bool                  `json:"success"`
	Workspaces []WorkspaceBasicInfo  `json:"workspaces,omitempty"`
	Error      string                `json:"error,omitempty"`
}

// WorkspaceBasicInfo contains basic workspace information
type WorkspaceBasicInfo struct {
	WorkspaceId    string     `json:"workspace_id"`
	Name           string     `json:"name"`
	Icon           string     `json:"icon,omitempty"`
	Color          string     `json:"color,omitempty"`
	TabIds         []string   `json:"tab_ids"`
	PinnedTabIds   []string   `json:"pinned_tab_ids,omitempty"`
	ActiveTabId    string     `json:"active_tab_id,omitempty"`
	TabsInfo       []TabInfo  `json:"tabs_info,omitempty"`
	TotalTabs      int        `json:"total_tabs"`
	TotalBlocks    int        `json:"total_blocks"`
}

// GetWorkspaceByNameAPIResponse represents the response for getting workspace by name
type GetWorkspaceByNameAPIResponse struct {
	Success   bool                 `json:"success"`
	Workspace *WorkspaceBasicInfo  `json:"workspace,omitempty"`
	Error     string               `json:"error,omitempty"`
}

// GetWorkspaceInfoAPIResponse represents the response for getting detailed workspace info
type GetWorkspaceInfoAPIResponse struct {
	Success   bool                 `json:"success"`
	Workspace *WorkspaceBasicInfo  `json:"workspace,omitempty"`
	Error     string               `json:"error,omitempty"`
}

// CreateTabAPIRequest represents the request for creating a new tab
type CreateTabAPIRequest struct {
	WorkspaceId string `json:"workspace_id"`
	TabName     string `json:"tab_name,omitempty"`    // Optional custom tab name
	Pinned      bool   `json:"pinned,omitempty"`      // Whether tab should be pinned
	Activate    bool   `json:"activate,omitempty"`    // Whether to activate the new tab
}

// CreateTabAPIResponse represents the response after creating a tab
type CreateTabAPIResponse struct {
	Success bool      `json:"success"`
	TabId   string    `json:"tab_id,omitempty"`
	Message string    `json:"message,omitempty"`
	Error   string    `json:"error,omitempty"`
	Tab     *TabInfo  `json:"tab,omitempty"`
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
	Success bool       `json:"success"`
	Tabs    []TabInfo  `json:"tabs,omitempty"`
	Error   string     `json:"error,omitempty"`
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
		TabId:      tabId,
		BlockDef:   blockDef,
		Magnified:  req.Magnified,
		Ephemeral:  req.Ephemeral,
	}

	// Set position if specified
	if req.Position != nil {
		createData.TargetBlockId = req.Position.TargetBlockId
		createData.TargetAction = req.Position.Action
	}

	// Create runtime options if not provided
	rtOpts := &waveobj.RuntimeOpts{}
	
	block, err := wcore.CreateBlock(ctx, createData.TabId, createData.BlockDef, rtOpts)
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
		Focused:    true,  // Focus the new widget
	}
	
	// Queue the layout action so the frontend knows how to display the block
	err = wcore.QueueLayoutActionForTab(ctx, tabId, *layoutAction)
	if err != nil {
		return &CreateWidgetAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to queue layout action: %s", err.Error()),
		}, nil
	}

	// Send database update events to notify frontend
	// This includes updates for the block, tab, and any other objects that were modified
	updates := waveobj.ContextGetUpdatesRtn(ctx)
	wps.Broker.SendUpdateEvents(updates)

	// Send focused layout state update to ensure the new widget is displayed
	// This is the minimal event needed to refresh the UI properly
	layoutStateId, err := wcore.GetLayoutIdForTab(ctx, tabId)
	if err == nil {
		wps.Broker.Publish(wps.WaveEvent{
			Event: wps.Event_WaveObjUpdate,
			Scopes: []string{
				waveobj.MakeORef(waveobj.OType_LayoutState, layoutStateId).String(),
			},
			Data: waveobj.WaveObjUpdate{
				UpdateType: waveobj.UpdateType_Update,
				OType:      waveobj.OType_LayoutState,
				OID:        layoutStateId,
			},
		})
	}

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
		WorkspaceId:   workspace.OID,
		Name:          workspace.Name,
		Icon:          workspace.Icon,
		Color:         workspace.Color,
		TabIds:        append(workspace.TabIds, workspace.PinnedTabIds...),
		ActiveTabId:   workspace.ActiveTabId,
		TabsInfo:      tabsInfo,
		TotalTabs:     len(tabsInfo),
		TotalBlocks:   totalBlocks,
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