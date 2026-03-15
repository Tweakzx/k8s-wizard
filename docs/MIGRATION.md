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
