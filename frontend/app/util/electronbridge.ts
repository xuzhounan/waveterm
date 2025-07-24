// Copyright 2024, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

/**
 * Electron Bridge for Wave Terminal Configuration
 * 提供主进程和渲染进程之间的配置通信
 */

export interface ElectronWaveConfig {
    port: number;
    authKey: string;
    dataDir: string;
    configDir: string;
    baseUrl: string;
}

// 检查是否在Electron环境中
export function isElectronEnvironment(): boolean {
    return typeof window !== 'undefined' && 
           typeof (window as any).api !== 'undefined';
}

// 从主进程获取Wave配置
export async function getWaveConfigFromElectron(): Promise<ElectronWaveConfig | null> {
    if (!isElectronEnvironment()) {
        return null;
    }

    try {
        const api = (window as any).api;
        if (api && api.getWaveConfig) {
            return api.getWaveConfig();
        }
    } catch (error) {
        console.warn('Failed to get Wave config from Electron:', error);
    }

    return null;
}

// 通知主进程配置变化
export async function notifyConfigChange(config: Partial<ElectronWaveConfig>): Promise<void> {
    if (!isElectronEnvironment()) {
        return;
    }

    try {
        const api = (window as any).api;
        if (api && api.updateWaveConfig) {
            await api.updateWaveConfig(config);
        }
    } catch (error) {
        console.warn('Failed to notify config change to Electron:', error);
    }
}