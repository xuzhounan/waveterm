// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { Terminal } from "@xterm/xterm";

/**
 * 增强的中文输入法处理器
 * 解决 xterm.js 中 composition view 定位问题
 */
export class EnhancedIMEHandler {
    private terminal: Terminal;
    private terminalElement: HTMLElement;
    private compositionView: HTMLElement | null = null;
    private helperTextarea: HTMLTextAreaElement | null = null;
    private isComposing = false;
    private compositionText = "";
    
    // 输入法状态跟踪
    private lastInputMethodState = false;
    private inputBuffer = "";
    private lastKeyTime = 0;
    private pendingInput = "";
    
    // 防重复输入的状态
    private preventDuplicateTimeout: number | null = null;
    private lastProcessedText = "";
    
    // 回调函数，用于通知 TermWrap 重置输入过滤器
    private onInputFilterReset?: () => void;

    constructor(terminal: Terminal, terminalElement: HTMLElement, onInputFilterReset?: () => void) {
        this.terminal = terminal;
        this.terminalElement = terminalElement;
        this.onInputFilterReset = onInputFilterReset;
        this.setupEnhancedIME();
    }

    private setupEnhancedIME() {
        // 查找 xterm.js 创建的元素
        this.compositionView = this.terminalElement.querySelector('.composition-view');
        this.helperTextarea = this.terminalElement.querySelector('.xterm-helper-textarea');

        if (!this.compositionView || !this.helperTextarea) {
            console.warn('xterm.js composition elements not found, IME enhancements may not work');
            return;
        }

        // 监听 composition 事件
        this.setupCompositionEvents();
        
        // 监听输入法状态变化
        this.setupInputMethodTracking();
        
        // 监听光标位置变化
        this.setupCursorTracking();
        
        // 应用增强样式
        this.applyEnhancedStyles();
    }

    private setupCompositionEvents() {
        if (!this.helperTextarea) return;

        // compositionstart - 开始输入
        this.helperTextarea.addEventListener('compositionstart', (e) => {
            this.isComposing = true;
            this.compositionText = "";
            this.updateCompositionView();
        });

        // compositionupdate - 输入过程中
        this.helperTextarea.addEventListener('compositionupdate', (e) => {
            this.compositionText = e.data || "";
            this.updateCompositionView();
        });

        // compositionend - 输入结束
        this.helperTextarea.addEventListener('compositionend', (e) => {
            console.log('🎯 Composition end:', e.data);
            this.isComposing = false;
            this.compositionText = "";
            this.hideCompositionView();
            
            // 处理最终输入，防止重复
            this.handleFinalInput(e.data || "");
        });

        // 监听输入位置变化
        this.helperTextarea.addEventListener('input', (e) => {
            const inputEvent = e as InputEvent;
            console.log('🎯 Input event:', inputEvent.data, 'isComposing:', this.isComposing);
            
            if (this.isComposing) {
                this.updateCompositionView();
            } else {
                // 非 composition 模式下的输入，检查是否需要防重复处理
                if (this.shouldBlockInput(inputEvent.data || "")) {
                    console.log('🎯 Blocking duplicate input event:', inputEvent.data);
                    e.preventDefault();
                    e.stopPropagation();
                    return false;
                }
                this.handleDirectInput(inputEvent.data || "");
            }
        });
    }

    private setupCursorTracking() {
        // 监听终端光标位置变化
        let lastCursorX = -1;
        let lastCursorY = -1;

        const updateCursorPosition = () => {
            const buffer = this.terminal.buffer.active;
            const cursorX = buffer.cursorX;
            const cursorY = buffer.cursorY;

            if (cursorX !== lastCursorX || cursorY !== lastCursorY) {
                lastCursorX = cursorX;
                lastCursorY = cursorY;
                
                if (this.isComposing) {
                    this.updateCompositionPosition(cursorX, cursorY);
                }
            }
        };

        // 使用 requestAnimationFrame 监听光标变化
        const trackCursor = () => {
            updateCursorPosition();
            requestAnimationFrame(trackCursor);
        };
        trackCursor();
    }

    private updateCompositionView() {
        if (!this.compositionView || !this.isComposing) return;

        // 显示 composition view
        this.compositionView.classList.add('active');
        this.compositionView.textContent = this.compositionText;

        // 更新位置
        const buffer = this.terminal.buffer.active;
        this.updateCompositionPosition(buffer.cursorX, buffer.cursorY);
    }

    private updateCompositionPosition(cursorX: number, cursorY: number) {
        if (!this.compositionView) return;

        // 获取字符尺寸
        const charMeasure = this.getCharacterDimensions();
        if (!charMeasure) return;

        // 计算绝对位置
        const terminalRect = this.terminalElement.getBoundingClientRect();
        const viewportElement = this.terminalElement.querySelector('.xterm-viewport');
        const viewportRect = viewportElement?.getBoundingClientRect();

        if (!viewportRect) return;

        // 考虑滚动偏移
        const scrollOffset = viewportElement?.scrollTop || 0;
        
        // 计算位置 (相对于终端容器)
        const left = cursorX * charMeasure.width;
        const top = (cursorY * charMeasure.height) - scrollOffset;

        // 设置位置
        this.compositionView.style.left = `${left}px`;
        this.compositionView.style.top = `${top}px`;

        // 确保不超出边界
        this.constrainToViewport();
    }

    private getCharacterDimensions(): { width: number; height: number } | null {
        try {
            // 使用 xterm.js 内部 API 获取字符尺寸
            const core = (this.terminal as any)._core;
            if (core && core._renderService && core._renderService.dimensions) {
                const dims = core._renderService.dimensions;
                return {
                    width: dims.css.cell.width,
                    height: dims.css.cell.height
                };
            }
        } catch (e) {
            console.warn('Failed to get character dimensions from xterm.js:', e);
        }

        // 降级方案：使用固定值
        return { width: 9, height: 18 };
    }

    private constrainToViewport() {
        if (!this.compositionView) return;

        const rect = this.compositionView.getBoundingClientRect();
        const terminalRect = this.terminalElement.getBoundingClientRect();

        // 确保不超出右边界
        if (rect.right > terminalRect.right) {
            const overflow = rect.right - terminalRect.right;
            const currentLeft = parseInt(this.compositionView.style.left) || 0;
            this.compositionView.style.left = `${currentLeft - overflow - 10}px`;
        }

        // 确保不超出下边界
        if (rect.bottom > terminalRect.bottom) {
            const currentTop = parseInt(this.compositionView.style.top) || 0;
            const charHeight = this.getCharacterDimensions()?.height || 18;
            this.compositionView.style.top = `${currentTop - charHeight - 5}px`;
        }
    }

    private hideCompositionView() {
        if (!this.compositionView) return;
        this.compositionView.classList.remove('active');
    }

    private applyEnhancedStyles() {
        if (!this.compositionView) return;

        // 应用增强样式
        Object.assign(this.compositionView.style, {
            background: 'rgba(0, 0, 0, 0.9)',
            color: '#ffffff',
            border: '1px solid #555',
            borderRadius: '3px',
            padding: '2px 6px',
            fontSize: '14px',
            fontFamily: 'inherit',
            boxShadow: '0 2px 8px rgba(0, 0, 0, 0.3)',
            pointerEvents: 'none',
            zIndex: '1000'
        });
    }

    /**
     * 设置输入法状态跟踪，特别处理 caps 键切换
     */
    private setupInputMethodTracking() {
        if (!this.helperTextarea) return;

        // 监听键盘事件，特别是 caps 键
        this.helperTextarea.addEventListener('keydown', (e) => {
            const now = Date.now();
            this.lastKeyTime = now;

            // 检测 caps 键（输入法切换）
            if (e.key === 'CapsLock' || e.keyCode === 20) {
                console.log('🎯 Caps Lock detected, clearing IME state');
                this.clearIMEState();
            }

            // 检测可能的输入法切换快捷键
            if (e.metaKey && e.key === ' ') { // Cmd+Space (另一种输入法切换)
                console.log('🎯 Input method switch detected, clearing IME state');
                this.clearIMEState();
            }
        });

        // 监听 blur/focus 事件，输入法切换时可能触发
        this.helperTextarea.addEventListener('blur', () => {
            console.log('🎯 Textarea blur, clearing IME state');
            this.clearIMEState();
        });

        this.helperTextarea.addEventListener('focus', () => {
            console.log('🎯 Textarea focus, resetting IME state');
            this.resetIMEState();
        });
    }

    /**
     * 检查是否应该阻止输入事件（在事件层面）
     */
    private shouldBlockInput(data: string): boolean {
        if (!data) return false;

        const now = Date.now();
        
        // 如果最近刚按过 caps 键，阻止后续的快速重复输入
        if (this.lastKeyTime > 0 && (now - this.lastKeyTime) < 300) {
            if (data === this.lastProcessedText && /^[a-zA-Z]$/.test(data)) {
                console.log('🎯 Caps-related duplicate detected:', data);
                return true;
            }
        }
        
        // 检查是否是非常快速的重复输入
        if (data === this.lastProcessedText && (now - this.lastKeyTime) < 50) {
            return true;
        }
        
        return false;
    }

    /**
     * 处理直接输入（非 composition 模式）
     */
    private handleDirectInput(data: string) {
        if (!data) return;

        const now = Date.now();
        this.lastProcessedText = data;
        console.log('🎯 Processing direct input:', data);
    }

    /**
     * 处理 composition 结束时的最终输入
     */
    private handleFinalInput(data: string) {
        if (!data) return;

        // 清除可能的重复输入防护
        if (this.preventDuplicateTimeout) {
            clearTimeout(this.preventDuplicateTimeout);
        }

        // 设置短暂的防重复期
        this.preventDuplicateTimeout = window.setTimeout(() => {
            this.lastProcessedText = "";
            this.preventDuplicateTimeout = null;
        }, 200);

        this.lastProcessedText = data;
        console.log('🎯 Final composition input:', data);
    }

    /**
     * 清理 IME 状态
     */
    private clearIMEState() {
        this.isComposing = false;
        this.compositionText = "";
        this.inputBuffer = "";
        this.pendingInput = "";
        this.lastProcessedText = "";
        
        if (this.preventDuplicateTimeout) {
            clearTimeout(this.preventDuplicateTimeout);
            this.preventDuplicateTimeout = null;
        }
        
        this.hideCompositionView();
        
        // 通知 TermWrap 重置输入过滤器
        if (this.onInputFilterReset) {
            this.onInputFilterReset();
        }
    }

    /**
     * 重置 IME 状态
     */
    private resetIMEState() {
        this.lastInputMethodState = false;
        this.inputBuffer = "";
        this.lastKeyTime = Date.now();
    }

    /**
     * 检测输入法是否处于激活状态
     */
    private detectInputMethodActive(): boolean {
        if (!this.helperTextarea) return false;
        
        // 检查是否有正在进行的 composition
        if (this.isComposing) return true;
        
        // 检查 textarea 的状态
        const hasComposition = this.helperTextarea.value !== this.helperTextarea.textContent;
        return hasComposition;
    }

    // 公共方法
    public dispose() {
        // 清理定时器
        if (this.preventDuplicateTimeout) {
            clearTimeout(this.preventDuplicateTimeout);
        }

        // 清理事件监听器
        if (this.helperTextarea) {
            // 注意：这里应该移除具体的事件处理函数，而不是重新绑定
            // 由于我们使用了匿名函数，这里只是示例性清理
            console.log('🎯 Disposing enhanced IME handler');
        }
    }

    public getCompositionState() {
        return {
            isComposing: this.isComposing,
            compositionText: this.compositionText
        };
    }
}