# Wave Terminal MCP 部署指南

本指南详细说明如何部署集成了MCP (Model Context Protocol) 功能的Wave Terminal，包括状态工具栏、呼吸灯和MCP客户端等新功能。

## 🚀 快速部署

### 1. 开发环境部署

#### 前置要求
- **Node.js**: 18+ 版本
- **Go**: 1.21+ 版本  
- **Git**: 用于版本控制
- **操作系统**: macOS, Linux, 或 Windows

#### 步骤
```bash
# 1. 克隆项目
git clone <your-wave-terminal-repo>
cd waveterm

# 2. 安装前端依赖
npm install

# 3. 构建前端 (开发版)
npm run build:dev

# 4. 启动Wave Terminal
npm run dev
```

### 2. 生产环境部署

#### 构建生产版本
```bash
# 1. 构建生产版前端
npm run build:prod

# 2. 构建Go后端
go build -o wave-terminal cmd/server/main-server.go

# 3. 启动生产服务器
./wave-terminal
```

## 🔧 MCP服务器部署

### 使用内置持久化脚本
```bash
# 启动MCP服务器
./persistent-server.sh start

# 查看服务器状态
./persistent-server.sh status

# 查看实时日志
./persistent-server.sh logs

# 停止服务器
./persistent-server.sh stop
```

### 手动启动MCP服务器
```bash
# 设置环境变量
export WAVETERM_DATA_DIR="/tmp/waveterm-mcp"
export WAVETERM_AUTH_KEY="your-auth-key-here"

# 启动服务器
go run cmd/server/main-server.go
```

## 🌐 云端部署

### 1. Docker 部署

#### 创建 Dockerfile
```dockerfile
FROM node:18-alpine AS frontend-builder

WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production

COPY frontend/ ./frontend/
COPY *.json ./
RUN npm run build:prod

FROM golang:1.21-alpine AS backend-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o wave-terminal cmd/server/main-server.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=backend-builder /app/wave-terminal .
COPY --from=frontend-builder /app/dist ./dist/

EXPOSE 8090 8091

CMD ["./wave-terminal"]
```

#### 构建和运行
```bash
# 构建Docker镜像
docker build -t wave-terminal-mcp .

# 运行容器
docker run -p 8090:8090 -p 8091:8091 \
  -e WAVETERM_DATA_DIR="/data" \
  -v wave-data:/data \
  wave-terminal-mcp
```

### 2. Docker Compose 部署

#### docker-compose.yml
```yaml
version: '3.8'

services:
  wave-terminal:
    build: .
    ports:
      - "8090:8090"
      - "8091:8091"
    environment:
      - WAVETERM_DATA_DIR=/data
      - WAVETERM_AUTH_KEY=${WAVETERM_AUTH_KEY}
    volumes:
      - wave-data:/data
      - ./logs:/logs
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - wave-terminal
    restart: unless-stopped

volumes:
  wave-data:
```

#### 启动服务
```bash
# 设置认证密钥
export WAVETERM_AUTH_KEY=$(openssl rand -hex 32)

# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f wave-terminal
```

### 3. Kubernetes 部署

#### deployment.yaml
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wave-terminal-mcp
  labels:
    app: wave-terminal
spec:
  replicas: 2
  selector:
    matchLabels:
      app: wave-terminal
  template:
    metadata:
      labels:
        app: wave-terminal
    spec:
      containers:
      - name: wave-terminal
        image: wave-terminal-mcp:latest
        imagePullPolicy: Always
        ports:
        - containerPort: 8090
        - containerPort: 8091
        env:
        - name: WAVETERM_DATA_DIR
          value: "/data"
        - name: WAVETERM_AUTH_KEY
          valueFrom:
            secretKeyRef:
              name: wave-terminal-secret
              key: auth-key
        volumeMounts:
        - name: data-volume
          mountPath: /data
        livenessProbe:
          httpGet:
            path: /api/v1/widgets/workspaces
            port: 8090
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /api/v1/widgets/workspaces
            port: 8090
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: data-volume
        persistentVolumeClaim:
          claimName: wave-terminal-pvc

---
apiVersion: v1
kind: Service
metadata:
  name: wave-terminal-service
spec:
  selector:
    app: wave-terminal
  ports:
  - name: http
    port: 80
    targetPort: 8090
  - name: websocket
    port: 8091
    targetPort: 8091
  type: LoadBalancer

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: wave-terminal-ingress
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - wave-terminal.yourdomain.com
    secretName: wave-terminal-tls
  rules:
  - host: wave-terminal.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: wave-terminal-service
            port:
              number: 80
```

#### 部署到Kubernetes
```bash
# 创建命名空间
kubectl create namespace wave-terminal

# 创建密钥
kubectl create secret generic wave-terminal-secret \
  --from-literal=auth-key=$(openssl rand -hex 32) \
  -n wave-terminal

# 应用配置
kubectl apply -f deployment.yaml -n wave-terminal

# 查看状态
kubectl get pods -n wave-terminal
kubectl logs -f deployment/wave-terminal-mcp -n wave-terminal
```

## 🔐 安全配置

### 1. 启用认证
```bash
# 在 pkg/web/widgetapi.go 中取消注释认证代码
# 第22-25行
if err := authkey.ValidateIncomingRequest(r); err != nil {
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
```

### 2. 生成安全密钥
```bash
# 方法1: OpenSSL
export WAVETERM_AUTH_KEY=$(openssl rand -hex 32)

# 方法2: Python
export WAVETERM_AUTH_KEY=$(python3 -c "import secrets; print(secrets.token_hex(32))")

# 方法3: Go
export WAVETERM_AUTH_KEY=$(go run -c "
package main
import (
    \"crypto/rand\"
    \"fmt\"
    \"encoding/hex\"
)
func main() {
    bytes := make([]byte, 32)
    rand.Read(bytes)
    fmt.Println(hex.EncodeToString(bytes))
}")
```

### 3. HTTPS/TLS 配置
```bash
# 生成自签名证书 (开发用)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# 或使用Let's Encrypt (生产用)
certbot certonly --standalone -d yourdomain.com
```

## 🚀 Claude Code 中的特殊部署

### 1. MCP 服务器集成

如果你想在Claude Code中直接使用这个增强版Wave Terminal作为MCP服务器：

#### 启动配置
```json
{
  "mcpServers": {
    "wave-terminal": {
      "command": "node",
      "args": ["/path/to/waveterm/mcp-bridge.js"],
      "env": {
        "WAVE_TERMINAL_URL": "http://localhost:8090",
        "WAVE_TERMINAL_AUTH_KEY": "your-auth-key"
      }
    }
  }
}
```

#### 创建MCP桥接脚本
```javascript
// mcp-bridge.js
const { MCPServer } = require('@modelcontextprotocol/sdk/server');
const { StdioServerTransport } = require('@modelcontextprotocol/sdk/server/stdio');

class WaveTerminalMCPServer extends MCPServer {
  constructor() {
    super({
      name: "wave-terminal",
      version: "1.0.0"
    });
    
    this.waveTerminalUrl = process.env.WAVE_TERMINAL_URL || "http://localhost:8090";
    this.authKey = process.env.WAVE_TERMINAL_AUTH_KEY;
  }

  async listTools() {
    return [
      {
        name: "create_widget",
        description: "Create a new widget in Wave Terminal",
        inputSchema: {
          type: "object",
          properties: {
            workspace_id: { type: "string" },
            widget_type: { type: "string" },
            title: { type: "string" },
            meta: { type: "object" }
          },
          required: ["workspace_id", "widget_type"]
        }
      },
      {
        name: "list_workspaces",
        description: "List all available workspaces",
        inputSchema: {
          type: "object",
          properties: {}
        }
      }
    ];
  }

  async callTool(name, args) {
    const headers = {};
    if (this.authKey) {
      headers['X-AuthKey'] = this.authKey;
    }

    switch (name) {
      case "create_widget":
        const response = await fetch(`${this.waveTerminalUrl}/api/v1/widgets`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            ...headers
          },
          body: JSON.stringify(args)
        });
        return [{ type: "text", text: await response.text() }];

      case "list_workspaces":
        const workspacesResponse = await fetch(`${this.waveTerminalUrl}/api/v1/widgets/workspaces`, {
          headers
        });
        return [{ type: "text", text: await workspacesResponse.text() }];

      default:
        throw new Error(`Unknown tool: ${name}`);
    }
  }
}

const server = new WaveTerminalMCPServer();
const transport = new StdioServerTransport();
server.connect(transport);
```

### 2. 在Claude Code中使用

一旦部署完成，你可以在Claude Code中这样使用：

```javascript
// 在Claude Code中调用Wave Terminal功能
const workspaces = await callTool("wave-terminal", "list_workspaces", {});
console.log("Available workspaces:", workspaces);

const newWidget = await callTool("wave-terminal", "create_widget", {
  workspace_id: "your-workspace-id",
  widget_type: "terminal",
  title: "AI Assistant Terminal",
  meta: { cwd: "/home/user/projects" }
});
console.log("Created widget:", newWidget);
```

## 📊 监控和维护

### 1. 健康检查
```bash
# 检查API健康状态
curl -f http://localhost:8090/api/v1/widgets/workspaces || exit 1

# 检查WebSocket连接
wscat -c ws://localhost:61270
```

### 2. 日志管理
```bash
# 查看应用日志
tail -f waveterm-server.log

# 日志轮转配置
logrotate -d /etc/logrotate.d/waveterm
```

### 3. 性能监控
```bash
# 监控资源使用
htop -p $(pgrep wave-terminal)

# 监控网络连接
netstat -tulpn | grep :8090
```

## 🔄 更新和升级

### 1. 滚动更新
```bash
# 备份当前数据
cp -r /tmp/waveterm-mcp /tmp/waveterm-mcp.backup.$(date +%Y%m%d)

# 更新代码
git pull origin main

# 重新构建
npm run build:prod
go build -o wave-terminal cmd/server/main-server.go

# 重启服务
./persistent-server.sh restart
```

### 2. 零停机更新 (Kubernetes)
```bash
# 更新镜像
kubectl set image deployment/wave-terminal-mcp \
  wave-terminal=wave-terminal-mcp:new-version \
  -n wave-terminal

# 查看滚动更新状态
kubectl rollout status deployment/wave-terminal-mcp -n wave-terminal
```

## 🛠️ 故障排除

### 常见问题

#### 1. 前端资源未找到
```bash
# 确保前端已构建
npm run build:dev

# 检查dist目录
ls -la dist/frontend/
```

#### 2. MCP服务器连接失败
```bash
# 检查服务器状态
./persistent-server.sh status

# 检查端口占用
lsof -i :8090

# 重启服务器
./persistent-server.sh restart
```

#### 3. 认证失败
```bash
# 检查认证密钥
echo $WAVETERM_AUTH_KEY

# 测试API访问
curl -H "X-AuthKey: $WAVETERM_AUTH_KEY" \
  http://localhost:8090/api/v1/widgets/workspaces
```

## 📞 技术支持

如果遇到部署问题，请检查：

1. **日志文件**: `waveterm-server.log`
2. **端口占用**: `netstat -tulpn | grep 6126`
3. **权限问题**: 确保有数据目录写入权限
4. **依赖版本**: 确认Node.js和Go版本符合要求

这个部署指南涵盖了从开发环境到生产环境的完整部署流程，特别针对MCP功能进行了优化配置。