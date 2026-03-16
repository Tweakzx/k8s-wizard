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

#### Complete Deployment Handler Example

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

func (h *DeploymentHandler) create(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    name := args["name"].(string)
    ns := args["namespace"].(string)
    image := args["image"].(string)
    replicas := int32(args["replicas"].(int))

    // Validate
    if name == "" {
        return tools.Result{}, fmt.Errorf("name is required")
    }
    if image == "" {
        return tools.Result{}, fmt.Errorf("image is required")
    }

    // Create deployment spec
    deployment := &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: ns,
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{
                MatchLabels: map[string]string{"app": name},
            },
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{
                    Labels: map[string]string{"app": name},
                },
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{
                        {
                            Name:  name,
                            Image: image,
                            Ports: []corev1.ContainerPort{{ContainerPort: 80}},
                        },
                    },
                },
            },
        },
    }

    // Generate YAML preview
    yaml, err := generateDeploymentYAML(name, ns, image, int(replicas))
    if err != nil {
        return tools.Result{}, fmt.Errorf("failed to generate YAML: %w", err)
    }

    // Create via client-go
    _, err = h.clientset.AppsV1().Deployments(ns).Create(ctx, deployment, metav1.CreateOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("create deployment failed: %w", err)
    }

    return tools.Result{
        Success:     true,
        Message:     fmt.Sprintf("✓ Created Deployment %s/%s (replicas: %d, image: %s)", ns, name, replicas, image),
        Preview:     yaml,
        NeedsConfirm: false,
    }, nil
}

func (h *DeploymentHandler) get(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    ns, _ := args["namespace"].(string)
    deployments, err := h.clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("get deployments failed: %w", err)
    }

    if len(deployments.Items) == 0 {
        if ns == "" {
            return tools.Result{Success: true, Message: "No deployments found in cluster"}, nil
        }
        return tools.Result{Success: true, Message: fmt.Sprintf("No deployments found in namespace %s", ns)}, nil
    }

    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("🚀 Deployments (%d):\n", len(deployments.Items)))
    for _, dep := range deployments.Items {
        sb.WriteString(fmt.Sprintf("  • %s (replicas: %d/%d)\n", dep.Name, dep.Status.ReadyReplicas, dep.Status.Replicas))
    }

    return tools.Result{
        Success: true,
        Message: sb.String(),
        NeedsConfirm: false,
    }, nil
}

func (h *DeploymentHandler) scale(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    name := args["name"].(string)
    ns := args["namespace"].(string)
    replicas := int32(args["replicas"].(int))

    // Validate
    if name == "" {
        return tools.Result{}, fmt.Errorf("name is required")
    }

    // Get and update
    dep, err := h.clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("get deployment failed: %w", err)
    }

    dep.Spec.Replicas = &replicas
    _, err = h.clientset.AppsV1().Deployments(ns).Update(ctx, dep, metav1.UpdateOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("scale deployment failed: %w", err)
    }

    return tools.Result{
        Success:     true,
        Message:     fmt.Sprintf("✓ Scaled Deployment %s/%s to %d replicas", ns, name, replicas),
        NeedsConfirm: true,
    }, nil
}

func (h *DeploymentHandler) delete(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    name := args["name"].(string)
    ns := args["namespace"].(string)

    // Validate
    if name == "" {
        return tools.Result{}, fmt.Errorf("name is required")
    }

    err := h.clientset.AppsV1().Deployments(ns).Delete(ctx, name, metav1.DeleteOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("delete deployment failed: %w", err)
    }

    return tools.Result{
        Success:     true,
        Message:     fmt.Sprintf("✓ Deleted Deployment %s/%s", ns, name),
        DangerLevel: tools.DangerHigh,
        NeedsConfirm: true,
    }, nil
}

// YAML generation helper
func generateDeploymentYAML(name, namespace, image string, replicas int) string {
    return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s
        ports:
        - containerPort: 80
`, name, namespace, replicas, name, name, image)
}
```

#### Pod Handler Implementation

```go
// pkg/k8s/handlers/pod.go

package handlers

import (
    "context"
    "fmt"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s-wizard/pkg/tools"
)

type PodHandler struct {
    *BaseHandler
}

func NewPodHandler(clientset kubernetes.Interface) *PodHandler {
    base := NewBaseHandler(clientset, "pod")
    base.ops = []Operation{
        {
            Name:        "get",
            Method:      "get",
            DangerLevel: tools.DangerNone,
            Description: "List pods in a namespace",
            Params: []Param{
                {Key: "namespace", Type: "string", Required: false, Default: "default"},
            },
        },
        {
            Name:        "describe",
            Method:      "describe",
            DangerLevel: tools.DangerNone,
            Description: "Get detailed pod information",
            Params: []Param{
                {Key: "name", Type: "string", Required: true},
                {Key: "namespace", Type: "string", Required: false, Default: "default"},
            },
        },
        {
            Name:        "delete",
            Method:      "delete",
            DangerLevel: tools.DangerMedium,
            Description: "Delete a pod",
            Params: []Param{
                {Key: "name", Type: "string", Required: true},
                {Key: "namespace", Type: "string", Required: false, Default: "default"},
            },
        },
    }
    return &PodHandler{BaseHandler: base}
}

func (h *PodHandler) get(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    ns, _ := args["namespace"].(string)
    pods, err := h.clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("get pods failed: %w", err)
    }

    if len(pods.Items) == 0 {
        if ns == "" {
            return tools.Result{Success: true, Message: "No pods found in cluster"}, nil
        }
        return tools.Result{Success: true, Message: fmt.Sprintf("No pods found in namespace %s", ns)}, nil
    }

    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("📦 Pods (%d):\n", len(pods.Items)))
    for _, pod := range pods.Items {
        status := string(pod.Status.Phase)
        sb.WriteString(fmt.Sprintf("  • %s (status: %s, age: %s)\n", pod.Name, status, pod.CreationTimestamp.Format("2006-01-02")))
    }

    return tools.Result{Success: true, Message: sb.String()}, nil
}

func (h *PodHandler) describe(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    name := args["name"].(string)
    ns, _ := args["namespace"].(string)

    pod, err := h.clientset.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("get pod failed: %w", err)
    }

    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("📦 Pod: %s/%s\n", ns, name))
    sb.WriteString(fmt.Sprintf("Status: %s\n", pod.Status.Phase))
    sb.WriteString(fmt.Sprintf("Node: %s\n", pod.Spec.NodeName))
    sb.WriteString(fmt.Sprintf("IP: %s\n", pod.Status.PodIP))
    sb.WriteString("\nContainers:\n")
    for _, c := range pod.Spec.Containers {
        sb.WriteString(fmt.Sprintf("  - %s: %s\n", c.Name, c.Image))
    }

    return tools.Result{Success: true, Message: sb.String()}, nil
}

func (h *PodHandler) delete(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    name := args["name"].(string)
    ns, _ := args["namespace"].(string)

    err := h.clientset.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("delete pod failed: %w", err)
    }

    return tools.Result{
        Success:     true,
        Message:     fmt.Sprintf("✓ Deleted Pod %s/%s", ns, name),
        DangerLevel: tools.DangerMedium,
        NeedsConfirm: true,
    }, nil
}
```

#### Service Handler Implementation

```go
// pkg/k8s/handlers/service.go

package handlers

import (
    "context"
    "fmt"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/util/intstr"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
    "k8s-wizard/pkg/tools"
)

type ServiceHandler struct {
    *BaseHandler
}

func NewServiceHandler(clientset kubernetes.Interface) *ServiceHandler {
    base := NewBaseHandler(clientset, "service")
    base.ops = []Operation{
        {
            Name:        "create",
            Method:      "create",
            DangerLevel: tools.DangerLow,
            Description: "Create a service",
            Params: []Param{
                {Key: "name", Type: "string", Required: true},
                {Key: "namespace", Type: "string", Required: false, Default: "default"},
                {Key: "port", Type: "int", Required: false, Default: 80},
                {Key: "type", Type: "string", Required: false, Default: "ClusterIP"},
            },
        },
        {
            Name:        "get",
            Method:      "get",
            DangerLevel: tools.DangerNone,
            Description: "List services",
            Params: []Param{
                {Key: "namespace", Type: "string", Required: false, Default: "default"},
            },
        },
        {
            Name:        "delete",
            Method:      "delete",
            DangerLevel: tools.DangerMedium,
            Description: "Delete a service",
            Params: []Param{
                {Key: "name", Type: "string", Required: true},
                {Key: "namespace", Type: "string", Required: false, Default: "default"},
            },
        },
    }
    return &ServiceHandler{BaseHandler: base}
}

func (h *ServiceHandler) create(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    name := args["name"].(string)
    ns, _ := args["namespace"].(string)
    port := int32(args["port"].(int))
    svcType, _ := args["type"].(string)

    service := &corev1.Service{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: ns,
        },
        Spec: corev1.ServiceSpec{
            Type: corev1.ServiceType(svcType),
            Ports: []corev1.ServicePort{
                {Port: port, TargetPort: intstr.FromInt(int(port))},
            },
            Selector: map[string]string{"app": name},
        },
    }

    _, err := h.clientset.CoreV1().Services(ns).Create(ctx, service, metav1.CreateOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("create service failed: %w", err)
    }

    return tools.Result{
        Success: true,
        Message: fmt.Sprintf("✓ Created Service %s/%s (type: %s, port: %d)", ns, name, svcType, port),
    }, nil
}

func (h *ServiceHandler) get(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    ns, _ := args["namespace"].(string)
    services, err := h.clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("get services failed: %w", err)
    }

    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("🔌 Services (%d):\n", len(services.Items)))
    for _, svc := range services.Items {
        sb.WriteString(fmt.Sprintf("  • %s (type: %s, clusterIP: %s)\n", svc.Name, svc.Spec.Type, svc.Spec.ClusterIP))
    }

    return tools.Result{Success: true, Message: sb.String()}, nil
}

func (h *ServiceHandler) delete(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    name := args["name"].(string)
    ns, _ := args["namespace"].(string)

    err := h.clientset.CoreV1().Services(ns).Delete(ctx, name, metav1.DeleteOptions{})
    if err != nil {
        return tools.Result{}, fmt.Errorf("delete service failed: %w", err)
    }

    return tools.Result{
        Success: true,
        Message: fmt.Sprintf("✓ Deleted Service %s/%s", ns, name),
        DangerLevel: tools.DangerMedium,
        NeedsConfirm: true,
    }, nil
}
```

#### Logs Handler Implementation

```go
// pkg/k8s/handlers/logs.go

package handlers

import (
    "bytes"
    "context"
    "fmt"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/client-go/kubernetes"
    "k8s-wizard/pkg/tools"
)

type LogsHandler struct {
    *BaseHandler
}

func NewLogsHandler(clientset kubernetes.Interface) *LogsHandler {
    base := NewBaseHandler(clientset, "logs")
    base.ops = []Operation{
        {
            Name:        "get",
            Method:      "get",
            DangerLevel: tools.DangerNone,
            Description: "Get pod logs",
            Params: []Param{
                {Key: "pod", Type: "string", Required: true},
                {Key: "namespace", Type: "string", Required: false, Default: "default"},
                {Key: "tail", Type: "int", Required: false, Default: 100},
            },
        },
    }
    return &LogsHandler{BaseHandler: base}
}

func (h *LogsHandler) get(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    pod := args["pod"].(string)
    ns, _ := args["namespace"].(string)
    tail := int64(args["tail"].(int))

    req := h.clientset.CoreV1().Pods(ns).GetLogs(pod, &corev1.PodLogOptions{
        TailLines: &tail,
    })

    logs, err := req.Stream(ctx)
    if err != nil {
        return tools.Result{}, fmt.Errorf("get logs failed: %w", err)
    }
    defer logs.Close()

    buf := new(bytes.Buffer)
    if _, err := buf.ReadFrom(logs); err != nil {
        return tools.Result{}, fmt.Errorf("read logs failed: %w", err)
    }

    return tools.Result{
        Success: true,
        Message: fmt.Sprintf("📋 Logs from %s/%s:\n%s", ns, pod, buf.String()),
    }, nil
}
```

#### Exec Handler Implementation

```go
// pkg/k8s/handlers/exec.go

package handlers

import (
    "bytes"
    "context"
    "fmt"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/remotecommand"
    "k8s.io/client-go/util/exec"
    "k8s.io/client-go/util/homedir"
    "k8s.io/client-go/util/keyutil"
    "os"
    "path/filepath"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/client-go/kubernetes/scheme"
    "k8s-wizard/pkg/tools"
)

type ExecHandler struct {
    *BaseHandler
    config *rest.Config
}

func NewExecHandler(clientset kubernetes.Interface) *ExecHandler {
    base := NewBaseHandler(clientset, "exec")

    // Load kubeconfig for exec operations
    var config *rest.Config
    var err error

    kubeconfig := os.Getenv("KUBECONFIG")
    if kubeconfig == "" {
        home := homedir.HomeDir()
        kubeconfig = filepath.Join(home, ".kube", "config")
    }

    config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
    if err != nil {
        config, err = rest.InClusterConfig()
    }

    base.ops = []Operation{
        {
            Name:        "execute",
            Method:      "execute",
            DangerLevel: tools.DangerMedium,
            Description: "Execute command in pod",
            Params: []Param{
                {Key: "pod", Type: "string", Required: true},
                {Key: "namespace", Type: "string", Required: false, Default: "default"},
                {Key: "command", Type: "string", Required: true},
                {Key: "container", Type: "string", Required: false},
            },
        },
    }
    return &ExecHandler{BaseHandler: base, config: config}
}
    *BaseHandler
}

func NewExecHandler(clientset kubernetes.Interface) *ExecHandler {
    base := NewBaseHandler(clientset, "exec")
    base.ops = []Operation{
        {
            Name:        "execute",
            Method:      "execute",
            DangerLevel: tools.DangerMedium,
            Description: "Execute command in pod",
            Params: []Param{
                {Key: "pod", Type: "string", Required: true},
                {Key: "namespace", Type: "string", Required: false, Default: "default"},
                {Key: "command", Type: "string", Required: true},
                {Key: "container", Type: "string", Required: false},
            },
        },
    }
    return &ExecHandler{BaseHandler: base}
}

func (h *ExecHandler) execute(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
    pod := args["pod"].(string)
    ns, _ := args["namespace"].(string)
    command := args["command"].(string)
    container, _ := args["container"].(string)

    // Build exec request
    execOptions := &corev1.PodExecOptions{
        Command:   []string{"sh", "-c", command},
        Stdout:    true,
        Stderr:    true,
        Container: container,
    }

    req := h.clientset.CoreV1().RESTClient().Post().
        Resource("pods").
        Namespace(ns).
        Name(pod).
        SubResource("exec").
        VersionedParams(execOptions, scheme.ParameterCodec)

    executor, err := remotecommand.NewSPDYExecutor(h.config, "POST", req.URL())
    if err != nil {
        return tools.Result{}, fmt.Errorf("create executor failed: %w", err)
    }

    var stdout, stderr bytes.Buffer
    err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
        Stdout: &stdout,
        Stderr: &stderr,
    })
    if err != nil {
        return tools.Result{}, fmt.Errorf("exec failed: %w", err)
    }

    output := stdout.String()
    if stderr.String() != "" {
        output += "\nStderr:\n" + stderr.String()
    }

    return tools.Result{
        Success: true,
        Message: fmt.Sprintf("⚡ Exec in %s/%s:\n%s", ns, pod, output),
        DangerLevel: tools.DangerMedium,
        NeedsConfirm: true,
    }, nil
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

#### Intent Prompt Template

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

#### Tool Descriptions Template

```yaml
# prompts/tools.yaml

name: tool_descriptions
version: "1.0"
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

#### Prompt Loader

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

// ToolDescription represents a tool for LLM prompting.
type ToolDescription struct {
    Name        string            `yaml:"name"`
    Description string            `yaml:"description"`
    Category   string            `yaml:"category"`
    Parameters []PromptParameter `yaml:"parameters"`
}

// PromptParameter represents a parameter in prompt.
type PromptParameter struct {
    Name        string      `yaml:"name"`
    Type        string      `yaml:"type"`
    Description string      `yaml:"description"`
    Required    bool        `yaml:"required"`
    Default     interface{} `yaml:"default,omitempty"`
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

// GetIntentPrompt returns intent parsing prompt.
func (l *Loader) GetIntentPrompt(userMessage string, toolRegistry *tools.Registry) (string, error) {
    prompt, ok := l.prompts["intent"]
    if !ok {
        return "", fmt.Errorf("intent prompt not found")
    }

    // Prepare template data
    data := make(map[string]interface{})
    data["UserMessage"] = userMessage

    // Use tool descriptions from registry (preferred) or fallback to loaded from YAML
    data["ToolDescriptions"] = toolRegistry.GetLLMDescriptions()

    // Render user prompt with data
    tmpl, err := template.New("intent").Parse(prompt.User)
    if err != nil {
        return "", fmt.Errorf("failed to parse prompt template: %w", err)
    }

    var rendered strings.Builder
    if err := tmpl.Execute(&rendered, data); err != nil {
        return "", fmt.Errorf("failed to render prompt: %w", err)
    }

    // Combine system and user prompts
    fullPrompt := prompt.System + "\n\n" + rendered.String()
    return fullPrompt, nil
}

// UpdateFromRegistry updates tool descriptions from registry.
func (l *Loader) UpdateFromRegistry(registry *tools.Registry) {
    // Rebuild tool descriptions from registry
    // This allows tools to be defined in code, not YAML
    tools := registry.List()

    l.categories = make(map[string][]ToolDescription)
    for _, tool := range tools {
        desc := ToolDescription{
            Name:        tool.Name(),
            Description: tool.Description(),
            Category:   tool.Category(),
        }
        for _, param := range tool.Parameters() {
            desc.Parameters = append(desc.Parameters, PromptParameter{
                Name:        param.Name,
                Type:        param.Type,
                Description: param.Description,
                Required:    param.Required,
                Default:     param.Default,
            })
        }

        l.categories[tool.Category()] = append(l.categories[tool.Category()], desc)
    }
}

// GetToolDescriptions returns formatted tool descriptions.
func (l *Loader) GetToolDescriptions(category string) []ToolDescription {
    if category == "" {
        // Return all tools
        var all []ToolDescription
        for _, tools := range l.categories {
            all = append(all, tools...)
        }
        return all
    }
    return l.categories[category]
}
```

#### Intent Parsing Integration

The `GetLLMDescriptions()` method from the tool registry formats tools for the LLM:

```go
// In the prompt loader's GetIntentPrompt method:
data["ToolDescriptions"] = toolRegistry.GetLLMDescriptions()
```

This ensures that when tools are registered in code (via handlers), they are automatically available for the LLM to reference. The LLM's response includes `tool_name` which is used to route to the appropriate tool or sub-graph.

### 5. Workflow Enhancements

#### Dependencies Structure Update

```go
// pkg/workflow/state.go (updated for new dependencies)

// Dependencies holds the dependencies needed by workflow nodes.
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

#### Sub-Graph Pattern

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

// LogsSubGraph implements logs viewing workflow.
type LogsSubGraph struct{}

func NewLogsSubGraph() *LogsSubGraph {
    return &LogsSubGraph{}
}

func (g *LogsSubGraph) Name() string {
    return "logs"
}

func (g *LogsSubGraph) Entry() string {
    return "validate_params"
}

func (g *LogsSubGraph) Exit() string {
    return lgg.END
}

func (g *LogsSubGraph) Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error) {
    sub := lgg.NewStateGraph[AgentState]()

    // Add nodes
    sub.AddNode("validate_params", "Validate log parameters",
        MakeValidateLogParamsNode())
    sub.AddNode("get_logs", "Fetch pod logs",
        MakeGetLogsNode(deps.K8sClient))
    sub.AddNode("format_logs", "Format and paginate logs",
        MakeFormatLogsNode())

    // Set entry
    sub.SetEntryPoint("validate_params")

    // Add conditional edge
    sub.AddConditionalEdge("validate_params", RouteAfterLogValidation)

    // Compile
    return sub.Compile()
}

// ExecSubGraph implements container command execution workflow.
type ExecSubGraph struct{}

func NewExecSubGraph() *ExecSubGraph {
    return &ExecSubGraph{}
}

func (g *ExecSubGraph) Name() string {
    return "exec"
}

func (g *ExecSubGraph) Entry() string {
    return "validate_params"
}

func (g *ExecSubGraph) Exit() string {
    return lgg.END
}

func (g *ExecSubGraph) Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error) {
    sub := lgg.NewStateGraph[AgentState]()

    // Add nodes
    sub.AddNode("validate_params", "Validate exec parameters",
        MakeValidateExecParamsNode())
    sub.AddNode("confirm_command", "Confirm dangerous command",
        MakeConfirmCommandNode())
    sub.AddNode("execute_command", "Execute command in container",
        MakeExecuteCommandNode(deps.K8sClient))
    sub.AddNode("stream_output", "Stream command output",
        MakeStreamOutputNode())

    // Set entry
    sub.SetEntryPoint("validate_params")

    // Add conditional edges
    sub.AddConditionalEdge("validate_params", RouteAfterExecValidation)
    sub.AddConditionalEdge("confirm_command", RouteAfterCommandConfirm)

    // Compile
    return sub.Compile()
}

// DiagnosticsSubGraph implements comprehensive diagnostics workflow.
type DiagnosticsSubGraph struct{}

func NewDiagnosticsSubGraph() *DiagnosticsSubGraph {
    return &DiagnosticsSubGraph{}
}

func (g *DiagnosticsSubGraph) Name() string {
    return "diagnostics"
}

func (g *DiagnosticsSubGraph) Entry() string {
    return "validate_target"
}

func (g *DiagnosticsSubGraph) Exit() string {
    return lgg.END
}

func (g *DiagnosticsSubGraph) Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error) {
    sub := lgg.NewStateGraph[AgentState]()

    // Add nodes
    sub.AddNode("validate_target", "Validate diagnostic target",
        MakeValidateDiagnosticTargetNode())
    sub.AddNode("check_pod_status", "Check pod status",
        MakeCheckPodStatusNode(deps.K8sClient))
    sub.AddNode("check_events", "Check recent events",
        MakeCheckEventsNode(deps.K8sClient))
    sub.AddNode("check_logs", "Check pod logs",
        MakeCheckLogsNode(deps.K8sClient))
    sub.AddNode("check_resources", "Check resource usage",
        MakeCheckResourcesNode(deps.K8sClient))
    sub.AddNode("compile_report", "Compile diagnostic report",
        MakeCompileDiagnosticReportNode())

    // Set entry
    sub.SetEntryPoint("validate_target")

    // Add edges
    sub.AddEdge("validate_target", "check_pod_status")
    sub.AddEdge("check_pod_status", "check_events")
    sub.AddEdge("check_events", "check_logs")
    sub.AddEdge("check_logs", "check_resources")
    sub.AddEdge("check_resources", "compile_report")

    // Compile
    return sub.Compile()
}

// Sub-Graph Manager

// pkg/workflow/subgraph_manager.go

package workflow

import (
    "fmt"
    lgg "github.com/smallnest/langgraphgo/graph"
)

// SubGraphManager manages available sub-graphs.
type SubGraphManager struct {
    subgraphs map[string]SubGraph
}

// NewSubGraphManager creates a sub-graph manager.
func NewSubGraphManager() *SubGraphManager {
    return &SubGraphManager{
        subgraphs: make(map[string]SubGraph),
    }
}

// Register adds a sub-graph to the manager.
func (m *SubGraphManager) Register(subgraph SubGraph) error {
    if _, exists := m.subgraphs[subgraph.Name()]; exists {
        return fmt.Errorf("sub-graph %s already registered", subgraph.Name())
    }
    m.subgraphs[subgraph.Name()] = subgraph
    return nil
}

// Get retrieves a sub-graph by name.
func (m *SubGraphManager) Get(name string) (SubGraph, bool) {
    sg, exists := m.subgraphs[name]
    return sg, exists
}

// List returns all registered sub-graphs.
func (m *SubGraphManager) List() []SubGraph {
    result := make([]SubGraph, 0, len(m.subgraphs))
    for _, sg := range m.subgraphs {
        result = append(result, sg)
    }
    return result
}

// InitializeStandardSubGraphs registers all standard sub-graphs.
func (m *SubGraphManager) InitializeStandardSubGraphs(deps *Dependencies) error {
    subgraphs := []SubGraph{
        NewLogsSubGraph(),
        NewExecSubGraph(),
        NewDiagnosticsSubGraph(),
    }

    for _, sg := range subgraphs {
        _, err := sg.Build(deps)
        if err != nil {
            return fmt.Errorf("failed to build sub-graph %s: %w", sg.Name(), err)
        }
        if err := m.Register(sg); err != nil {
            return err
        }
    }

    return nil
}
```

#### Example Sub-Graph Nodes

```go
// pkg/workflow/logs_nodes.go

package workflow

import (
    "context"
    "fmt"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/client-go/kubernetes"
    "k8s-wizard/pkg/k8s"
)

// MakeValidateLogParamsNode validates log parameters.
func MakeValidateLogParamsNode() NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        if state.Action == nil {
            return state, nil
        }

        // Check required parameters
        if state.Action.Name == "" {
            state.Error = fmt.Errorf("pod name is required for logs")
            state.Status = StatusError
            return state, nil
        }

        return state, nil
    }
}

// MakeGetLogsNode fetches pod logs.
func MakeGetLogsNode(client k8s.Client) NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        // Extract parameters
        name := state.Action.Name
        ns := state.Action.Namespace
        if ns == "" {
            ns = "default"
        }

        // Optional parameters
        tailLines := 100
        if tl, ok := state.Action.Params["tailLines"].(int); ok {
            tailLines = tl
        }
        container := ""
        if c, ok := state.Action.Params["container"].(string); ok {
            container = c
        }

        logs, err := client.GetPodLogs(ctx, ns, name, container, tailLines)
        if err != nil {
            state.Error = err
            state.Status = StatusError
            return state, nil
        }

        state.Result = logs
        state.Status = StatusExecuted
        return state, nil
    }
}
```

#### Routing Helper Functions

```go
// pkg/workflow/routing.go (additions)

import (
    "context"
    "fmt"
    "strings"
    corev1 "k8s.io/api/core/v1"
    lgg "github.com/smallnest/langgraphgo/graph"
    "k8s-wizard/pkg/k8s"
)

// buildToolArgs converts K8sAction to tool arguments.
func buildToolArgs(action *K8sAction) map[string]interface{} {
    args := make(map[string]interface{})
    args["namespace"] = action.Namespace
    args["name"] = action.Name
    for k, v := range action.Params {
        args[k] = v
    }
    return args
}

// RouteAfterLogValidation determines next node after log validation.
func RouteAfterLogValidation(ctx context.Context, state AgentState) string {
    if state.Error != nil {
        return lgg.END
    }
    return "get_logs"
}

// RouteAfterExecValidation determines next node after exec validation.
func RouteAfterExecValidation(ctx context.Context, state AgentState) string {
    if state.Error != nil {
        return lgg.END
    }

    // Dangerous commands need confirmation
    if isDangerousCommand(state.Action) {
        return "confirm_command"
    }
    return "execute_command"
}

func isDangerousCommand(action *K8sAction) bool {
    // Check for dangerous commands (rm, rm -rf, etc.)
    if command, ok := action.Params["command"].(string); ok {
        dangerous := []string{"rm", "rm -rf", "dd", "mkfs", "fdisk"}
        for _, d := range dangerous {
            if strings.Contains(command, d) {
                return true
            }
        }
    }
    return false
}

// RouteAfterCommandConfirm determines next node after command confirmation.
func RouteAfterCommandConfirm(ctx context.Context, state AgentState) string {
    if state.Confirm == nil || !*state.Confirm {
        return lgg.END // Wait for user confirmation
    }
    return "execute_command"
}

// Diagnostic Sub-Graph Nodes

// MakeValidateDiagnosticTargetNode validates diagnostic target parameters.
func MakeValidateDiagnosticTargetNode() NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        if state.Action == nil {
            return state, nil
        }

        // Check required parameters
        if state.Action.Name == "" {
            state.Error = fmt.Errorf("pod name is required for diagnostics")
            state.Status = StatusError
            return state, nil
        }

        return state, nil
    }
}

// MakeCheckPodStatusNode checks pod status.
func MakeCheckPodStatusNode(client k8s.Client) NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        name := state.Action.Name
        ns := state.Action.Namespace
        if ns == "" {
            ns = "default"
        }

        pod, err := client.GetPod(ctx, ns, name)
        if err != nil {
            state.Error = err
            state.Status = StatusError
            return state, nil
        }

        var sb strings.Builder
        sb.WriteString(fmt.Sprintf("Pod Status: %s\n", string(pod.Status.Phase)))
        sb.WriteString(fmt.Sprintf("Pod IP: %s\n", pod.Status.PodIP))
        sb.WriteString(fmt.Sprintf("Node: %s\n", pod.Spec.NodeName))
        sb.WriteString(fmt.Sprintf("Ready: %d/%d\n",
            countReadyContainers(pod.Status.ContainerStatuses),
            len(pod.Spec.Containers)))

        state.Result = sb.String()
        return state, nil
    }
}

// MakeCheckEventsNode checks recent events for the pod.
func MakeCheckEventsNode(client k8s.Client) NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        name := state.Action.Name
        ns := state.Action.Namespace
        if ns == "" {
            ns = "default"
        }

        events, err := client.GetEvents(ctx, ns, name)
        if err != nil {
            state.Error = err
            state.Status = StatusError
            return state, nil
        }

        var sb strings.Builder
        sb.WriteString("Recent Events:\n")
        for _, event := range events {
            sb.WriteString(fmt.Sprintf("  %s: %s (%s)\n",
                event.LastTimestamp.Format("15:04:05"),
                event.Message,
                event.Type))
        }

        state.Result += "\n" + sb.String()
        return state, nil
    }
}

// MakeCheckLogsNode checks pod logs (last 50 lines).
func MakeCheckLogsNode(client k8s.Client) NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        name := state.Action.Name
        ns := state.Action.Namespace
        if ns == "" {
            ns = "default"
        }

        logs, err := client.GetPodLogs(ctx, ns, name, "", 50)
        if err != nil {
            state.Error = err
            state.Status = StatusError
            return state, nil
        }

        state.Result += "\nRecent Logs (last 50 lines):\n" + logs
        return state, nil
    }
}

// MakeCheckResourcesNode checks resource usage.
func MakeCheckResourcesNode(client k8s.Client) NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        name := state.Action.Name
        ns := state.Action.Namespace
        if ns == "" {
            ns = "default"
        }

        metrics, err := client.GetPodMetrics(ctx, ns, name)
        if err != nil {
            state.Error = err
            state.Status = StatusError
            return state, nil
        }

        var sb strings.Builder
        sb.WriteString("Resource Usage:\n")
        for _, container := range metrics.Containers {
            sb.WriteString(fmt.Sprintf("  %s:\n", container.Name))
            sb.WriteString(fmt.Sprintf("    CPU: %s\n", container.Usage.Cpu().String()))
            sb.WriteString(fmt.Sprintf("    Memory: %s\n", container.Usage.Memory().String()))
        }

        state.Result += "\n" + sb.String()
        return state, nil
    }
}

// MakeCompileDiagnosticReportNode compiles all diagnostic information into a report.
func MakeCompileDiagnosticReportNode() NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        // All diagnostic information is already accumulated in state.Result
        state.Status = StatusExecuted
        return state, nil
    }
}

// Helper function to count ready containers
func countReadyContainers(statuses []corev1.ContainerStatus) int {
    ready := 0
    for _, cs := range statuses {
        if cs.Ready {
            ready++
        }
    }
    return ready
}
```

#### Type Definitions (From Existing Codebase)

```go
// pkg/workflow/state.go (existing types - referenced throughout spec)

type K8sAction struct {
    Action         string                 `json:"action"`
    Resource       string                 `json:"resource"`
    Name           string                 `json:"name"`
    Namespace      string                 `json:"namespace"`
    Params         map[string]interface{} `json:"params"`
    IsK8sOperation bool                   `json:"is_k8s_operation"`
    Reply          string                 `json:"reply"`
}

const (
    StatusPending      = "pending"
    StatusNeedsInfo    = "needs_info"
    StatusNeedsConfirm = "needs_confirm"
    StatusExecuted     = "executed"
    StatusError        = "error"
    StatusChat         = "chat"
)

// api/models/types (existing models referenced)

// From api/models/requests.go:
type ClarificationRequest struct { ... }
type ActionPreview struct { ... }
```

#### Context-Aware State Management

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
        `, threadID, entry.Role, entry.Content, entry.Timestamp.Format(time.RFC3339), actionJSON)
        if err != nil {
            // Log error but continue
            continue
        }
    }
}

// Helper function to parse action from JSON (simplified)
func parseActionFromJSON(jsonStr string) *K8sAction {
    // Simplified parser - in production use encoding/json
    action := &K8sAction{
        Action:    "",
        Resource:  "",
        Namespace: "",
    }
    // Parse the JSON string and populate action fields
    // This is a placeholder for actual JSON parsing
    return action
}
```

#### Handler Registry Implementation

```go
// pkg/k8s/handlers/registry.go

package handlers

import (
    "fmt"
    "k8s.io/client-go/kubernetes"
    "k8s-wizard/pkg/tools"
)

// HandlerRegistry manages all K8s resource handlers.
type HandlerRegistry struct {
    handlers map[string]Handler
}

// NewHandlerRegistry creates an empty handler registry.
func NewHandlerRegistry() *HandlerRegistry {
    return &HandlerRegistry{
        handlers: make(map[string]Handler),
    }
}

// Register adds a handler to the registry.
func (r *HandlerRegistry) Register(handler Handler) error {
    r.handlers[handler.Resource()] = handler
    return nil
}

// RegisterWithTools registers a handler and all its tools.
func (r *HandlerRegistry) RegisterWithTools(handler Handler, toolRegistry *tools.Registry) error {
    // Register the handler first
    if err := r.Register(handler); err != nil {
        return err
    }

    // Type assertion to get RegisterTools method
    // Handlers must implement RegisterTools(registry *tools.Registry) error
    type registrable interface {
        RegisterTools(*tools.Registry) error
    }

    if reg, ok := handler.(registrable); ok {
        return reg.RegisterTools(toolRegistry)
    }

    return fmt.Errorf("handler does not support tool registration: %s", handler.Resource())
}

// Get retrieves a handler by resource type.
func (r *HandlerRegistry) Get(resource string) (Handler, bool) {
    handler, exists := r.handlers[resource]
    return handler, exists
}

// InitializeStandardHandlers registers all standard K8s handlers.
func (r *HandlerRegistry) InitializeStandardHandlers(clientset kubernetes.Interface, toolRegistry *tools.Registry) error {
    handlers := []Handler{
        NewDeploymentHandler(clientset),
        NewPodHandler(clientset),
        NewServiceHandler(clientset),
    }

    for _, handler := range handlers {
        if err := r.RegisterWithTools(handler, toolRegistry); err != nil {
            return fmt.Errorf("failed to register %s tools: %w", handler.Resource(), err)
        }
    }

    return nil
}
```

#### Complete Sub-Graph Node Implementations

```go
// pkg/workflow/logs_nodes.go (additional nodes)

// MakeFormatLogsNode formats logs for display.
func MakeFormatLogsNode() NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        if state.Result == "" {
            return state, nil
        }

        // Format logs for display
        // Add line numbers, timestamps, etc.
        formatted := state.Result

        state.Result = formatted
        state.Status = StatusExecuted
        return state, nil
    }
}

// pkg/workflow/exec_nodes.go (additional nodes)

// MakeValidateExecParamsNode validates exec parameters.
func MakeValidateExecParamsNode() NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        if state.Action == nil {
            return state, nil
        }

        // Check required parameters
        if state.Action.Name == "" {
            state.Error = fmt.Errorf("pod name is required for exec")
            state.Status = StatusError
            return state, nil
        }

        command, _ := state.Action.Params["command"].(string)
        if command == "" {
            state.Error = fmt.Errorf("command is required for exec")
            state.Status = StatusError
            return state, nil
        }

        return state, nil
    }
}

// MakeConfirmCommandNode generates confirmation prompt for dangerous commands.
func MakeConfirmCommandNode() NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        if state.Action == nil {
            return state, nil
        }

        command, _ := state.Action.Params["command"].(string)

        if !isDangerousCommand(state.Action) {
            // Safe command, skip confirmation
            return state, nil
        }

        // Generate confirmation request
        state.ClarificationRequest = &models.ClarificationRequest{
            Type:   "confirm",
            Title:  "⚠️ 危险操作确认",
            Action: "exec",
            Fields: []models.ClarificationField{
                {
                    Key:         "confirm",
                    Label:       "确认执行此命令",
                    Type:        "boolean",
                    Required:    true,
                    Group:       "basic",
                },
            },
        }
        state.NeedsClarification = true
        state.Status = StatusNeedsConfirm

        return state, nil
    }
}

// MakeExecuteCommandNode executes command in container.
func MakeExecuteCommandNode(client k8s.Client) NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        command, _ := state.Action.Params["command"].(string)
        container, _ := state.Action.Params["container"].(string)
        name := state.Action.Name
        ns := state.Action.Namespace
        if ns == "" {
            ns = "default"
        }

        output, err := client.ExecPod(ctx, ns, name, container, command)
        if err != nil {
            state.Error = err
            state.Status = StatusError
            return state, nil
        }

        state.Result = output
        state.Status = StatusExecuted
        return state, nil
    }
}

// MakeStreamOutputNode streams command output.
func MakeStreamOutputNode() NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        // Output is already in state.Result from ExecuteCommandNode
        state.Status = StatusExecuted
        return state, nil
    }
}
```

#### Sub-Graph Routing Logic

```go
// pkg/workflow/graph.go (updated section)

func mapActionToSubGraph(action *K8sAction) string {
    // Map actions to their complex workflows
    switch action.Action {
    case "logs":
        return "logs"
    case "exec":
        return "exec"
    case "diagnose":
        return "diagnostics"
    default:
        return ""
    }
}

// Updated routing after intent parsing
func RouteAfterParse(ctx context.Context, state AgentState) string {
    // If there was an error, end
    if state.Error != nil {
        return lgg.END
    }

    // If not a K8s operation, return chat reply
    if !state.IsK8sOperation {
        return lgg.END
    }

    // Check if this action maps to a sub-graph
    subGraphName := mapActionToSubGraph(state.Action)
    if subGraphName != "" {
        // Set a flag to indicate sub-graph should be used
        state.UseSubGraph = true
        state.TargetSubGraph = subGraphName
    }

    // Otherwise, proceed to merge form data
    return "merge_form"
}
```

#### State Update for Sub-Graph Support

```go
// pkg/workflow/state.go (updated)

type AgentState struct {
    // === Existing fields ===
    UserMessage string
    FormData    map[string]interface{}
    Confirm     *bool
    ThreadID    string

    Action         *K8sAction
    IsK8sOperation bool

    ClarificationRequest *models.ClarificationRequest
    NeedsClarification   bool

    ActionPreview *models.ActionPreview

    Result string
    Error  error

    Status string

    Reply string

    // === New fields for tool routing ===
    UseToolRouter   bool   // Whether to route through tool registry
    UseSubGraph     bool   // Whether to route to a sub-graph
    TargetSubGraph  string // Name of target sub-graph
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

### Phase 2: Resource Handlers & Prompts (3-4 weeks)

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

### Phase 3: Sub-Graphs & Context (3-4 weeks)

**Goal**: Add complex workflow support and context awareness.

| Task | Effort | Dependencies |
|------|---------|--------------|
| Implement `SubGraph` interface | 1 day | None |
| Implement `LogsSubGraph` | 3 days | Interface |
| Implement `ExecSubGraph` | 3 days | Logs |
| Implement `DiagnosticsSubGraph` | 3 days | Exec |
| Create `SubGraphManager` | 1 day | All sub-graphs |
| Implement `ContextManager` | 2 days | None |
| Update routing for sub-graphs | 1 day | Manager |
| Add unit tests | 2 days | All components |
| **Total** | **~3.5 weeks** | |

**Milestone**: Complex workflows and context awareness working.

### Phase 4: Integration & Testing (2-3 weeks)

**Goal**: Ensure everything works together.

| Task | Effort | Dependencies |
|------|---------|--------------|
| Update main graph routing | 1 day | All phases complete |
| Add sub-graph routing node | 1 day | Graph updated |
| Coexistence testing | 2 days | Routing updated |
| End-to-end integration tests | 3 days | All phases complete |
| Performance benchmarking | 1 day | Integration working |
| Update documentation | 2 days | Integration working |
| Update ARCHITECTURE.md | 1 day | Code complete |
| Update ROADMAP.md | 0.5 day | Architecture updated |
| Create migration guide | 1 day | Documentation updated |
| **Total** | **~2.5 weeks** | |

**Milestone**: Production-ready extensibility system.

#### Coexistence Strategy

During migration, old and new code paths should coexist:

```go
// pkg/workflow/routing.go (updated)

func RouteAfterParse(ctx context.Context, state AgentState) string {
    // If there was an error, end
    if state.Error != nil {
        return lgg.END
    }

    // If not a K8s operation, return chat reply
    if !state.IsK8sOperation {
        return lgg.END
    }

    // New path: Use tool registry if available
    if state.Deps.ToolRegistry != nil {
        // Set flag for tool router node
        state.UseToolRouter = true
        return "merge_form"
    }

    // Legacy path: Use existing routing
    return "merge_form"
}

func MakeToolRouterNode(registry *tools.Registry) NodeFunc {
    return func(ctx context.Context, state AgentState) (AgentState, error) {
        // Only execute via tool router if flag is set
        if !state.UseToolRouter || state.Action == nil {
            return state, nil
        }

        // Convert K8sAction to tool execution
        toolName := fmt.Sprintf("%s_%s", state.Action.Action, state.Action.Resource)
        args := buildToolArgs(state.Action)

        result, err := registry.Execute(ctx, toolName, args)
        if err != nil {
            state.Error = err
            state.Status = StatusError
            return state, nil
        }

        state.Result = result.Message
        if result.Preview != "" {
            state.ActionPreview = &models.ActionPreview{
                YAML:        result.Preview,
                DangerLevel: string(result.DangerLevel),
                Summary:     result.Message,
            }
        }

        state.Status = StatusExecuted
        return state, nil
    }
}
```

This gradual migration approach allows:
- Old code to work until fully tested
- New features to use tools immediately
- Feature flags to control routing
- Safe rollback if issues arise

---

## Assumptions

This design makes several assumptions about the existing codebase:

1. **LangGraphGo Sub-Graph Support**
   - Assumption: LangGraphGo supports nested sub-graphs or can implement them via routing pattern
   - If not supported: The routing pattern (manual sub-graph invocation) will be used

2. **Checkpoint Storage**
   - Assumption: Existing `CheckpointerManager` from `pkg/workflow/checkpointer.go` provides session persistence
   - Context persistence will use the same mechanism or in-memory storage if unavailable

3. **Existing Type Definitions**
   - Assumption: `K8sAction`, `AgentState`, `models.*` types already exist in codebase and are stable
   - New fields will be added to `AgentState` (UseSubGraph, TargetSubGraph)

4. **LLM Integration**
   - Assumption: LLM client can handle text-only prompts (no streaming requirement for Phase 1-2)
   - Tool descriptions will be injected into prompts as plain text

5. **Testing Infrastructure**
   - Assumption: Existing test setup (Kind cluster, test utilities) is available
   - No additional infrastructure required for unit tests

6. **Deployment Model**
   - Assumption: Single K8s cluster with standard client-go configuration
   - No multi-cluster or RBAC constraints in current environment

7. **Go Version**
   - Assumption: Go 1.24+ is available (already in use per go.mod)
   - Generics and other modern Go features can be used if beneficial

If any of these assumptions are incorrect, the design may need adjustment during implementation.

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

## Testing Strategy

### Unit Testing

Each component should have unit tests covering:

| Component | Coverage Goal | Key Scenarios |
|-----------|----------------|----------------|
| `pkg/tools/` | 90%+ | Registry, tool registration, execution, LLM description formatting |
| `pkg/prompts/` | 80%+ | Template loading, rendering, error handling |
| `pkg/k8s/handlers/` | 85%+ | Each operation, validation, edge cases |
| `pkg/workflow/context/` | 80%+ | Context retrieval, history management, truncation |
| `pkg/workflow/subgraph_*` | 85%+ | Sub-graph execution, routing, error handling |

### Integration Testing

End-to-end scenarios:

1. **Simple Operations**: Create, get, scale, delete deployments
2. **Complex Workflows**: Logs with filtering, exec with command validation
3. **Context Awareness**: Multi-turn conversations with references ("that one", "it")
4. **Error Handling**: Invalid parameters, missing resources, API failures
5. **Coexistence**: Verify old and new paths both work during migration

### Performance Benchmarks

Target performance for each operation:

| Operation | Target | Notes |
|-----------|--------|-------|
| Tool registration | < 1ms | Should be O(1) |
| Tool execution (get) | < 500ms | Read operations only |
| Tool execution (create) | < 2s | Depends on K8s API |
| Context retrieval | < 10ms | Memory-based lookup |
| Prompt rendering | < 5ms | Simple template substitution |

---

## Non-Goals (What This Design Does NOT Address)

This design focuses on extensibility through incremental refactoring. The following are explicitly **NOT** goals of this design:

1. **Dynamic Tool Discovery via Reflection/Plugins**
   - Tools are defined in Go code at compile time
   - No runtime plugin loading or hot-swapping
   - Adding a new tool requires code recompilation

2. **External Tool Definitions**
   - Tool definitions come from handler code, not YAML/JSON files
   - LLM prompt templates are externalized, but tool metadata is code-based
   - Future: Could evolve to external tool definitions

3. **Multi-Cluster Support**
   - This design maintains single-cluster K8s client
   - Future: Cluster-aware context and routing

4. **Authentication/RBAC**
   - No permission-aware routing or user-based tool filtering
   - Future: Permission checks before tool execution

5. **Streaming Response**
   - LLM calls are blocking (request/response pattern)
   - Future: Server-sent events or WebSockets for streaming

6. **Operation History/Undo**
   - Context management tracks conversation but not all operations
   - Future: Full audit log with undo capability

7. **Resource Templates**
   - Operations use programmatic resource creation
   - Future: YAML template system for complex resources

These are areas that could be addressed in future iterations but would increase complexity beyond what's appropriate for the current scope and team size.

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

**Total Timeline**: 10-13.5 weeks for full implementation
**Approach**: Incremental, non-breaking, maintainable for solo developer
**Philosophy**: Add, don't replace; interface-based; backward compatible

---

**Next Steps**:
1. Review this design document
2. Create implementation plan using `superpowers:writing-plans` skill
3. Begin with Phase 1: Tool System Foundation
