// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { modalsModel } from "@/app/store/modalmodel";
import { RpcApi } from "@/app/store/wshclientapi";
import { TabRpcClient } from "@/app/store/wshrpcutil";
import { WOS } from "@/store/global";

/**
 * Opens a modal to edit the custom name of a block
 * @param blockId - The ID of the block to edit
 * @param blockType - The type of block (e.g., "term", "web", "preview")
 * @param currentName - The current custom name of the block
 */
export function editBlockCustomName(blockId: string, blockType: string, currentName: string = "") {
    modalsModel.pushModal("BlockNameEditModal", {
        blockId,
        blockType,
        currentName,
        onSave: async (newName: string) => {
            // If empty string, remove the custom name
            const metaUpdate = newName.trim() ? { name: newName.trim() } : { name: null };
            
            try {
                await RpcApi.SetMetaCommand(TabRpcClient, {
                    oref: WOS.makeORef("block", blockId),
                    meta: metaUpdate,
                });
            } catch (error) {
                console.error("Failed to update block name:", error);
                // TODO: Add user notification for error
            }
        }
    });
}