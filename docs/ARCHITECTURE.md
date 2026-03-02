# K8s Wizard 架构设计

本文档描述 K8s Agent 的设计思路和架构规划。

## 目录

- [设计原则](#设计原则)
- [整体架构](#整体架构)
- [核心组件](#核心组件)
- [包结构](#包结构)
- [工作流引擎](#工作流引擎)
- [数据流](#数据流)
- [安全设计](#安全设计)
- [扩展性设计](#扩展性设计)
- [日志系统](#日志系统)
- [技术选型](#技术选型)

---

## 设计原则

### 1. Safety First（安全优先）

- 所有写操作默认需要确认，用户可预览 YAML
- 危险操作（删除、强制重启等）需要二次确认
- 支持操作回滚

### 2. Human-in-the-Loop（人在回路）

- LLM 是 Copilot 不是 Autopilot
- AI 提供建议和预览，人类做最终决策
- 关键操作需要人工审批

### 3. Graceful Degradation（优雅降级）

- LLM 失败时返回错误信息
- 部分功能不可用不影响核心功能

### 4. Auditability（可审计性）

- 所有操作记录日志
- 支持日志落盘和轮转
- 变更历史可查询

---

## 整体架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                           用户层 (User Layer)                         │
│   CLI / Web UI / Slack Bot / IDE Plugin / API Gateway               │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                      API 层 (Gin Framework)                           │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│   │ ChatHandler │  │ ConfigHandler│  │ResourcesHdlr│                 │
│   └──────┬──────┘  └──────┬──────┘  └──────┬──────┘                 │
│          │                │                │                         │
│   ┌──────▼────────────────▼────────────────▼──────┐                 │
│   │              Middleware (CORS, Logging)        │                 │
│   └────────────────────────────────────────────────┘                 │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                      Agent 层 (GraphAgent)                            │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │                    Workflow Engine (langgraphgo)              │   │
│   │  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐         │   │
│   │  │ Parse   │→ │ Merge   │→ │ Check   │→ │ Preview │         │   │
│   │  │ Intent  │  │ Form    │  │ Clarify │  │ Generate│         │   │
│   │  └─────────┘  └─────────┘  └─────────┘  └─────────┘         │   │
│   │        │                                   │                  │   │
│   │        └───────────────────────────────────┘                  │   │
│   │                          │                                    │   │
│   │  ┌─────────┐              │                                   │   │
│   │  │ Execute │←─────────────┘                                   │   │
│   │  │ Action  │                                                  │   │
│   │  └─────────┘                                                  │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                                                                       │
│   ┌────────────────────┐  ┌────────────────────┐                    │
│   │  LLM Client        │  │  K8s Client        │                    │
│   │  (GLM/DeepSeek/etc)│  │  (client-go)       │                    │
│   └────────────────────┘  └────────────────────┘                    │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                      基础设施层 (Infrastructure)                       │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│   │   Logger    │  │  Config     │  │ Checkpointer│                 │
│   │ (slog+lumberjack) │(JSON File)│  │  (SQLite)   │                 │
│   └─────────────┘  └─────────────┘  └─────────────┘                 │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 核心组件

### 1. GraphAgent (pkg/agent)

核心 Agent 实现，封装工作流引擎提供统一接口。

```go
type GraphAgent struct {
    graph     *lgg.StateRunnable[workflow.AgentState]
    deps      *workflow.Dependencies
    mu        sync.RWMutex
    modelName string
}

// 主要方法
func (a *GraphAgent) ProcessCommand(ctx context.Context, userMsg string) (string, error)
func (a *GraphAgent) ProcessCommandWithClarification(...) (result, clarification, preview, error)
func (a *GraphAgent) SetModel(modelName string) error
func (a *GraphAgent) GetModelName() string
```

### 2. Workflow Engine (pkg/workflow)

基于 langgraphgo 的工作流引擎，定义状态流转。

#### 状态定义 (state.go)

```go
type AgentState struct {
    // 输入
    UserMessage string
    FormData    map[string]interface{}
    Confirm     *bool
    ThreadID    string

    // 解析结果
    Action         *K8sAction
    IsK8sOperation bool

    // 澄清
    ClarificationRequest *models.ClarificationRequest
    NeedsClarification   bool

    // 预览
    ActionPreview *models.ActionPreview

    // 执行
    Result string
    Error  error

    // 状态
    Status string // pending, needs_info, needs_confirm, executed, error, chat
}
```

#### 节点实现 (nodes.go)

```go
// 节点工厂函数
func MakeParseIntentNode(llmClient llm.Client) NodeFunc
func MakeMergeFormNode() NodeFunc
func MakeCheckClarifyNode() NodeFunc
func MakeGeneratePreviewNode() NodeFunc
func MakeExecuteNode(client *kubernetes.Clientset) NodeFunc
```

#### 路由函数 (routing.go)

```go
func RouteAfterParse(ctx context.Context, state AgentState) string
func RouteAfterClarify(ctx context.Context, state AgentState) string
func RouteAfterPreview(ctx context.Context, state AgentState) string
```

### 3. LLM Client (pkg/llm)

统一的 LLM 客户端接口，支持多种提供商。

```go
type Client interface {
    Chat(ctx context.Context, prompt string) (string, error)
    GetModel() string
}

// 支持的 API 格式
// - OpenAI Completions (GLM, DeepSeek)
// - Anthropic (Claude)
```

### 4. Checkpointer (pkg/workflow)

会话持久化，基于 SQLite 实现。

```go
type CheckpointerManager struct {
    store   *sqlite.SqliteCheckpointStore
    dataDir string
}

func (m *CheckpointerManager) GetStore() lgg.CheckpointStore
func (m *CheckpointerManager) ClearSession(ctx context.Context, threadID string) error
```

---

## 包结构

```
pkg/
├── agent/                    # Agent 实现
│   └── agent.go              # GraphAgent, GraphAgentWithCheckpointer
│
├── workflow/                 # 工作流引擎
│   ├── state.go              # AgentState, K8sAction, Dependencies
│   ├── nodes.go              # 节点工厂函数
│   ├── routing.go            # 路由决策函数
│   ├── graph.go              # 图构建器
│   └── checkpointer.go       # 会话持久化
│
├── llm/                      # LLM 客户端
│   └── client.go             # OpenAI/Anthropic 兼容客户端
│
├── config/                   # 配置管理
│   └── config.go             # 配置加载、模型发现
│
└── logger/                   # 日志系统
    └── logger.go             # 结构化日志、文件轮转
```

---

## 工作流引擎

### 节点图

```
                    ┌─────────────────┐
                    │  parse_intent   │
                    │  (解析意图)      │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
              ▼              │              ▼
        ┌─────────┐          │        ┌─────────┐
        │  END    │          │        │ END     │
        │ (Chat)  │          │        │ (Error) │
        └─────────┘          │        └─────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             │
        ┌───────────┐                       │
        │ merge_form│                       │
        │ (合并表单) │                       │
        └─────┬─────┘                       │
              │                             │
              ▼                             │
        ┌─────────────┐                     │
        │check_clarify│                     │
        │ (检查澄清)   │                     │
        └──────┬──────┘                     │
               │                            │
     ┌─────────┴─────────┐                  │
     │                   │                  │
     ▼                   ▼                  │
┌─────────┐        ┌───────────────┐        │
│   END   │        │generate_preview│        │
│(需澄清) │        │  (生成预览)    │        │
└─────────┘        └───────┬───────┘        │
                           │                │
                 ┌─────────┴─────────┐      │
                 │                   │      │
                 ▼                   ▼      │
           ┌─────────┐          ┌─────────┐ │
           │ execute │          │   END   │ │
           │ (执行)  │          │(需确认) │ │
           └────┬────┘          └─────────┘ │
                │                           │
                ▼                           │
           ┌─────────┐                      │
           │   END   │◄─────────────────────┘
           │ (完成)  │
           └─────────┘
```

### 状态流转

```go
// 状态常量
const (
    StatusPending      = "pending"       // 初始状态
    StatusNeedsInfo    = "needs_info"    // 需要澄清
    StatusNeedsConfirm = "needs_confirm" // 需要确认
    StatusExecuted     = "executed"      // 已执行
    StatusError        = "error"         // 错误
    StatusChat         = "chat"          // 闲聊
)
```

---

## 数据流

### 创建操作流程

```
用户: "部署一个 nginx"
    │
    ▼
┌─────────────────┐
│  Parse Intent   │ → K8sAction{Action: "create", Resource: "deployment", 
│                   │         Name: "nginx", Params: 缺少 replicas, image}
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Merge Form    │ → 如果有 FormData，合并到 Action
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Check Clarify  │ → 检查是否需要澄清（缺少必填字段）
└────────┬────────┘
         │ 需要澄清
         ▼
返回 ClarificationRequest (表单)
         │
         │ 用户填写表单
         ▼
┌─────────────────┐
│ Generate Preview│ → 生成 YAML 预览和危险等级
└────────┬────────┘
         │
         ▼
返回 ActionPreview (等待确认)
         │
         │ 用户确认
         ▼
┌─────────────────┐
│     Execute     │ → client-go: Deployments.Create()
└────────┬────────┘
         │
         ▼
响应: "✓ 已创建 Deployment default/nginx"
```

---

## 安全设计

### 风险等级

| 等级 | 操作类型 | 确认要求 |
|------|---------|---------|
| Low | get, list, describe | 无 |
| Medium | create, scale | 预览确认 |
| High | update, delete | 二次确认 |

### 实现方式

```go
// 生成预览时设置危险等级
func generateActionPreview(action *K8sAction) *models.ActionPreview {
    switch action.Action {
    case "create":
        return &ActionPreview{DangerLevel: "low", ...}
    case "scale":
        return &ActionPreview{DangerLevel: "medium", ...}
    case "delete":
        return &ActionPreview{DangerLevel: "high", ...}
    }
}
```

---

## 扩展性设计

### 1. 插件化节点

```go
// 节点工厂函数签名
type NodeFunc func(ctx context.Context, state AgentState) (AgentState, error)

// 自定义节点
func MakeCustomNode(deps *Dependencies) NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        // 自定义逻辑
        return state, nil
    }
}

// 添加到图
g.AddNode("custom_node", "Custom Node", MakeCustomNode(deps))
```

### 2. 多 LLM 提供商

```go
// LLM 接口
type Client interface {
    Chat(ctx context.Context, prompt string) (string, error)
    GetModel() string
}

// 支持的提供商
// - GLM (智谱)
// - DeepSeek
// - Claude (Anthropic)
// - 可扩展其他 OpenAI 兼容 API
```

### 3. 多集群支持（规划中）

```go
type ClusterManager struct {
    clusters map[string]*ClusterConnection
}
```

---

## 日志系统

### 架构

```
┌─────────────────────────────────────────────────────┐
│                   Logger Package                      │
│  ┌──────────────────────────────────────────────┐   │
│  │              slog.Logger                       │   │
│  │  ┌─────────────┐  ┌───────────────────────┐  │   │
│  │  │ JSON Handler│  │   MultiWriter         │  │   │
│  │  └─────────────┘  │  ┌─────────┐ ┌──────┐│  │   │
│  │                   │  │ Console │ │ File ││  │   │
│  │                   │  └─────────┘ └──────┘│  │   │
│  │                   └───────────────────────┘  │   │
│  └──────────────────────────────────────────────┘   │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │           Lumberjack (Log Rotation)           │   │
│  │  • MaxSize: 100MB                             │   │
│  │  • MaxBackups: 3                              │   │
│  │  • MaxAge: 30 days                            │   │
│  │  • Compress: true                             │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

### 使用方式

```go
// 初始化
log, _ := logger.Init(&logger.Config{
    EnableFile: true,
    FilePath:   "~/.k8s-wizard/logs/k8s-wizard.log",
    Level:      "info",
    Format:     "json",
    Console:    true,
})
defer log.Close()

// 记录日志
logger.Info("操作完成", "action", "create", "resource", "deployment/nginx")
logger.Error("操作失败", "error", err, "action", "delete")
```

### 日志格式

```json
{
  "time": "2025-03-01T17:30:00+08:00",
  "level": "INFO",
  "msg": "操作完成",
  "action": "create",
  "resource": "deployment/nginx"
}
```

---

## 技术选型

| 层级 | 技术选择 | 说明 |
|------|---------|------|
| **前端** | React 18 + TypeScript + Tailwind | 响应式 UI，组件化开发 |
| **后端** | Go 1.24 + Gin | 高性能，原生 K8s 支持 |
| **K8s 客户端** | client-go | 官方 Go 客户端 |
| **LLM** | GLM / DeepSeek / Claude | 国内友好，多模型支持 |
| **工作流引擎** | langgraphgo | 状态图工作流 |
| **会话存储** | SQLite | 嵌入式数据库，无需额外依赖 |
| **日志** | slog + lumberjack | 结构化日志，自动轮转 |
| **部署** | Helm + K8s | 云原生部署 |

---

## 测试覆盖

| 包 | 测试文件 | 覆盖率 |
|----|---------|--------|
| `pkg/agent` | `agent_test.go` | 46%+ |
| `pkg/workflow` | `*_test.go` | 61%+ |
| `pkg/config` | `config_test.go` | 49%+ |
| `pkg/llm` | `client_test.go` | 84%+ |

---

## 参考资料

- [Kubernetes API Reference](https://kubernetes.io/docs/reference/kubernetes-api/)
- [client-go Documentation](https://github.com/kubernetes/client-go)
- [langgraphgo](https://github.com/smallnest/langgraphgo)
- [LLM Prompt Engineering Best Practices](https://platform.openai.com/docs/guides/prompt-engineering)
