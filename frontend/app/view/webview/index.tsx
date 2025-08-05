// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { BlockNodeModel } from "@/app/block/blocktypes";
import { WebViewModel } from "./webview";
import { ImprovedWebViewModel } from "./webview-improved";

// Feature flag for testing improved WebView implementation
// Set to true to use WebContentsView instead of webview tag
const USE_IMPROVED_WEBVIEW = false; // Start with false, can be toggled for testing

/**
 * Factory function to create the appropriate WebView model
 * based on the feature flag.
 */
export function createWebViewModel(blockId: string, nodeModel: BlockNodeModel): ViewModel {
    if (USE_IMPROVED_WEBVIEW) {
        console.log("[WebView] Using improved WebContentsView implementation for block:", blockId);
        return new ImprovedWebViewModel(blockId, nodeModel);
    } else {
        console.log("[WebView] Using legacy webview tag implementation for block:", blockId);
        return new WebViewModel(blockId, nodeModel);
    }
}

// Re-export for backwards compatibility
export { WebViewModel } from "./webview";
export { ImprovedWebViewModel } from "./webview-improved";