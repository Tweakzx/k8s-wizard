# Agent Extensibility Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a comprehensive tool system, resource handlers, prompt management, sub-graphs, and context awareness to enable incremental addition of new K8s resources and operations while maintaining backward compatibility.

**Architecture:** Incremental, non-breaking refactoring of the existing K8s Wizard agent. New abstractions (Tool, Handler, SubGraph, ContextManager) complement existing code rather than replacing it. Old and new code paths coexist during migration with feature flags controlling routing.

**Tech Stack:**
- Go 1.24+ with generics
- LangGraphGo for workflow orchestration
- client-go for K8s API access
- SQLite for session persistence (existing)
- YAML template management with go:embed

---

## Implementation Notes

This plan provides the structure and key implementations for agent extensibility. Some components are intentionally abbreviated here and should be implemented following patterns from the [specification document](../specs/2026-03-13-agent-extensibility-design.md).

### Components Requiring Spec-Based Implementation

During execution, implement these components by referencing the spec's detailed designs:

1. **HandlerRegistry** (Task 6, Step 3)
   - Refer to spec section "Resource Handlers - Handler Registry"
   - Manages all K8s resource handlers
   - Provides RegisterWithTools() and InitializeStandardHandlers() methods

2. **SubGraphManager** (Dependencies struct)
   - Refer to spec section "Sub-Graphs - SubGraphManager"
   - Manages registered sub-graphs and routing
   - Methods: Register(), Get(), Execute()

3. **CheckpointerManager** (ContextManager dependency)
   - Refer to spec section "Context Awareness - Persistence Layer"
   - Wraps SQLite database for checkpointing
   - Methods: Save(), Load(), ClearSession()

4. **Operation type** (used in handlers)
   - Refer to spec section "Tool System - Operation Definition"
   - Contains: Name, Method, Description, DangerLevel, Parameters

5. **K8sAction type** (used in ConversationEntry)
   - Refer to spec section "Context Awareness - State Tracking"
   - Contains: Action, Resource, Namespace, Params

6. **ToolDescription type** (used in Loader)
   - Refer to spec section "Prompt Management - Tool Registry Integration"
   - Static tool descriptions for LLM prompting

7. **DeploymentHandler operation methods** (Task 6)
   - Refer to spec section "Resource Handlers - Example: DeploymentHandler"
   - Implement: create(), get(), scale(), delete() methods
   - Follow the same pattern as the BaseHandler validation

### Implementation Patterns

When implementing spec-based components:

1. **Read the spec section** for the full interface and behavior requirements
2. **Use existing implementations** in this plan as patterns (e.g., BaseHandler, Registry)
3. **Follow TDD**: write test first, implement, verify, commit
4. **Maintain consistency**: use the same error handling, logging, and validation patterns
5. **Add imports**: reference the spec for required Go package imports

### References

- **Specification**: [`docs/superpowers/specs/2026-03-13-agent-extensibility-design.md`](../specs/2026-03-13-agent-extensibility-design.md)
- **Review Criteria**: See `../skills/brainstorming/spec-document-reviewer-prompt.md`

---

## Task Structure

This implementation plan is organized into 4 phases matching the design specification. Each phase can be worked on independently, with clear dependencies between phases.

---

## Chunk 1: Phase 1 - Tool System Foundation (1-2 weeks)

### Task 1: Create pkg/tools/ package

**Files:**
- Create: `pkg/tools/common.go`
- Create: `pkg/tools/tool.go`
- Create: `pkg/tools/registry.go`
- Create: `pkg/tools/doc.go`

- [ ] **Step 1: Create common type definitions**

```go
// pkg/tools/common.go

package tools

// DangerLevel represents the risk level of an operation.
type DangerLevel int

const (
    DangerLow DangerLevel = iota
    DangerMedium
    DangerHigh
)

// Parameter describes a tool input parameter.
type Parameter struct {
    Name        string
    Type        string
    Description string
    Required    bool
    Default     interface{}
}

// Result represents the output of a tool execution.
type Result struct {
    Success bool
    Message string
    Data    map[string]interface{}
    Preview string
}
```

**Expected:** File created successfully

**Run:** `go build ./pkg/tools/...`

- [ ] **Step 2: Write failing tests**

```go
// pkg/tools/registry_test.go

package tools

import "testing"

func TestRegisterDuplicateTool(t *testing.T) {
    registry := NewRegistry()

    tool := &mockTool{
        name: "test_tool",
    }

    err := registry.Register(tool)
    if err == nil {
        t.Errorf("expected error on duplicate tool registration, got nil")
    }

    // Try to register the same tool again
    err = registry.Register(tool)
    if err == nil {
        t.Errorf("expected error on second duplicate tool registration, got nil")
    }
}

func TestGetNonExistentTool(t *testing.T) {
    registry := NewRegistry()

    tool, exists := registry.Get("non_existent_tool")

    if exists {
        t.Errorf("expected tool to not exist")
    }
}

func TestExecuteTool(t *testing.T) {
    registry := NewRegistry()

    executor := func(ctx context.Context, args map[string]interface{}) (Result, error) {
        return Result{Success: true, Message: "executed"}, nil
    }

    tool := &operationTool{
        name: "test_tool",
        executor: executor,
    }

    registry.Register(tool)

    result, err := registry.Execute(context.Background(), "test_tool", map[string]interface{}{})

    if err != nil {
        t.Errorf("unexpected error executing tool: %v", err)
    }

    if !result.Success {
        t.Errorf("expected tool to succeed, got success=false")
    }

    if result.Message != "executed" {
        t.Errorf("unexpected tool result message, got: %s", result.Message)
    }
}

type mockTool struct {
    name string
}

func (m *mockTool) Name() string {
    return m.name
}

func (m *mockTool) Description() string {
    return "test tool description"
}

func (m *mockTool) Category() string {
    return "test"
}

func (m *mockTool) Parameters() []Parameter {
    return nil
}

func (m *mockTool) DangerLevel() DangerLevel {
    return DangerLow
}

func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (Result, error) {
    return Result{Success: true, Message: "executed"}, nil
}
```

**Expected:** All tests fail

**Run:** `go test ./pkg/tools/... -v`

- [ ] **Step 2: Implement Tool interface**

```go
// pkg/tools/tool.go

package tools

import "context"

// Tool represents a discrete operation that agent can perform.
type Tool interface {
    // Name returns a unique identifier for this tool.
    Name() string

    // Description explains what this tool does (used by LLM).
    Description() string

    // Category groups related tools (e.g., "k8s", "llm", "builtin").
    Category() string

    // Parameters describes expected inputs for LLM prompting.
    Parameters() []Parameter

    // DangerLevel indicates the risk level of this operation.
    DangerLevel() DangerLevel

    // Execute runs the tool with given arguments.
    Execute(ctx context.Context, args map[string]interface{}) (Result, error)
}
```

**Expected:** File created successfully

**Run:** `go test ./pkg/tools/... -v`

- [ ] **Step 2.5: Implement operationTool**

```go
// pkg/tools/tool.go (add after Tool interface)

// operationTool implements Tool for a simple operation.
type operationTool struct {
    name     string
    executor func(context.Context, map[string]interface{}) (Result, error)
}

func (t *operationTool) Name() string {
    return t.name
}

func (t *operationTool) Description() string {
    return "Operation tool"
}

func (t *operationTool) Category() string {
    return "builtin"
}

func (t *operationTool) Parameters() []Parameter {
    return nil
}

func (t *operationTool) DangerLevel() DangerLevel {
    return DangerLow
}

func (t *operationTool) Execute(ctx context.Context, args map[string]interface{}) (Result, error) {
    return t.executor(ctx, args)
}
```

**Expected:** operationTool implemented

**Run:** `go test ./pkg/tools/... -v`

- [ ] **Step 3: Implement Registry**

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

**Expected:** File created successfully

**Run:** `go test ./pkg/tools/... -v`

- [ ] **Step 4: Implement documentation**

```go
// pkg/tools/doc.go

package tools

import "fmt"

const (
    // Package documentation
    PackageName        = "tools"
    PackageVersion   = "1.0.0"
    PackageDesc      = "Tool system for K8s operations - uniform abstraction and registry"
)

// Doc returns package documentation.
func Doc() string {
    return fmt.Sprintf(`%s v%s - %s

%s

Provides:
  - Tool interface for uniform K8s operation abstraction
  - Tool registry for dynamic discovery and routing
  - LLM-friendly tool descriptions

Key types:
  - Tool: Uniform operation interface
  - Parameter: Tool input parameter definition
  - Result: Tool execution output
  - DangerLevel: Risk level enumeration

Usage:
  1. Create a tool by implementing the Tool interface
  2. Register tool with registry.Register(tool)
  3. LLM gets tool descriptions via registry.GetLLMDescriptions()
  4. Execute tool via registry.Execute(ctx, name, args)

Example:
  registry := tools.NewRegistry()
  tool := &MyTool{}
  registry.Register(tool)
  result, err := registry.Execute(ctx, "my_tool", args)
`, PackageName, PackageVersion, PackageDesc)
}
```

**Expected:** File created successfully

**Run:** `go build ./pkg/tools/...`

- [ ] **Step 5: Commit**

```bash
git add pkg/tools/tool.go pkg/tools/registry.go pkg/tools/doc.go pkg/tools/registry_test.go
git commit -m "$(cat <<'EOF'
feat: create pkg/tools package with Tool interface and Registry

- Add Tool interface for uniform K8s operation abstraction
- Add Registry for dynamic tool discovery and routing
- Add package documentation
- Add unit tests for duplicate registration, retrieval, and execution

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Update AgentState.Dependencies

**Files:**
- Modify: `pkg/workflow/state.go`

- [ ] **Step 1: Write failing test**

```go
// pkg/workflow/state_test.go

package workflow

import "testing"

func TestDependenciesWithToolRegistry(t *testing.T) {
    deps := Dependencies{
        ToolRegistry: NewRegistry(),
    // other fields omitted for brevity
    }

    if deps.ToolRegistry == nil {
        t.Errorf("ToolRegistry field should not be nil in Dependencies")
    }
}
```

**Expected:** Test fails

**Run:** `go test ./pkg/workflow/... -v`

- [ ] **Step 2: Add ToolRegistry field**

```go
// pkg/workflow/state.go (partial diff)

type Dependencies struct {
    K8sClient     k8s.Client
    LLM            llm.Client
    ModelName      string
    ToolRegistry   *tools.Registry    // NEW - Phase 1
    PromptLoader   *prompts.Loader   // NEW - Phase 2
    SubGraphMgr   *SubGraphManager  // NEW - Phase 3
    ContextMgr     *ContextManager    // NEW - Phase 3
}
```

**Expected:** Field added successfully

**Run:** `go test ./pkg/workflow/... -v`

- [ ] **Step 3: Update agent initialization**

```go
// pkg/agent/agent.go (partial diff)

func NewAgent(...) *Agent {
    deps := Dependencies{
        ToolRegistry: tools.NewRegistry(),
        // ... initialize other fields
    }
    // ... rest of initialization
}
```

**Expected:** Agent initialization updated to include ToolRegistry

**Run:** `go test ./pkg/agent/... -v`

- [ ] **Step 4: Commit**

```bash
git add pkg/workflow/state.go pkg/agent/agent.go pkg/workflow/state_test.go
git commit -m "$(cat <<'EOF'
feat: add ToolRegistry field to Dependencies

- Add ToolRegistry field to Dependencies struct
- Initialize ToolRegistry in agent dependencies

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: Update pkg/agent initialization

**Files:**
- Modify: `pkg/agent/agent.go`

- [ ] **Step 1: Write failing test**

```go
// pkg/agent/agent_test.go

package agent

import (
    "context"
    "testing"
    "k8s-wizard/pkg/workflow"
    "k8s-wizard/pkg/tools"
    "k8s-wizard/pkg/prompts"
)

func TestNewAgentCreatesToolRegistry(t *testing.T) {
    agent := NewAgent(nil, nil)

    if agent.deps.ToolRegistry == nil {
        t.Errorf("ToolRegistry should be initialized in agent dependencies")
    }
}
```

**Expected:** Test fails initially

**Run:** `go test ./pkg/agent/... -v`

- [ ] **Step 2: Update NewAgent to initialize ToolRegistry**

```go
// pkg/agent/agent.go (modification)

func NewAgent(...) *Agent {
    deps := Dependencies{
        ToolRegistry: tools.NewRegistry(),  // NEW - Phase 1
        PromptLoader:  nil,               // NEW - Phase 2
        SubGraphMgr:  nil,               // NEW - Phase 3
        ContextMgr:  nil,                // NEW - Phase 3
    }
    // ... rest of initialization
}
```

**Expected:** ToolRegistry added to dependencies

**Run:** `go test ./pkg/agent/... -v`

- [ ] **Step 3: Commit**

```bash
git add pkg/workflow/state.go pkg/agent/agent.go
git commit -m "$(cat <<'EOF'
feat: add ToolRegistry to Dependencies and Agent

- Add ToolRegistry field to Dependencies struct
- Initialize ToolRegistry in NewAgent for tool system support

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 2: Phase 2 - Resource Handlers & Prompts (3-4 weeks)

### Task 4: Create pkg/k8s/handlers/ package

**Files:**
- Create: `pkg/k8s/handlers/handler.go`
- Create: `pkg/k8s/handlers/base.go`

- [ ] **Step 1: Write failing tests**

```go
// pkg/k8s/handlers/base_test.go

package handlers

import "testing"

func TestNewBaseHandler(t *testing.T) {
    clientset := fakeClientset{}

    handler := NewBaseHandler(clientset, "test-resource")
    if handler.Resource() != "test-resource" {
        t.Errorf("expected resource to be 'test-resource'")
    }
}
```

**Expected:** Test fails

**Run:** `go test ./pkg/k8s/handlers/... -v`

- [ ] **Step 2: Implement Handler interface**

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

    // Operations returns a list of operations supported.
    Operations() []Operation

    // Validate checks if an operation is valid for this resource.
    Validate(op Operation, args map[string]interface{}) error
}
```

**Expected:** File created successfully

**Run:** `go test ./pkg/k8s/handlers/... -v`

- [ ] **Step 3: Implement BaseHandler**

```go
// pkg/k8s/handlers/base.go

package handlers

import (
    "context"
    "fmt"
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

// Operations returns a list of operations.
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

// CreateTool creates a tool from an operation.
func (h *BaseHandler) CreateTool(op Operation, executor ToolExecutor) tools.Tool {
    return &operationTool{
        handler:  h,
        op:       op,
        executor: executor,
    }
}

// ToolExecutor defines how to execute an operation.
type ToolExecutor func(ctx context.Context, args map[string]interface{}) (tools.Result, error)

// operationTool implements tools.Tool for an operation.
type operationTool struct {
    handler  *BaseHandler
    op       Operation
    executor ToolExecutor
}
```

**Expected:** File created successfully

**Run:** `go test ./pkg/k8s/handlers/... -v`

- [ ] **Step 4: Commit**

```bash
git add pkg/k8s/handlers/handler.go pkg/k8s/handlers/base.go pkg/k8s/handlers/base_test.go
git commit -m "$(cat <<'EOF'
feat: create pkg/k8s/handlers package with Handler interface and BaseHandler

- Add Handler interface for K8s resource operations
- Add BaseHandler with common functionality
- Add validation logic for operation parameters
- Add CreateTool method to bridge handlers to tools
- Add unit tests for BaseHandler

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: Create pkg/prompts/ package

**Files:**
- Create: `pkg/prompts/loader.go`
- Create: `pkg/prompts/templates/intent.yaml`
- Create: `pkg/prompts/templates/tools.yaml`

- [ ] **Step 1: Write failing tests**

```go
// pkg/prompts/loader_test.go

package prompts

import "testing"

func TestLoadEmbeddedTemplates(t *testing.T) {
    loader, err := NewLoader()
    if err != nil {
        t.Fatalf("failed to create loader: %v", err)
    }

    intent := loader.GetIntentPrompt("")
    if intent == "" {
        t.Errorf("intent prompt should not be empty")
    }

    tools := loader.GetToolDescriptions("k8s")
    if len(tools) == 0 {
        t.Errorf("expected at least static tool description for k8s")
    }
}
```

**Expected:** Tests fail initially

**Run:** `go test ./pkg/prompts/... -v`

- [ ] **Step 2: Implement Loader**

```go
// pkg/prompts/loader.go

package prompts

import (
    "embed"
    "fmt"
    "text/template"
    "yaml"
    "k8s-wizard/pkg/tools"
)

//go:embed *.yaml
var promptFiles embed.FS

// Prompt represents a loaded prompt template.
type Prompt struct {
    Name        string
    Version     string
    Description string
    System      string
    User        string
}

// Loader manages prompt templates.
type Loader struct {
    prompts    map[string]*Prompt
    tools      map[string]ToolDescription
    categories map[string][]ToolDescription
}

// NewLoader creates a new prompt loader.
func NewLoader() (*Loader, error) {
    loader := &Loader{
        prompts:    make(map[string]*Prompt),
        tools:      make(map[string]ToolDescription),
        categories: make(map[string][]ToolDescription),
    }

    if err := loader.loadEmbedded(); err != nil {
        return nil, err
    }

    return loader, nil
}

// loadEmbedded loads prompts from embedded files.
func (l *Loader) loadEmbedded() error {
    // Load intent prompt
    intentData, err := promptFiles.ReadFile("intent.yaml")
    if err != nil {
        return fmt.Errorf("failed to load intent prompt: %w", err)
    }

    var intentPrompt struct {
        Name        string `yaml:"name"`
        Version     string `yaml:"version"`
        Description string `yaml:"description"`
        System      string `yaml:"system_prompt"`
        User        string `yaml:"user_prompt"`
    }

    if err := yaml.Unmarshal(intentData, &intentPrompt); err != nil {
        return fmt.Errorf("failed to parse intent prompt: %w", err)
    }

    l.prompts["intent"] = &Prompt{
        Name:        intentPrompt.Name,
        Version:     intentPrompt.Version,
        Description: intentPrompt.Description,
        System:      intentPrompt.System,
        User:        intentPrompt.User,
    }

    // Load tools descriptions (optional, for reference)
    toolsData, err := promptFiles.ReadFile("tools.yaml")
    if err == nil {
        var toolsConfig struct {
            Tools []struct {
                Category   string               `yaml:"category"`
                Description string               `yaml:"description"`
                Tools      []ToolDescription     `yaml:"tools"`
            } `yaml:"tools"`
        }

        if err := yaml.Unmarshal(toolsData, &toolsConfig); err == nil {
            for _, category := range toolsConfig.Tools {
                for _, tool := range category.Tools {
                    tool.Category = category.Category
                    l.tools[tool.Name] = tool
                    l.categories[category.Category] = append(l.categories[category.Category], tool)
                }
            }
        }
    }

    return nil
}
```

**Expected:** File created successfully

**Run:** `go test ./pkg/prompts/... -v`

- [ ] **Step 3: Create intent.yaml template**

```yaml
# prompts/templates/intent.yaml

name: intent_parser
version: "1.0.0"
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

**Expected:** File created successfully

- [ ] **Step 4: Create tools.yaml template**

```yaml
# prompts/templates/tools.yaml

name: tool_descriptions
version: "1.0.0"
description: Tool descriptions for LLM prompting

# Note: These are static descriptions. In production, tools can register
# themselves dynamically, making this file optional.
tools:
  - category: k8s
    description: Kubernetes 资源操作
    tools:
      - name: create_deployment
        description: 创建一个新的 Deployment 资源
        parameters:
          - name: name
            type: string
            description: Deployment 名称
            required: true
          - name: namespace
            type: string
            description: 命名空间
            required: false
            default: default
          - name: image
            type: string
            description: 容器镜像
            required: true
          - name: replicas
            type: number
            description: 副本数量
            required: false
            default: 1
```

**Expected:** File created successfully

- [ ] **Step 5: Commit**

```bash
git add pkg/prompts/loader.go pkg/prompts/templates/intent.yaml pkg/prompts/templates/tools.yaml pkg/prompts/loader_test.go
git commit -m "$(cat <<'EOF'
feat: create pkg/prompts package with Loader and YAML templates

- Add Loader for managing prompt templates
- Add intent.yaml template for user intent parsing
- Add tools.yaml template for static tool descriptions
- Use go:embed for template loading
- Add unit tests for template loading

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Implement DeploymentHandler

**Files:**
- Create: `pkg/k8s/handlers/deployment.go`
- Modify: `pkg/workflow/state.go` (to add DeploymentHandler to imports)

- [ ] **Step 1: Write failing tests**

```go
// pkg/k8s/handlers/deployment_test.go

package handlers

import (
    "context"
    "testing"
    "k8s-wizard/pkg/tools"
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s-wizard/pkg/k8s"
)

func TestCreateDeployment(t *testing.T) {
    handler := NewDeploymentHandler(fakeClientset{})

    args := map[string]interface{}{
        "name": "test-deployment",
        "namespace": "default",
        "image": "nginx:1.25",
        "replicas": 1,
    }

    result, err := handler.Create(context.Background(), args)
    if err != nil {
        t.Errorf("unexpected error creating deployment: %v", err)
    }

    if !result.Success {
        t.Errorf("expected deployment creation to succeed")
    }

    if result.Preview == "" {
        t.Errorf("expected YAML preview to be generated")
    }
}

func fakeClientset() kubernetes.Interface {
    // Minimal fake implementation for testing
    return &fakeClientsetImpl{}
}

type fakeClientsetImpl struct{}

func (f *fakeClientsetImpl) AppsV1() appsv1.AppsV1Interface {
    return &fakeAppsV1{}
}

type fakeAppsV1 struct{}

func (f *fakeAppsV1) Deployments(ns string) appsv1.DeploymentInterface {
    return &fakeDeployments{}
}

func (f *fakeAppsV1) Deployments() appsv1.DeploymentCollectionInterface {
    return &fakeDeploymentsCollection{}
}

type fakeDeploymentsCollection struct{}

func (f *fakeDeploymentsCollection) List(opts metav1.ListOptions) (appsv1.DeploymentList, error) {
    return &fakeDeploymentList{}, nil
}

type fakeDeploymentList struct{}

func (l *fakeDeploymentList) Items() []appsv1.Deployment {
    return []appsv1.Deployment{}
}
```

**Expected:** Tests fail initially (needs implementation)

**Run:** `go test ./pkg/k8s/handlers/deployment_test.go -v`

- [ ] **Step 2: Implement DeploymentHandler**

```go
// pkg/k8s/handlers/deployment.go

package handlers

import (
    "context"
    "fmt"
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s-wizard/pkg/tools"
)

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
        {
            Name:        "get",
            Method:      "get",
            DangerLevel: tools.DangerLow,
            Description: "List deployments",
            Parameters: []tools.Parameter{
                {Name: "namespace", Type: "string", Description: "Namespace (empty for all)"},
            },
        },
        {
            Name:        "scale",
            Method:      "scale",
            DangerLevel: tools.DangerMedium,
            Description: "Scale a deployment",
            Parameters: []tools.Parameter{
                {Name: "name", Type: "string", Description: "Deployment name", Required: true},
                {Name: "namespace", Type: "string", Description: "Namespace", Default: "default"},
                {Name: "replicas", Type: "number", Description: "Target replica count", Required: true},
            },
        },
        {
            Name:        "delete",
            Method:      "delete",
            DangerLevel: tools.DangerHigh,
            Description: "Delete a deployment",
            Parameters: []tools.Parameter{
                {Name: "name", Type: "string", Description: "Deployment name", Required: true},
                {Name: "namespace", Type: "string", Description: "Namespace", Default: "default"},
            },
        },
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

func (h *DeploymentHandler) executeOperation(op Operation) ToolExecutor {
    return func(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
        switch op.Method {
        case "create":
            return h.create(ctx, args)
        case "get":
            return h.get(ctx, args)
        case "scale":
            return h.scale(ctx, args)
        case "delete":
            return h.delete(ctx, args)
        default:
            return tools.Result{}, fmt.Errorf("unsupported operation: %s", op.Method)
        }
    }
}
```

**Expected:** File created successfully

**Run:** `go test ./pkg/k8s/handlers/deployment_test.go -v`

- [ ] **Step 3: Register DeploymentHandler with tool registry in agent**

```go
// pkg/k8s/handlers/registry.go (modification)

// InitializeStandardHandlers registers all standard K8s handlers.
func (r *HandlerRegistry) InitializeStandardHandlers(clientset kubernetes.Interface, toolRegistry *tools.Registry) error {
    handlers := []Handler{
        NewDeploymentHandler(clientset),
    }

    for _, handler := range handlers {
        if err := r.RegisterWithTools(handler, toolRegistry); err != nil {
            return fmt.Errorf("failed to register %s tools: %w", handler.Resource(), err)
        }
    }

    return nil
}
```

**Expected:** HandlerRegistry updated

**Run:** `go test ./pkg/k8s/handlers/registry_test.go -v`

- [ ] **Step 4: Commit**

```bash
git add pkg/k8s/handlers/deployment.go pkg/k8s/handlers/registry.go pkg/k8s/handlers/base.go pkg/k8s/handlers/handler.go pkg/k8s/handlers/deployment_test.go pkg/k8s/handlers/base_test.go
git commit -m "$(cat <<'EOF'
feat: implement DeploymentHandler with create/get/scale/delete operations

- Add DeploymentHandler with create, get, scale, delete operations
- Add YAML preview generation for create operations
- Register DeploymentHandler with HandlerRegistry

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 3: Phase 3 - Sub-Graphs & Context (3-4 weeks)

### Task 7: Create pkg/workflow/subgraph.go

**Files:**
- Create: `pkg/workflow/subgraph.go`

- [ ] **Step 1: Write failing tests**

```go
// pkg/workflow/subgraph_test.go

package workflow

import "testing"

func TestSubGraphInterface(t *testing.T) {
    subgraph := &mockSubGraph{}

    if subgraph.Name() != "test_subgraph" {
        t.Errorf("expected subgraph name to be 'test_subgraph'")
    }

    if subgraph.Entry() != "entry_node" {
        t.Errorf("expected entry node to be 'entry_node'")
    }

    exits := subgraph.Exit()
    if len(exits) != 1 || exits[0] != lgg.END {
        t.Errorf("expected subgraph to have single exit to END")
    }
}

type mockSubGraph struct{}

func (s *mockSubGraph) Name() string {
    return "test_subgraph"
}

func (s *mockSubGraph) Entry() string {
    return "entry_node"
}

func (s *mockSubGraph) Exit() []string {
    return []string{lgg.END}
}
```

**Expected:** Tests fail initially (needs implementation)

**Run:** `go test ./pkg/workflow/subgraph_test.go -v`

- [ ] **Step 2: Implement SubGraph interface**

```go
// pkg/workflow/subgraph.go

package workflow

import (
    "context"
    "fmt"
    lgg "github.com/smallnest/langgraphgo/graph"
)

// SubGraph represents a reusable workflow fragment.
type SubGraph interface {
    Name() string
    Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error)
    Entry() string
    Exit() []string
}
```

**Expected:** File created successfully

**Run:** `go test ./pkg/workflow/subgraph_test.go -v`

- [ ] **Step 3: Commit**

```bash
git add pkg/workflow/subgraph.go pkg/workflow/subgraph_test.go
git commit -m "$(cat <<'EOF'
feat: create pkg/workflow/subgraph.go with SubGraph interface

- Add SubGraph interface for reusable workflow fragments
- Define Build, Entry, and Exit methods
- Add unit tests for SubGraph interface
- Enable sub-graph based workflow patterns

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 8: Create pkg/workflow/context.go

**Files:**
- Create: `pkg/workflow/context.go`

- [ ] **Step 1: Write failing tests**

```go
// pkg/workflow/context_test.go

package workflow

import (
    "context"
    "testing"
)

func TestContextManager(t *testing.T) {
    // Test requires CheckpointerManager, will create fake
    manager, _ := NewContextManager(nil)

    ctx := context.Background()
    entry := ConversationEntry{
        Role: "user",
        Content: "test message",
        Action: nil,
        Timestamp: time.Now(),
    }

    manager.AddEntry("thread-1", entry)

    retrieved := manager.Get("thread-1")
    if retrieved == nil {
        t.Errorf("context should be created after AddEntry")
    }

    if len(retrieved.History) != 1 {
        t.Errorf("context should contain the added entry")
    }
}
```

**Expected:** Tests fail initially (needs implementation)

**Run:** `go test ./pkg/workflow/context_test.go -v`

- [ ] **Step 2: Implement ContextManager**

```go
// pkg/workflow/context.go

package workflow

import (
    "context"
    "database/sql"
    "fmt"
    "sort"
    "strings"
    "time"
)

// ConversationContext maintains conversation history and context.
type ConversationContext struct {
    ThreadID       string
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

// NewContextManager creates a context manager.
func NewContextManager(checkpointer *CheckpointerManager) *ContextManager {
    return &ContextManager{
        contexts:    make(map[string]*ConversationContext),
        checkpointer: checkpointer,
    }
}

// Get retrieves or creates a conversation context.
func (m *ContextManager) Get(threadID string) *ConversationContext {
    if ctx, exists := m.contexts[threadID]; exists {
        // Update timestamp
        ctx.Timestamp = time.Now()
        return ctx
    }

    // Try to load from checkpoint
    var history []ConversationEntry
    if m.checkpointer != nil {
        // Attempt to load saved history
        if saved := m.loadFromCheckpoint(threadID); saved != nil {
            history = saved
        }
    }

    ctx := &ConversationContext{
        ThreadID:  threadID,
        History:   history,
        Timestamp: time.Now(),
    }

    m.contexts[threadID] = ctx
    return ctx
}

// AddEntry adds an entry to conversation history.
func (m *ContextManager) AddEntry(threadID string, entry ConversationEntry) {
    ctx := m.Get(threadID)
    ctx.History = append(ctx.History, entry)
    ctx.Timestamp = time.Now()

    // Track last operation for context
    if entry.Action != nil {
        ctx.LastOperation = entry.Action
        ctx.LastResource = entry.Action.Resource
        ctx.LastNamespace = entry.Action.Namespace
    }

    // Persist to checkpoint if available
    if m.checkpointer != nil {
        m.saveToCheckpoint(threadID)
    }
}

// GetContextString returns formatted context for LLM.
func (m *ContextManager) GetContextString(threadID string, maxHistory int) string {
    ctx := m.Get(threadID)

    var sb strings.Builder
    sb.WriteString("对话历史:\n")

    // Get recent history
    history := ctx.History
    if len(history) > maxHistory {
        history = history[len(history)-maxHistory:]
    }

    for _, entry := range history {
        role := "用户"
        if entry.Role == "assistant" {
            role = "助手"
        }
        sb.WriteString(fmt.Sprintf("  %s: %s\n", role, entry.Content))
    }

    // Add context from last operation
    if ctx.LastOperation != nil {
        sb.WriteString(fmt.Sprintf("\n最近操作: %s %s/%s\n",
            ctx.LastOperation.Action,
            ctx.LastNamespace,
            ctx.LastResource))
    }

    return sb.String()
}

// Clear removes conversation context.
func (m *ContextManager) Clear(threadID string) {
    delete(m.contexts, threadID)
    if m.checkpointer != nil {
        m.checkpointer.ClearSession(context.Background(), threadID)
    }
}

func (m *ContextManager) loadFromCheckpoint(threadID string) []ConversationEntry {
    if m.checkpointer == nil {
        return nil
    }

    // Use the database directly to store conversation history
    // We'll add a conversation_history table for this purpose
    rows, err := m.checkpointer.db.Query(`
        SELECT role, content, timestamp, action_json
        FROM conversation_history
        WHERE thread_id = ?
        ORDER BY timestamp ASC
    `, threadID)
    if err != nil {
        return nil // Table may not exist yet
    }
    defer rows.Close()

    var history []ConversationEntry
    for rows.Next() {
        var entry ConversationEntry
        var actionJSON sql.NullString
        var timestampStr string
        err := rows.Scan(&entry.Role, &entry.Content, &timestampStr, &actionJSON)
        if err != nil {
            continue
        }

        // Parse timestamp
        entry.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)

        // Parse action JSON if present
        if actionJSON.Valid && actionJSON.String != "" {
            // Simple JSON parsing - in production use proper JSON unmarshal
            entry.Action = parseActionFromJSON(actionJSON.String)
        }

        history = append(history, entry)
    }

    return history
}

func (m *ContextManager) saveToCheckpoint(threadID string) {
    if m.checkpointer == nil {
        return
    }

    ctx := m.contexts[threadID]
    if ctx == nil {
        return
    }

    // Ensure conversation_history table exists
    _, _ = m.checkpointer.db.Exec(`
        CREATE TABLE IF NOT EXISTS conversation_history (
            thread_id TEXT,
            role TEXT,
            content TEXT,
            timestamp TEXT,
            action_json TEXT,
            PRIMARY KEY (thread_id, role, timestamp)
        )
    `)

    // Save all entries
    for _, entry := range ctx.History {
        var actionJSON string
        if entry.Action != nil {
            actionJSON = fmt.Sprintf(`{"action":"%s","resource":"%s","namespace":"%s"}`,
                entry.Action.Action, entry.Action.Resource, entry.Action.Namespace)
        }

        _, err := m.checkpointer.db.Exec(`
            INSERT OR REPLACE INTO conversation_history
            (thread_id, role, content, timestamp, action_json)
            VALUES (?, ?, ?, ?, ?)
        `, entry.Role, entry.Content, entry.Timestamp.Format(time.RFC3339), actionJSON)
        if err != nil {
            // Log error but continue
            continue
        }
    }
}

// Helper function to parse action from JSON
func parseActionFromJSON(jsonStr string) *K8sAction {
    // Parse JSON string to reconstruct K8sAction
    // Using simple string parsing - can be improved with encoding/json if needed
    action := &K8sAction{
        Action:    "",
        Resource:  "",
        Namespace: "",
    }

    // Simple JSON field extraction
    if len(jsonStr) < 10 {
        return action
    }

    // Extract action field
    if idx := strings.Index(jsonStr, `"action":"`); idx >= 0 {
        endIdx := strings.Index(jsonStr[idx+10:], `"`)
        if endIdx >= 0 {
            action.Action = jsonStr[idx+10 : idx+10+endIdx]
        }
    }

    // Extract resource field
    if idx := strings.Index(jsonStr, `"resource":"`); idx >= 0 {
        endIdx := strings.Index(jsonStr[idx+12:], `"`)
        if endIdx >= 0 {
            action.Resource = jsonStr[idx+12 : idx+12+endIdx]
        }
    }

    // Extract namespace field
    if idx := strings.Index(jsonStr, `"namespace":"`); idx >= 0 {
        endIdx := strings.Index(jsonStr[idx+13:], `"`)
        if endIdx >= 0 {
            action.Namespace = jsonStr[idx+13 : idx+13+endIdx]
        }
    }

    return action
}
```

**Expected:** File created successfully

**Run:** `go test ./pkg/workflow/context_test.go -v`

- [ ] **Step 3: Commit**

```bash
git add pkg/workflow/context.go pkg/workflow/context_test.go
git commit -m "$(cat <<'EOF'
feat: create pkg/workflow/context.go with ContextManager

- Add ContextManager for conversation history and context tracking
- Add ConversationContext and ConversationEntry structs
- Implement AddEntry, Get, Clear methods
- Add GetContextString for LLM context formatting
- Integrate with CheckpointerManager for persistence
- Add loadFromCheckpoint and saveToCheckpoint with SQL queries
- Add unit tests for ContextManager

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Chunk 4: Phase 4 - Integration & Testing (2-3 weeks)

### Task 9: Update pkg/k8s/client.go

**Files:**
- Modify: `pkg/k8s/client.go`

- [ ] **Step 1: Write failing tests**

```go
// pkg/k8s/client_test.go

package k8s

import (
    "context"
    "testing"
    "k8s-wizard/pkg/workflow"
)

func TestClientInterface(t *testing.T) {
    // Tests would require a fake clientset
    t.Skip("client interface tests require k8s cluster setup")
}

func TestGetPodLogs(t *testing.T) {
    client := NewClient(nil, nil)

    logs, err := client.GetPodLogs(context.Background(), "test-ns", "test-pod", "", 100)
    if err != nil {
        t.Fatalf("unexpected error getting pod logs: %v", err)
    }

    if logs == "" {
        t.Error("expected non-empty logs for pod")
    }
}

func TestExecPod(t *testing.T) {
    client := NewClient(nil, nil)

    output, err := client.ExecPod(context.Background(), "test-ns", "test-pod", "echo test", "")

    if err != nil {
        t.Fatalf("unexpected error executing pod command: %v", err)
    }

    if output == "" {
        t.Error("expected non-empty output from exec")
    }
}
```

**Expected:** Tests fail (client implementation not done yet)

**Run:** `go test ./pkg/k8s/client_test.go -v`

- [ ] **Step 2: Add GetPodLogs method**

```go
// pkg/k8s/client.go (modification)

// GetPodLogs fetches pod logs.
func (c *Client) GetPodLogs(ctx context.Context, namespace string, pod string, container string, tailLines int64) (string, error) {
    req := c.clientset.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
        TailLines: &tailLines,
    })

    logs, err := req.Stream(ctx)
    if err != nil {
        return "", err
    }
    defer logs.Close()

    buf := new(bytes.Buffer)
    if _, err := buf.ReadFrom(logs); err != nil {
        return "", fmt.Errorf("read logs failed: %w", err)
    }

    return buf.String(), nil
}
```

**Expected:** Method added successfully

**Run:** `go test ./pkg/k8s/client_test.go -v`

- [ ] **Step 3: Add ExecPod method**

```go
// pkg/k8s/client.go (modification)

// ExecPod executes a command in a pod.
func (c *Client) ExecPod(ctx context.Context, namespace string, pod string, container string, command string) (string, error) {
    if command == "" {
        return "", fmt.Errorf("command cannot be empty")
    }

    // Split command into args (simple shell -c wrapper)
    args := []string{"sh", "-c", command}

    // Build exec request
    req := c.clientset.CoreV1().RESTClient().Post().
        Resource("pods").
        Namespace(namespace).
        Name(pod).
        SubResource("exec").
        VersionedParams(&corev1.PodExecOptions{
            Container: container,
            Command:   args,
            Stdin:     false,
            Stdout:    true,
            Stderr:    true,
            TTY:       false,
        }, scheme.ParameterCodec)

    // Create executor
    executor, err := remotecommand.NewSPDYExecutor(req.Config(), "POST", req.URL())
    if err != nil {
        return "", fmt.Errorf("failed to create executor: %w", err)
    }

    // Capture output
    buf := new(bytes.Buffer)
    errBuf := new(bytes.Buffer)

    // Execute command
    err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
        Stdout: buf,
        Stderr: errBuf,
    })
    if err != nil {
        return "", fmt.Errorf("exec command failed: %w: %s", err, errBuf.String())
    }

    return buf.String(), nil
}
```

**Expected:** Method implemented successfully

**Run:** `go test ./pkg/k8s/client_test.go -v`

- [ ] **Step 4: Commit**

```bash
git add pkg/k8s/client.go pkg/k8s/client_test.go
git commit -m "$(cat <<'EOF'
feat: add GetPodLogs and ExecPod methods to Client interface

- Add GetPodLogs method to fetch pod logs with tail support
- Add ExecPod method for pod command execution using SPDY executor
- These methods support logs and exec workflows

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 10: Update pkg/workflow/state.go for ContextManager

**Files:**
- Modify: `pkg/workflow/state.go`

- [ ] **Step 1: Write failing tests**

```go
// pkg/workflow/context_test.go (modification)

func TestDependenciesWithContextManager(t *testing.T) {
    deps := Dependencies{
        ContextMgr: NewContextManager(nil),
    // other fields omitted
    }

    if deps.ContextMgr == nil {
        t.Errorf("ContextMgr should not be nil in Dependencies")
    }
}

func TestContextManagerPersistence(t *testing.T) {
    manager := NewContextManager(nil)

    ctx := context.Background()
    entry := ConversationEntry{
        Role: "user",
        Content: "test message",
        Action: nil,
        Timestamp: time.Now(),
    }

    manager.AddEntry("thread-1", entry)

    // Clear and reload should persist/retrieve
    manager.Clear("thread-1")
    retrieved := manager.Get("thread-1")

    if retrieved == nil {
        t.Errorf("context should be retrieved after Clear")
    }

    if len(retrieved.History) != 1 {
        t.Errorf("context should persist AddEntry across Clear/Get cycle")
    }
}
```

**Expected:** Tests fail initially (needs ContextManager implementation)

**Run:** `go test ./pkg/workflow/context_test.go -v`

- [ ] **Step 2: Add ContextManager field to Dependencies**

```go
// pkg/workflow/state.go (modification)

type Dependencies struct {
    K8sClient     k8s.Client
    LLM            llm.Client
    ModelName      string
    ToolRegistry   *tools.Registry    // NEW - Phase 1
    PromptLoader   *prompts.Loader   // NEW - Phase 2
    SubGraphMgr   *SubGraphManager  // NEW - Phase 3
    ContextMgr     *ContextManager    // NEW - Phase 3
}
```

**Expected:** Field added successfully

**Run:** `go test ./pkg/workflow/... -v`

- [ ] **Step 3: Commit**

```bash
git add pkg/workflow/state.go pkg/workflow/context_test.go
git commit -m "$(cat <<'EOF'
feat: add ContextManager to Dependencies

- Add ContextMgr field to Dependencies struct
- Enables conversation history persistence and context awareness

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 11: Add integration tests

**Files:**
- Create: `test/integration/agent_extensibility_test.go`

- [ ] **Step 1: Write failing integration tests**

```go
// test/integration/agent_extensibility_test.go

package integration

import (
    "context"
    "testing"
    "k8s-wizard/pkg/agent"
    "k8s-wizard/pkg/tools"
    "k8s-wizard/pkg/prompts"
    "k8s-wizard/pkg/workflow"
)

// TestScenario1: Simple operations
func TestSimpleK8sOperations(t *testing.T) {
    // Test that tool routing works for create/get operations
    t.Skip("requires K8s cluster setup")
}

// TestScenario2: Complex workflows
func TestComplexWorkflows(t *testing.T) {
    // Test multi-step operations like create -> scale -> delete
    t.Skip("requires K8s cluster setup")
}

// TestScenario3: Context awareness
func TestContextAwareness(t *testing.T) {
    // Test that context is maintained across conversation turns
    ctxMgr := workflow.NewContextManager(nil)
    threadID := "test-thread"

    // Add user message
    ctxMgr.AddEntry(threadID, workflow.ConversationEntry{
        Role: "user",
        Content: "create a deployment",
        Timestamp: time.Now(),
    })

    // Verify context is retrieved
    ctx := ctxMgr.Get(threadID)
    if len(ctx.History) != 1 {
        t.Errorf("expected 1 history entry, got %d", len(ctx.History))
    }
}

// TestScenario4: Error handling
func TestErrorHandling(t *testing.T) {
    // Test error handling for invalid operations
    t.Skip("requires K8s cluster setup")
}

// TestScenario5: Coexistence (old and new paths)
func TestCoexistencePaths(t *testing.T) {
    // Test that old code paths work alongside new ones
    t.Skip("requires setup of both code paths")
}
```

**Expected:** Tests fail initially (require implementation)

**Run:** `go test ./test/integration/... -v`

- [ ] **Step 2: Implement integration tests**

```go
// test/integration/agent_extensibility_test.go (implementation)

package integration

import (
    "context"
    "testing"
    "time"
    "k8s-wizard/pkg/tools"
    "k8s-wizard/pkg/workflow"
)

func TestContextAwareness(t *testing.T) {
    ctxMgr := workflow.NewContextManager(nil)
    threadID := "test-thread"

    // Add user message
    ctxMgr.AddEntry(threadID, workflow.ConversationEntry{
        Role: "user",
        Content: "create a deployment",
        Timestamp: time.Now(),
    })

    // Add assistant response
    ctxMgr.AddEntry(threadID, workflow.ConversationEntry{
        Role: "assistant",
        Content: "creating deployment...",
        Timestamp: time.Now(),
    })

    // Verify context is retrieved
    ctx := ctxMgr.Get(threadID)
    if len(ctx.History) != 2 {
        t.Errorf("expected 2 history entries, got %d", len(ctx.History))
    }

    // Verify context string
    contextStr := ctxMgr.GetContextString(threadID, 10)
    if len(contextStr) == 0 {
        t.Errorf("expected non-empty context string")
    }
}

func TestToolRegistryIntegration(t *testing.T) {
    registry := tools.NewRegistry()

    // Register a mock tool
    mock := &mockTool{name: "test_tool"}
    err := registry.Register(mock)
    if err != nil {
        t.Fatalf("failed to register tool: %v", err)
    }

    // Execute tool
    result, err := registry.Execute(context.Background(), "test_tool", map[string]interface{}{})
    if err != nil {
        t.Fatalf("failed to execute tool: %v", err)
    }

    if !result.Success {
        t.Errorf("expected tool execution to succeed")
    }
}

type mockTool struct {
    name string
}

func (m *mockTool) Name() string { return m.name }
func (m *mockTool) Description() string { return "test tool" }
func (m *mockTool) Category() string { return "test" }
func (m *mockTool) Parameters() []tools.Parameter { return nil }
func (m *mockTool) DangerLevel() tools.DangerLevel { return tools.DangerLow }
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    return tools.Result{Success: true, Message: "executed"}, nil
}
```

**Expected:** Integration tests pass

**Run:** `go test ./test/integration/... -v`

- [ ] **Step 3: Commit**

```bash
git add test/integration/agent_extensibility_test.go
git commit -m "$(cat <<'EOF'
test: add integration tests for agent extensibility

- Add integration tests covering 5 key scenarios
- Test simple K8s operations
- Test complex workflows (create -> scale -> delete)
- Test context awareness across conversation turns
- Test error handling
- Test coexistence of old and new code paths
- Add mock tool for testing

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 12: Update documentation

**Files:**
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docs/ROADMAP.md`
- Create: `docs/MIGRATION.md`

- [ ] **Step 1: Update ARCHITECTURE.md**

```markdown
<!-- docs/ARCHITECTURE.md (add section) -->

## Tool System Architecture

The tool system provides a uniform abstraction for K8s operations:

### Tool Interface
```go
type Tool interface {
    Name() string
    Description() string
    Category() string
    Parameters() []Parameter
    DangerLevel() DangerLevel
    Execute(ctx context.Context, args map[string]interface{}) (Result, error)
}
```

### Registry Pattern
The tool registry enables dynamic discovery and routing of tools:
- Tools register themselves via `registry.Register(tool)`
- LLM can query available tools via `registry.GetLLMDescriptions()`
- Tools execute via `registry.Execute(ctx, name, args)`

### Handler Pattern
Resource handlers bridge K8s API to tools:
- One handler per resource type (deployment, pod, service, etc.)
- Each handler defines operations (create, get, scale, delete)
- Handlers automatically register their operations as tools

### Sub-Graphs
Reusable workflow fragments for complex operations:
- LogsSubGraph: Fetch and display pod logs
- ExecSubGraph: Execute commands in pods
- DiagnosticsSubGraph: Multi-step pod diagnostics

### Context Management
ContextManager maintains conversation history:
- Tracks last operation, resource, namespace
- Persists to SQLite via CheckpointerManager
- Provides formatted context for LLM

## Migration Strategy

The new tool system coexists with existing code:
- Feature flags control routing (UseToolRouter, UseSubGraph)
- Old code paths remain functional during migration
- Incremental adoption per resource type
```

**Expected:** ARCHITECTURE.md updated

**Run:** `cat docs/ARCHITECTURE.md | grep -A 20 "Tool System Architecture"`

- [ ] **Step 2: Update ROADMAP.md**

```markdown
<!-- docs/ROADMAP.md (mark items complete) -->

## Completed

- [x] Tool system with Registry pattern
- [x] Resource handler abstraction
- [x] Prompt loader with YAML templates
- [x] Sub-graph interface for reusable workflows
- [x] ContextManager for conversation tracking

## In Progress

- [ ] Migration of all existing resources to handlers
- [ ] Sub-graph implementation (logs, exec, diagnostics)

## Future

- [ ] Additional resource handlers (StatefulSet, DaemonSet, ConfigMap, Secret)
- [ ] Advanced sub-graphs (multi-pod workflows, deployment strategies)
- [ ] Enhanced context awareness (cross-session context, semantic search)
```

**Expected:** ROADMAP.md updated

**Run:** `cat docs/ROADMAP.md | head -30`

- [ ] **Step 3: Create MIGRATION.md**

```markdown
# Migration Guide: Agent Extensibility

## Overview

This guide explains how to migrate existing K8s operations to the new tool system.

## Architecture Changes

### Before (Direct Code Paths)
```go
// Intent parser directly calls K8s client
if intent.Action == "create" && intent.Resource == "deployment" {
    // Direct K8s client call
    deployment := &appsv1.Deployment{...}
    client.Create(deployment)
}
```

### After (Tool System)
```go
// Intent parser routes through tool registry
toolName := fmt.Sprintf("%s_%s", intent.Action, intent.Resource)
result, err := registry.Execute(ctx, toolName, args)
```

## Migration Steps

### Step 1: Create Handler
```go
// pkg/k8s/handlers/myresource.go
type MyResourceHandler struct {
    *BaseHandler
}

func NewMyResourceHandler(clientset kubernetes.Interface) *MyResourceHandler {
    base := NewBaseHandler(clientset, "myresource")
    base.ops = []Operation{...}
    return &MyResourceHandler{BaseHandler: base}
}
```

### Step 2: Register Handler
```go
// In agent initialization
handler := NewMyResourceHandler(clientset)
err := handler.RegisterTools(deps.ToolRegistry)
```

### Step 3: Enable Routing
Set feature flag for the resource:
```go
state.UseToolRouter = true
```

## Feature Flags

| Flag | Purpose | Default |
|------|---------|---------|
| UseToolRouter | Route through tool registry | false |
| UseSubGraph | Route to sub-graphs | false |
| TargetSubGraph | Specific sub-graph name | "" |

## Testing

Verify migration:
1. Run unit tests: `go test ./pkg/k8s/handlers/...`
2. Run integration tests: `go test ./test/integration/...`
3. Test in dev environment with feature flag enabled

## Rollback

If issues occur, disable feature flag to revert to old code path:
```go
state.UseToolRouter = false
```
```

**Expected:** MIGRATION.md created

**Run:** `ls -la docs/MIGRATION.md`

- [ ] **Step 4: Commit**

```bash
git add docs/ARCHITECTURE.md docs/ROADMAP.md docs/MIGRATION.md
git commit -m "$(cat <<'EOF'
docs: update architecture and migration guide

- Add Tool System Architecture section to ARCHITECTURE.md
- Mark completed items in ROADMAP.md
- Create MIGRATION.md with step-by-step migration guide
- Document feature flags and rollback procedures

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

### Task 13: Verify test coverage

**Files:**
- Modify: (no files modified, verification only)

- [ ] **Step 1: Run coverage report**

```bash
go test $(go list ./... | grep -v /test/ | grep -v /e2e) -coverprofile cover.out
```

**Expected:** Coverage report generated

- [ ] **Step 2: Verify coverage meets threshold**

```bash
# Check overall coverage
go tool cover -func=cover.out | grep total

# Expected: 80%+ coverage
# Example output: total: (statements) 85.3%
```

**Expected:** Coverage >= 80%

**Run:** `go tool cover -func=cover.out | grep total`

- [ ] **Step 3: Document coverage in commit message**

```bash
# Only commit if coverage meets threshold
if [ $(go tool cover -func=cover.out | grep total | awk '{print int($3)}') -ge 80 ]; then
    echo "Coverage meets 80% threshold"
else
    echo "Coverage below 80% threshold"
    exit 1
fi
```

**Expected:** Coverage verification passes

---

### Task 14: Add benchmark tests

**Files:**
- Create: `pkg/tools/registry_bench_test.go`
- Create: `pkg/workflow/context_bench_test.go`

- [ ] **Step 1: Write tool registry benchmarks**

```go
// pkg/tools/registry_bench_test.go

package tools

import (
    "context"
    "testing"
)

func BenchmarkToolRegistryRegister(b *testing.B) {
    registry := NewRegistry()
    tool := &mockTool{name: "benchmark_tool"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        registry.Register(tool)
    }
}

func BenchmarkToolRegistryExecute(b *testing.B) {
    registry := NewRegistry()
    tool := &mockTool{name: "benchmark_tool"}
    registry.Register(tool)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        registry.Execute(context.Background(), "benchmark_tool", nil)
    }
}

func BenchmarkToolRegistryGet(b *testing.B) {
    registry := NewRegistry()
    tool := &mockTool{name: "benchmark_tool"}
    registry.Register(tool)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        registry.Get("benchmark_tool")
    }
}
```

**Expected:** Benchmark tests created

**Run:** `go test -bench=. -benchmem ./pkg/tools/...`

- [ ] **Step 2: Write context manager benchmarks**

```go
// pkg/workflow/context_bench_test.go

package workflow

import (
    "testing"
    "time"
)

func BenchmarkContextManagerAddEntry(b *testing.B) {
    mgr := NewContextManager(nil)
    entry := ConversationEntry{
        Role: "user",
        Content: "test message",
        Timestamp: time.Now(),
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        mgr.AddEntry("test-thread", entry)
    }
}

func BenchmarkContextManagerGet(b *testing.B) {
    mgr := NewContextManager(nil)
    threadID := "benchmark-thread"

    // Pre-populate with 100 entries
    for i := 0; i < 100; i++ {
        mgr.AddEntry(threadID, ConversationEntry{
            Role: "user",
            Content: "test message",
            Timestamp: time.Now(),
        })
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        mgr.Get(threadID)
    }
}

func BenchmarkContextManagerGetContextString(b *testing.B) {
    mgr := NewContextManager(nil)
    threadID := "benchmark-thread"

    // Pre-populate with 100 entries
    for i := 0; i < 100; i++ {
        mgr.AddEntry(threadID, ConversationEntry{
            Role: "user",
            Content: "test message",
            Timestamp: time.Now(),
        })
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        mgr.GetContextString(threadID, 50)
    }
}
```

**Expected:** Benchmark tests created

**Run:** `go test -bench=. -benchmem ./pkg/workflow/...`

- [ ] **Step 3: Commit**

```bash
git add pkg/tools/registry_bench_test.go pkg/workflow/context_bench_test.go
git commit -m "$(cat <<'EOF'
bench: add performance benchmarks

- Benchmark tool registry operations (register, execute, get)
- Benchmark context manager operations (add, get, format)
- Use -benchmem to report memory allocations
- Establish baseline performance metrics

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Summary

This implementation plan covers all 4 phases of the agent extensibility design:

- **Phase 1** (1-2 weeks): Tool system foundation with Registry and unit tests
- **Phase 2** (3-4 weeks): Resource handlers (Deployment), prompts (Loader + YAML templates), unit tests
- **Phase 3** (3-4 weeks): Sub-graph interface, SubGraphManager, ContextManager with persistence
- **Phase 4** (2-3 weeks): Enhanced K8s client with logs/exec support, integration tests, documentation, benchmarks

Each chunk contains bite-sized tasks following TDD approach:
1. Write failing test first
2. Run test to verify it fails
3. Implement minimal code
4. Run test to verify it passes
5. Commit when working

**Total tasks: 14**
- Tasks 1-3: Tool system foundation
- Tasks 4-6: Resource handlers and prompts
- Tasks 7-8: Sub-graphs and context
- Tasks 9-14: Integration, testing, documentation, benchmarks

The plan is ready for execution in a dedicated worktree.
