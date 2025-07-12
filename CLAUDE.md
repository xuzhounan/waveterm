Modern Terminal - Claude Code 分阶段开发计划

## 🏗️ 当前架构概览

基于 `/frontend/app/view/modernterm/` 的完整组件化架构：
- ✅ **组件体系完整**: ModernTerminalBlock + CommandInput/Output + 完整测试覆盖
- ✅ **用户交互基础**: 命令历史导航、多行粘贴、输入法支持、快捷键
- ✅ **执行机制**: 基于 TermWshClient + RpcApi 的流式命令执行
- ⚠️ **输出处理**: 基础 ANSI 清理，需要升级为颜色渲染
- ❌ **会话持久**: 每条命令独立进程，无上下文保持
- ❌ **PTY 支持**: 无真正终端交互，不支持控制字符

## 📋 分阶段开发计划

### 🚀 Phase 1: 输出体验优化 (优先级: ⭐⭐⭐)

**目标**: 基于现有架构快速提升用户体验

#### 1.1 ANSI 颜色渲染增强
```typescript
// 当前: CommandOutput.tsx 中的基础清理
const cleanOutput = finalOutput.replace(/\x1b\[[0-9;]*m/g, '');

// 升级为: 完整的 ANSI-to-HTML 转换
const coloredOutput = parseAnsiToHtml(finalOutput);
```

**技术选型**: `ansi-to-html` 库
**影响文件**: `CommandOutput.tsx`
**风险**: 低 - 不涉及架构变更

#### 1.2 流式输出性能优化
- 优化现有防抖策略，减少渲染卡顿
- 调整输出更新频率和批处理逻辑
- 改进超长输出的处理性能

#### 1.3 输入交互细节完善
- ✅ 命令历史导航 (已实现)
- ✅ 多行粘贴预览 (已实现)  
- 🔄 Shift+Enter 多行输入增强
- 🔄 自动滚动到底部优化

---

### 🔧 Phase 2: 会话架构升级 (优先级: ⭐⭐⭐⭐)

**目标**: 实现持续会话支持，突破独立进程限制

#### 2.1 PersistentShellSession 实现
```typescript
class PersistentShellSession {
  private shell: ChildProcess;
  private workingDir: string;
  private envVars: Record<string, string>;
  
  async executeCommand(cmd: string): Promise<CommandResult>;
  async changeDirectory(path: string): Promise<void>;
  async setEnvironment(key: string, value: string): Promise<void>;
}
```

**技术方案**: 
- 基于现有 `TermWshClient` 扩展
- 维护单一 `spawn('zsh')` 进程
- 通过特殊标记符分隔命令输出

#### 2.2 命令上下文保持
- `cd` 命令影响后续执行路径
- `export` 环境变量持久化
- 工作目录状态同步

#### 2.3 交互式程序支持基础
- Python REPL 模式检测
- Node.js 交互式环境
- 多行输入缓冲和执行

---

### 🎛️ Phase 3: 完整终端体验 (优先级: ⭐⭐⭐⭐⭐)

**目标**: 接近原生终端的完整交互体验

#### 3.1 node-pty 集成
```typescript
import * as pty from 'node-pty';

class PtyTerminalController {
  private ptyProcess: pty.IPty;
  
  setupPty(): void;
  handleInput(data: string): void;
  handleResize(cols: number, rows: number): void;
}
```

**技术选型**: `node-pty` > `xterm.js`（更符合现有架构）

#### 3.2 控制字符支持
- Ctrl+C 进程中断
- Tab 自动补全
- 方向键命令行编辑
- Ctrl+L 清屏等

#### 3.3 终端尺寸管理
- 动态计算终端行列数
- 窗口大小变化响应
- 输出区域自适应

#### 3.4 完整测试和文档
- 维护现有测试覆盖标准
- PTY 集成的 Mock 和测试
- 性能基准和优化

---

## 🎯 实施建议

### 优先启动任务
1. **Phase 1.1 - ANSI 颜色渲染** (影响面小，效果明显)
2. **Phase 1.2 - 流式输出优化** (基于现有实现改进)

### 风险控制
- Phase 2 和 Phase 3 有依赖关系，建议配合实施
- 保持向后兼容，不破坏现有 RPC 调用机制
- 每个 Phase 完成后进行完整回归测试

### 技术债务清理
- 统一 ANSI 处理逻辑
- 优化组件间状态同步
- 改进错误处理和用户反馈

---

## 🏁 最终目标

将 Modern Terminal 打造成具备以下能力的 Wave 内置终端组件：
- ✅ **颜色输出和语法高亮** - 完整 ANSI 支持
- ✅ **会话持久化** - 支持 cd、export、REPL 等
- ✅ **原生交互体验** - PTY + 控制字符支持  
- ✅ **完整历史管理** - 命令历史、复制粘贴、导出记录
- ✅ **高性能渲染** - 流式输出、大量数据处理优化
