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
	Name             string                 `json:"name"`                       // 标签页名称
	Pinned           bool                   `json:"pinned,omitempty"`           // 是否固定
	Meta             MetaMapType            `json:"meta,omitempty"`             // 标签页元数据
	
	// 布局和块信息
	LayoutState      *SavedLayoutState      `json:"layoutstate,omitempty"`     // 保存的布局状态
	Blocks           []*SavedBlock          `json:"blocks,omitempty"`          // 保存的块配置
}

// SavedLayoutState 表示保存的布局状态
type SavedLayoutState struct {
	RootNode          any                 `json:"rootnode,omitempty"`         // 根节点树结构
	MagnifiedNodeId   string              `json:"magnifiednodeid,omitempty"`  // 放大的节点ID
	FocusedNodeId     string              `json:"focusednodeid,omitempty"`    // 聚焦的节点ID
	LeafOrder         *[]LeafOrderEntry   `json:"leaforder,omitempty"`        // 叶子节点顺序
	Meta              MetaMapType         `json:"meta,omitempty"`             // 布局元数据
}

// SavedBlock 表示保存的块配置
type SavedBlock struct {
	OriginalOID     string         `json:"originaloid"`            // 原始块ID（用于布局树引用）
	ParentORef      string         `json:"parentoref,omitempty"`   // 父对象引用
	RuntimeOpts     *RuntimeOpts   `json:"runtimeopts,omitempty"`  // 运行时选项
	Stickers        []*StickerType `json:"stickers,omitempty"`     // 标签贴纸
	Meta            MetaMapType    `json:"meta"`                   // 元数据（核心配置）
	SubBlockIds     []string       `json:"subblockids,omitempty"`  // 子块ID列表（映射到新ID）
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