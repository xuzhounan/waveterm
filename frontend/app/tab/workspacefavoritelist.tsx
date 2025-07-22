// Copyright 2025, Command Line
// SPDX-License-Identifier: Apache-2.0

import { IconButton } from "@/element/iconbutton";
import { makeIconClass } from "@/util/util";
import { useState, useCallback } from "react";
import "./workspacefavoritelist.scss";

interface WorkspaceFavorite {
    favoriteid: string;
    name: string;
    description?: string;
    icon: string;
    color: string;
    tags?: string[];
    createdat: string;
    updatedat: string;
    usagecount: number;
}

interface WorkspaceFavoriteListProps {
    favorites: WorkspaceFavorite[];
    onSelect?: (favoriteId: string) => void;
    onDelete?: (favoriteId: string) => void;
    onEdit?: (favoriteId: string) => void;
    loading?: boolean;
}

export const WorkspaceFavoriteList = ({ 
    favorites, 
    onSelect, 
    onDelete, 
    onEdit,
    loading = false 
}: WorkspaceFavoriteListProps) => {
    const [expandedFavorite, setExpandedFavorite] = useState<string | null>(null);

    const toggleExpanded = useCallback((favoriteId: string) => {
        setExpandedFavorite(prev => prev === favoriteId ? null : favoriteId);
    }, []);

    const handleSelect = useCallback((favoriteId: string) => {
        onSelect?.(favoriteId);
    }, [onSelect]);

    const handleDelete = useCallback((favoriteId: string, e: React.MouseEvent) => {
        e.stopPropagation();
        if (confirm("确定要删除这个收藏配置吗？")) {
            onDelete?.(favoriteId);
        }
    }, [onDelete]);

    const handleEdit = useCallback((favoriteId: string, e: React.MouseEvent) => {
        e.stopPropagation();
        onEdit?.(favoriteId);
    }, [onEdit]);

    if (loading) {
        return (
            <div className="workspace-favorite-list loading">
                <div className="loading-spinner">加载中...</div>
            </div>
        );
    }

    if (!favorites || favorites.length === 0) {
        return (
            <div className="workspace-favorite-list empty">
                <div className="empty-state">
                    <i className="fa fa-bookmark-o"></i>
                    <p>暂无收藏的工作区配置</p>
                    <p className="hint">保存当前工作区为收藏，以便快速创建相似的工作环境</p>
                </div>
            </div>
        );
    }

    return (
        <div className="workspace-favorite-list">
            {favorites.map((favorite) => {
                const isExpanded = expandedFavorite === favorite.favoriteid;
                const editIconDecl: IconButtonDecl = {
                    elemtype: "iconbutton",
                    icon: "pencil",
                    title: "编辑收藏",
                    className: "edit-btn",
                    click: (e) => handleEdit(favorite.favoriteid, e),
                };
                const deleteIconDecl: IconButtonDecl = {
                    elemtype: "iconbutton",
                    icon: "trash",
                    title: "删除收藏",
                    className: "delete-btn",
                    click: (e) => handleDelete(favorite.favoriteid, e),
                };

                return (
                    <div 
                        key={favorite.favoriteid} 
                        className={`favorite-item ${isExpanded ? 'expanded' : ''}`}
                        onClick={() => handleSelect(favorite.favoriteid)}
                    >
                        <div className="favorite-header">
                            <div className="favorite-icon-name">
                                <i 
                                    className={makeIconClass(favorite.icon, false)} 
                                    style={{ color: favorite.color }}
                                />
                                <div className="favorite-info">
                                    <div className="favorite-name">{favorite.name}</div>
                                    {favorite.description && (
                                        <div className="favorite-description">{favorite.description}</div>
                                    )}
                                </div>
                            </div>
                            
                            <div className="favorite-actions">
                                <div className="usage-count">
                                    使用 {favorite.usagecount} 次
                                </div>
                                <IconButton decl={editIconDecl} />
                                <IconButton decl={deleteIconDecl} />
                                <button 
                                    className="expand-btn"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        toggleExpanded(favorite.favoriteid);
                                    }}
                                >
                                    <i className={`fa fa-chevron-${isExpanded ? 'up' : 'down'}`} />
                                </button>
                            </div>
                        </div>

                        {isExpanded && (
                            <div className="favorite-details">
                                {favorite.tags && favorite.tags.length > 0 && (
                                    <div className="favorite-tags">
                                        <span className="tags-label">标签：</span>
                                        {favorite.tags.map((tag, index) => (
                                            <span key={index} className="tag">
                                                {tag}
                                            </span>
                                        ))}
                                    </div>
                                )}
                                
                                <div className="favorite-layout-info">
                                    <div className="layout-stats">
                                        <span className="stat-item">
                                            <i className="fa fa-window-maximize"></i>
                                            {favorite.defaulttabs?.length || 0} 个标签页
                                        </span>
                                        <span className="stat-item">
                                            <i className="fa fa-th"></i>
                                            {favorite.defaulttabs?.reduce((total, tab) => 
                                                total + (tab.blocks?.length || 0), 0) || 0} 个窗口
                                        </span>
                                        {favorite.widgetconfigs && Object.keys(favorite.widgetconfigs).length > 0 && (
                                            <span className="stat-item">
                                                <i className="fa fa-puzzle-piece"></i>
                                                {Object.keys(favorite.widgetconfigs).length} 个小组件
                                            </span>
                                        )}
                                    </div>
                                    {favorite.defaulttabs && favorite.defaulttabs.length > 0 && (
                                        <div className="tabs-preview">
                                            <span className="preview-label">标签页预览：</span>
                                            {favorite.defaulttabs.slice(0, 3).map((tab, index) => (
                                                <span key={index} className="tab-preview">
                                                    {tab.pinned && <i className="fa fa-thumbtack"></i>}
                                                    {tab.name}
                                                </span>
                                            ))}
                                            {favorite.defaulttabs.length > 3 && (
                                                <span className="more-tabs">+{favorite.defaulttabs.length - 3}</span>
                                            )}
                                        </div>
                                    )}
                                </div>
                                
                                <div className="favorite-meta">
                                    <div className="meta-item">
                                        <span className="meta-label">创建时间：</span>
                                        <span className="meta-value">
                                            {new Date(favorite.createdat).toLocaleDateString('zh-CN')}
                                        </span>
                                    </div>
                                    <div className="meta-item">
                                        <span className="meta-label">更新时间：</span>
                                        <span className="meta-value">
                                            {new Date(favorite.updatedat).toLocaleDateString('zh-CN')}
                                        </span>
                                    </div>
                                </div>
                                
                                <div className="select-hint">
                                    点击创建基于此配置的新工作区
                                </div>
                            </div>
                        )}
                    </div>
                );
            })}
        </div>
    );
};