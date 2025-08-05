# WebView 改进实现测试说明

## 实现完成状态

已完成以下集成工作：

1. ✅ **主进程集成**
   - `emain/emain.ts` - 添加了 WebView IPC 初始化
   - `emain/emain-webview-manager.ts` - 完整的 WebContentsView 管理器

2. ✅ **预加载脚本**
   - `emain/preload.ts` - 暴露了所有 WebView API 方法

3. ✅ **前端集成**
   - `frontend/app/view/webview/index.tsx` - 创建了切换逻辑
   - `frontend/app/view/webview/webview-improved.tsx` - 新的 WebView 实现
   - `frontend/app/block/block.tsx` - 集成了工厂函数

4. ✅ **TypeScript 类型**
   - `frontend/types/electron-api.d.ts` - 完整的 API 类型定义

## 测试步骤

### 1. 启用新实现

编辑 `frontend/app/view/webview/index.tsx`：

```typescript
// 将此值改为 true 以启用新的 WebContentsView 实现
const USE_IMPROVED_WEBVIEW = true;
```

### 2. 构建并运行

```bash
# 开发模式运行
task dev
```

### 3. 创建 Web 视图测试

1. 启动应用后，创建新的 block（点击 "+"）
2. 选择 "Web" 类型
3. 观察控制台输出，应该看到：
   ```
   [WebView] Using improved WebContentsView implementation for block: <blockId>
   ```

### 4. 功能测试清单

- [ ] **基础导航**
  - [ ] 输入 URL 并按回车导航
  - [ ] 测试 https://google.com
  - [ ] 测试 http://localhost:3000（本地服务器）
  
- [ ] **导航控制**
  - [ ] 后退按钮
  - [ ] 前进按钮
  - [ ] 刷新按钮
  - [ ] 主页按钮

- [ ] **搜索功能**
  - [ ] Cmd/Ctrl + F 打开搜索
  - [ ] 输入搜索词
  - [ ] 上一个/下一个结果

- [ ] **缩放控制**
  - [ ] 右键菜单 > 设置缩放因子
  - [ ] 测试不同缩放级别

- [ ] **其他功能**
  - [ ] 开发者工具（右键菜单）
  - [ ] 音频静音控制（访问 YouTube）
  - [ ] 外部浏览器打开

### 5. 性能对比

记录以下指标：

1. **内存使用**
   - 打开 5 个 Web 视图
   - 使用活动监视器/任务管理器查看内存

2. **加载时间**
   - 记录页面加载完成时间

3. **响应性**
   - 导航操作的响应速度

## 调试提示

### 查看日志

1. **主进程日志**
   - 在终端中查看 WebViewManager 日志
   - 搜索 `[WebViewManager]` 前缀

2. **渲染进程日志**
   - 打开开发者工具（Cmd/Ctrl + Option + I）
   - 查看 Console 标签

3. **IPC 通信**
   - 在 Network 标签中查看 IPC 消息

### 常见问题

1. **WebView 不显示**
   - 检查是否有错误日志
   - 确认 `setupWebViewIPC()` 被调用

2. **导航失败**
   - 检查 URL 格式
   - 查看主进程错误日志

3. **事件未触发**
   - 确认事件监听器正确注册
   - 检查 blockId 是否正确

## 回滚方法

如果遇到问题，可以快速回滚：

1. 编辑 `frontend/app/view/webview/index.tsx`
2. 将 `USE_IMPROVED_WEBVIEW` 设为 `false`
3. 重新运行应用

## 反馈

请记录测试结果和任何问题：

- 功能是否正常：✅/❌
- 性能表现：更好/相同/更差
- 具体问题描述
- 改进建议