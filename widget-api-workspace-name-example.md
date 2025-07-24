# Widget API - Get Workspace by Name

这个功能允许你通过工作区名称来获取工作区信息，而不需要知道具体的工作区ID。

## API端点

```
GET /api/v1/widgets/workspace/name/{workspace_name}
```

## 功能特性

- **大小写不敏感**：支持大小写不敏感的名称匹配
- **URL编码支持**：支持包含空格等特殊字符的工作区名称
- **错误处理**：适当的HTTP状态码和错误消息

## 使用示例

### 1. 获取名为"Default"的工作区信息

```bash
curl -X GET "http://localhost:8090/api/v1/widgets/workspace/name/Default"
```

**响应示例（成功）：**
```json
{
  "success": true,
  "workspace": {
    "workspace_id": "workspace-123",
    "name": "Default",
    "tab_ids": ["tab-1", "tab-2"],
    "active_tab_id": "tab-1"
  }
}
```

### 2. 获取包含空格的工作区名称

```bash
curl -X GET "http://localhost:8090/api/v1/widgets/workspace/name/My%20Workspace"
```

### 3. 工作区不存在的情况

```bash
curl -X GET "http://localhost:8090/api/v1/widgets/workspace/name/NonExistent"
```

**响应示例（未找到）：**
```json
{
  "success": false,
  "error": "workspace with name 'NonExistent' not found"
}
```

## 常见用例

### 1. 在指定工作区创建Widget

```bash
# 1. 首先通过名称获取工作区ID
WORKSPACE_ID=$(curl -s -X GET "http://localhost:8090/api/v1/widgets/workspace/name/MyWorkspace" | jq -r '.workspace.workspace_id')

# 2. 使用获取到的工作区ID创建widget
curl -X POST "http://localhost:8090/api/v1/widgets" \
  -H "Content-Type: application/json" \
  -d "{
    \"workspace_id\": \"$WORKSPACE_ID\",
    \"widget_type\": \"terminal\",
    \"title\": \"My Terminal\"
  }"
```

### 2. 批量操作特定工作区

```bash
#!/bin/bash

WORKSPACE_NAME="Development"

# 获取工作区信息
WORKSPACE_INFO=$(curl -s -X GET "http://localhost:8090/api/v1/widgets/workspace/name/$WORKSPACE_NAME")

if echo "$WORKSPACE_INFO" | jq -r '.success' | grep -q true; then
    WORKSPACE_ID=$(echo "$WORKSPACE_INFO" | jq -r '.workspace.workspace_id')
    echo "Found workspace '$WORKSPACE_NAME' with ID: $WORKSPACE_ID"
    
    # 在该工作区创建多个widgets...
    for widget_type in terminal web files; do
        curl -X POST "http://localhost:8090/api/v1/widgets" \
          -H "Content-Type: application/json" \
          -d "{
            \"workspace_id\": \"$WORKSPACE_ID\",
            \"widget_type\": \"$widget_type\",
            \"title\": \"Auto-created $widget_type\"
          }"
    done
else
    echo "Workspace '$WORKSPACE_NAME' not found"
fi
```

## 错误处理

| 状态码 | 情况 | 响应示例 |
|--------|------|----------|
| 200 | 成功找到工作区 | `{"success": true, "workspace": {...}}` |
| 404 | 工作区不存在 | `{"success": false, "error": "workspace with name 'XYZ' not found"}` |
| 400 | 工作区名称为空 | `{"success": false, "error": "workspace_name is required"}` |
| 500 | 服务器内部错误 | `{"success": false, "error": "Internal server error: ..."}` |

## 测试

运行提供的测试脚本来验证功能：

```bash
./test-widget-api-workspace-name.sh
```

该脚本会测试各种场景，包括：
- 正常的工作区名称查找
- 不存在的工作区名称
- 空名称处理
- URL编码的名称
- 大小写不敏感匹配

## 集成说明

这个API端点完全兼容现有的Widget API系统，可以与其他端点配合使用：

1. `GET /api/v1/widgets/workspaces` - 列出所有工作区
2. `GET /api/v1/widgets/workspace/name/{name}` - 根据名称获取工作区（新功能）
3. `GET /api/v1/widgets/workspace/{id}` - 根据ID获取工作区widgets
4. `POST /api/v1/widgets` - 创建widget

通过这种方式，用户可以更方便地管理和操作工作区，无需记住复杂的工作区ID。