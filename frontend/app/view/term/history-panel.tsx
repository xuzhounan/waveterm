// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import clsx from "clsx";
import * as React from "react";

interface HistoryPanelProps {
    history: string[];
    isVisible: boolean;
    onClose: () => void;
    onSelectCommand: (command: string) => void;
    onClearHistory: () => void;
}

const HistoryPanel: React.FC<HistoryPanelProps> = ({
    history,
    isVisible,
    onClose,
    onSelectCommand,
    onClearHistory,
}) => {
    const panelRef = React.useRef<HTMLDivElement>(null);

    // 点击面板外部关闭
    React.useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (panelRef.current && !panelRef.current.contains(event.target as Node)) {
                onClose();
            }
        };

        if (isVisible) {
            document.addEventListener("mousedown", handleClickOutside);
        }

        return () => {
            document.removeEventListener("mousedown", handleClickOutside);
        };
    }, [isVisible, onClose]);

    // ESC键关闭
    React.useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === "Escape") {
                onClose();
            }
        };

        if (isVisible) {
            document.addEventListener("keydown", handleKeyDown);
        }

        return () => {
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [isVisible, onClose]);

    if (!isVisible) {
        return null;
    }

    return (
        <div className="history-panel-overlay">
            <div ref={panelRef} className="history-panel">
                <header className="history-panel-header">
                    <h3>命令历史</h3>
                    <div className="history-panel-actions">
                        <button
                            className="history-panel-button history-panel-clear"
                            onClick={onClearHistory}
                            title="清空历史"
                        >
                            <i className="fa fa-trash" />
                        </button>
                        <button
                            className="history-panel-button history-panel-close"
                            onClick={onClose}
                            title="关闭"
                        >
                            <i className="fa fa-times" />
                        </button>
                    </div>
                </header>
                <div className="history-panel-content">
                    {history.length === 0 ? (
                        <div className="history-panel-empty">
                            <p>暂无命令历史</p>
                            <p className="history-panel-tip">在终端中执行命令后，命令会自动记录在这里</p>
                        </div>
                    ) : (
                        <ul className="history-panel-list">
                            {history.map((command, index) => (
                                <li
                                    key={index}
                                    className="history-panel-item"
                                    onClick={() => onSelectCommand(command)}
                                    title={`点击复制: ${command}`}
                                >
                                    <span className="history-panel-index">{history.length - index}</span>
                                    <span className="history-panel-command">{command}</span>
                                    <i className="fa fa-copy history-panel-copy-icon" />
                                </li>
                            ))}
                        </ul>
                    )}
                </div>
                <footer className="history-panel-footer">
                    <small>显示最近 {Math.min(history.length, 20)} 条命令 • 点击命令可复制到剪贴板</small>
                </footer>
            </div>
        </div>
    );
};

export { HistoryPanel };