// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { Terminal } from "@xterm/xterm";
import * as keyutil from "@/util/keyutil";

/**
 * 增强的快捷键绑定处理器
 * 支持 macOS 风格的快捷键和高级编辑功能
 */
export class EnhancedKeybindingsHandler {
    private terminal: Terminal;
    private sendDataToShell: (data: string) => void;

    constructor(terminal: Terminal, sendDataHandler: (data: string) => void) {
        this.terminal = terminal;
        this.sendDataToShell = sendDataHandler;
    }

    /**
     * 处理增强的快捷键
     * @param event 键盘事件
     * @returns 是否处理了该事件
     */
    public handleKeydown(event: KeyboardEvent): boolean {
        const waveEvent = keyutil.adaptFromReactOrNativeKeyEvent(event);
        if (waveEvent.type !== "keydown") {
            return false;
        }

        // 特殊处理：caps 键输入法切换检测
        if (this.handleInputMethodSwitch(event)) {
            return true; // 阻止进一步处理，让 IME 系统接管
        }

        // macOS 风格快捷键
        if (this.handleMacOSKeybindings(waveEvent, event)) {
            return true;
        }

        // 通用快捷键
        if (this.handleUniversalKeybindings(waveEvent, event)) {
            return true;
        }

        // 编辑增强快捷键
        if (this.handleEditingKeybindings(waveEvent, event)) {
            return true;
        }

        return false;
    }

    private handleMacOSKeybindings(waveEvent: any, event: KeyboardEvent): boolean {
        // Cmd + Backspace - 删除到行首
        if (keyutil.checkKeyPressed(waveEvent, "Cmd:Backspace")) {
            this.deleteToBeginningOfLine();
            event.preventDefault();
            return true;
        }

        // Cmd + Delete - 删除到行尾
        if (keyutil.checkKeyPressed(waveEvent, "Cmd:Delete")) {
            this.deleteToEndOfLine();
            event.preventDefault();
            return true;
        }

        // Option + 左箭头 - 按词向左移动
        if (keyutil.checkKeyPressed(waveEvent, "Option:ArrowLeft")) {
            this.moveCursorWordLeft();
            event.preventDefault();
            return true;
        }

        // Option + 右箭头 - 按词向右移动
        if (keyutil.checkKeyPressed(waveEvent, "Option:ArrowRight")) {
            this.moveCursorWordRight();
            event.preventDefault();
            return true;
        }

        // Option + Backspace - 删除上一个词
        if (keyutil.checkKeyPressed(waveEvent, "Option:Backspace")) {
            this.deleteWordBackward();
            event.preventDefault();
            return true;
        }

        // Option + Delete - 删除下一个词
        if (keyutil.checkKeyPressed(waveEvent, "Option:Delete")) {
            this.deleteWordForward();
            event.preventDefault();
            return true;
        }

        // Cmd + 左箭头 - 移动到行首
        if (keyutil.checkKeyPressed(waveEvent, "Cmd:ArrowLeft")) {
            this.moveCursorToBeginningOfLine();
            event.preventDefault();
            return true;
        }

        // Cmd + 右箭头 - 移动到行尾
        if (keyutil.checkKeyPressed(waveEvent, "Cmd:ArrowRight")) {
            this.moveCursorToEndOfLine();
            event.preventDefault();
            return true;
        }

        // Cmd + A - 全选当前行
        if (keyutil.checkKeyPressed(waveEvent, "Cmd:a")) {
            this.selectCurrentLine();
            event.preventDefault();
            return true;
        }

        return false;
    }

    private handleUniversalKeybindings(waveEvent: any, event: KeyboardEvent): boolean {
        // Ctrl + A - 移动到行首 (Unix style)
        if (keyutil.checkKeyPressed(waveEvent, "Ctrl:a")) {
            this.moveCursorToBeginningOfLine();
            event.preventDefault();
            return true;
        }

        // Ctrl + E - 移动到行尾 (Unix style)
        if (keyutil.checkKeyPressed(waveEvent, "Ctrl:e")) {
            this.moveCursorToEndOfLine();
            event.preventDefault();
            return true;
        }

        // Ctrl + U - 删除到行首 (Unix style)
        if (keyutil.checkKeyPressed(waveEvent, "Ctrl:u")) {
            this.deleteToBeginningOfLine();
            event.preventDefault();
            return true;
        }

        // Ctrl + K - 删除到行尾 (Unix style)
        if (keyutil.checkKeyPressed(waveEvent, "Ctrl:k")) {
            this.deleteToEndOfLine();
            event.preventDefault();
            return true;
        }

        // Ctrl + W - 删除上一个词 (Unix style)
        if (keyutil.checkKeyPressed(waveEvent, "Ctrl:w")) {
            this.deleteWordBackward();
            event.preventDefault();
            return true;
        }

        return false;
    }

    private handleEditingKeybindings(waveEvent: any, event: KeyboardEvent): boolean {
        // Home - 移动到行首
        if (keyutil.checkKeyPressed(waveEvent, "Home")) {
            this.moveCursorToBeginningOfLine();
            event.preventDefault();
            return true;
        }

        // End - 移动到行尾
        if (keyutil.checkKeyPressed(waveEvent, "End")) {
            this.moveCursorToEndOfLine();
            event.preventDefault();
            return true;
        }

        // Ctrl + 左箭头 - 按词向左移动 (Windows/Linux style)
        if (keyutil.checkKeyPressed(waveEvent, "Ctrl:ArrowLeft")) {
            this.moveCursorWordLeft();
            event.preventDefault();
            return true;
        }

        // Ctrl + 右箭头 - 按词向右移动 (Windows/Linux style)
        if (keyutil.checkKeyPressed(waveEvent, "Ctrl:ArrowRight")) {
            this.moveCursorWordRight();
            event.preventDefault();
            return true;
        }

        return false;
    }

    // 光标移动方法
    private moveCursorToBeginningOfLine() {
        this.sendDataToShell('\x01'); // Ctrl+A
    }

    private moveCursorToEndOfLine() {
        this.sendDataToShell('\x05'); // Ctrl+E
    }

    private moveCursorWordLeft() {
        this.sendDataToShell('\x1bb'); // Alt+b (bash word backward)
    }

    private moveCursorWordRight() {
        this.sendDataToShell('\x1bf'); // Alt+f (bash word forward)
    }

    // 删除方法
    private deleteToBeginningOfLine() {
        this.sendDataToShell('\x15'); // Ctrl+U
    }

    private deleteToEndOfLine() {
        this.sendDataToShell('\x0b'); // Ctrl+K
    }

    private deleteWordBackward() {
        this.sendDataToShell('\x17'); // Ctrl+W
    }

    private deleteWordForward() {
        this.sendDataToShell('\x1bd'); // Alt+d (bash delete word forward)
    }

    // 选择方法
    private selectCurrentLine() {
        // 先移动到行首，然后选择到行尾
        this.moveCursorToBeginningOfLine();
        // 小延迟后选择到行尾
        setTimeout(() => {
            // 使用 Shift+End 选择到行尾
            this.sendDataToShell('\x1b[1;2F'); // Shift+End
        }, 10);
    }

    // 工具方法
    public getCurrentCursorPosition(): { x: number; y: number } {
        const buffer = this.terminal.buffer.active;
        return {
            x: buffer.cursorX,
            y: buffer.cursorY
        };
    }

    public getCurrentLine(): string {
        const buffer = this.terminal.buffer.active;
        const line = buffer.getLine(buffer.cursorY);
        return line?.translateToString() || '';
    }

    /**
     * 处理输入法切换相关的按键
     * @param event 键盘事件
     * @returns 是否需要特殊处理
     */
    private handleInputMethodSwitch(event: KeyboardEvent): boolean {
        // 检测 caps 键
        if (event.key === 'CapsLock' || event.keyCode === 20) {
            console.log('🎯 CapsLock detected in keybindings, preparing for IME switch');
            // 不阻止事件，让 IME 处理器处理
            return false;
        }

        // 检测其他输入法切换快捷键
        if ((event.metaKey && event.key === ' ') || (event.ctrlKey && event.shiftKey)) {
            console.log('🎯 Input method switch shortcut detected');
            return false;
        }

        return false;
    }

    /**
     * 检查当前是否在命令行输入状态
     */
    public isInInputMode(): boolean {
        const currentLine = this.getCurrentLine();
        // 简单检测：如果当前行包含常见的提示符字符
        return /[\$#%>]/.test(currentLine) || currentLine.trim().length > 0;
    }
}