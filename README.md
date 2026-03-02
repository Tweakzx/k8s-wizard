# K8s Wizard

通过自然语言管理 Kubernetes 集群的 AI 助手。

## 特性

- **自然语言交互** - 用中文或英文描述你想要做的事情，无需记忆 kubectl 命令
- **智能澄清** - 当信息不足时，自动生成表单收集必要信息
- **操作预览** - 执行前预览 YAML 配置，确认后再执行
- **动态模型切换** - 前端实时切换 LLM 模型（GLM、DeepSeek、Claude）
- **安全确认** - 危险操作（如删除）需要明确确认
- **会话持久化** - 支持多轮对话，自动保存会话状态
- **日志落盘** - 完整的日志系统，支持文件输出和日志轮转

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
| 工作流引擎 | langgraphgo |
| 日志 | slog + lumberjack |

## 架构设计

K8s Wizard 采用分层架构设计，核心组件包括：

```
┌─────────────────────────────────────────────────────────────┐
│                        API Layer (Gin)                       │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────┐ │
│  │  Chat   │  │Resources│  │ Config  │  │    Health       │ │
│  │ Handler │  │ Handler │  │ Handler │  │    Handler      │ │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────────┬────────┘ │
└───────┼────────────┼────────────┼─────────────────┼─────────┘
        │            │            │                 │
┌───────┴────────────┴────────────┴─────────────────┴─────────┐
│                      Agent Layer                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                    GraphAgent                          │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  │  │
│  │  │ Process │→ │ Clarify │→ │ Preview │→ │ Execute │  │  │
│  │  │ Intent  │  │  Check  │  │ Generate│  │ Action  │  │  │
│  │  └─────────┘  └─────────┘  └─────────┘  └─────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
        │                              │
        ▼                              ▼
┌─────────────────┐          ┌─────────────────┐
│   LLM Client    │          │   K8s Client    │
│ (GLM/DeepSeek)  │          │   (client-go)   │
└─────────────────┘          └─────────────────┘
```

### 核心包结构

```
pkg/
├── agent/           # Agent 核心
│   ├── agent.go     # GraphAgent 实现 + 接口定义
│   └── ...          # 工厂函数、Checkpoint 支持
│
├── workflow/        # 工作流引擎
│   ├── state.go     # 状态定义 (AgentState, K8sAction)
│   ├── nodes.go     # 节点工厂 (Parse, Clarify, Preview, Execute)
│   ├── routing.go   # 路由函数
│   ├── graph.go     # 图构建器
│   └── checkpointer.go  # 会话持久化
│
├── llm/             # LLM 客户端
│   └── client.go    # OpenAI/Anthropic 兼容 API
│
├── config/          # 配置管理
│   └── config.go    # 配置加载、模型发现
│
└── logger/          # 日志系统
    └── logger.go    # 结构化日志 + 文件轮转
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
│   ├── agent/                # Agent 实现
│   │   └── agent.go          # GraphAgent + 接口 + 工厂
│   ├── workflow/             # 工作流引擎
│   │   ├── state.go          # 状态定义
│   │   ├── nodes.go          # 节点实现
│   │   ├── routing.go        # 路由逻辑
│   │   ├── graph.go          # 图构建
│   │   └── checkpointer.go   # 会话持久化
│   ├── llm/                  # LLM 客户端
│   │   └── client.go
│   ├── config/               # 配置管理
│   │   └── config.go
│   └── logger/               # 日志系统
│       └── logger.go
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
  "api": {"port": 8080, "host": "0.0.0.0"},
  "log": {
    "enableFile": true,
    "filePath": "",
    "maxSize": 100,
    "maxBackups": 3,
    "maxAge": 30,
    "compress": true,
    "level": "info",
    "format": "json",
    "console": true
  }
}
```

### 日志配置

| 字段 | 说明 | 默认值 |
|------|------|--------|
| `enableFile` | 启用文件日志 | `true` |
| `filePath` | 日志文件路径 | `~/.k8s-wizard/logs/k8s-wizard.log` |
| `maxSize` | 单文件最大 MB | `100` |
| `maxBackups` | 保留旧文件数 | `3` |
| `maxAge` | 保留天数 | `30` |
| `compress` | 压缩旧文件 | `true` |
| `level` | 日志级别 | `info` |
| `format` | 输出格式 | `json` |
| `console` | 同时输出到控制台 | `true` |

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

### 日志文件位置

默认日志文件位于 `~/.k8s-wizard/logs/k8s-wizard.log`

```bash
# 查看日志
tail -f ~/.k8s-wizard/logs/k8s-wizard.log

# 查看最近 100 行
tail -100 ~/.k8s-wizard/logs/k8s-wizard.log
```

## 开发路线图

详见 [docs/ROADMAP.md](docs/ROADMAP.md)

- [x] 自然语言解析
- [x] 智能澄清流程
- [x] 动态模型切换
- [x] YAML 预览确认
- [x] 会话持久化（SQLite Checkpointer）
- [x] 结构化日志系统
- [ ] 多轮会话支持
- [ ] 流式响应
- [ ] 更多 K8s 资源

## 测试

项目包含完整的单元测试：

```bash
# 运行所有测试
make test

# 运行测试并查看覆盖率
go test ./... -coverprofile=cover.out
go tool cover -html=cover.out
```

### 测试覆盖

| 包 | 覆盖率 |
|----|--------|
| `pkg/agent` | 46%+ |
| `pkg/config` | 49%+ |
| `pkg/llm` | 84%+ |
| `pkg/workflow` | 61%+ |

## 许可证

MIT

## 贡献

欢迎提交 Issue 和 Pull Request！
