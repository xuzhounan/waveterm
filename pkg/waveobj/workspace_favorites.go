// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package waveobj

import (
	"time"
)

// WorkspaceFavorite 表示一个收藏的工作区配置模板
type WorkspaceFavorite struct {
	FavoriteId  string      `json:"favoriteid"`  // 收藏配置的唯一ID
	Name        string      `json:"name"`        // 收藏配置的名称
	Description string      `json:"description,omitempty"` // 可选的描述
	Icon        string      `json:"icon"`        // 工作区图标
	Color       string      `json:"color"`       // 工作区颜色
	Tags        []string    `json:"tags,omitempty"` // 可选的标签
	CreatedAt   time.Time   `json:"createdat"`   // 创建时间
	UpdatedAt   time.Time   `json:"updatedat"`   // 更新时间
	UsageCount  int         `json:"usagecount"`  // 使用次数统计
	
	// 工作区的默认配置
	DefaultTabs []DefaultTabConfig `json:"defaulttabs,omitempty"` // 默认标签页配置
	
	// 工作区级别的小组件配置
	WidgetConfigs map[string]WidgetConfig `json:"widgetconfigs,omitempty"`
	
	// 元数据配置
	Meta MetaMapType `json:"meta,omitempty"`
}

// DefaultTabConfig 表示默认标签页的配置
type DefaultTabConfig struct {
	Name     string      `json:"name"`               // 标签页名称
	Pinned   bool        `json:"pinned,omitempty"`   // 是否固定
	BlockDef BlockDef    `json:"blockdef"`           // 块定义
	Meta     MetaMapType `json:"meta,omitempty"`     // 标签页元数据
}

// WidgetConfig 表示小组件配置
type WidgetConfig struct {
	DisplayOrder  float64  `json:"display:order,omitempty"`
	DisplayHidden bool     `json:"display:hidden,omitempty"`
	Icon          string   `json:"icon,omitempty"`
	Color         string   `json:"color,omitempty"`
	Label         string   `json:"label,omitempty"`
	Description   string   `json:"description,omitempty"`
	Magnified     bool     `json:"magnified,omitempty"`
	BlockDef      BlockDef `json:"blockdef"`
}

// WorkspaceFavoriteList 收藏列表类型
type WorkspaceFavoriteList []*WorkspaceFavorite