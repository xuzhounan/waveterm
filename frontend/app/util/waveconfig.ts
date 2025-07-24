// Copyright 2024, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

/**
 * Wave Terminal Configuration Utilities
 * 统一环境配置和运行时检测
 */

export interface WaveServerConfig {
    baseUrl: string;
    port: number;
    authKey?: string;
    isRunning: boolean;
}

// 获取当前运行环境的配置
export async function detectWaveServerConfig(): Promise<WaveServerConfig | null> {
    // 尝试从window对象获取运行时配置（如果在Electron环境中）
    const electronConfig = await tryGetElectronConfig();
    if (electronConfig) {
        return electronConfig;
    }

    // 回退到端口扫描
    return await scanForWaveServer();
}

// 尝试从Electron主进程获取配置
async function tryGetElectronConfig(): Promise<WaveServerConfig | null> {
    try {
        const { getWaveConfigFromElectron } = await import('./electronbridge');
        const config = await getWaveConfigFromElectron();
        
        if (config) {
            return {
                baseUrl: config.baseUrl || `http://localhost:${config.port}`,
                port: config.port,
                authKey: config.authKey,
                isRunning: true,
            };
        }
    } catch (error) {
        console.log('Electron config not available, falling back to port scan');
    }
    return null;
}

// 扫描常用端口查找Wave服务器
async function scanForWaveServer(): Promise<WaveServerConfig | null> {
    const commonPorts = [
        61269, // 默认生产端口
        51920, // 常见动态端口
        50531, // 常见动态端口
        57029, // 常见动态端口
        8080,  // 开发端口
        3000,  // 开发端口
    ];

    for (const port of commonPorts) {
        try {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 1000);

            const response = await fetch(`http://localhost:${port}/api/v1/widgets`, {
                method: 'HEAD', // 使用HEAD请求减少网络开销
                signal: controller.signal,
                cache: 'no-cache',
            });

            clearTimeout(timeoutId);

            if (response.ok) {
                return {
                    baseUrl: `http://localhost:${port}`,
                    port: port,
                    isRunning: true,
                };
            }
        } catch (error) {
            // 继续尝试下一个端口
            continue;
        }
    }

    return null;
}

// 缓存配置以避免重复检测
let cachedConfig: WaveServerConfig | null = null;
let configCacheTime = 0;
const CACHE_DURATION = 30000; // 30秒缓存

export async function getWaveServerConfig(): Promise<WaveServerConfig | null> {
    const now = Date.now();
    
    // 如果缓存仍然有效，直接返回
    if (cachedConfig && (now - configCacheTime) < CACHE_DURATION) {
        return cachedConfig;
    }

    // 重新检测配置
    cachedConfig = await detectWaveServerConfig();
    configCacheTime = now;
    
    return cachedConfig;
}

// 清除配置缓存（在连接失败时调用）
export function clearWaveServerConfigCache(): void {
    cachedConfig = null;
    configCacheTime = 0;
}

// 构建API URL
export async function buildApiUrl(endpoint: string): Promise<string | null> {
    const config = await getWaveServerConfig();
    if (!config) {
        return null;
    }
    
    const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
    return `${config.baseUrl}/${cleanEndpoint}`;
}

// Wave API 调用包装器
export async function waveApiFetch(endpoint: string, options: RequestInit = {}): Promise<Response | null> {
    const url = await buildApiUrl(endpoint);
    if (!url) {
        throw new Error('Wave server not found. Please ensure Wave Terminal is running.');
    }

    try {
        const response = await fetch(url, {
            ...options,
            headers: {
                'Cache-Control': 'no-cache',
                ...options.headers,
            },
        });

        return response;
    } catch (error) {
        // 如果连接失败，清除缓存以便下次重新检测
        clearWaveServerConfigCache();
        throw error;
    }
}