// Copyright 2025, Command Line
// SPDX-License-Identifier: Apache-2.0

import {
    ExpandableMenu,
    ExpandableMenuItem,
    ExpandableMenuItemGroup,
    ExpandableMenuItemGroupTitle,
    ExpandableMenuItemLeftElement,
    ExpandableMenuItemRightElement,
} from "@/element/expandablemenu";
import { Popover, PopoverButton, PopoverContent } from "@/element/popover";
import { fireAndForget, makeIconClass, useAtomValueSafe } from "@/util/util";
import clsx from "clsx";
import { atom, PrimitiveAtom, useAtom, useAtomValue, useSetAtom } from "jotai";
import { splitAtom } from "jotai/utils";
import { OverlayScrollbarsComponent } from "overlayscrollbars-react";
import { CSSProperties, forwardRef, useCallback, useEffect } from "react";
import WorkspaceSVG from "../asset/workspace.svg";
import { IconButton } from "../element/iconbutton";
import { atoms, getApi } from "../store/global";
import { WorkspaceService } from "../store/services";
import { getObjectValue, makeORef } from "../store/wos";
import { waveEventSubscribe } from "../store/wps";
import { WorkspaceEditor } from "./workspaceeditor";
import { WorkspaceFavoriteEditor } from "./workspacefavoriteeditor";
import { WorkspaceFavoriteList } from "./workspacefavoritelist";
import "./workspaceswitcher.scss";

type WorkspaceListEntry = {
    windowId: string;
    workspace: Workspace;
};

type WorkspaceList = WorkspaceListEntry[];
const workspaceMapAtom = atom<WorkspaceList>([]);
const workspaceSplitAtom = splitAtom(workspaceMapAtom);
const editingWorkspaceAtom = atom<string>();
const showFavoritesAtom = atom<boolean>(false);
const showFavoriteEditorAtom = atom<boolean>(false);
const favoritesAtom = atom<any[]>([]);
const favoritesLoadingAtom = atom<boolean>(false);
const WorkspaceSwitcher = forwardRef<HTMLDivElement>((_, ref) => {
    const setWorkspaceList = useSetAtom(workspaceMapAtom);
    const activeWorkspace = useAtomValueSafe(atoms.workspace);
    const workspaceList = useAtomValue(workspaceSplitAtom);
    const setEditingWorkspace = useSetAtom(editingWorkspaceAtom);
    const [showFavorites, setShowFavorites] = useAtom(showFavoritesAtom);
    const [showFavoriteEditor, setShowFavoriteEditor] = useAtom(showFavoriteEditorAtom);
    const [favorites, setFavorites] = useAtom(favoritesAtom);
    const [favoritesLoading, setFavoritesLoading] = useAtom(favoritesLoadingAtom);

    const updateWorkspaceList = useCallback(async () => {
        const workspaceList = await WorkspaceService.ListWorkspaces();
        if (!workspaceList) {
            return;
        }
        const newList: WorkspaceList = [];
        for (const entry of workspaceList) {
            // This just ensures that the atom exists for easier setting of the object
            getObjectValue(makeORef("workspace", entry.workspaceid));
            newList.push({
                windowId: entry.windowid,
                workspace: await WorkspaceService.GetWorkspace(entry.workspaceid),
            });
        }
        setWorkspaceList(newList);
    }, []);

    useEffect(
        () =>
            waveEventSubscribe({
                eventType: "workspace:update",
                handler: () => fireAndForget(updateWorkspaceList),
            }),
        []
    );

    useEffect(() => {
        fireAndForget(updateWorkspaceList);
    }, []);

    const onDeleteWorkspace = useCallback((workspaceId: string) => {
        getApi().deleteWorkspace(workspaceId);
    }, []);

    const isActiveWorkspaceSaved = !!(activeWorkspace.name && activeWorkspace.icon);

    const workspaceIcon = isActiveWorkspaceSaved ? (
        <i className={makeIconClass(activeWorkspace.icon, false)} style={{ color: activeWorkspace.color }}></i>
    ) : (
        <WorkspaceSVG />
    );

    const saveWorkspace = () => {
        fireAndForget(async () => {
            await WorkspaceService.UpdateWorkspace(activeWorkspace.oid, "", "", "", true);
            await updateWorkspaceList();
            setEditingWorkspace(activeWorkspace.oid);
        });
    };

    const loadFavorites = useCallback(async () => {
        setFavoritesLoading(true);
        try {
            const favoriteList = await WorkspaceService.ListWorkspaceFavorites();
            setFavorites(favoriteList || []);
        } catch (error) {
            console.error("Failed to load workspace favorites:", error);
            setFavorites([]);
        } finally {
            setFavoritesLoading(false);
        }
    }, [setFavorites, setFavoritesLoading]);

    const handleSaveAsFavorite = useCallback(async (favoriteName: string, description: string, tags: string[]) => {
        try {
            await WorkspaceService.SaveWorkspaceAsFavorite(activeWorkspace.oid, favoriteName, description, tags);
            setShowFavoriteEditor(false);
            await loadFavorites();
            // TODO: Show success notification
        } catch (error) {
            console.error("Failed to save workspace as favorite:", error);
            // TODO: Show error notification
        }
    }, [activeWorkspace.oid, setShowFavoriteEditor, loadFavorites]);

    const handleSelectFavorite = useCallback(async (favoriteId: string) => {
        try {
            const newWorkspaceId = await WorkspaceService.CreateWorkspaceFromFavorite(favoriteId);
            getApi().switchWorkspace(newWorkspaceId);
            // Close the popover
            document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
        } catch (error) {
            console.error("Failed to create workspace from favorite:", error);
            // TODO: Show error notification
        }
    }, []);

    const handleDeleteFavorite = useCallback(async (favoriteId: string) => {
        try {
            await WorkspaceService.DeleteWorkspaceFavorite(favoriteId);
            await loadFavorites();
            // TODO: Show success notification
        } catch (error) {
            console.error("Failed to delete workspace favorite:", error);
            // TODO: Show error notification
        }
    }, [loadFavorites]);

    const handleShowFavorites = useCallback(() => {
        setShowFavorites(true);
        setShowFavoriteEditor(false);
        setEditingWorkspace(null);
        fireAndForget(loadFavorites);
    }, [setShowFavorites, setShowFavoriteEditor, setEditingWorkspace, loadFavorites]);

    const handleBackToWorkspaces = useCallback(() => {
        setShowFavorites(false);
        setShowFavoriteEditor(false);
    }, [setShowFavorites, setShowFavoriteEditor]);

    return (
        <Popover
            className="workspace-switcher-popover"
            placement="bottom-start"
            onDismiss={() => setEditingWorkspace(null)}
            ref={ref}
        >
            <PopoverButton
                className="workspace-switcher-button grey"
                as="div"
                onClick={() => {
                    fireAndForget(updateWorkspaceList);
                }}
            >
                <span className="workspace-icon">{workspaceIcon}</span>
            </PopoverButton>
            <PopoverContent className={`workspace-switcher-content ${showFavoriteEditor ? 'favorite-editor-mode' : showFavorites ? 'favorites-list-mode' : ''}`}>
                {showFavoriteEditor ? (
                    <WorkspaceFavoriteEditor
                        workspaceId={activeWorkspace.oid}
                        workspaceName={activeWorkspace.name}
                        onSave={handleSaveAsFavorite}
                        onCancel={() => setShowFavoriteEditor(false)}
                    />
                ) : showFavorites ? (
                    <>
                        <div className="favorites-header">
                            <button className="back-button" onClick={handleBackToWorkspaces}>
                                <i className="fa fa-arrow-left"></i>
                                返回
                            </button>
                            <div className="title">工作区收藏</div>
                        </div>
                        <OverlayScrollbarsComponent className={"scrollable favorites-scrollable"} options={{ scrollbars: { autoHide: "leave" } }}>
                            <WorkspaceFavoriteList
                                favorites={favorites}
                                loading={favoritesLoading}
                                onSelect={handleSelectFavorite}
                                onDelete={handleDeleteFavorite}
                            />
                        </OverlayScrollbarsComponent>
                    </>
                ) : (
                    <>
                        <div className="title">{isActiveWorkspaceSaved ? "切换工作区" : "打开工作区"}</div>
                        <OverlayScrollbarsComponent className={"scrollable"} options={{ scrollbars: { autoHide: "leave" } }}>
                            <ExpandableMenu noIndent singleOpen>
                                {workspaceList.map((entry, i) => (
                                    <WorkspaceSwitcherItem key={i} entryAtom={entry} onDeleteWorkspace={onDeleteWorkspace} />
                                ))}
                            </ExpandableMenu>
                        </OverlayScrollbarsComponent>

                        <div className="actions">
                            {isActiveWorkspaceSaved ? (
                                <>
                                    <ExpandableMenuItem onClick={() => getApi().createWorkspace()}>
                                        <ExpandableMenuItemLeftElement>
                                            <i className="fa-sharp fa-solid fa-plus"></i>
                                        </ExpandableMenuItemLeftElement>
                                        <div className="content">创建新工作区</div>
                                    </ExpandableMenuItem>
                                    <ExpandableMenuItem onClick={handleShowFavorites}>
                                        <ExpandableMenuItemLeftElement>
                                            <i className="fa-sharp fa-solid fa-bookmark"></i>
                                        </ExpandableMenuItemLeftElement>
                                        <div className="content">从收藏创建</div>
                                    </ExpandableMenuItem>
                                    <ExpandableMenuItem onClick={() => setShowFavoriteEditor(true)}>
                                        <ExpandableMenuItemLeftElement>
                                            <i className="fa-sharp fa-solid fa-heart"></i>
                                        </ExpandableMenuItemLeftElement>
                                        <div className="content">收藏当前工作区</div>
                                    </ExpandableMenuItem>
                                </>
                            ) : (
                                <ExpandableMenuItem onClick={() => saveWorkspace()}>
                                    <ExpandableMenuItemLeftElement>
                                        <i className="fa-sharp fa-solid fa-floppy-disk"></i>
                                    </ExpandableMenuItemLeftElement>
                                    <div className="content">保存工作区</div>
                                </ExpandableMenuItem>
                            )}
                        </div>
                    </>
                )}
            </PopoverContent>
        </Popover>
    );
});

const WorkspaceSwitcherItem = ({
    entryAtom,
    onDeleteWorkspace,
}: {
    entryAtom: PrimitiveAtom<WorkspaceListEntry>;
    onDeleteWorkspace: (workspaceId: string) => void;
}) => {
    const activeWorkspace = useAtomValueSafe(atoms.workspace);
    const [workspaceEntry, setWorkspaceEntry] = useAtom(entryAtom);
    const [editingWorkspace, setEditingWorkspace] = useAtom(editingWorkspaceAtom);

    const workspace = workspaceEntry.workspace;
    const isCurrentWorkspace = activeWorkspace.oid === workspace.oid;

    const setWorkspace = useCallback((newWorkspace: Workspace) => {
        setWorkspaceEntry({ ...workspaceEntry, workspace: newWorkspace });
        if (newWorkspace.name != "") {
            fireAndForget(() =>
                WorkspaceService.UpdateWorkspace(
                    workspace.oid,
                    newWorkspace.name,
                    newWorkspace.icon,
                    newWorkspace.color,
                    false
                )
            );
        }
    }, []);

    const isActive = !!workspaceEntry.windowId;
    const editIconDecl: IconButtonDecl = {
        elemtype: "iconbutton",
        className: "edit",
        icon: "pencil",
        title: "Edit workspace",
        click: (e) => {
            e.stopPropagation();
            if (editingWorkspace === workspace.oid) {
                setEditingWorkspace(null);
            } else {
                setEditingWorkspace(workspace.oid);
            }
        },
    };
    const windowIconDecl: IconButtonDecl = {
        elemtype: "iconbutton",
        className: "window",
        noAction: true,
        icon: isCurrentWorkspace ? "check" : "window",
        title: isCurrentWorkspace ? "This is your current workspace" : "This workspace is open",
    };

    const isEditing = editingWorkspace === workspace.oid;

    return (
        <ExpandableMenuItemGroup
            key={workspace.oid}
            isOpen={isEditing}
            className={clsx({ "is-current": isCurrentWorkspace })}
        >
            <ExpandableMenuItemGroupTitle
                onClick={() => {
                    getApi().switchWorkspace(workspace.oid);
                    // Create a fake escape key event to close the popover
                    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
                }}
            >
                <div
                    className="menu-group-title-wrapper"
                    style={
                        {
                            "--workspace-color": workspace.color,
                        } as CSSProperties
                    }
                >
                    <ExpandableMenuItemLeftElement>
                        <i
                            className={clsx("left-icon", makeIconClass(workspace.icon, true))}
                            style={{ color: workspace.color }}
                        />
                    </ExpandableMenuItemLeftElement>
                    <div className="label">{workspace.name}</div>
                    <ExpandableMenuItemRightElement>
                        <div className="icons">
                            <IconButton decl={editIconDecl} />
                            {isActive && <IconButton decl={windowIconDecl} />}
                        </div>
                    </ExpandableMenuItemRightElement>
                </div>
            </ExpandableMenuItemGroupTitle>
            <ExpandableMenuItem>
                <WorkspaceEditor
                    title={workspace.name}
                    icon={workspace.icon}
                    color={workspace.color}
                    focusInput={isEditing}
                    onTitleChange={(title) => setWorkspace({ ...workspace, name: title })}
                    onColorChange={(color) => setWorkspace({ ...workspace, color })}
                    onIconChange={(icon) => setWorkspace({ ...workspace, icon })}
                    onDeleteWorkspace={() => onDeleteWorkspace(workspace.oid)}
                />
            </ExpandableMenuItem>
        </ExpandableMenuItemGroup>
    );
};

export { WorkspaceSwitcher };
