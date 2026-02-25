# K8s Wizard 🧙

通过自然语言管理 Kubernetes 集群的 AI 助手。

## 项目价值

- 🎯 **降低使用门槛** - 无需学习复杂的 kubectl 命令，支持自然语言（中文/英文）
- 🤖 **AI 驱动** - 支持多种 LLM 模型（GLM、DeepSeek、Claude）
- 📱 **统一架构** - 前后端整合在同一项目中
- ⚡ **核心功能** - 部署、查看、扩缩容、删除等 K8s 操作
- ⚙️ **灵活配置** - 支持 JSON 配置文件和环境变量

## 技术栈

### 前端
- React 18 + TypeScript
- Vite
- Tailwind CSS

### 后端
- Go 1.24
- Gin Web 框架
- client-go (Kubernetes Go 客户端)

## 配置系统（参考 OpenClaw）

K8s Wizard 采用类似 OpenClaw 的配置系统，支持通过 JSON 配置文件或环境变量管理模型设置。

### 配置文件位置

默认配置文件位置（按优先级排序）：
1. `~/.config/k8s-wizard/config.json` (XDG 标准位置)
2. `~/.k8s-wizard/config.json` (传统位置)

### 配置文件示例

```json
{
  "meta": {
    "version": "1.0.0"
  },
  "models": {
    "mode": "merge",
    "providers": {
      "glm": {
        "baseUrl": "https://open.bigmodel.cn/api/paas/v4",
        "auth": "api-key",
        "api": "openai-completions",
        "models": [
          {
            "id": "glm-4-flash",
            "name": "GLM-4 Flash",
            "reasoning": true,
            "contextWindow": 128000,
            "maxTokens": 131072
          }
        ]
      }
    }
  },
  "agents": {
    "defaults": {
      "model": {
        "primary": "glm/glm-4-flash"
      }
    }
  },
  "api": {
    "port": 8080,
    "host": "0.0.0.0"
  }
}
```

### 模型字符串格式

模型采用 `provider/model-id` 格式：
- `glm/glm-4-flash` - GLM-4 Flash
- `deepseek/deepseek-chat` - DeepSeek Chat
- `claude/claude-sonnet-4-20250514` - Claude Sonnet 4

### 环境变量

环境变量可以覆盖配置文件中的设置：

| 变量 | 说明 |
|------|------|
| `GLM_API_KEY` | GLM API Key |
| `DEEPSEEK_API_KEY` | DeepSeek API Key |
| `ANTHROPIC_API_KEY` | Claude API Key |
| `K8S_WIZARD_MODEL` | 覆盖默认模型（如 `deepseek/deepseek-chat`）|
| `PORT` | API 服务端口（默认 8080）|
| `KUBECONFIG` | K8s 配置文件路径 |

## 快速开始

### 前置要求

- Go 1.24+
- Node.js 18+ (用于前端开发)
- Kubernetes 集群
- LLM API Key（GLM、DeepSeek 或 Claude）

### 1. 克隆项目

```bash
cd /home/shawn/github/agents-workplace/k8s-wizard
```

### 2. 配置 API Key（选择一个）

**方式 1：环境变量（推荐）**

```bash
# GLM (智谱）- 国内 API，速度快
export GLM_API_KEY=your-glm-api-key

# 或 DeepSeek - 国内 API，性价比高
export DEEPSEEK_API_KEY=your-deepseek-api-key

# 或 Claude - 需要科学上网
export ANTHROPIC_API_KEY=your-anthropic-api-key
```

**方式 2：凭证文件**

创建 `~/.k8s-wizard/credentials.json`:
```json
{
  "profiles": {
    "glm:default": {
      "apiKey": "your-glm-api-key"
    },
    "deepseek:default": {
      "apiKey": "your-deepseek-api-key"
    },
    "claude:default": {
      "apiKey": "your-anthropic-api-key"
    }
  }
}
```

### 3. 安装依赖

```bash
make install
```

### 4. 运行开发服务器

```bash
# 同时启动前后端（推荐）
make dev

# 或分别启动
make dev:api  # 后端: http://localhost:8080
make dev:web  # 前端: http://localhost:5173
```

访问 http://localhost:5173

## 项目结构

```
k8s-wizard/
├── api/                  # 后端 API (Go)
│   ├── main.go          # API 入口
│   ├── handlers/        # 请求处理器
│   │   ├── chat.go    # 聊天处理
│   │   ├── health.go   # 健康检查
│   │   ├── resources.go # 资源获取
│   │   └── config.go  # 配置信息
│   ├── middleware/      # 中间件
│   │   └── cors.go    # CORS 配置
│   └── models/         # 数据模型
│       └── requests.go # 请求/响应模型
│
├── web/                  # 前端 React
│   ├── src/
│   │   ├── components/   # UI 组件
│   │   ├── pages/        # 页面
│   │   ├── services/     # API 服务
│   │   ├── hooks/        # React Hooks
│   │   └── types/        # TypeScript 类型
│   └── package.json
│
├── pkg/                  # 共享包
│   ├── agent/           # Agent 核心逻辑
│   │   └── agent.go
│   └── config/          # 配置管理
│       └── config.go
│
├── Makefile            # 构建脚本
├── go.mod             # Go 模块配置
└── README.md          # 项目文档
```

## API 端点

| 方法 | 路径 | 说明 |
|------|-------|------|
| GET | `/health` | 健康检查 |
| POST | `/api/chat` | 聊天接口 |
| GET | `/api/resources` | 获取资源列表 |
| GET | `/api/config/model` | 获取当前模型信息 |
| GET | `/api/config` | 获取完整配置 |

## 使用示例

### 通过 Web UI

1. 在输入框输入自然语言指令
2. 例如："部署一个 nginx"
3. 查看返回的执行结果

### 支持的操作

| 操作类型 | 示例指令 |
|---------|---------|
| 部署 | "部署一个 nginx"、"创建一个 redis" |
| 查看 | "查看所有 pod"、"显示所有 deployment" |
| 扩缩容 | "扩容到 5 个副本"、"缩容到 2 个副本" |
| 删除 | "删除名为 test 的 pod"、"移除 nginx" |

## 可用命令

```bash
make dev          # 同时启动前后端
make dev:api     # 仅启动后端
make dev:web     # 仅启动前端
make build       # 构建前后端
make build:api   # 仅构建后端
make build:web   # 仅构建前端
make clean       # 清理构建产物
make install     # 安装所有依赖
```

## 故障排查

### 后端无法启动

确保设置了 API Key 环境变量或创建了凭证文件：
```bash
export GLM_API_KEY=your-api-key
```

### 前端显示"未连接"

检查后端服务是否正常运行在 http://localhost:8080

### 切换模型

修改 `~/.k8s-wizard/config.json` 中的 `agents.defaults.model.primary` 字段：
```json
{
  "agents": {
    "defaults": {
      "model": {
        "primary": "deepseek/deepseek-chat"
      }
    }
  }
}
```

或使用环境变量临时覆盖：
```bash
export K8S_WIZARD_MODEL="deepseek/deepseek-chat"
make dev:api
```

## 许可证

MIT

## 贡献

欢迎提交 Issue 和 Pull Request！
