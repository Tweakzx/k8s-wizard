# Agent Extensibility Design

## Overview

This document describes a comprehensive design for improving the extensibility of the K8s Wizard agent, enabling incremental addition of new K8s resources, operations, and complex workflows while maintaining backward compatibility.

**Date**: 2026-03-13
**Version**: 1.0
**Status**: Draft

---

## Background

K8s Wizard is a Kubernetes-native AI assistant built with Go and LangGraphGo. The current implementation has:

- **~6,800 lines** of Go code across `pkg/agent`, `pkg/workflow`, `pkg/k8s`, `pkg/llm`
- **Fixed workflow graph** with hardcoded nodes for specific operations
- **Embedded prompts** in Go source code
- **Switch-based routing** scattered across multiple files
- **Limited resource support** (pod, deployment, service, namespace, node)
- **Roadmap** with ambitious plans for logs, exec, describe, multi-cluster, templates, and more

**Problem**: Adding new features requires modifying multiple files, making incremental development difficult for a solo developer.

**Goal**: Enable incremental extensibility while maintaining stability and keeping the design practical for one-person development.

---

## Approach Selection

### Chosen Approach: Evolutionary Refactoring

After evaluating three approaches (Evolutionary Refactoring, Tool/Plugin Architecture, Modular Component System), we selected **Evolutionary Refactoring** for the following reasons:

1. **Incremental** - Can implement features one at a time without rewriting everything first
2. **Practical** - Addresses real pain points with modest investment
3. **Maintainable** - Clear structure without excessive abstraction
4. **Solo-developer friendly** - Fits a single developer's pace and capabilities
5. **Future-proof** - Sets foundation for evolving to a full plugin system if needed later

### Alternative Approaches (Not Selected)

| Approach | Pros | Cons | Why Not Selected |
|-----------|--------|--------|-----------------|
| **Tool/Plugin Architecture** | Clean separation, dynamic tool discovery | Significant upfront refactoring, higher complexity overhead | Too complex for solo developer |
| **Modular Component System** | Maximum flexibility, swappable implementations | Highest complexity, steep learning curve, more boilerplate | Overkill for current needs |

---

## Design

### 1. Architecture Overview

**Core Design Principles**:

1. **Add, Don't Replace** - New abstractions complement existing code
2. **Interface-Based** - Components communicate through well-defined interfaces
3. **Incremental** - Each feature can be added independently
4. **Zero Breaking Changes** - Existing functionality continues working

**High-Level Architecture**:

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Agent Layer                                │
│   ┌─────────────────────────────────────────────────────────┐     │
│   │              Workflow Engine (langgraphgo)                │     │
│   │  ┌─────────┐  ┌─────────┐  ┌─────────────────┐   │     │
│   │  │ Intent   │→ │ Clarify │→ │   Tool Router   │   │     │
│   │  │  Node    │  │  Node   │  │  (NEW - Phase 1)│   │     │
│   │  └─────────┘  └─────────┘  └────────┬────────┘   │     │
│   │                                 │                    │     │
│   │  ┌────────────────────────────────┼─────────────────┐   │     │
│   │  │         Tool Registry        │                 │   │     │
│   │  │  ┌────────────────────────┐ │                 │   │     │
│   │  │  │ DeploymentHandler     │ │                 │   │     │
│   │  │  │ PodHandler           │ │                 │   │     │
│   │  │  │ LogsHandler         │ │                 │   │     │
│   │  │  │ ExecHandler         │ │                 │   │     │
│   │  │  │ ...                 │ │                 │   │     │
│   │  │  └────────────────────────┘ │                 │   │     │
│   │  └────────────────────────────────┘                 │     │
│   └─────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────┘
```

**Key Components**:

| Component | Responsibility | Phase |
|-----------|-----------------|--------|
| **Tool Interface** | Uniform abstraction for all K8s operations | 1 |
| **Tool Registry** | Dynamic discovery and routing to handlers | 1 |
| **Resource Handlers** | Implementation per K8s resource type | 2 |
| **Prompt Templates** | Externalized prompt management | 2 |
| **Sub-graphs** | Complex multi-step workflows | 3 |

### 2. Tool Interface & Registry

#### Tool Interface

```go
// pkg/tools/tool.go

package tools

import "context"

// Tool represents a discrete operation that the agent can perform.
type Tool interface {
    // Name returns the unique identifier for this tool.
    Name() string

    // Description explains what this tool does (used by LLM).
    Description() string

    // Category groups related tools (e.g., "k8s", "llm", "builtin").
    Category() string

    // Parameters describes the expected inputs for LLM prompting.
    Parameters() []Parameter

    // DangerLevel indicates the risk level of this operation.
    DangerLevel() DangerLevel

    // Execute runs the tool with the given arguments.
    Execute(ctx context.Context, args map[string]interface{}) (Result, error)
}

// Parameter describes a tool input parameter.
type Parameter struct {
    Name        string      `json:"name"`
    Type        string      `json:"type"`        // string, number, boolean, object, array
    Description string      `json:"description"`
    Required    bool        `json:"required"`
    Default     interface{} `json:"default,omitempty"`
}

// Result represents the output of a tool execution.
type Result struct {
    Success     bool        `json:"success"`
    Message     string      `json:"message"`
    Data        interface{} `json:"data,omitempty"`      // Structured data
    Preview     string      `json:"preview,omitempty"`    // YAML/JSON for user confirmation
    NeedsConfirm bool       `json:"needs_confirm"`      // Whether user approval is required
}

// DangerLevel represents the risk level of an operation.
type DangerLevel string

const (
    DangerLow    DangerLevel = "low"    // Read operations
    DangerMedium DangerLevel = "medium" // Create/Scale
    DangerHigh   DangerLevel = "high"   // Delete/Destructive
)
```

#### Tool Registry

```go
// pkg/tools/registry.go

package tools

import (
    "context"
    "fmt"
    "sort"
    "strings"
    "sync"
)

// Registry manages tool discovery and routing.
type Registry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
    return &Registry{
        tools: make(map[string]Tool),
    }
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool Tool) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.tools[tool.Name()]; exists {
        return fmt.Errorf("tool %s already registered", tool.Name())
    }

    r.tools[tool.Name()] = tool
    return nil
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    tool, exists := r.tools[name]
    return tool, exists
}

// List returns all registered tools sorted by category and name.
func (r *Registry) List() []Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    result := make([]Tool, 0, len(r.tools))
    for _, tool := range r.tools {
        result = append(result, tool)
    }

    // Sort by category, then name
    sort.Slice(result, func(i, j int) bool {
        if result[i].Category() != result[j].Category() {
            return result[i].Category() < result[j].Category()
        }
        return result[i].Name() < result[j].Name()
    })

    return result
}

// ListByCategory returns tools in a specific category.
func (r *Registry) ListByCategory(category string) []Tool {
    r.mu.RLock()
    defer r.mu.RUnlock()

    var result []Tool
    for _, tool := range r.tools {
        if tool.Category() == category {
            result = append(result, tool)
        }
    }
    return result
}

// Execute calls a tool by name with given arguments.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}) (Result, error) {
    tool, exists := r.Get(name)
    if !exists {
        return Result{}, fmt.Errorf("tool %s not found", name)
    }

    return tool.Execute(ctx, args)
}

// GetLLMDescriptions returns tool info formatted for LLM prompting.
func (r *Registry) GetLLMDescriptions() string {
    tools := r.List()

    var sb strings.Builder
    for _, tool := range tools {
        sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))

        for _, param := range tool.Parameters() {
            req := ""
            if param.Required {
                req = " (required)"
            }
            sb.WriteString(fmt.Sprintf("  • %s: %s%s\n", param.Name, param.Type, req))
        }
    }

    return sb.String()
}
```

### 3. Resource Handler Pattern

#### Resource Handler Interface

```go
// pkg/k8s/handlers/handler.go

package handlers

import (
    "context"
    "fmt"
    "k8s-wizard/pkg/tools"
)

// Handler provides operations for a specific K8s resource type.
type Handler interface {
    // Resource returns the resource type this handler manages.
    Resource() string

    // Operations returns the list of operations supported.
    Operations() []Operation

    // Validate checks if an operation is valid for this resource.
    Validate(op Operation, args map[string]interface{}) error
}

// Operation represents a specific action on a resource.
type Operation struct {
    Name        string
    Method      string    // create, get, list, delete, update, scale, describe, logs, exec, apply
    DangerLevel tools.DangerLevel
    Description string
    Parameters  []tools.Parameter
}
```

#### Base Handler Implementation

```go
// pkg/k8s/handlers/base.go

package handlers

import (
    "context"
    "fmt"
    "k8s.io/client-go/kubernetes"
    "k8s-wizard/pkg/tools"
)

// BaseHandler provides common functionality for resource handlers.
type BaseHandler struct {
    clientset kubernetes.Interface
    resource  string
    ops       []Operation
}

// NewBaseHandler creates a base handler.
func NewBaseHandler(clientset kubernetes.Interface, resource string) *BaseHandler {
    return &BaseHandler{
        clientset: clientset,
        resource:  resource,
    }
}

// Resource implements Handler.
func (h *BaseHandler) Resource() string {
    return h.resource
}

// Operations returns the list of operations.
func (h *BaseHandler) Operations() []Operation {
    return h.ops
}

// Validate provides basic validation.
func (h *BaseHandler) Validate(op Operation, args map[string]interface{}) error {
    for _, param := range op.Parameters {
        if param.Required {
            if _, exists := args[param.Name]; !exists {
                return fmt.Errorf("required parameter %s missing", param.Name)
            }
        }
    }
    return nil
}

// ToolExecutor defines how to execute an operation.
type ToolExecutor func(ctx context.Context, args map[string]interface{}) (tools.Result, error)

// operationTool implements tools.Tool for an operation.
type operationTool struct {
    handler  *BaseHandler
    op       Operation
    executor ToolExecutor
}

func (t *operationTool) Name() string {
    return fmt.Sprintf("%s_%s", t.op.Method, t.handler.resource)
}

func (t *operationTool) Description() string {
    return t.op.Description
}

func (t *operationTool) Category() string {
    return "k8s"
}

func (t *operationTool) Parameters() []tools.Parameter {
    return t.op.Parameters
}

func (t *operationTool) DangerLevel() tools.DangerLevel {
    return t.op.DangerLevel
}

func (t *operationTool) Execute(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    return t.executor(ctx, args)
}
```

#### Example: Deployment Handler Structure

```go
// pkg/k8s/handlers/deployment.go (summary)

package handlers

type DeploymentHandler struct {
    *BaseHandler
}

func NewDeploymentHandler(clientset kubernetes.Interface) *DeploymentHandler {
    base := NewBaseHandler(clientset, "deployment")
    base.ops = []Operation{
        {
            Name:        "create",
            Method:      "create",
            DangerLevel: tools.DangerLow,
            Description: "Create a new deployment",
            Parameters: []tools.Parameter{
                {Name: "name", Type: "string", Description: "Deployment name", Required: true},
                {Name: "namespace", Type: "string", Description: "Namespace", Default: "default"},
                {Name: "image", Type: "string", Description: "Container image", Required: true},
                {Name: "replicas", Type: "number", Description: "Number of replicas", Default: 1},
            },
        },
        // ... scale, get, delete operations
    }
    return &DeploymentHandler{BaseHandler: base}
}

func (h *DeploymentHandler) RegisterTools(registry *tools.Registry) error {
    for _, op := range h.Operations() {
        tool := h.CreateTool(op, h.executeOperation(op))
        if err := registry.Register(tool); err != nil {
            return err
        }
    }
    return nil
}
```

### 4. Prompt Management

#### Prompt Structure

```
prompts/
├── intent.yaml          # Main intent parsing prompt
├── intent_schema.json  # Schema for validation (optional)
├── tools.yaml          # Tool descriptions for LLM
└── loader.go          # Runtime YAML loader with go:embed
```

#### Intent Prompt Template (simplified)

```yaml
# prompts/intent.yaml

name: intent_parser
version: "1.0"
description: Parses user intent and extracts K8s operation details

system_prompt: |
  你是一个智能的 Kubernetes 助手。理解用户的自然语言指令，判断意图并提取关键信息。

user_prompt: |
  用户指令: {{.UserMessage}}

  可用的工具:
  {{.ToolDescriptions}}

  请返回 JSON（只返回 JSON，不要其他文字）:
  {
    "action": "操作类型",
    "resource": "资源类型",
    "name": "资源名称",
    "namespace": "命名空间",
    "params": {},
    "is_k8s_operation": true/false,
    "tool_name": "具体要调用的工具名称"
  }

  规则:
  1. tool_name 是必须的，格式为 "操作_资源"，例如: "create_deployment", "get_pod", "scale_deployment"
  2. 如果是 K8s 相关操作，设置 is_k8s_operation=true
  3. 如果是闲聊、打招呼、提问等，设置 is_k8s_operation=false 并在 reply 中回复用户
  4. params 包含操作所需的所有参数，对应工具的 parameters 定义
```

#### Prompt Loader

```go
// pkg/prompts/loader.go (summary)

package prompts

import (
    "embed"
    "fmt"
    "text/template"
    "yaml"
)

//go:embed *.yaml
var promptFiles embed.FS

// Loader manages prompt templates.
type Loader struct {
    prompts    map[string]*Prompt
    tools      map[string][]ToolDescription
    categories map[string][]ToolDescription
}

func NewLoader() (*Loader, error) {
    // Load from embedded files
    // ...
}

// GetIntentPrompt returns intent parsing prompt.
func (l *Loader) GetIntentPrompt(userMessage string, toolRegistry *tools.Registry) (string, error) {
    prompt, ok := l.prompts["intent"]
    if !ok {
        return "", fmt.Errorf("intent prompt not found")
    }

    data := make(map[string]interface{})
    data["UserMessage"] = userMessage
    data["ToolDescriptions"] = toolRegistry.GetLLMDescriptions()

    // Render template with data
    // ...
    return fullPrompt, nil
}

// UpdateFromRegistry updates tool descriptions from registry.
func (l *Loader) UpdateFromRegistry(registry *tools.Registry) {
    // Rebuild tool descriptions from registry
    // This allows tools to be defined in code, not YAML
    tools := registry.List()
    // ...
}
```

### 5. Workflow Enhancements

#### Sub-Graph Pattern

```go
// pkg/workflow/subgraph.go (summary)

package workflow

// SubGraph represents a reusable workflow fragment.
type SubGraph interface {
    Name() string
    Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error)
    Entry() string
    Exit() []string
}

// DiagnosticsSubGraph implements complex diagnostics workflow.
type DiagnosticsSubGraph struct{}

func NewDiagnosticsSubGraph() *DiagnosticsSubGraph {
    return &DiagnosticsSubGraph{}
}

func (g *DiagnosticsSubGraph) Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error) {
    sub := lgg.NewStateGraph[AgentState]()

    // Add nodes: check_pod_status, check_logs, check_events, analyze_issues
    // Add conditional edges for routing
    // Compile
    return sub.Compile()
}

// LogsSubGraph, ExecSubGraph follow similar pattern...
```

#### Sub-Graph Manager

```go
// pkg/workflow/subgraph_manager.go (summary)

package workflow

// SubGraphManager manages available sub-graphs.
type SubGraphManager struct {
    subgraphs map[string]SubGraph
}

func NewSubGraphManager() *SubGraphManager {
    return &SubGraphManager{
        subgraphs: make(map[string]SubGraph),
    }
}

func (m *SubGraphManager) Register(subgraph SubGraph) error {
    if _, exists := m.subgraphs[subgraph.Name()]; exists {
        return fmt.Errorf("sub-graph %s already registered", subgraph.Name())
    }
    m.subgraphs[subgraph.Name()] = subgraph
    return nil
}

func (m *SubGraphManager) Get(name string) (SubGraph, bool) {
    sg, exists := m.subgraphs[name]
    return sg, exists
}
```

#### Context-Aware State Management

```go
// pkg/workflow/context.go (summary)

package workflow

import "time"

// ConversationContext maintains conversation history and context.
type ConversationContext struct {
    ThreadID       string
    UserMessage    string
    History        []ConversationEntry
    LastOperation  *K8sAction
    LastResource   string
    LastNamespace string
    Timestamp     time.Time
}

// ConversationEntry represents a single conversation turn.
type ConversationEntry struct {
    Role      string    // "user" or "assistant"
    Content   string
    Action    *K8sAction
    Timestamp time.Time
}

// ContextManager manages conversation contexts per thread.
type ContextManager struct {
    contexts    map[string]*ConversationContext
    checkpointer *CheckpointerManager
}

func NewContextManager(checkpointer *CheckpointerManager) *ContextManager {
    return &ContextManager{
        contexts:    make(map[string]*ConversationContext),
        checkpointer: checkpointer,
    }
}

func (m *ContextManager) Get(threadID string) *ConversationContext {
    // Retrieve or create context
    // ...
}

func (m *ContextManager) AddEntry(threadID string, entry ConversationEntry) {
    // Track last operation for context
    // Enable "that one", "it" references
}

func (m *ContextManager) GetContextString(threadID string, maxHistory int) string {
    // Format conversation history for LLM
    // Include last operation context
}
```

---

## Package Structure

### Final Structure

```
k8s-wizard/
├── pkg/
│   ├── agent/                    # (minimal changes)
│   │   ├── agent.go
│   │   └── agent_test.go
│   │
│   ├── tools/                    # (NEW - Phase 1)
│   │   ├── tool.go
│   │   ├── registry.go
│   │   ├── registry_test.go
│   │   └── doc.go
│   │
│   ├── prompts/                  # (NEW - Phase 2)
│   │   ├── loader.go
│   │   ├── loader_test.go
│   │   ├── prompts.go
│   │   └── templates/
│   │       ├── intent.yaml
│   │       └── tools.yaml
│   │
│   ├── k8s/
│   │   ├── client.go            # (existing, minimal changes)
│   │   ├── client_test.go
│   │   ├── handlers/            # (NEW - Phase 2)
│   │   │   ├── handler.go
│   │   │   ├── base.go
│   │   │   ├── registry.go
│   │   │   ├── deployment.go
│   │   │   ├── pod.go
│   │   │   ├── service.go
│   │   │   ├── logs.go
│   │   │   ├── exec.go
│   │   │   └── _test/
│   │   └── context.go            # (NEW - Phase 3)
│   │
│   ├── workflow/
│   │   ├── state.go             # (existing, add new deps)
│   │   ├── state_test.go
│   │   ├── nodes.go             # (existing, refactored)
│   │   ├── nodes_test.go
│   │   ├── routing.go           # (existing)
│   │   ├── routing_test.go
│   │   ├── graph.go            # (existing, enhanced)
│   │   ├── graph_test.go
│   │   ├── subgraph.go          # (NEW - Phase 3)
│   │   ├── subgraph_manager.go   # (NEW - Phase 3)
│   │   ├── logs_nodes.go         # (NEW - Phase 3)
│   │   └── context.go           # (NEW - Phase 3)
│   │
│   ├── llm/                      # (existing)
│   ├── config/                   # (existing)
│   └── logger/                   # (existing)
│
├── docs/
│   ├── superpowers/
│   │   └── specs/
│   │       └── 2026-03-13-agent-extensibility-design.md
│   ├── ARCHITECTURE.md         # (update)
│   └── ROADMAP.md             # (update)
│
└── cmd/
    └── k8s-wizard/
        └── main.go            # (update initialization)
```

---

## Migration Plan

### Phase 1: Tool System Foundation (1-2 weeks)

**Goal**: Implement tool interface and registry without breaking existing code.

| Task | Effort | Dependencies |
|------|---------|--------------|
| Create `pkg/tools/` package | 2 days | None |
| Implement `Tool` interface | 1 day | Package created |
| Implement `Registry` | 2 days | Interface |
| Add unit tests for registry | 1 day | Registry |
| Update `AgentState.Dependencies` | 0.5 day | None |
| Update agent initialization | 1 day | Dependencies updated |
| **Total** | **~1 week** | |

**Milestone**: Tool system in place, ready for handlers.

### Phase 2: Resource Handlers & Prompts (2-3 weeks)

**Goal**: Migrate existing operations to handler pattern and externalize prompts.

| Task | Effort | Dependencies |
|------|---------|--------------|
| Create `pkg/k8s/handlers/` package | 1 day | Phase 1 complete |
| Implement `Handler` interface and base | 1 day | Package created |
| Implement `DeploymentHandler` | 2 days | Base |
| Implement `PodHandler` | 2 days | Deployment |
| Implement `ServiceHandler` | 2 days | Pod |
| Implement `LogsHandler` | 3 days | Base |
| Implement `ExecHandler` | 3 days | Base |
| Add unit tests for each handler | 2 days | All handlers |
| Create `HandlerRegistry` | 1 day | Handlers |
| Create `pkg/prompts/` package | 0.5 day | None |
| Create YAML templates | 1 day | Package created |
| Implement `Loader` | 1.5 days | Templates |
| Update `MakeParseIntentNode` | 0.5 day | Loader |
| Update workflow to use handlers | 1 day | Registry |
| Update initialization | 0.5 day | Node updated |
| **Total** | **~3.5 weeks** | |

**Milestone**: All existing operations migrated to handlers, prompts externalized.

### Phase 3: Sub-Graphs & Context (2-3 weeks)

**Goal**: Add complex workflow support and context awareness.

| Task | Effort | Dependencies |
|------|---------|--------------|
| Implement `SubGraph` interface | 1 day | None |
| Implement `LogsSubGraph` | 2 days | Interface |
| Implement `ExecSubGraph` | 3 days | Logs |
| Implement `DiagnosticsSubGraph` | 3 days | Exec |
| Create `SubGraphManager` | 1 day | All sub-graphs |
| Implement `ContextManager` | 2 days | None |
| Update routing for sub-graphs | 1 day | Manager |
| Add unit tests | 2 days | All components |
| **Total** | **~2.5 weeks** | |

**Milestone**: Complex workflows and context awareness working.

### Phase 4: Integration & Testing (1-2 weeks)

**Goal**: Ensure everything works together.

| Task | Effort | Dependencies |
|------|---------|--------------|
| End-to-end integration tests | 2 days | All phases complete |
| Update documentation | 2 days | Integration working |
| Update ARCHITECTURE.md | 1 day | Code complete |
| Update ROADMAP.md | 0.5 day | Architecture updated |
| Create migration guide | 0.5 day | Documentation updated |
| **Total** | **~1 week** | |

**Milestone**: Production-ready extensibility system.

---

## Risk Mitigation

| Risk | Mitigation |
|-------|------------|
| **Breaking existing functionality** | Keep old code path, add new alongside, gradual migration |
| **Tool registration complexity** | Start with few tools, add more incrementally |
| **LangGraphGo sub-graph support** | If not supported, implement as routing pattern first |
| **Performance overhead** | Benchmark tool execution, optimize hot paths |
| **Test coverage regression** | Run tests after each phase, maintain 80%+ coverage |
| **Solo developer overwhelm** | Incremental phases, clear milestones, can pause between phases |

---

## Success Criteria

- ✅ All existing operations work unchanged
- ✅ New resource types can be added by creating a handler file
- ✅ New operations can be added by implementing handler method
- ✅ Prompts are editable without recompilation
- ✅ Sub-graphs enable complex workflows
- ✅ Context awareness for multi-turn conversations
- ✅ Test coverage maintained at 80%+
- ✅ Documentation updated (ARCHITECTURE.md, ROADMAP.md)
- ✅ Migration guide created

---

## Future Enhancements (Beyond Scope)

These are not part of this design but represent natural evolution:

1. **Tool Discovery** - Automatic discovery of handlers via Go plugins or reflection
2. **Tool Configuration** - External tool definitions in YAML/JSON files
3. **Multi-Cluster Support** - Cluster-aware context and routing
4. **Resource Templates** - Pre-defined templates for common deployments
5. **Operation History** - Undo/redo functionality
6. **RBAC Integration** - Permission-aware tool routing
7. **Metrics & Observability** - Per-tool performance tracking
8. **Streaming** - Real-time response streaming

---

## Summary

This design enables **incremental extensibility** across all key areas of K8s Wizard:

| Area | Phase | Key Benefit |
|-------|--------|-------------|
| **Tool System** | 1 | Uniform abstraction, easy routing, LLM integration |
| **Resource Handlers** | 2 | One file per resource, clear operations, easy to add |
| **Prompts** | 2 | Externalized, editable, versionable, no recompile needed |
| **Sub-Graphs** | 3 | Complex workflows, reusability, multi-step operations |
| **Context** | 3 | Multi-turn conversations, memory, contextual references |

**Total Timeline**: 8-10 weeks for full implementation
**Approach**: Incremental, non-breaking, maintainable for solo developer
**Philosophy**: Add, don't replace; interface-based; backward compatible

---

**Next Steps**:
1. Review this design document
2. Create implementation plan using `superpowers:writing-plans` skill
3. Begin with Phase 1: Tool System Foundation
