# Workspace级Widget配置系统设计文档

## 🎯 设计目标

扩展Wave的配置系统，支持每个workspace拥有独立的widget配置，同时保持系统级widget作为共享基础。

## 📋 技术方案

### 1. 数据结构扩展

#### 1.1 后端配置类型扩展
```go
// pkg/wconfig/settingsconfig.go
type FullConfigType struct {
    Settings         SettingsType                            `json:"settings" merge:"meta"`
    MimeTypes        map[string]MimeTypeConfigType           `json:"mimetypes"`
    DefaultWidgets   map[string]WidgetConfigType             `json:"defaultwidgets"`
    Widgets          map[string]WidgetConfigType             `json:"widgets"`
    WorkspaceWidgets map[string]map[string]WidgetConfigType  `json:"workspacewidgets"` // 新增
    Presets          map[string]waveobj.MetaMapType          `json:"presets"`
    TermThemes       map[string]TermThemeType                `json:"termthemes"`
    Connections      map[string]ConnKeywords                 `json:"connections"`
    Bookmarks        map[string]WebBookmark                  `json:"bookmarks"`
    ConfigErrors     []ConfigError                           `json:"configerrors" configfile:"-"`
}
```

#### 1.2 前端类型定义
```typescript
// frontend/types/types.ts
type FullConfigType = {
    settings: SettingsType;
    defaultwidgets: { [key: string]: WidgetConfigType };
    widgets: { [key: string]: WidgetConfigType };
    workspacewidgets: { [workspaceId: string]: { [key: string]: WidgetConfigType } }; // 新增
    // ... 其他字段
};
```

### 2. 配置文件组织

#### 2.1 文件结构
```
~/.config/wave/
├── settings.json
├── widgets.json                    # 系统级共享widgets
├── workspaces/                     # 新增workspace配置目录
│   ├── {workspace-id-1}/
│   │   └── widgets.json           # workspace特定widgets
│   ├── {workspace-id-2}/
│   │   └── widgets.json
│   └── ...
```

#### 2.2 配置优先级
1. **Workspace级widget配置** (最高优先级)
2. **系统级widget配置** (中等优先级)  
3. **默认widget配置** (最低优先级)

### 3. 配置合并逻辑

#### 3.1 Widget合并规则
```typescript
function mergeWidgetConfigs(
    defaultWidgets: Record<string, WidgetConfigType>,
    systemWidgets: Record<string, WidgetConfigType>,
    workspaceWidgets: Record<string, WidgetConfigType>
): Record<string, WidgetConfigType> {
    const merged = { ...defaultWidgets };
    
    // 应用系统级配置
    Object.entries(systemWidgets).forEach(([key, config]) => {
        merged[key] = { ...merged[key], ...config };
    });
    
    // 应用workspace级配置 (最高优先级)
    Object.entries(workspaceWidgets).forEach(([key, config]) => {
        merged[key] = { ...merged[key], ...config };
    });
    
    return merged;
}
```

#### 3.2 配置覆盖策略
- **完全覆盖**: workspace可以定义全新的widget
- **部分覆盖**: workspace可以只修改特定字段(如url、label等)
- **显示控制**: workspace可以隐藏系统级widget
- **顺序调整**: workspace可以重新排序widget显示顺序

### 4. 实现步骤

#### 4.1 后端实现
1. **扩展配置读取逻辑**
   ```go
   // pkg/wconfig/settingsconfig.go
   func readWorkspaceWidgetConfigs(workspaceId string) (map[string]WidgetConfigType, []ConfigError) {
       configPath := filepath.Join("workspaces", workspaceId, "widgets.json")
       return readConfigPart(configPath, false)
   }
   
   func ReadFullConfig() FullConfigType {
       // ... 现有逻辑
       
       // 读取所有workspace widget配置
       workspaceWidgets := make(map[string]map[string]WidgetConfigType)
       workspaceDirs := getWorkspaceConfigDirs()
       for _, workspaceId := range workspaceDirs {
           wsWidgets, errs := readWorkspaceWidgetConfigs(workspaceId)
           fullConfig.ConfigErrors = append(fullConfig.ConfigErrors, errs...)
           if len(wsWidgets) > 0 {
               workspaceWidgets[workspaceId] = wsWidgets
           }
       }
       fullConfig.WorkspaceWidgets = workspaceWidgets
       
       return fullConfig
   }
   ```

2. **配置写入接口**
   ```go
   func SetWorkspaceWidgetConfig(workspaceId string, widgetKey string, config WidgetConfigType) error {
       configDir := filepath.Join(wavebase.GetWaveConfigDir(), "workspaces", workspaceId)
       if err := os.MkdirAll(configDir, 0755); err != nil {
           return err
       }
       
       configPath := "workspaces/" + workspaceId + "/widgets.json"
       m, cerrs := ReadWaveHomeConfigFile(configPath)
       if len(cerrs) > 0 && !os.IsNotExist(cerrs[0]) {
           return fmt.Errorf("error reading workspace config: %v", cerrs[0])
       }
       if m == nil {
           m = make(waveobj.MetaMapType)
       }
       
       m[widgetKey] = config
       return WriteWaveHomeConfigFile(configPath, m)
   }
   ```

#### 4.2 前端实现
1. **扩展配置原子系统**
   ```typescript
   // frontend/app/store/global.ts
   const workspaceWidgetsAtom: Atom<Record<string, WidgetConfigType>> = atom((get) => {
       const fullConfig = get(atoms.fullConfigAtom);
       const workspace = get(atoms.workspaceAtom);
       if (!workspace || !fullConfig) return {};
       
       const defaultWidgets = fullConfig.defaultwidgets || {};
       const systemWidgets = fullConfig.widgets || {};
       const workspaceWidgets = fullConfig.workspacewidgets?.[workspace.oid] || {};
       
       return mergeWidgetConfigs(defaultWidgets, systemWidgets, workspaceWidgets);
   });
   ```

2. **更新Widget渲染组件**
   ```typescript
   // frontend/app/workspace/workspace.tsx
   const WorkspaceComponent = () => {
       // 替换原有的 fullConfig?.widgets
       const widgets = useAtomValue(atoms.workspaceWidgetsAtom);
       const sortedWidgets = sortByDisplayOrder(widgets);
       
       // ... 其余渲染逻辑保持不变
   };
   ```

#### 4.3 配置管理UI
1. **Workspace设置面板**
   ```typescript
   // 新增组件: WorkspaceWidgetSettings.tsx
   const WorkspaceWidgetSettings = () => {
       const workspace = useAtomValue(atoms.workspaceAtom);
       const [workspaceWidgets, setWorkspaceWidgets] = useState({});
       
       const handleWidgetConfigChange = async (widgetKey: string, config: WidgetConfigType) => {
           await RpcApi.SetWorkspaceWidgetConfig(workspace.oid, widgetKey, config);
           // 触发配置重新加载
       };
       
       return (
           <div>
               {/* Widget配置表单界面 */}
           </div>
       );
   };
   ```

2. **右键菜单扩展**
   ```typescript
   // 在workspace.tsx中添加右键菜单项
   const contextMenuItems = [
       {
           label: "编辑Workspace Widgets",
           click: () => openWorkspaceWidgetSettings()
       },
       {
           label: "编辑全局 Widgets", 
           click: () => openGlobalWidgetSettings()
       }
   ];
   ```

### 5. 使用场景示例

#### 5.1 项目特定配置
```json
// ~/.config/wave/workspaces/project-a-workspace-id/widgets.json
{
    "local_web": {
        "display:order": 1,
        "icon": "brands@chrome",
        "label": "Local Dev",
        "color": "#4285f4",
        "blockdef": {
            "meta": {
                "view": "web",
                "url": "http://localhost:3000",
                "pinnedurl": "http://localhost:3000"
            }
        }
    },
    "project_terminal": {
        "display:order": 2,
        "icon": "terminal",
        "label": "Project Shell",
        "blockdef": {
            "meta": {
                "view": "term",
                "cmd:env": {
                    "PROJECT_ROOT": "/path/to/project"
                },
                "cmd:initscript": "cd $PROJECT_ROOT && npm run dev"
            }
        }
    },
    "defwidget@files": {
        "display:hidden": true  // 在此workspace隐藏文件浏览器
    }
}
```

#### 5.2 不同端口项目
```json
// ~/.config/wave/workspaces/project-b-workspace-id/widgets.json  
{
    "local_web": {
        "label": "Project B Dev",
        "blockdef": {
            "meta": {
                "url": "http://localhost:8090"
            }
        }
    }
}
```

### 6. 向后兼容性

- 现有的系统级widget配置保持完全兼容
- 没有workspace特定配置的workspace将继续使用系统级配置
- 配置升级是透明的，不需要用户手动迁移

### 7. 技术优势

1. **灵活性**: 每个workspace可以有独特的工具配置
2. **共享性**: 系统级widget仍然作为共享基础
3. **优先级明确**: 清晰的配置覆盖层次
4. **扩展性**: 未来可以进一步扩展为基于项目、连接等维度的配置
5. **性能**: 配置合并在内存中进行，不影响运行时性能

## 🚀 实施计划

1. **Phase 1**: 后端配置系统扩展
2. **Phase 2**: 前端原子系统更新
3. **Phase 3**: UI组件更新
4. **Phase 4**: 配置管理界面
5. **Phase 5**: 测试和文档完善

此设计方案既保持了现有系统的稳定性，又为用户提供了强大的定制能力，完全满足不同项目需要不同widget配置的需求。

## 🧪 测试验证

### 后端测试
1. **配置加载测试**: 验证`ReadFullConfig()`正确加载workspace widget配置
2. **配置写入测试**: 验证`SetWorkspaceWidgetConfig()`正确创建和更新配置文件
3. **错误处理测试**: 验证无效配置的错误处理和回退机制

### 前端测试
1. **配置合并测试**: 验证`workspaceWidgetsAtom`正确合并多层配置
2. **UI更新测试**: 验证workspace切换时widget列表正确更新
3. **RPC调用测试**: 验证`SetWorkspaceWidgetConfigCommand`正常工作

### 集成测试
1. **端到端配置流程**: 从UI修改到配置文件更新的完整流程
2. **多workspace隔离测试**: 验证不同workspace的配置互不影响
3. **配置热重载测试**: 验证配置变更时的实时更新

## 📝 实现总结

已完成的功能模块：

### ✅ 后端实现
- 扩展`FullConfigType`支持`WorkspaceWidgets`字段
- 实现workspace配置目录扫描(`getWorkspaceConfigDirs`)
- 实现workspace widget配置读取(`readWorkspaceWidgetConfigs`)
- 实现workspace widget配置写入(`SetWorkspaceWidgetConfig`)
- 添加RPC接口`SetWorkspaceWidgetConfigCommand`

### ✅ 前端实现  
- 新增`workspaceWidgetsAtom`配置合并原子
- 更新Widget组件使用合并后的配置
- 扩展右键菜单支持workspace配置编辑
- 添加RPC客户端调用方法

### ✅ 类型系统
- 扩展`FullConfigType`类型定义
- 添加`WorkspaceWidgetConfigRequest`RPC类型
- 保持类型安全的配置操作

### 🔄 代码生成
注意：部分自动生成的代码（`wshclient.go`, `wshclientapi.ts`, `gotypes.d.ts`）需要运行相应的生成命令来同步最新的类型定义和RPC方法。

## 📖 用户使用指南

### 基本用法
1. 右键点击workspace widget栏
2. 选择"Edit Workspace widgets.json"  
3. 参考`docs/workspace-widgets-example.json`配置格式
4. 保存文件后配置自动生效

### 配置语法
```json
{
  "custom_widget_name": {
    "display:order": 1,
    "icon": "icon-name",
    "label": "Display Label", 
    "color": "#hex-color",
    "blockdef": {
      "meta": {
        "view": "web|term|preview",
        "url": "http://localhost:3000"
      }
    }
  }
}
```

### 高级功能
- **覆盖系统widget**: 使用相同的widget key覆盖系统配置
- **隐藏系统widget**: 设置`"display:hidden": true`
- **环境变量**: 在terminal widget中使用`"cmd:env"`设置项目环境
- **初始化脚本**: 使用`"cmd:initscript"`设置启动命令

这个workspace级widget配置系统为Wave用户提供了前所未有的定制灵活性，让每个项目都能拥有最适合的工具配置。