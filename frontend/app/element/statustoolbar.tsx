// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import * as React from "react";
import { ServerStatusLight } from "./serverstatuslight";
import { MCPClient } from "./mcpclient";
import { MCPServerControl } from "./mcpservercontrol";
import "./statustoolbar.scss";

interface StatusToolbarProps {
    className?: string;
}

const StatusToolbar: React.FC<StatusToolbarProps> = ({ className }) => {
    return (
        <div className={`status-toolbar ${className || ""}`}>
            <div className="status-toolbar-left">
                <span className="status-info">Wave Terminal</span>
            </div>
            
            <div className="status-toolbar-center">
                {/* 中央区域，未来可以添加其他组件 */}
            </div>
            
            <div className="status-toolbar-right">
                <MCPServerControl className="toolbar-mcp-server-control" />
                <MCPClient className="toolbar-mcp-client" />
                <ServerStatusLight size="small" className="toolbar-server-status" />
            </div>
        </div>
    );
};

export { StatusToolbar };