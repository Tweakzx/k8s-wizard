# K8s Wizard Development Roadmap

## 📊 Current Status (v0.1.0)

### ✅ Implemented Features

| Feature | Status | Description |
|---------|--------|-------------|
| **Natural Language K8s Operations** | ✅ | Deploy, scale, delete, list resources via chat |
| **Multi-LLM Support** | ✅ | GLM, DeepSeek, Claude (dynamic model switching) |
| **Dynamic Model Discovery** | ✅ | Fetch available models from provider API |
| **Smart Clarification Flow** | ✅ | Form-based info collection when input incomplete |
| **Action Preview** | ✅ | YAML preview before execution |
| **Web UI** | ✅ | React + Tailwind CSS chat interface |
| **Model Selector** | ✅ | Switch models in frontend |
| **API Key Filtering** | ✅ | Only show models with configured keys |

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (React + Vite)                 │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────┐ │
│  │ ChatPage│  │ModelSel │  │ActionForm│  │ ActionPreview   │ │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────────┬────────┘ │
│       └────────────┴────────────┴─────────────────┘          │
│                         │ REST API                           │
└─────────────────────────┼───────────────────────────────────┘
                          │
┌─────────────────────────┼───────────────────────────────────┐
│                    Backend (Go + Gin)                        │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────────┐ │
│  │  Agent  │  │Clarify  │  │  K8s    │  │   LLM Client    │ │
│  │         │  │  Logic  │  │ Client  │  │ (GLM/DeepSeek)  │ │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────────┬────────┘ │
│       └────────────┴────────────┴─────────────────┘          │
│                         │                                    │
└─────────────────────────┼───────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   ┌────┴────┐      ┌─────┴─────┐     ┌─────┴─────┐
   │ K8s API │      │ GLM API   │     │ DeepSeek  │
   │ Cluster │      │ (Coding)  │     │   API     │
   └─────────┘      └───────────┘     └───────────┘
```

---

## 🗺️ Roadmap

### Phase 1: Core Enhancements (v0.2.0)

**Goal**: Improve user experience and reliability

| Feature | Priority | Effort | Description |
|---------|----------|--------|-------------|
| **Session Management** | 🔴 High | 2d | Multi-turn conversation with context |
| **Streaming Response** | 🔴 High | 1d | Typewriter effect for AI responses |
| **Markdown Rendering** | 🔴 High | 0.5d | Render code blocks, tables, lists |
| **Safety Confirmation** | 🔴 High | 1d | Confirm before destructive operations |
| **Error Recovery** | 🟡 Medium | 1d | Retry failed operations |
| **Loading States** | 🟡 Medium | 0.5d | Better UX during API calls |

#### 1.1 Session Management
```
Current: Each request is stateless
Target:  Maintain conversation history per session

Features:
- Session ID tracking
- Context window management
- Reference previous operations ("把那个 deployment 扩容")
- Session persistence (localStorage)
```

#### 1.2 Streaming Response
```
Current: Wait for full response
Target:  Stream tokens as they arrive

Implementation:
- SSE (Server-Sent Events) or WebSocket
- Typewriter effect in frontend
- Cancel streaming support
```

#### 1.3 Safety Confirmation
```
Dangerous Operations (require explicit confirmation):
- Delete deployment/pod
- Scale to 0 replicas
- Delete namespace
- Any destructive operation

Implementation:
- dangerLevel: "low" | "medium" | "high"
- Dry-run mode by default
- Explicit user confirmation UI
```

---

### Phase 2: Extended K8s Support (v0.3.0)

**Goal**: Support more K8s resources

| Resource | Priority | Description |
|----------|----------|-------------|
| **Logs** | 🔴 High | `kubectl logs` equivalent |
| **Exec** | 🔴 High | `kubectl exec` into containers |
| **Describe** | 🔴 High | Detailed resource info |
| **Namespace** | 🟡 Medium | Create/delete namespaces |
| **ConfigMap** | 🟡 Medium | Manage config maps |
| **Secret** | 🟡 Medium | Manage secrets |
| **Ingress** | 🟡 Medium | Manage ingress rules |
| **PVC** | 🟡 Medium | Manage persistent volumes |

#### Example Commands to Support
```
"查看 nginx 的日志"
"进入 nginx 容器执行命令"
"详细描述 nginx deployment"
"创建一个 configmap"
"查看所有 ingress"
```

---

### Phase 3: Advanced Features (v0.4.0)

**Goal**: Power user features

| Feature | Priority | Description |
|---------|----------|-------------|
| **YAML Editor** | 🟡 Medium | Edit YAML directly |
| **Resource Templates** | 🟡 Medium | Pre-defined templates |
| **History** | 🟡 Medium | Operation history & undo |
| **Multi-cluster** | 🟢 Low | Switch between clusters |
| **RBAC** | 🟢 Low | Permission management |
| **Dark Mode** | 🟢 Low | UI theme |

#### 3.1 Resource Templates
```yaml
# Pre-defined templates for common deployments
templates:
  - name: "nginx-deployment"
    resource: deployment
    template: |
      apiVersion: apps/v1
      kind: Deployment
      spec:
        replicas: 1
        containers:
        - name: {{name}}
          image: nginx:latest
```

#### 3.2 Operation History
```
Features:
- Log all operations with timestamps
- Undo capability (where applicable)
- Export history as JSON
- Filter by resource type, action, date
```

---

### Phase 4: Enterprise Features (v0.5.0)

**Goal**: Production readiness

| Feature | Priority | Description |
|---------|----------|-------------|
| **Authentication** | 🔴 High | User login, API keys |
| **Audit Logging** | 🔴 High | Who did what when |
| **Rate Limiting** | 🟡 Medium | Prevent abuse |
| **Metrics** | 🟡 Medium | Prometheus integration |
| **High Availability** | 🟡 Medium | Multi-instance support |
| **Backup/Restore** | 🟢 Low | Backup configurations |

---

### Phase 5: AI Enhancements (v0.6.0)

**Goal**: Smarter AI assistant

| Feature | Priority | Description |
|---------|----------|-------------|
| **Context Awareness** | 🔴 High | Remember previous context |
| **Proactive Suggestions** | 🟡 Medium | Suggest actions based on cluster state |
| **Natural Language Diff** | 🟡 Medium | "What changed since yesterday?" |
| **Troubleshooting** | 🟡 Medium | "Why is my pod crashing?" |
| **Cost Estimation** | 🟢 Low | Estimate resource costs |

#### 5.1 Context Awareness
```
User: "部署 nginx"
AI: Creates deployment...

User: "扩容到 5 个"  ← No need to specify "nginx" again
AI: Understands context, scales nginx to 5

User: "它的日志"  ← "它" refers to nginx
AI: Shows nginx logs
```

#### 5.2 Proactive Suggestions
```
AI monitors cluster and suggests:
- "Deployment 'api' has high restart count"
- "Pod 'nginx-xxx' is in CrashLoopBackOff"
- "No resource limits set on deployment 'web'"
```

---

## 📅 Timeline

```
2025 Q1 (v0.2.0)
├── Session Management
├── Streaming Response
├── Markdown Rendering
└── Safety Confirmation

2025 Q2 (v0.3.0)
├── Logs Support
├── Exec Support
├── Describe Support
└── Extended Resources

2025 Q3 (v0.4.0)
├── YAML Editor
├── Templates
├── History
└── Multi-cluster

2025 Q4 (v0.5.0)
├── Authentication
├── Audit Logging
├── Metrics
└── HA Support

2026 Q1 (v0.6.0)
├── Context Awareness
├── Proactive Suggestions
├── Troubleshooting
└── Cost Estimation
```

---

## 🎯 Success Metrics

| Metric | Current | Target (v0.5.0) |
|--------|---------|-----------------|
| Supported K8s Resources | 3 | 12+ |
| LLM Providers | 3 | 5+ |
| Response Time | 2-5s | <3s |
| User Satisfaction | - | 4.5/5 |
| GitHub Stars | - | 500+ |

---

## 🤝 Contributing

### How to Contribute
1. Fork the repository
2. Create a feature branch
3. Submit a PR with tests
4. Join our discussions

### Priority Areas for Contributions
- 📝 Documentation
- 🐛 Bug fixes
- ✨ New K8s resource support
- 🌐 Internationalization
- 🧪 Test coverage

---

## 📋 Technical Debt

| Item | Priority | Description |
|------|----------|-------------|
| Unit Tests | 🔴 High | Add comprehensive tests |
| Error Handling | 🟡 Medium | Consistent error responses |
| Logging | 🟡 Medium | Structured logging |
| API Documentation | 🟡 Medium | OpenAPI/Swagger docs |
| Performance | 🟢 Low | Optimize LLM calls |

---

## 🔗 Related Projects

- [kubectl-ai](https://github.com/GoogleCloudPlatform/kubectl-ai) - Google's kubectl AI
- [k8sgpt](https://github.com/k8sgpt/k8sgpt) - K8s diagnostics with AI
- [kube-copilot](https://github.com/awesome-kube/copilot) - K8s AI assistant

---

*Last Updated: 2025-02-25*
*Version: 0.1.0*
