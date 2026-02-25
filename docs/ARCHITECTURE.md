# K8s Wizard 架构设计

本文档描述 K8s Agent 的设计思路和架构规划。

## 目录

- [设计原则](#设计原则)
- [整体架构](#整体架构)
- [核心组件](#核心组件)
- [数据流](#数据流)
- [安全设计](#安全设计)
- [扩展性设计](#扩展性设计)
- [技术选型](#技术选型)

---

## 设计原则

### 1. Safety First（安全优先）

- 所有写操作默认 dry-run，需要用户确认
- 危险操作（删除、强制重启等）需要二次确认
- 支持操作回滚

### 2. Human-in-the-Loop（人在回路）

- LLM 是 Copilot 不是 Autopilot
- AI 提供建议和预览，人类做最终决策
- 关键操作需要人工审批

### 3. Graceful Degradation（优雅降级）

- LLM 失败时回退到规则引擎
- 网络异常时支持离线模式
- 部分功能不可用不影响核心功能

### 4. Auditability（可审计性）

- 所有操作记录审计日志
- 支持操作回放和追踪
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
│                      网关层 (Gateway Layer)                           │
│   • 认证授权 (RBAC)    • 限流控制    • 审计日志    • 请求路由         │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                   意图理解层 (Intent Understanding)                    │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│   │ LLM Parser  │→ │ Validator   │→ │ Disambiguator│                 │
│   │  意图解析   │  │  参数验证   │  │  模糊消歧   │                  │
│   └─────────────┘  └─────────────┘  └─────────────┘                 │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                     规划层 (Planning Layer)                           │
│   • 生成操作计划    • 风险评估    • 依赖分析    • 影响范围计算         │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                     安全层 (Safety Layer)                             │
│   • Dry-run 预览    • 权限检查    • 破坏性操作确认    • 审批流程       │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                     执行层 (Execution Layer)                          │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │
│   │  client-go  │  │    Helm     │  │   kubectl   │                 │
│   │  原生 API   │  │  Chart 管理 │  │  命令行     │                  │
│   └─────────────┘  └─────────────┘  └─────────────┘                 │
└───────────────────────────────┬─────────────────────────────────────┘
                                │
┌───────────────────────────────▼─────────────────────────────────────┐
│                     结果层 (Result Layer)                             │
│   • 结果格式化    • 错误恢复    • 状态跟踪    • 回滚支持              │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 核心组件

### 1. 意图理解器 (Intent Parser)

负责将自然语言转换为结构化的操作意图。

```go
type Intent struct {
    // 核心信息
    Action      string            // create, get, update, delete, scale, describe...
    Resource    string            // pod, deployment, service, configmap...
    Name        string            // 资源名称
    Namespace   string            // 命名空间（空=所有命名空间）
    Params      map[string]any    // 操作参数
    
    // 元信息
    Confidence  float64           // 置信度 0.0-1.0
    Ambiguities []string          // 模糊点列表，需要澄清
    RawQuery    string            // 原始用户输入
    
    // 上下文
    SessionID   string            // 会话 ID
    Context     *ConversationContext
    
    // 安全信息
    RiskLevel   RiskLevel         // low, medium, high, critical
    Reversible  bool              // 是否可逆
    ImpactScope ImpactScope       // 影响范围
}
```

**工作流程**：

```
用户输入 → LLM 解析 → 置信度评估 → 模糊检测 → 结构化 Intent
                ↓
         置信度 < 0.7? → 触发澄清流程
```

### 2. 澄清管理器 (Clarification Manager)

当意图不明确时，收集必要信息。

```go
type ClarificationRequest struct {
    Type        string              // form, options, confirm
    Title       string
    Description string
    Fields      []ClarificationField
    Timeout     time.Duration
}

type ClarificationField struct {
    Key         string
    Label       string
    Type        string              // text, number, select, multiselect
    Required    bool
    Default     interface{}
    Options     []Option
    Validator   ValidatorFunc
}
```

**澄清策略**：

| 场景 | 策略 |
|------|------|
| 缺少必要参数 | 动态生成表单 |
| 多个匹配资源 | 展示选项列表 |
| 危险操作确认 | 二次确认弹窗 |
| 模糊指令 | 提供操作建议 |

### 3. 操作规划器 (Action Planner)

将意图转换为可执行的操作计划。

```go
type ActionPlan struct {
    ID          string
    Intent      *Intent
    Steps       []ExecutionStep
    Dependencies []string           // 依赖的其他操作
    RollbackPlan []RollbackStep    // 回滚计划
    
    // 风险评估
    RiskAssessment *RiskAssessment
    
    // 预览信息
    YAMLPreview   string
    DiffPreview   string
    Summary       string
}

type ExecutionStep struct {
    Type        string              // k8s_api, helm, kubectl
    Action      string
    Resource    string
    Params      map[string]any
    DryRun      bool
}
```

### 4. 安全控制器 (Safety Controller)

确保所有操作符合安全策略。

```go
type SafetyPolicy struct {
    // 操作分级
    OperationLevels map[string]RiskLevel
    
    // 命名空间限制
    AllowedNamespaces   []string
    ProtectedNamespaces []string    // 禁止操作的命名空间
    
    // 资源保护
    ProtectedResources []ResourcePattern
    
    // 确认策略
    RequireConfirmation map[RiskLevel]bool
    RequireApproval     map[RiskLevel]bool
    
    // 限流
    RateLimits map[string]RateLimit
}

type RiskAssessment struct {
    Level           RiskLevel
    Factors         []RiskFactor
    ImpactResources []string
    Reversible      bool
    RollbackComplexity string
}
```

**风险等级**：

| 等级 | 操作类型 | 确认要求 |
|------|---------|---------|
| Low | get, list, describe | 无 |
| Medium | create, scale | 预览确认 |
| High | update, delete | 二次确认 |
| Critical | 强制删除、批量操作 | 审批流程 |

### 5. 执行引擎 (Execution Engine)

执行操作计划并管理状态。

```go
type Executor interface {
    // 预览（dry-run）
    Preview(ctx context.Context, plan *ActionPlan) (*PreviewResult, error)
    
    // 执行
    Execute(ctx context.Context, plan *ActionPlan) (*ExecutionResult, error)
    
    // 状态查询
    Status(ctx context.Context, executionID string) (*ExecutionStatus, error)
    
    // 回滚
    Rollback(ctx context.Context, executionID string) error
    
    // 取消
    Cancel(ctx context.Context, executionID string) error
}

type ExecutionResult struct {
    ID          string
    Status      ExecutionStatus    // pending, running, succeeded, failed
    Steps       []StepResult
    StartTime   time.Time
    EndTime     time.Time
    Error       error
    Output      string
    Resources   []ResourceRef      // 受影响的资源
}
```

### 6. 会话管理器 (Session Manager)

管理对话上下文和状态。

```go
type Session struct {
    ID           string
    UserID       string
    CreatedAt    time.Time
    ExpiresAt    time.Time
    
    // 对话历史
    Messages     []Message
    
    // 工作上下文
    CurrentNamespace string
    RecentResources  []ResourceRef
    LastIntent       *Intent
    
    // 状态
    State        SessionState       // active, waiting_input, executing
    PendingAction *ActionPlan
}

// 上下文感知
func (s *Session) ResolveReference(ref string) *ResourceRef {
    // "它" → 上次提到的资源
    // "那个 namespace" → 上次操作的命名空间
    // "刚才的 deployment" → 最近操作的 deployment
}
```

---

## 数据流

### 查询操作流程

```
用户: "查看所有 pod"
    │
    ▼
┌─────────────────┐
│  Intent Parser  │ → Intent{Action: "get", Resource: "pod", Namespace: ""}
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Validator    │ → 检查权限、参数有效性
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Safety Checker  │ → Risk: Low, 无需确认
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│    Executor     │ → client-go: Pods.List(all namespaces)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Result Handler  │ → 格式化输出，添加命名空间标签
└─────────────────┘
         │
         ▼
响应: "📦 集群中的 Pod (共 11 个):
      • [kube-system] coredns-xxx (Running)
      • [default] nginx-xxx (Running)
      ..."
```

### 创建操作流程

```
用户: "部署一个 nginx"
    │
    ▼
┌─────────────────┐
│  Intent Parser  │ → Intent{Action: "create", Resource: "deployment", 
│                   │         Name: "nginx", Params: 缺少 replicas, image}
└────────┬────────┘
         │ 置信度低，触发澄清
         ▼
┌─────────────────┐
│ Clarification   │ → 生成表单，收集缺失信息
└────────┬────────┘
         │
         ▼
用户填写表单 → Intent 更新完整
         │
         ▼
┌─────────────────┐
│  Action Planner │ → 生成 ActionPlan + YAML 预览
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Safety Checker  │ → Risk: Medium, 需要确认
└────────┬────────┘
         │
         ▼
用户确认 → 执行
         │
         ▼
┌─────────────────┐
│    Executor     │ → client-go: Deployments.Create()
└────────┬────────┘
         │
         ▼
响应: "✓ 已创建 Deployment default/nginx"
```

---

## 安全设计

### 1. 认证授权

```go
// 用户身份
type UserIdentity struct {
    ID          string
    Name        string
    Roles       []string
    Permissions []Permission
}

// RBAC 权限检查
func CheckPermission(user *UserIdentity, intent *Intent) error {
    // 1. 检查操作权限
    // 2. 检查命名空间权限
    // 3. 检查资源权限
}
```

### 2. 操作审计

```go
type AuditLog struct {
    ID          string
    Timestamp   time.Time
    UserID      string
    SessionID   string
    
    // 操作信息
    Intent      *Intent
    ActionPlan  *ActionPlan
    
    // 结果
    Status      string
    Error       string
    
    // 资源变更
    Changes     []ResourceChange
}

// 所有写操作记录审计日志
func Audit(intent *Intent, result *ExecutionResult) {
    // 1. 记录操作详情
    // 2. 记录资源变更
    // 3. 存储到持久化存储
}
```

### 3. 敏感信息保护

```go
// 敏感信息检测
func DetectSecrets(params map[string]any) []string {
    // 检测 API Key、密码、Token 等
}

// 敏感信息脱敏
func MaskSecrets(output string) string {
    // 替换敏感信息为 ***
}
```

---

## 扩展性设计

### 1. 插件化资源支持

```go
// 资源处理器接口
type ResourceHandler interface {
    // 支持的资源类型
    Types() []string
    
    // 解析意图
    ParseIntent(intent *Intent) error
    
    // 生成操作计划
    Plan(intent *Intent) (*ActionPlan, error)
    
    // 执行操作
    Execute(ctx context.Context, plan *ActionPlan) (*ExecutionResult, error)
    
    // 格式化结果
    Format(result *ExecutionResult) string
}

// 资源处理器注册
func RegisterHandler(handler ResourceHandler) {
    // 注册到处理器映射表
}
```

### 2. 多集群支持

```go
type ClusterManager struct {
    clusters map[string]*ClusterConnection
}

type ClusterConnection struct {
    Name       string
    Config     *rest.Config
    Client     *kubernetes.Clientset
    
    // 集群信息
    Version    string
    NodeCount  int
}

// 多集群操作
func ExecuteOnCluster(clusterName string, intent *Intent) (*ExecutionResult, error) {
    // 1. 获取集群连接
    // 2. 执行操作
    // 3. 返回结果
}
```

### 3. LLM 提供商抽象

```go
type LLMProvider interface {
    // 解析意图
    ParseIntent(ctx context.Context, query string, session *Session) (*Intent, error)
    
    // 生成回复
    GenerateReply(ctx context.Context, prompt string) (string, error)
    
    // 模型信息
    ModelName() string
    ProviderName() string
}

// 支持的提供商
// - GLM (智谱)
// - DeepSeek
// - Claude (Anthropic)
// - OpenAI
// - 本地模型 (Ollama)
```

---

## 技术选型

| 层级 | 技术选择 | 说明 |
|------|---------|------|
| **前端** | React 18 + TypeScript + Tailwind | 响应式 UI，组件化开发 |
| **后端** | Go 1.24 + Gin | 高性能，原生 K8s 支持 |
| **K8s 客户端** | client-go | 官方 Go 客户端 |
| **LLM** | GLM / DeepSeek / Claude | 国内友好，多模型支持 |
| **会话存储** | Redis | 高性能，支持过期 |
| **审计存储** | PostgreSQL | 可靠的关系型存储 |
| **部署** | Helm + K8s Operator | 云原生部署 |

---

## 未来规划

### 短期 (v0.2.0)

- [ ] 多轮会话支持
- [ ] 流式响应（打字机效果）
- [ ] Markdown 渲染
- [ ] 更多资源类型（CRD 支持）

### 中期 (v0.3.0)

- [ ] 多集群管理
- [ ] Helm Chart 支持
- [ ] 操作回滚
- [ ] 审批流程

### 长期 (v1.0.0)

- [ ] 自然语言生成 K8s YAML
- [ ] 智能故障诊断
- [ ] 自动化运维建议
- [ ] VS Code 插件

---

## 参考资料

- [Kubernetes API Reference](https://kubernetes.io/docs/reference/kubernetes-api/)
- [client-go Documentation](https://github.com/kubernetes/client-go)
- [LLM Prompt Engineering Best Practices](https://platform.openai.com/docs/guides/prompt-engineering)
