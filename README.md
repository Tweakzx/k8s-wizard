# K8s Wizard

通过自然语言管理 Kubernetes 集群的 AI 助手。

## 特性

- **自然语言交互** - 用中文或英文描述你想要做的事情，无需记忆 kubectl 命令
- **智能澄清** - 当信息不足时，自动生成表单收集必要信息
- **操作预览** - 执行前预览 YAML 配置，确认后再执行
- **动态模型切换** - 前端实时切换 LLM 模型（GLM、DeepSeek、Claude）
- **安全确认** - 危险操作（如删除）需要明确确认

## 截图

```
┌─────────────────────────────────────────────────────────────┐
│  K8s Wizard                              [glm/glm-4-flash ▼]│
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  用户: 部署一个 nginx 应用                                   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │  📦 创建 Deployment                                   │   │
│  │                                                       │   │
│  │  应用名称: [nginx        ]                            │   │
│  │  镜像地址: [nginx:latest ]                            │   │
│  │  副本数:   [  1  ] [-][+]                             │   │
│  │  命名空间: [default     ]                             │   │
│  │                                                       │   │
│  │                          [取消] [确认]                │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 技术栈

| 层级 | 技术 |
|------|------|
| 前端 | React 18, TypeScript, Vite, Tailwind CSS |
| 后端 | Go 1.24, Gin |
| K8s | client-go |
| LLM | GLM, DeepSeek, Claude |

## 架构设计

K8s Wizard 采用分层架构设计，核心组件包括：

- **意图理解层** - LLM 解析自然语言，提取操作意图
- **安全层** - 风险评估、权限检查、操作确认
- **执行层** - client-go 调用 K8s API
- **结果层** - 格式化输出、错误处理

```
用户输入 → 意图解析 → 风险评估 → 执行操作 → 返回结果
              ↓
         需要澄清? → 生成表单 → 用户填写
              ↓
         危险操作? → 预览确认 → 用户确认
```

> 详细架构设计请参考 [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

## 快速开始

### 前置要求

- Go 1.24+
- Node.js 18+
- Kubernetes 集群（或 kubectl 配置）
- LLM API Key（GLM / DeepSeek / Claude 任选）

### 1. 获取代码

```bash
git clone https://github.com/your-org/k8s-wizard.git
cd k8s-wizard
```

### 2. 配置 API Key

**方式一：环境变量（推荐）**

```bash
# GLM（推荐国内用户）
export GLM_API_KEY=your-glm-api-key

# 或 DeepSeek
export DEEPSEEK_API_KEY=your-deepseek-api-key

# 或 Claude（需要科学上网）
export ANTHROPIC_API_KEY=your-anthropic-api-key
```

**方式二：凭证文件**

创建 `~/.k8s-wizard/credentials.json`：

```json
{
  "profiles": {
    "glm:default": {
      "apiKey": "your-glm-api-key"
    }
  }
}
```

> 详细配置请参考 [docs/LLM_SETUP.md](docs/LLM_SETUP.md)

### 3. 启动服务

```bash
# 安装依赖并启动（前后端同时启动）
make dev
```

- 前端: http://localhost:5173
- 后端: http://localhost:8080

### 4. 使用

打开浏览器访问 http://localhost:5173，输入自然语言指令：

- "部署一个 nginx 应用"
- "查看所有 pod"
- "把 nginx 扩容到 5 个副本"
- "删除 test 命名空间下的 deployment"

## 项目结构

```
k8s-wizard/
├── api/                      # 后端 API
│   ├── main.go               # 入口
│   ├── handlers/             # 请求处理器
│   │   ├── chat.go           # 聊天 + 澄清流程
│   │   ├── config.go         # 模型配置/切换
│   │   ├── resources.go      # K8s 资源查询
│   │   └── health.go         # 健康检查
│   ├── middleware/           # 中间件
│   │   └── cors.go
│   └── models/               # 数据模型
│       └── requests.go
│
├── pkg/                      # 核心包
│   ├── agent/                # Agent 逻辑
│   │   ├── agent.go          # LLM 交互
│   │   └── clarify.go        # 澄清 + 预览
│   └── config/               # 配置管理
│       └── config.go
│
├── web/                      # 前端
│   ├── src/
│   │   ├── components/       # UI 组件
│   │   │   ├── ActionForm.tsx    # 动态表单
│   │   │   ├── ActionPreview.tsx # YAML 预览
│   │   │   ├── ModelSelector.tsx # 模型切换
│   │   │   └── ...
│   │   ├── pages/
│   │   ├── hooks/
│   │   ├── services/
│   │   └── types/
│   └── package.json
│
├── docs/                     # 文档
│   ├── ARCHITECTURE.md      # 架构设计
│   ├── LLM_SETUP.md         # LLM 配置指南
│   └── ROADMAP.md           # 开发路线图
├── Makefile
└── go.mod
```

## API 端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |
| POST | `/api/chat` | 聊天接口（支持澄清流程） |
| GET | `/api/resources` | 获取 K8s 资源列表 |
| GET | `/api/config/model` | 获取当前模型和可用模型 |
| PUT | `/api/config/model` | 切换模型 |
| GET | `/api/config` | 获取完整配置 |

### 聊天接口示例

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"content": "部署一个 nginx"}'
```

**响应（需要澄清）**：

```json
{
  "status": "needs_info",
  "clarification": {
    "type": "form",
    "title": "📦 创建 Deployment",
    "fields": [
      {"key": "name", "label": "应用名称", "type": "text", "required": true},
      {"key": "image", "label": "镜像地址", "type": "text", "required": true},
      {"key": "replicas", "label": "副本数", "type": "number", "default": 1}
    ]
  }
}
```

**响应（需要确认）**：

```json
{
  "status": "needs_confirm",
  "actionPreview": {
    "type": "create",
    "resource": "deployment/nginx",
    "yaml": "apiVersion: apps/v1\nkind: Deployment\n...",
    "dangerLevel": "low",
    "summary": "创建 Deployment nginx (副本: 1, 镜像: nginx:latest)"
  }
}
```

## 支持的操作

| 操作 | 示例指令 |
|------|---------|
| 部署 | "部署一个 nginx"、"创建 redis 应用" |
| 查看 | "查看所有 pod"、"显示 deployment 列表" |
| 扩缩容 | "扩容到 5 个副本"、"nginx 缩容到 2" |
| 删除 | "删除 nginx deployment"、"移除 test pod" |

## 配置

### 配置文件位置

按优先级排序：

1. `~/.config/k8s-wizard/config.json` (XDG 标准)
2. `~/.k8s-wizard/config.json`

### 配置示例

```json
{
  "meta": {"version": "1.0.0"},
  "models": {
    "mode": "merge",
    "providers": {
      "glm": {
        "baseUrl": "https://open.bigmodel.cn/api/coding/paas/v4",
        "auth": "api-key",
        "api": "openai-completions",
        "models": [{"id": "glm-4-flash", "name": "GLM-4 Flash"}]
      }
    }
  },
  "agents": {
    "defaults": {"model": {"primary": "glm/glm-4-flash"}}
  },
  "api": {"port": 8080, "host": "0.0.0.0"}
}
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `GLM_API_KEY` | GLM API Key |
| `DEEPSEEK_API_KEY` | DeepSeek API Key |
| `ANTHROPIC_API_KEY` | Claude API Key |
| `K8S_WIZARD_MODEL` | 覆盖默认模型 |
| `PORT` | API 端口（默认 8080） |
| `KUBECONFIG` | K8s 配置路径 |

## 命令

```bash
make dev          # 启动开发服务器（前后端）
make dev:api      # 仅启动后端
make dev:web      # 仅启动前端
make build        # 构建生产版本
make build:api    # 仅构建后端
make build:web    # 仅构建前端
make run          # 构建并运行后端
make clean        # 清理构建产物
make install      # 安装依赖
make test         # 运行测试
make lint         # 代码检查
```

## 支持的 LLM

| 提供商 | 环境变量 | 获取地址 |
|--------|----------|----------|
| GLM (智谱) | `GLM_API_KEY` | https://open.bigmodel.cn/ |
| DeepSeek | `DEEPSEEK_API_KEY` | https://platform.deepseek.com/ |
| Claude | `ANTHROPIC_API_KEY` | https://console.anthropic.com/ |

> 详细配置请参考 [docs/LLM_SETUP.md](docs/LLM_SETUP.md)

## 故障排查

### 后端无法启动

确保设置了 API Key：

```bash
export GLM_API_KEY=your-api-key
make dev:api
```

### 前端显示"未连接"

检查后端是否运行在 http://localhost:8080

### 模型列表为空

确保配置了对应提供商的 API Key，只有配置了 Key 的提供商才会显示模型。

## 开发路线图

详见 [docs/ROADMAP.md](docs/ROADMAP.md)

- [x] 自然语言解析
- [x] 智能澄清流程
- [x] 动态模型切换
- [x] YAML 预览确认
- [ ] 多轮会话支持
- [ ] 流式响应
- [ ] 更多 K8s 资源

## 许可证

MIT

## 贡献

欢迎提交 Issue 和 Pull Request！
