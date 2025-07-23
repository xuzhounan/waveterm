// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

// Widget API service for creating widgets in workspaces via REST API
package widgetapiservice

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/wavetermdev/waveterm/pkg/service/workspaceservice"
	"github.com/wavetermdev/waveterm/pkg/wcore"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wconfig"
	"github.com/wavetermdev/waveterm/pkg/wps"
	"github.com/wavetermdev/waveterm/pkg/wshrpc"
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
	WorkspaceId string `json:"workspace_id"`
	Name        string `json:"name"`
	TabIds      []string `json:"tab_ids"`
	ActiveTabId string `json:"active_tab_id,omitempty"`
}

// CreateWidget creates a new widget in the specified workspace
func (ws *WidgetAPIService) CreateWidget(ctx context.Context, req CreateWidgetAPIRequest) (*CreateWidgetAPIResponse, error) {
	log.Printf("WidgetAPIService.CreateWidget called with workspace_id=%s, widget_type=%s", req.WorkspaceId, req.WidgetType)

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

	blockRef, err := wcore.CreateBlock(ctx, createData.TabId, createData.BlockDef, createData.RtOpts)
	if err != nil {
		return &CreateWidgetAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to create block: %s", err.Error()),
		}, nil
	}

	// Send update event
	wps.Broker.Publish(wps.WaveEvent{
		Event: "block:create",
		Scopes: []string{
			fmt.Sprintf("workspace:%s", req.WorkspaceId),
			fmt.Sprintf("tab:%s", tabId),
		},
		Data: map[string]any{
			"blockid":     blockRef.OID,
			"tabid":       tabId,
			"workspaceid": req.WorkspaceId,
		},
	})

	// Prepare response
	widgetInfo := &WidgetInfo{
		BlockId:     blockRef.OID,
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
		BlockId: blockRef.OID,
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

	return &GetWorkspaceWidgetsAPIResponse{
		Success: true,
		Widgets: widgetConfig,
	}, nil
}

// ListWorkspaces returns a list of all available workspaces
func (ws *WidgetAPIService) ListWorkspaces(ctx context.Context) (*ListWorkspacesAPIResponse, error) {
	log.Printf("WidgetAPIService.ListWorkspaces called")

	workspaceInfos, err := workspaceservice.WorkspaceServiceInstance.ListWorkspaces(ctx)
	if err != nil {
		return &ListWorkspacesAPIResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to list workspaces: %s", err.Error()),
		}, nil
	}

	workspaces := make([]WorkspaceBasicInfo, 0, len(workspaceInfos))
	for _, info := range workspaceInfos {
		if info.WorkspaceData == nil {
			continue
		}
		
		workspaces = append(workspaces, WorkspaceBasicInfo{
			WorkspaceId: info.WorkspaceData.OID,
			Name:        info.WorkspaceData.Name,
			TabIds:      info.WorkspaceData.TabIds,
			ActiveTabId: info.WorkspaceData.ActiveTabId,
		})
	}

	return &ListWorkspacesAPIResponse{
		Success:    true,
		Workspaces: workspaces,
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