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

// SaveWorkspaceAsFavorite 将当前工作区保存为收藏配置
func SaveWorkspaceAsFavorite(ctx context.Context, workspaceId string, favoriteName string, description string, tags []string) (*waveobj.WorkspaceFavorite, error) {
	if favoriteName == "" {
		return nil, fmt.Errorf("favorite name cannot be empty")
	}

	// 获取工作区信息
	workspace, err := GetWorkspace(ctx, workspaceId)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %w", err)
	}

	// 获取工作区的标签页信息
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

		// 创建默认块定义（简化版本，主要保存视图类型）
		var blockDef waveobj.BlockDef
		if len(tab.BlockIds) > 0 {
			// 获取第一个块的配置作为模板
			block, err := wstore.DBGet[*waveobj.Block](ctx, tab.BlockIds[0])
			if err == nil && block != nil {
				blockDef = waveobj.BlockDef{
					Meta: block.Meta,
				}
			}
		}

		defaultTab := waveobj.DefaultTabConfig{
			Name:     tab.Name,
			Pinned:   isPinned,
			BlockDef: blockDef,
			Meta:     tab.Meta,
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

	// 创建收藏配置
	now := time.Now()
	favorite := &waveobj.WorkspaceFavorite{
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

	// 保存到配置文件
	err = saveFavoriteToConfig(favorite)
	if err != nil {
		return nil, fmt.Errorf("failed to save favorite: %w", err)
	}

	log.Printf("saved workspace %s as favorite: %s", workspaceId, favoriteName)
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

	// 应用收藏配置中的默认标签页
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
			tabId, err := CreateTab(ctx, workspace.OID, defaultTab.Name, false, defaultTab.Pinned, false)
			if err != nil {
				log.Printf("warning: failed to create tab %s: %v", defaultTab.Name, err)
				continue
			}

			// 应用标签页的元数据配置
			if len(defaultTab.Meta) > 0 {
				tabORef := waveobj.ORef{OType: "tab", OID: tabId}
				wstore.UpdateObjectMeta(ctx, tabORef, defaultTab.Meta, true)
			}

			// 如果有块定义，创建对应的块
			if len(defaultTab.BlockDef.Meta) > 0 {
				// 这里可以根据BlockDef创建具体的块
				// 目前简化处理，主要应用元数据
				tab, err := wstore.DBGet[*waveobj.Tab](ctx, tabId)
				if err == nil && tab != nil {
					tab.Meta = waveobj.MergeMeta(tab.Meta, defaultTab.BlockDef.Meta, true)
					wstore.DBUpdate(ctx, tab)
				}
			}
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