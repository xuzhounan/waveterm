// Copyright 2025, Command Line
// SPDX-License-Identifier: Apache-2.0

import { Button } from "@/element/button";
import { Input } from "@/element/input";
import { useState, useRef, useEffect, useCallback } from "react";
import "./workspacefavoriteeditor.scss";

interface WorkspaceFavoriteEditorProps {
    workspaceId: string;
    workspaceName: string;
    currentFavorite?: any; // 当前已存在的收藏信息
    onSave?: (favoriteName: string, description: string, tags: string[]) => void;
    onCancel?: () => void;
}

export const WorkspaceFavoriteEditor = ({ 
    workspaceId, 
    workspaceName, 
    currentFavorite,
    onSave, 
    onCancel 
}: WorkspaceFavoriteEditorProps) => {
    const [favoriteName, setFavoriteName] = useState(currentFavorite?.name || workspaceName || "");
    const [description, setDescription] = useState(currentFavorite?.description || "");
    const [tagsText, setTagsText] = useState(currentFavorite?.tags ? currentFavorite.tags.join(', ') : "");
    const nameInputRef = useRef<HTMLInputElement>(null);

    useEffect(() => {
        if (nameInputRef.current) {
            nameInputRef.current.focus();
            nameInputRef.current.select();
        }
    }, []);

    const handleSave = useCallback(() => {
        if (!favoriteName.trim()) {
            return;
        }
        
        const tags = tagsText
            .split(',')
            .map(tag => tag.trim())
            .filter(tag => tag.length > 0);
            
        onSave?.(favoriteName.trim(), description.trim(), tags);
    }, [favoriteName, description, tagsText, onSave]);

    const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleSave();
        } else if (e.key === 'Escape') {
            e.preventDefault();
            onCancel?.();
        }
    }, [handleSave, onCancel]);

    return (
        <div className="workspace-favorite-editor">
            <div className="editor-header">
                <h3>{currentFavorite ? '更新工作区收藏' : '保存工作区为收藏'}</h3>
            </div>
            
            <div className="editor-form">
                <div className="form-group">
                    <label>收藏名称 *</label>
                    <Input
                        ref={nameInputRef}
                        value={favoriteName}
                        onChange={setFavoriteName}
                        onKeyDown={handleKeyDown}
                        placeholder="输入收藏名称"
                        maxLength={50}
                        required
                    />
                </div>

                <div className="form-group">
                    <label>描述</label>
                    <Input
                        value={description}
                        onChange={setDescription}
                        onKeyDown={handleKeyDown}
                        placeholder="可选的描述信息"
                        maxLength={200}
                    />
                </div>

                <div className="form-group">
                    <label>标签</label>
                    <Input
                        value={tagsText}
                        onChange={setTagsText}
                        onKeyDown={handleKeyDown}
                        placeholder="用逗号分隔的标签，如：开发,前端,项目"
                        maxLength={100}
                    />
                </div>
            </div>

            <div className="editor-actions">
                <Button className="secondary" onClick={onCancel}>
                    取消
                </Button>
                <Button 
                    className="primary" 
                    onClick={handleSave}
                    disabled={!favoriteName.trim()}
                >
                    {currentFavorite ? '更新收藏' : '保存收藏'}
                </Button>
            </div>
        </div>
    );
};