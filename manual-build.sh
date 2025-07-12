#!/bin/bash

# 手动构建 Wave Terminal macOS 应用
set -e

echo "🚀 手动构建 Wave Terminal..."

# 清理
rm -rf make/
rm -rf dist/mac-arm64/

# 确保构建完成
npm run build:prod

# 创建应用目录结构
APP_DIR="make/Wave.app"
CONTENTS_DIR="$APP_DIR/Contents"
RESOURCES_DIR="$CONTENTS_DIR/Resources"
MACOS_DIR="$CONTENTS_DIR/MacOS"

echo "📁 创建应用目录结构..."
mkdir -p "$RESOURCES_DIR"
mkdir -p "$MACOS_DIR"

# 复制 Electron 可执行文件
echo "📦 复制 Electron 可执行文件..."
ELECTRON_PATH="/Users/xzn/Desktop/code-project/waveterm/node_modules/electron/dist/Electron.app"
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

# 修复 package.json 中的路径
echo "🔧 修复 package.json 路径..."
sed -i '' 's|"main": "./dist/main/index.js"|"main": "./main/index.js"|g' "$RESOURCES_DIR/app/package.json"

# 复制应用图标
echo "🎨 复制应用图标..."
if [ -f "build/icon.icns" ]; then
    cp build/icon.icns "$RESOURCES_DIR/icon.icns"
    echo "✅ 图标复制完成"
else
    echo "⚠️  警告: 图标文件不存在"
fi

# 创建 Info.plist
echo "📝 创建 Info.plist..."
cat > "$CONTENTS_DIR/Info.plist" << EOF
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
    <key>CFBundleIconFile</key>
    <string>icon.icns</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15.0</string>
    <key>NSMainNibFile</key>
    <string>MainMenu</string>
    <key>NSPrincipalClass</key>
    <string>NSApplication</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF

# 设置可执行权限
chmod +x "$MACOS_DIR/Wave"
chmod +x "$RESOURCES_DIR/app/dist/bin/"* 2>/dev/null || true

# 创建 ZIP 包
echo "🗜️  创建 ZIP 包..."
cd make/
zip -r "Wave-darwin-arm64-0.11.3.zip" Wave.app
cd ..

echo "✅ 构建完成！"
echo "📍 应用位置: make/Wave.app"
echo "📍 ZIP 包: make/Wave-darwin-arm64-0.11.3.zip"

# 测试应用
echo "🧪 测试应用是否可运行..."
if [ -f "$MACOS_DIR/Wave" ]; then
    echo "✅ 可执行文件存在"
    echo "ℹ️  可以尝试运行: open make/Wave.app"
else
    echo "❌ 可执行文件缺失"
fi