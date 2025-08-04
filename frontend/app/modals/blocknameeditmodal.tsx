// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { Modal } from "@/app/modals/modal";
import { modalsModel } from "@/store/modalmodel";
import * as keyutil from "@/util/keyutil";
import { useCallback, useState } from "react";
import "./blocknameeditmodal.scss";

interface BlockNameEditModalProps {
    blockId: string;
    blockType: string;
    currentName: string;
    onSave: (newName: string) => void;
}

const BlockNameEditModal = ({ blockId, blockType, currentName, onSave }: BlockNameEditModalProps) => {
    const [blockName, setBlockName] = useState(currentName);

    const handleSave = useCallback(() => {
        onSave(blockName);
        modalsModel.popModal();
    }, [blockName, onSave]);

    const handleCancel = useCallback(() => {
        modalsModel.popModal();
    }, []);

    const handleKeyDown = useCallback(
        (waveEvent: WaveKeyboardEvent): boolean => {
            if (keyutil.checkKeyPressed(waveEvent, "Escape")) {
                handleCancel();
                return true;
            }
            if (keyutil.checkKeyPressed(waveEvent, "Enter")) {
                handleSave();
                return true;
            }
            return false;
        },
        [handleCancel, handleSave]
    );

    const getTitle = () => {
        switch (blockType) {
            case "web":
                return "Edit Web View Name";
            case "term":
                return "Edit Terminal Name";
            case "preview":
                return "Edit Preview Name";
            case "sysinfo":
                return "Edit System Info Name";
            case "waveai":
                return "Edit Wave AI Name";
            default:
                return "Edit Block Name";
        }
    };

    return (
        <Modal
            onOk={handleSave}
            onCancel={handleCancel}
            onClose={handleCancel}
            okLabel="Save"
            cancelLabel="Cancel"
        >
            <div className="block-name-edit-modal">
                <div className="modal-header">{getTitle()}</div>
                <div className="modal-body">
                    <div className="input-group">
                        <label htmlFor="block-name">Name:</label>
                        <input
                            id="block-name"
                            type="text"
                            value={blockName}
                            onChange={(e) => setBlockName(e.target.value)}
                            onKeyDown={(e) => keyutil.keydownWrapper(handleKeyDown)(e)}
                            placeholder="Enter name..."
                            autoFocus
                            maxLength={100}
                        />
                    </div>
                    <div className="hint">Leave empty to use default name</div>
                </div>
            </div>
        </Modal>
    );
};

BlockNameEditModal.displayName = "BlockNameEditModal";

export { BlockNameEditModal };