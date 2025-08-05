# WebView 测试 URL 建议

## 问题说明

你遇到的错误 `ERR_CONNECTION_REFUSED` 表示 `http://localhost:3000` 上没有运行的服务。这是预设配置中的一个示例 URL。

## 推荐的测试 URL

### 1. 公共网站（推荐用于初始测试）

```
https://www.google.com
https://github.com
https://www.wikipedia.org
https://news.ycombinator.com
https://example.com
```

### 2. 媒体测试网站

```
https://www.youtube.com           # 测试视频播放和音频控制
https://soundcloud.com           # 测试音频播放
https://vimeo.com               # 测试视频播放
```

### 3. Web 应用测试

```
https://web.whatsapp.com        # 测试复杂 Web 应用
https://discord.com/app         # 测试实时通信应用
https://app.slack.com          # 测试企业应用
```

### 4. 本地开发服务器

如果你有本地开发服务器，可以测试：

```bash
# 启动一个简单的 HTTP 服务器
python3 -m http.server 8000

# 然后访问
http://localhost:8000
```

或者使用 Node.js：

```bash
# 安装 http-server
npm install -g http-server

# 在任意目录启动
http-server -p 8080

# 访问
http://localhost:8080
```

## 修改默认 URL

### 方法 1：通过设置界面
1. 打开 Wave Terminal 设置
2. 查找 "Web Default URL" 设置
3. 修改为你想要的默认 URL

### 方法 2：直接设置
在创建 Web 视图后，直接在地址栏输入你想访问的 URL。

### 方法 3：修改配置文件
如果需要永久修改默认 URL，可以：

1. 找到设置文件位置（通常在 `~/.config/waveterm/` 或类似目录）
2. 修改 `web:defaulturl` 设置

## 测试步骤

1. **创建 Web 视图**
   - 点击 "+" 创建新 block
   - 选择 "Web"

2. **输入测试 URL**
   - 在地址栏输入上述任意 URL
   - 按回车键导航

3. **验证功能**
   - 页面应该正常加载
   - 导航按钮应该可用
   - 刷新功能正常

## 故障排除

### 如果所有 URL 都无法访问

1. **检查网络连接**
   - 确认你的网络连接正常
   - 尝试在普通浏览器中访问相同 URL

2. **检查代理设置**
   - Wave Terminal 可能继承了系统代理设置
   - 检查是否有代理配置影响连接

3. **查看控制台错误**
   - 打开开发者工具（Cmd/Ctrl + Option + I）
   - 查看 Console 标签中的错误信息

### 特定于新 WebView 实现的问题

如果使用新的 WebContentsView 实现时遇到问题：

1. **确认实现已启用**
   - 检查控制台是否显示：`[WebView] Using improved WebContentsView implementation`

2. **检查主进程日志**
   - 查找 `[WebViewManager]` 相关日志

3. **验证 IPC 通信**
   - 在 Network 标签查看 IPC 消息

## 性能测试 URL

用于测试 WebView 性能的网站：

```
https://browserbench.org/Speedometer2.1/    # JavaScript 性能测试
https://webglsamples.org/aquarium/aquarium.html    # WebGL 性能测试
https://www.youtube.com/watch?v=dQw4w9WgXcQ    # 视频播放性能
```