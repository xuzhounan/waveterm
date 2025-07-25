// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import react from "@vitejs/plugin-react-swc";
import { defineConfig } from "electron-vite";
import flow from "rollup-plugin-flow";
import { ViteImageOptimizer } from "vite-plugin-image-optimizer";
import { viteStaticCopy } from "vite-plugin-static-copy";
import svgr from "vite-plugin-svgr";
import tsconfigPaths from "vite-tsconfig-paths";

export default defineConfig({
    main: {
        root: ".",
        build: {
            rollupOptions: {
                input: {
                    index: "emain/emain.ts",
                },
            },
            outDir: "dist/main",
        },
        plugins: [tsconfigPaths(), flow()],
        css: {
            preprocessorOptions: {
                scss: {
                    silenceDeprecations: ["mixed-decls", "legacy-js-api"],
                    api: "modern-compiler",
                },
            },
        },
        resolve: {
            alias: {
                "@": "frontend",
            },
        },
        define: {
            "process.env.WS_NO_BUFFER_UTIL": "true",
            "process.env.WS_NO_UTF_8_VALIDATE": "true",
        },
    },
    preload: {
        root: ".",
        build: {
            sourcemap: true,
            rollupOptions: {
                input: {
                    index: "emain/preload.ts",
                    "preload-webview": "emain/preload-webview.ts",
                },
                output: {
                    format: "cjs",
                },
            },
            outDir: "dist/preload",
        },
        plugins: [tsconfigPaths(), flow()],
        css: {
            preprocessorOptions: {
                scss: {
                    silenceDeprecations: ["mixed-decls", "legacy-js-api"],
                    api: "modern-compiler",
                },
            },
        },
    },
    renderer: {
        root: ".",
        build: {
            target: "es6",
            sourcemap: true,
            outDir: "dist/frontend",
            rollupOptions: {
                input: {
                    index: "index.html",
                },
            },
        },
        server: {
            open: false,
        },
        optimizeDeps: {
            exclude: ["parse5", "entities"],
        },
        css: {
            preprocessorOptions: {
                scss: {
                    silenceDeprecations: ["mixed-decls", "legacy-js-api"],
                    api: "modern-compiler",
                },
            },
        },
        plugins: [
            ViteImageOptimizer(),
            tsconfigPaths(),
            flow(),
            svgr({
                svgrOptions: { exportType: "default", ref: true, svgo: false, titleProp: true },
                include: "**/*.svg",
            }),
            react({}),
            viteStaticCopy({
                targets: [{ src: "node_modules/monaco-editor/min/vs/*", dest: "monaco" }],
            }),
        ],
    },
});
