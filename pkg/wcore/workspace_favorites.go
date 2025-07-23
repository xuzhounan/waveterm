// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package wcore

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wconfig"
	"github.com/wavetermdev/waveterm/pkg/wstore"
	"github.com/wavetermdev/waveterm/pkg/util/utilfn"
)

const WorkspaceFavoritesConfigFile = "workspace-favorites.json"

// SaveWorkspaceAsFavorite 将当前工作区保存为收藏配置，如果同名收藏已存在则更新
func SaveWorkspaceAsFavorite(ctx context.Context, workspaceId string, favoriteName string, description string, tags []string) (*waveobj.WorkspaceFavorite, error) {
	if favoriteName == "" {
		return nil, fmt.Errorf("favorite name cannot be empty")
	}

	// 检查是否已存在同名收藏，如果存在则更新
	existingFavorites, err := ListWorkspaceFavorites()
	if err != nil {
		log.Printf("warning: failed to check existing favorites: %v", err)
	}
	
	var existingFavorite *waveobj.WorkspaceFavorite
	if existingFavorites != nil {
		for _, favorite := range existingFavorites {
			if favorite.Name == favoriteName {
				existingFavorite = favorite
				break
			}
		}
	}

	// 获取工作区信息
	workspace, err := GetWorkspace(ctx, workspaceId)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %w", err)
	}

	// 获取工作区的标签页信息，包括完整的布局和块配置
	var defaultTabs []waveobj.DefaultTabConfig
	allTabIds := append(workspace.PinnedTabIds, workspace.TabIds...)
	for _, tabId := range allTabIds {
		tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
		if err != nil || tab == nil {
			log.Printf("warning: could not get tab %s: %v", tabId, err)
			continue
		}
		
		isPinned := false
		for _, pinnedId := range workspace.PinnedTabIds {
			if pinnedId == tabId {
				isPinned = true
				break
			}
		}

		// 保存完整的布局状态
		var savedLayoutState *waveobj.SavedLayoutState
		if tab.LayoutState != "" {
			layoutState, err := wstore.DBGet[*waveobj.LayoutState](ctx, tab.LayoutState)
			if err == nil && layoutState != nil {
				savedLayoutState = &waveobj.SavedLayoutState{
					RootNode:         layoutState.RootNode,
					MagnifiedNodeId:  layoutState.MagnifiedNodeId,
					FocusedNodeId:    layoutState.FocusedNodeId,
					LeafOrder:        layoutState.LeafOrder,
					Meta:             layoutState.Meta,
				}
			}
		}

		// 保存所有块的完整配置
		var savedBlocks []*waveobj.SavedBlock
		for _, blockId := range tab.BlockIds {
			block, err := wstore.DBGet[*waveobj.Block](ctx, blockId)
			if err != nil || block == nil {
				log.Printf("warning: could not get block %s: %v", blockId, err)
				continue
			}

			savedBlock := &waveobj.SavedBlock{
				OriginalOID:  blockId,
				ParentORef:   block.ParentORef,
				RuntimeOpts:  block.RuntimeOpts,
				Stickers:     block.Stickers,
				Meta:         block.Meta,
				SubBlockIds:  block.SubBlockIds,
			}
			savedBlocks = append(savedBlocks, savedBlock)
		}

		defaultTab := waveobj.DefaultTabConfig{
			Name:        tab.Name,
			Pinned:      isPinned,
			Meta:        tab.Meta,
			LayoutState: savedLayoutState,
			Blocks:      savedBlocks,
		}
		defaultTabs = append(defaultTabs, defaultTab)
	}

	// 获取工作区级别的小组件配置
	config := wconfig.GetWatcher().GetFullConfig()
	widgetConfigs := make(map[string]waveobj.WidgetConfig)
	if workspaceWidgets, exists := config.WorkspaceWidgets[workspaceId]; exists {
		for key, widgetConfig := range workspaceWidgets {
			widgetConfigs[key] = waveobj.WidgetConfig{
				DisplayOrder:  widgetConfig.DisplayOrder,
				DisplayHidden: widgetConfig.DisplayHidden,
				Icon:          widgetConfig.Icon,
				Color:         widgetConfig.Color,
				Label:         widgetConfig.Label,
				Description:   widgetConfig.Description,
				Magnified:     widgetConfig.Magnified,
				BlockDef:      widgetConfig.BlockDef,
			}
		}
	}

	// 创建或更新收藏配置
	now := time.Now()
	var favorite *waveobj.WorkspaceFavorite
	
	if existingFavorite != nil {
		// 更新现有收藏，保留使用次数和创建时间
		favorite = &waveobj.WorkspaceFavorite{
			FavoriteId:    existingFavorite.FavoriteId,
			Name:          favoriteName,
			Description:   description,
			Icon:          workspace.Icon,
			Color:         workspace.Color,
			Tags:          tags,
			CreatedAt:     existingFavorite.CreatedAt,
			UpdatedAt:     now,
			UsageCount:    existingFavorite.UsageCount,
			DefaultTabs:   defaultTabs,
			WidgetConfigs: widgetConfigs,
			Meta:          workspace.Meta,
		}
		log.Printf("updating existing workspace favorite: %s", favoriteName)
	} else {
		// 创建新收藏
		favorite = &waveobj.WorkspaceFavorite{
			FavoriteId:    uuid.NewString(),
			Name:          favoriteName,
			Description:   description,
			Icon:          workspace.Icon,
			Color:         workspace.Color,
			Tags:          tags,
			CreatedAt:     now,
			UpdatedAt:     now,
			UsageCount:    0,
			DefaultTabs:   defaultTabs,
			WidgetConfigs: widgetConfigs,
			Meta:          workspace.Meta,
		}
		log.Printf("creating new workspace favorite: %s", favoriteName)
	}

	// 保存到配置文件
	err = saveFavoriteToConfig(favorite)
	if err != nil {
		return nil, fmt.Errorf("failed to save favorite: %w", err)
	}

	if existingFavorite != nil {
		log.Printf("updated workspace %s as favorite: %s", workspaceId, favoriteName)
	} else {
		log.Printf("saved workspace %s as favorite: %s", workspaceId, favoriteName)
	}
	return favorite, nil
}

// ListWorkspaceFavorites 获取所有工作区收藏配置
func ListWorkspaceFavorites() ([]*waveobj.WorkspaceFavorite, error) {
	favoritesData, cerrs := wconfig.ReadWaveHomeConfigFile(WorkspaceFavoritesConfigFile)
	if len(cerrs) > 0 {
		// 如果文件不存在，返回空列表
		for _, err := range cerrs {
			if err.Err != "" && (contains(err.Err, "no such file") || contains(err.Err, "cannot find")) {
				return []*waveobj.WorkspaceFavorite{}, nil
			}
		}
		return nil, fmt.Errorf("error reading favorites config: %v", cerrs[0])
	}

	if favoritesData == nil {
		return []*waveobj.WorkspaceFavorite{}, nil
	}

	var favorites []*waveobj.WorkspaceFavorite
	for _, favoriteData := range favoritesData {
		var favorite waveobj.WorkspaceFavorite
		err := utilfn.ReUnmarshal(&favorite, favoriteData)
		if err != nil {
			log.Printf("warning: failed to unmarshal favorite: %v", err)
			continue
		}
		favorites = append(favorites, &favorite)
	}

	// 按名称排序
	sort.Slice(favorites, func(i, j int) bool {
		return favorites[i].Name < favorites[j].Name
	})

	return favorites, nil
}

// GetWorkspaceFavorite 获取指定的工作区收藏配置
func GetWorkspaceFavorite(favoriteId string) (*waveobj.WorkspaceFavorite, error) {
	favorites, err := ListWorkspaceFavorites()
	if err != nil {
		return nil, err
	}

	for _, favorite := range favorites {
		if favorite.FavoriteId == favoriteId {
			return favorite, nil
		}
	}

	return nil, fmt.Errorf("favorite not found: %s", favoriteId)
}

// CreateWorkspaceFromFavorite 从收藏配置创建新的工作区
func CreateWorkspaceFromFavorite(ctx context.Context, favoriteId string) (*waveobj.Workspace, error) {
	favorite, err := GetWorkspaceFavorite(favoriteId)
	if err != nil {
		return nil, fmt.Errorf("favorite not found: %w", err)
	}

	// 创建基础工作区
	workspace, err := CreateWorkspace(ctx, favorite.Name, favorite.Icon, favorite.Color, false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	// 应用收藏配置中的默认标签页（包括完整的布局恢复）
	if len(favorite.DefaultTabs) > 0 {
		// 删除默认创建的标签页
		if len(workspace.TabIds) > 0 {
			_, err = DeleteTab(ctx, workspace.OID, workspace.TabIds[0], false)
			if err != nil {
				log.Printf("warning: failed to delete default tab: %v", err)
			}
		}

		// 创建收藏配置中定义的标签页
		for _, defaultTab := range favorite.DefaultTabs {
			// 恢复完整的标签页布局和块配置
			restoredTabId, err := restoreTabFromFavorite(ctx, workspace.OID, defaultTab)
			if err != nil {
				log.Printf("warning: failed to restore tab %s: %v", defaultTab.Name, err)
				continue
			}
			
			log.Printf("successfully restored tab %s with layout", restoredTabId)
		}

		// 刷新工作区信息
		workspace, err = GetWorkspace(ctx, workspace.OID)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh workspace: %w", err)
		}

		// 设置第一个标签页为活动标签页
		if len(workspace.PinnedTabIds) > 0 {
			SetActiveTab(ctx, workspace.OID, workspace.PinnedTabIds[0])
		} else if len(workspace.TabIds) > 0 {
			SetActiveTab(ctx, workspace.OID, workspace.TabIds[0])
		}
	}

	// 应用工作区级别的小组件配置
	if len(favorite.WidgetConfigs) > 0 {
		for widgetKey, widgetConfig := range favorite.WidgetConfigs {
			// 转换为wconfig.WidgetConfigType
			configType := wconfig.WidgetConfigType{
				DisplayOrder:  widgetConfig.DisplayOrder,
				DisplayHidden: widgetConfig.DisplayHidden,
				Icon:          widgetConfig.Icon,
				Color:         widgetConfig.Color,
				Label:         widgetConfig.Label,
				Description:   widgetConfig.Description,
				Magnified:     widgetConfig.Magnified,
				BlockDef:      widgetConfig.BlockDef,
			}
			
			err = wconfig.SetWorkspaceWidgetConfig(workspace.OID, widgetKey, configType)
			if err != nil {
				log.Printf("warning: failed to set widget config %s: %v", widgetKey, err)
			}
		}
	}

	// 更新使用次数
	err = incrementFavoriteUsage(favoriteId)
	if err != nil {
		log.Printf("warning: failed to increment favorite usage: %v", err)
	}

	log.Printf("created workspace from favorite: %s -> %s", favorite.Name, workspace.OID)
	return workspace, nil
}

// DeleteWorkspaceFavorite 删除工作区收藏配置
func DeleteWorkspaceFavorite(favoriteId string) error {
	favoritesData, cerrs := wconfig.ReadWaveHomeConfigFile(WorkspaceFavoritesConfigFile)
	if len(cerrs) > 0 {
		return fmt.Errorf("error reading favorites config: %v", cerrs[0])
	}

	if favoritesData == nil {
		return fmt.Errorf("favorite not found: %s", favoriteId)
	}

	delete(favoritesData, favoriteId)
	
	err := wconfig.WriteWaveHomeConfigFile(WorkspaceFavoritesConfigFile, favoritesData)
	if err != nil {
		return fmt.Errorf("failed to save favorites config: %w", err)
	}

	log.Printf("deleted workspace favorite: %s", favoriteId)
	return nil
}

// UpdateWorkspaceFavorite 更新工作区收藏配置
func UpdateWorkspaceFavorite(favoriteId string, name string, description string, tags []string) error {
	favorite, err := GetWorkspaceFavorite(favoriteId)
	if err != nil {
		return err
	}

	if name != "" {
		favorite.Name = name
	}
	favorite.Description = description
	favorite.Tags = tags
	favorite.UpdatedAt = time.Now()

	err = saveFavoriteToConfig(favorite)
	if err != nil {
		return fmt.Errorf("failed to update favorite: %w", err)
	}

	log.Printf("updated workspace favorite: %s", favoriteId)
	return nil
}

// 内部辅助函数
func saveFavoriteToConfig(favorite *waveobj.WorkspaceFavorite) error {
	favoritesData, cerrs := wconfig.ReadWaveHomeConfigFile(WorkspaceFavoritesConfigFile)
	if len(cerrs) > 0 {
		// 如果文件不存在，创建新的配置
		for _, err := range cerrs {
			if err.Err != "" && (contains(err.Err, "no such file") || contains(err.Err, "cannot find")) {
				favoritesData = make(waveobj.MetaMapType)
				break
			}
		}
		if favoritesData == nil {
			return fmt.Errorf("error reading favorites config: %v", cerrs[0])
		}
	}

	if favoritesData == nil {
		favoritesData = make(waveobj.MetaMapType)
	}

	// 将收藏配置转换为MetaMapType
	var favoriteMap waveobj.MetaMapType
	err := utilfn.ReUnmarshal(&favoriteMap, favorite)
	if err != nil {
		return fmt.Errorf("failed to marshal favorite: %w", err)
	}

	favoritesData[favorite.FavoriteId] = favoriteMap
	
	err = wconfig.WriteWaveHomeConfigFile(WorkspaceFavoritesConfigFile, favoritesData)
	if err != nil {
		return fmt.Errorf("failed to save favorites config: %w", err)
	}

	return nil
}

func incrementFavoriteUsage(favoriteId string) error {
	favorite, err := GetWorkspaceFavorite(favoriteId)
	if err != nil {
		return err
	}

	favorite.UsageCount++
	favorite.UpdatedAt = time.Now()

	return saveFavoriteToConfig(favorite)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && s[len(s)-len(substr):] == substr) || 
		(len(s) > len(substr) && s[:len(substr)] == substr) || 
		(len(s) > len(substr) && len(substr) > 0 && s[len(s)-len(substr):] == substr) ||
		findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// restoreTabFromFavorite 从收藏配置恢复标签页的完整布局
func restoreTabFromFavorite(ctx context.Context, workspaceId string, defaultTab waveobj.DefaultTabConfig) (string, error) {
	// 1. 创建基础标签页和布局状态
	layoutStateId := uuid.NewString()
	tabId := uuid.NewString()
	
	tab := &waveobj.Tab{
		OID:         tabId,
		Version:     0,
		Name:        defaultTab.Name,
		LayoutState: layoutStateId,
		BlockIds:    []string{},
		Meta:        defaultTab.Meta,
	}
	
	// 2. 恢复布局状态
	var layoutState *waveobj.LayoutState
	if defaultTab.LayoutState != nil {
		layoutState = &waveobj.LayoutState{
			OID:                   layoutStateId,
			Version:               0,
			RootNode:              defaultTab.LayoutState.RootNode,
			MagnifiedNodeId:       defaultTab.LayoutState.MagnifiedNodeId,
			FocusedNodeId:         defaultTab.LayoutState.FocusedNodeId,
			LeafOrder:             defaultTab.LayoutState.LeafOrder,
			PendingBackendActions: nil,
			Meta:                  defaultTab.LayoutState.Meta,
		}
	} else {
		// 创建空的布局状态
		layoutState = &waveobj.LayoutState{
			OID:     layoutStateId,
			Version: 0,
		}
	}
	
	// 3. 恢复块配置，创建旧ID到新ID的映射
	oldToNewBlockIds := make(map[string]string)
	var newBlocks []*waveobj.Block
	
	for _, savedBlock := range defaultTab.Blocks {
		newBlockId := uuid.NewString()
		oldToNewBlockIds[savedBlock.OriginalOID] = newBlockId
		
		// 创建新的块
		newBlock := &waveobj.Block{
			OID:         newBlockId,
			Version:     0,
			ParentORef:  fmt.Sprintf("tab:%s", tabId),
			RuntimeOpts: savedBlock.RuntimeOpts,
			Stickers:    savedBlock.Stickers,
			Meta:        savedBlock.Meta,
			SubBlockIds: []string{}, // 稍后更新
		}
		newBlocks = append(newBlocks, newBlock)
		tab.BlockIds = append(tab.BlockIds, newBlockId)
	}
	
	// 4. 更新子块ID映射
	for i, savedBlock := range defaultTab.Blocks {
		if len(savedBlock.SubBlockIds) > 0 {
			var newSubBlockIds []string
			for _, oldSubBlockId := range savedBlock.SubBlockIds {
				if newSubBlockId, exists := oldToNewBlockIds[oldSubBlockId]; exists {
					newSubBlockIds = append(newSubBlockIds, newSubBlockId)
				}
			}
			newBlocks[i].SubBlockIds = newSubBlockIds
		}
	}
	
	// 5. 更新布局树中的块ID引用
	if layoutState.RootNode != nil {
		layoutState.RootNode = updateLayoutNodeBlockIds(layoutState.RootNode, oldToNewBlockIds)
	}
	
	// 6. 更新聚焦和放大节点ID
	if layoutState.FocusedNodeId != "" {
		if newId, exists := oldToNewBlockIds[layoutState.FocusedNodeId]; exists {
			layoutState.FocusedNodeId = newId
		}
	}
	if layoutState.MagnifiedNodeId != "" {
		if newId, exists := oldToNewBlockIds[layoutState.MagnifiedNodeId]; exists {
			layoutState.MagnifiedNodeId = newId
		}
	}
	
	// 7. 将标签页添加到工作区
	workspace, err := wstore.DBGet[*waveobj.Workspace](ctx, workspaceId)
	if err != nil {
		return "", fmt.Errorf("workspace not found: %w", err)
	}
	
	if defaultTab.Pinned {
		workspace.PinnedTabIds = append(workspace.PinnedTabIds, tabId)
	} else {
		workspace.TabIds = append(workspace.TabIds, tabId)
	}
	
	// 8. 保存到数据库
	err = wstore.DBInsert(ctx, tab)
	if err != nil {
		return "", fmt.Errorf("failed to insert tab: %w", err)
	}
	
	err = wstore.DBInsert(ctx, layoutState)
	if err != nil {
		return "", fmt.Errorf("failed to insert layout state: %w", err)
	}
	
	for _, block := range newBlocks {
		err = wstore.DBInsert(ctx, block)
		if err != nil {
			log.Printf("warning: failed to insert block %s: %v", block.OID, err)
		}
	}
	
	err = wstore.DBUpdate(ctx, workspace)
	if err != nil {
		return "", fmt.Errorf("failed to update workspace: %w", err)
	}
	
	log.Printf("restored tab %s with %d blocks and layout", tabId, len(newBlocks))
	return tabId, nil
}

// updateLayoutNodeBlockIds 递归更新布局节点中的块ID引用
func updateLayoutNodeBlockIds(node any, idMapping map[string]string) any {
	if node == nil {
		return nil
	}
	
	// 布局节点通常是map或slice结构，我们需要递归处理
	switch v := node.(type) {
	case map[string]any:
		result := make(map[string]any)
		for key, value := range v {
			if key == "blockid" || key == "blockId" {
				// 更新块ID引用
				if strValue, ok := value.(string); ok {
					if newId, exists := idMapping[strValue]; exists {
						result[key] = newId
					} else {
						result[key] = value
					}
				} else {
					result[key] = value
				}
			} else {
				// 递归处理其他字段
				result[key] = updateLayoutNodeBlockIds(value, idMapping)
			}
		}
		return result
		
	case []any:
		var result []any
		for _, item := range v {
			result = append(result, updateLayoutNodeBlockIds(item, idMapping))
		}
		return result
		
	default:
		// 基本类型直接返回
		return node
	}
}