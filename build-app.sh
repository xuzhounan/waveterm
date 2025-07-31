#!/bin/bash

# Wave Terminal 应用程序构建脚本
# 
# ⚠️  DEPRECATED: This manual build script is deprecated.
# Please use the official Task-based build system instead.
#
# For development: task dev
# For packaging:   task package
# For full build:  task build:backend && npm run build:prod
#
# The official build system provides:
# - Cross-platform support
# - Proper version management
# - Integrated dependency handling
# - Electron Builder integration
#
# If you need a custom build, please modify Taskfile.yml instead.

echo "❌ This manual build script has been deprecated."
echo ""
echo "Please use the official Task-based build system:"
echo "  📦 For packaging:     task package"
echo "  🚀 For development:   task dev"
echo "  🔨 For backend only:  task build:backend"
echo "  ⚡ For frontend only: npm run build:prod"
echo ""
echo "The Task system provides cross-platform builds, proper version"
echo "management, and integration with electron-builder."
echo ""
echo "See BUILD.md for complete instructions."
exit 1

# Legacy manual build code below (kept for reference)
# Remove this section after confirming Task-based builds work properly

set -e

echo "🚀 开始构建 Wave Terminal 应用程序..."

# 清理之前的构建
echo "📁 清理构建目录..."
rm -rf make/
rm -rf dist/mac-arm64/

# 构建 Go 后端和工具
echo "🛠️  编译 Go 后端和工具..."
if command -v go &> /dev/null; then
    # 编译 wavesrv 后端
    cd cmd/server
    CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o ../../bin/wavesrv.arm64 .
    cd ../..
    
    # 编译 wsh 命令
    go build -o bin/wsh cmd/wsh/main-wsh.go
    
    echo "✅ Go 后端和工具编译完成"
else
    echo "⚠️  警告: Go 未安装，跳过后端编译"
fi

# 构建生产版本
echo "🔨 构建生产版本..."
npm run build:prod

# 安装原生依赖
echo "📦 安装原生依赖..."
npm run postinstall

# 使用手动构建方法（解决 ASAR 路径问题）
echo "🎯 使用手动构建方法..."

# 创建应用目录结构
APP_DIR="make/Wave.app"
CONTENTS_DIR="$APP_DIR/Contents"
RESOURCES_DIR="$CONTENTS_DIR/Resources"
MACOS_DIR="$CONTENTS_DIR/MacOS"

echo "📁 创建应用目录结构..."
mkdir -p "$RESOURCES_DIR"
mkdir -p "$MACOS_DIR"

# 复制 Electron 可执行文件
echo "📦 复制 Electron 框架..."
ELECTRON_PATH="node_modules/electron/dist/Electron.app"
if [ -d "$ELECTRON_PATH" ]; then
    cp -R "$ELECTRON_PATH/Contents/MacOS/Electron" "$MACOS_DIR/Wave"
    cp -R "$ELECTRON_PATH/Contents/Frameworks" "$CONTENTS_DIR/"
    cp -R "$ELECTRON_PATH/Contents/Resources"/* "$RESOURCES_DIR/"
else
    echo "❌ Electron 可执行文件未找到"
    exit 1
fi

# 复制应用文件
echo "📄 复制应用文件..."
cp -R dist/ "$RESOURCES_DIR/app/"
cp package.json "$RESOURCES_DIR/app/"

# 复制后端文件
echo "🔍 复制后端文件..."
mkdir -p "$RESOURCES_DIR/app/bin"

# 复制 wavesrv 后端
if [ -f "bin/wavesrv.arm64" ]; then
    cp bin/wavesrv.arm64 "$RESOURCES_DIR/app/bin/"
    chmod +x "$RESOURCES_DIR/app/bin/wavesrv.arm64"
    echo "✅ wavesrv 后端文件已复制"
else
    echo "❌ 错误: wavesrv 后端文件缺失，应用无法正常运行"
    exit 1
fi

# 复制 wsh 命令
if [ -f "bin/wsh" ]; then
    cp bin/wsh "$RESOURCES_DIR/app/bin/"
    chmod +x "$RESOURCES_DIR/app/bin/wsh"
    echo "✅ wsh 命令已复制"
else
    echo "❌ 错误: wsh 命令缺失，终端功能可能无法正常工作"
    exit 1
fi

# 注意：跳过 npm install，Electron 应用在打包时已包含所需依赖
echo "📦 跳过依赖安装（已在构建时包含）"

# 修复 package.json 中的主入口路径
echo "🔧 修复主入口路径..."
sed -i '' 's|"main": "./dist/main/index.js"|"main": "./main/index.js"|' "$RESOURCES_DIR/app/package.json"

# 创建 Info.plist
echo "📝 创建 Info.plist..."
cat > "$CONTENTS_DIR/Info.plist" << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDisplayName</key>
    <string>Wave</string>
    <key>CFBundleExecutable</key>
    <string>Wave</string>
    <key>CFBundleIdentifier</key>
    <string>dev.commandline.waveterm</string>
    <key>CFBundleName</key>
    <string>Wave</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>0.11.3</string>
    <key>CFBundleVersion</key>
    <string>0.11.3</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15.0</string>
    <key>NSMainNibFile</key>
    <string>MainMenu</string>
    <key>NSPrincipalClass</key>
    <string>NSApplication</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
</dict>
</plist>
EOF

# 复制图标
echo "🎨 复制应用图标..."
if [ -f "build/icon.icns" ]; then
    cp build/icon.icns "$RESOURCES_DIR/icon.icns"
    echo "✅ 图标复制完成"
else
    echo "⚠️  警告: 图标文件不存在"
fi

# 设置权限
echo "🔐 设置权限..."
chmod +x "$MACOS_DIR/Wave"
chmod +x "$RESOURCES_DIR/app/dist/bin/"* 2>/dev/null || true

# 创建 ZIP 包
echo "🗜️  创建分发包..."
cd make/
zip -r "Wave-darwin-arm64-0.11.3.zip" Wave.app > /dev/null
cd ..

# 显示构建结果
if [ -d "make" ] && [ "$(ls -A make)" ]; then
    echo ""
    echo "🎉 构建完成！生成的文件："
    ls -la make/
    echo ""
    echo "📍 应用位置: ./make/Wave.app"
    echo "📍 ZIP 包: ./make/Wave-darwin-arm64-0.11.3.zip"
    
    # 显示大小信息
    APP_SIZE=$(du -sh make/Wave.app | cut -f1)
    ZIP_SIZE=$(du -sh make/Wave-darwin-arm64-0.11.3.zip | cut -f1)
    echo "📊 应用大小: $APP_SIZE"
    echo "📊 压缩包大小: $ZIP_SIZE"
    echo ""
    echo "🚀 运行应用: open make/Wave.app"
else
    echo ""
    echo "⚠️  构建失败"
fi

echo ""
echo "✨ 构建脚本执行完成！"