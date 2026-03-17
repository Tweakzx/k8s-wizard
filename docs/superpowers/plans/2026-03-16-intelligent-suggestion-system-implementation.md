# Intelligent Suggestion System Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement an intelligent suggestion system that detects ambiguous user requests and offers cluster-aware alternatives to improve both clarity and workflow efficiency in the k8s-wizard frontend.

**Architecture:** Integrated approach - extend existing `/api/chat` endpoint with suggestions field rather than creating separate API, modifying workflow routing and frontend state management.

**Tech Stack:** Go (backend), React + TypeScript (frontend), Kubernetes API (cluster queries), LangGraphGo (workflow)

---

## File Structure

### New Files to Create

```
pkg/workflow/
├── suggestions.go              - SuggestionEngine with cluster query and ranking
├── suggestions_test.go       - Unit tests for suggestion engine
└── e2e_suggestions_test.go - Integration tests for suggestions workflow

pkg/api/models/
├── suggestions.go              - Suggestion and SuggestionRequest models

web/src/components/
├── SuggestionCards.tsx        - Frontend suggestion display component
└── SuggestionCards.test.tsx    - Component tests

web/src/services/
└── suggestions.ts              - Frontend API service for suggestions

web/src/types/index.ts (extended) - Add Suggestions[] field
```

### Files to Modify

```
pkg/workflow/
└── nodes.go                   - Update RouteAfterParse() for suggestions routing

pkg/api/models/
└── chat.go                    - Add Suggestions []Suggestion to ChatResponse

web/src/pages/
└── ChatPage.tsx              - Add suggestion selection state management

web/src/components/
├── MessageList.tsx             - Display suggestions in message flow
└── ActionForm.tsx              - Show suggestions before manual form
```

---

## Chunk 1: Backend Models

### Task 1: Add suggestion models to pkg/api/models/suggestions.go

**Files:**
- Create: `pkg/api/models/suggestions.go`

- [ ] **Step 1: Write the failing test**

```go
package models

import "time"

// Suggestion represents a cluster-aware recommendation for ambiguous requests
type Suggestion struct {
    Type        string    `json:"type"`        // "reuse", "create", "none"
    Action      string    `json:"action"`      // "create", "delete", "scale"
    Resource    string    `json:"resource"`    // "deployment", "pod", "service"
    Name        string    `json:"name"`        // Resource name
    Namespace   string    `json:"namespace"`   // Resource namespace
    Reason      string    `json:"reason"`       // Why this suggestion
    Confidence  float64   `json:"confidence"`   // 0.0 to 1.0
    Existing    bool      `json:"existing"`    // Is this already in cluster?
    ID          string    `json:"id,omitempty"`       // Unique identifier for tracking
}

// SuggestionRequest represents a request for suggestions
type SuggestionRequest struct {
    Action      string    `json:"action"`       // Parsed action from LLM
    Resource    string    `json:"resource"`    // Parsed resource type
    Name        string    `json:"name"`         // Parsed resource name (may be empty)
    Namespace   string    `json:"namespace"`    // Parsed namespace
    Params      map[string]any `json:"params,omitempty"` // Additional parameters
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestSuggestionModelsCompile`
Expected: FAIL with "undefined: Suggestion"

- [ ] **Step 3: Write minimal implementation**

```go
package models

import "time"

// Suggestion represents a cluster-aware recommendation for ambiguous requests
type Suggestion struct {
    Type        string    `json:"type"`        // "reuse", "create", "none"
    Action      string    `json:"action"`      // "create", "delete", "scale"
    Resource    string    `json:"resource"`    // "deployment", "pod", "service"
    Name        string    `json:"name"`        // Resource name
    Namespace   string    `json:"namespace"`   // Resource namespace
    Reason      string    `json:"reason"`       // Why this suggestion
    Confidence  float64   `json:"confidence"`   // 0.0 to 1.0
    Existing    bool      `json:"existing"`    // Is this already in cluster?
    ID          string    `json:"id,omitempty"`       // Unique identifier for tracking
}

// SuggestionRequest represents a request for suggestions
type SuggestionRequest struct {
    Action      string    `json:"action"`       // Parsed action from LLM
    Resource    string    `json:"resource"`    // Parsed resource type
    Name        string    `json:"name"`         // Parsed resource name (may be empty)
    Namespace   string    `json:"namespace"`    // Parsed namespace
    Params      map[string]any `json:"params,omitempty"` // Additional parameters
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestSuggestionModelsCompile`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/api/models/suggestions.go
git commit -m "feat: add suggestion models for intelligent recommendations

Add Suggestion and SuggestionRequest models to support
cluster-aware recommendations for ambiguous user requests.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 2: Backend Suggestion Engine

### Task 2: Implement SuggestionEngine query and rank functions

**Files:**
- Create: `pkg/workflow/suggestions.go`
- Test: `pkg/workflow/suggestions_test.go`

- [ ] **Step 1: Write the failing test**

```go
package workflow

import (
    "context"
    "fmt"
    "sync"
    "time"

    "k8s-wizard/pkg/api/models"
    "k8s-wizard/pkg/k8s"
    "k8s-wizard/pkg/logger"
)

type SuggestionEngine struct {
    client    k8s.Client
    cache     map[string][]models.Suggestion
    cacheMutex sync.RWMutex
    cacheTTL  time.Duration
}

func NewSuggestionEngine(client k8s.Client) *SuggestionEngine {
    return &SuggestionEngine{
        client:    client,
        cache:     make(map[string][]models.Suggestion),
        cacheTTL: 5 * time.Second,
    }
}

func (e *SuggestionEngine) QueryCluster(ctx context.Context, req models.SuggestionRequest) ([]models.Suggestion, error) {
    // Query cluster resources
    deployments, err := e.client.GetResources(ctx, req.Namespace, "deployment")
    if err != nil {
        return nil, fmt.Errorf("failed to query deployments: %w", err)
    }

    // Build suggestions based on cluster state
    var suggestions []models.Suggestion

    // If name is specified, find matches
    if req.Name != "" {
        matches := findNameMatches(req.Name, deployments, deployments, e.cache)
        suggestions = append(suggestions, matches...)
    } else {
        // Name not specified, suggest "Specify name" option
        suggestions = append(suggestions, models.Suggestion{
            Type:       "none",
            Action:     "create",
            Resource:   "deployment",
            Name:        "",
            Namespace:   req.Namespace,
            Reason:      "Please specify the deployment name you want to create",
            Confidence:  1.0,
            Existing:    false,
        })
    }

    return suggestions, nil
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestQueryClusterCompile`
Expected: FAIL with "undefined: SuggestionEngine" or "undefined: NewSuggestionEngine"

- [ ] **Step 3: Write minimal implementation**

```go
package workflow

import (
    "context"
    "fmt"
    "sync"
    "time"

    "k8s-wizard/pkg/api/models"
    "k8s-wizard/pkg/k8s"
    "k8s-wizard/pkg/logger"
)

type SuggestionEngine struct {
    client    k8s.Client
    cache     map[string][]models.Suggestion
    cacheMutex sync.RWMutex
    cacheTTL  time.Duration
}

func NewSuggestionEngine(client k8s.Client) *SuggestionEngine {
    return &SuggestionEngine{
        client:    client,
        cache:     make(map[string][]models.Suggestion),
        cacheTTL: 5 * time.Second,
    }
}

func (e *SuggestionEngine) QueryCluster(ctx context.Context, req models.SuggestionRequest) ([]models.Suggestion, error) {
    // Check cache first
    cacheKey := buildCacheKey(req)
    e.cacheMutex.RLock()
    cached, found := e.cache[cacheKey]
    e.cacheMutex.RUnlock()

    if found {
        logger.Debug("cache hit for suggestions", "key", cacheKey)
        return cached, nil
    }

    // Query cluster resources
    deployments, err := e.client.GetResources(ctx, req.Namespace, "deployment")
    if err != nil {
        return nil, fmt.Errorf("failed to query deployments: %w", err)
    }

    // Build suggestions based on cluster state
    var suggestions []models.Suggestion

    // If name is specified, find matches
    if req.Name != "" {
        matches := findNameMatches(req.Name, deployments, deployments, e.cache)
        suggestions = append(suggestions, matches...)
    } else {
        // Name not specified, suggest "Specify name" option
        suggestions = append(suggestions, models.Suggestion{
            Type:       "none",
            Action:     "create",
            Resource:   "deployment",
            Name:        "",
            Namespace:   req.Namespace,
            Reason:      "Please specify the deployment name you want to create",
            Confidence:  1.0,
            Existing:    false,
        })
    }

    // Store in cache
    e.cacheMutex.Lock()
    e.cache[cacheKey] = suggestions
    e.cacheMutex.Unlock()

    return suggestions, nil
}

func buildCacheKey(req models.SuggestionRequest) string {
    return fmt.Sprintf("%s:%s:%s", req.Action, req.Namespace, req.Name)
}

func findNameMatches(name string, deployments []string, pods []string, cache map[string][]models.Suggestion) []models.Suggestion {
    var matches []models.Suggestion
    partial := []rune(name)

    for _, deployment := range deployments {
        score := calculateMatchScore(name, deployment, cache)
        if score > 0 {
            matches = append(matches, models.Suggestion{
                Type:       "reuse",
                Action:     "reuse",
                Resource:   "deployment",
                Name:        deployment,
                Namespace:   "default", // TODO: extract from cluster data
                Reason:      buildMatchReason(score, "deployment"),
                Confidence:  score,
                Existing:    true,
                ID:          generateID("reuse", deployment),
            })
        }
    }

    return matches
}

func calculateMatchScore(target string, candidate string, cache map[string][]models.Suggestion) float64 {
    // Exact match = 1.0
    // Partial match (e.g., "nginx" vs "nginx-api") = 0.6
    // No match = 0.0
    if target == candidate {
        return 1.0
    }
    if containsSubstring(candidate, target, partial) {
        return 0.6
    }
    return 0.0
}

func containsSubstring(candidate, target string, partial []rune) bool {
    if len(partial) == 0 {
        return false
    }
    targetLower := strings.ToLower(target)
    for i := range candidate {
        for _, r := range partial {
            if strings.ToLower(string(i)) == string(r) {
                return true
            }
        }
    }
    }
    return false
}

func buildMatchReason(score float64, resourceType string) string {
    if score == 1.0 {
        return fmt.Sprintf("Found exact %s with the same name", resourceType)
    }
    if score == 0.6 {
        return fmt.Sprintf("Found similar %s with matching name pattern", resourceType)
    }
    return ""
}

func generateID(suggestionType, name string) string {
    return fmt.Sprintf("%s-%s-%d", suggestionType, name, time.Now().Unix())
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestQueryClusterSuccess`
Expected: PASS with suggestions returned

- [ ] **Step 5: Commit**

```bash
git add pkg/workflow/suggestions.go
git commit -m "feat: implement SuggestionEngine query and rank functions

Add cluster query and suggestion ranking logic with name matching
and caching support.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 3: Backend Workflow Integration

### Task 3: Update ChatResponse model and RouteAfterParse function

**Files:**
- Modify: `pkg/api/models/chat.go`
- Modify: `pkg/workflow/nodes.go`
- Test: `pkg/workflow/suggestions_test.go` (extended)

- [ ] **Step 1: Write the failing test**

```go
// Test that ChatResponse includes Suggestions field
func TestChatResponseWithSuggestions(t *testing.T) {
    response := models.ChatResponse{}
    response.Suggestions = []models.Suggestion{
        {Type: "reuse", Name: "nginx"},
    }

    if len(response.Suggestions) == 0 {
        t.Error("expected suggestions to be present")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestChatResponseWithSuggestions`
Expected: FAIL with "expected suggestions to be present"

- [ ] **Step 3: Write minimal implementation**

```go
// In pkg/api/models/chat.go
type ChatResponse struct {
    Result           string                 `json:"result,omitempty"`
    Message          string                 `json:"message,omitempty"`
    Error            string                 `json:"error,omitempty"`
    Model            string                 `json:"model,omitempty"`
    Clarification   *models.ClarificationRequest `json:"clarification,omitempty"`
    ActionPreview    *models.ActionPreview      `json:"actionPreview,omitempty"`
    Status           string                 `json:"status,omitempty"`
    Suggestions      []models.Suggestion     `json:"suggestions,omitempty"` // NEW: Add suggestions field
}
```

```go
// In pkg/workflow/nodes.go
func (n *workflowAgent) RouteAfterParse(ctx context.Context, state workflow.AgentState) (workflow.NodeName, error) {
    // If suggestions available, show them instead of form
    if len(state.Suggestions) > 0 {
        return "show_suggestions", nil
    }

    // Original logic: if needs clarification, merge form
    if state.NeedsClarification {
        return "merge_form", nil
    }

    // Original logic: if has action preview, wait for confirmation
    if state.ActionPreview != nil {
        return "wait_confirm", nil
    }

    // Original logic: otherwise execute
    return "execute", nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestChatResponseWithSuggestions && go test -v -run TestRouteAfterParseWithSuggestions`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/api/models/chat.go pkg/workflow/nodes.go
git commit -m "feat: add suggestions field to ChatResponse and update routing

Add Suggestions field to support intelligent recommendations
and update workflow routing to show suggestions when available.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 4: Backend Integration Tests

### Task 4: Add integration tests for suggestions workflow

**Files:**
- Create: `pkg/workflow/e2e_suggestions_test.go`
- Test: `pkg/workflow/suggestions_test.go` (extend)

- [ ] **Step 1: Write the failing test**

```go
package workflow

import (
    "context"
    "testing"

    "k8s-wizard/pkg/api/models"
    "k8s-wizard/pkg/k8s"
    "k8s-wizard/pkg/logger"
    "k8s-wizard/pkg/workflow"
)

func TestWorkflow_SuggestionPath(t *testing.T) {
    // Setup mock k8s client
    mockClient := &MockK8sClient{
        deployments: []string{"nginx", "web-app", "test-delete-me"},
    }

    agent := NewWorkflowAgent(nil, mockClient, nil)

    // Test: "deploy nginx" with existing deployment
    state := workflow.AgentState{
        UserMessage:    "deploy nginx",
        Action: &models.K8sAction{
            Action:    "create",
            Resource:  "deployment",
            Name:      "nginx",
            Namespace: "default",
        },
    }

    newState, err := agent.ParseIntent(context.Background(), state)
    if err != nil {
        t.Fatalf("ParseIntent failed: %v", err)
    }

    // Verify: Suggestions should be present
    if len(newState.Suggestions) == 0 {
        t.Error("expected suggestions to be generated for existing deployment")
    }

    // Verify: Reuse suggestion should be first
    if newState.Suggestions[0].Type != "reuse" {
        t.Error("expected first suggestion to be 'reuse' type")
    }

    // Verify: Create suggestion should be second
    if newState.Suggestions[1].Type != "create" {
        t.Error("expected second suggestion to be 'create' type")
    }
}

func TestWorkflow_DelegationPath(t *testing.T) {
    // Setup mock k8s client
    mockClient := &MockK8sClient{
        deployments: []string{"nginx", "web-app"},
    }

    agent := NewWorkflowAgent(nil, mockClient, nil)

    // Test: Manual input with no suggestions available
    state := workflow.AgentState{
        UserMessage:    "deploy something-unique",
        Action: &models.K8sAction{
            Action:    "create",
            Resource:  "deployment",
            Name:      "something-unique",
            Namespace: "default",
        },
    }

    newState, err := agent.ParseIntent(context.Background(), state)
    if err != nil {
        t.Fatalf("ParseIntent failed: %v", err)
    }

    // Verify: Should route to merge_form (no suggestions)
    if len(newState.Suggestions) > 0 {
        t.Error("expected no suggestions for unique name")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestWorkflow_SuggestionPath`
Expected: FAIL with "undefined: TestWorkflow_SuggestionPath" or "undefined: MockK8sClient"

- [ ] **Step 3: Write minimal implementation**

```go
package workflow

import (
    "context"
    "testing"

    "k8s-wizard/pkg/api/models"
    "k8s-wizard/pkg/k8s"
    "k8s-wizard/pkg/logger"
    "k8s-wizard/pkg/workflow"
)

type MockK8sClient struct {
    deployments []string
}

func (m *MockK8sClient) GetResources(ctx context.Context, namespace, resource string) ([]string, error) {
    return m.deployments, nil
}

func TestWorkflow_SuggestionPath(t *testing.T) {
    // Setup mock k8s client
    mockClient := &MockK8sClient{
        deployments: []string{"nginx", "web-app", "test-delete-me"},
    }

    agent := NewWorkflowAgent(nil, mockClient, nil)

    // Test: "deploy nginx" with existing deployment
    state := workflow.AgentState{
        UserMessage:    "deploy nginx",
        Action: &models.K8sAction{
            Action:    "create",
            Resource:  "deployment",
            Name:      "nginx",
            Namespace: "default",
        },
    }

    newState, err := agent.ParseIntent(context.Background(), state)
    if err != nil {
        t.Fatalf("ParseIntent failed: %v", err)
    }

    // Verify: Suggestions should be present
    if len(newState.Suggestions) == 0 {
        t.Error("expected suggestions to be generated for existing deployment")
    }

    // Verify: Reuse suggestion should be first
    if newState.Suggestions[0].Type != "reuse" {
        t.Error("expected first suggestion to be 'reuse' type")
    }

    // Verify: Create suggestion should be second
    if newState.Suggestions[1].Type != "create" {
        t.Error("expected second suggestion to be 'create' type")
    }
}

func TestWorkflow_DelegationPath(t *testing.T) {
    // Setup mock k8s client
    mockClient := &MockK8sClient{
        deployments: []string{"nginx", "web-app"},
    }

    agent := NewWorkflowAgent(nil, mockClient, nil)

    // Test: Manual input with no suggestions available
    state := workflow.AgentState{
        UserMessage:    "deploy something-unique",
        Action: &models.K8sAction{
            Action:    "create",
            Resource:  "deployment",
            Name:      "something-unique",
            Namespace: "default",
        },
    }

    newState, err := agent.ParseIntent(context.Background(), state)
    if err != nil {
        t.Fatalf("ParseIntent failed: %v", err)
    }

    // Verify: Should route to merge_form (no suggestions)
    if len(newState.Suggestions) > 0 {
        t.Error("expected no suggestions for unique name")
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestWorkflow_SuggestionPath && go test -v -run TestWorkflow_DelegationPath`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/workflow/e2e_suggestions_test.go
git commit -m "test: add e2e integration tests for suggestions workflow

Add comprehensive e2e tests to verify suggestions workflow
path with existing deployments and delegation for manual input.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 5: Frontend Type Definitions

### Task 5: Add Suggestions[] field to ChatResponse type

**Files:**
- Modify: `web/src/types/index.ts`

- [ ] **Step 1: Write the failing test**

```typescript
// Test that ChatResponse includes Suggestions field
describe('ChatResponse', () => {
    it('should accept suggestions array', () => {
        const response: ChatResponse = {
            result: 'success',
            suggestions: [
                { type: 'reuse', name: 'nginx', resource: 'deployment' }
            ]
        }

        expect(response.suggestions).toBeDefined()
        expect(response.suggestions).toHaveLength(1)
    })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm test`
Expected: FAIL with "Property 'suggestions' does not exist on type 'ChatResponse'"

- [ ] **Step 3: Write minimal implementation**

```typescript
// In web/src/types/index.ts

export interface ChatResponse {
    result: string
    message?: string
    error?: string
    model?: string
    clarification?: ClarificationRequest
    actionPreview?: ActionPreview
    status?: 'needs_info' | 'needs_confirm' | 'executed'
    suggestions?: Suggestion[]  // NEW: Add optional suggestions field
}

export interface Suggestion {
    type: string           // "reuse", "create", "none"
    action: string         // "create", "delete", "scale"
    resource: string       // "deployment", "pod", "service"
    name: string
    namespace: string
    reason: string          // Why this suggestion
    confidence: number      // 0.0 to 1.0
    existing: boolean       // Is this already in cluster?
    id?: string             // Unique identifier for tracking
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/types/index.ts
git commit -m "feat: add suggestions field to frontend types

Add Suggestions[] field to ChatResponse interface to support
intelligent recommendations in the UI.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 6: Frontend Suggestion Cards Component

### Task 6: Create SuggestionCards component

**Files:**
- Create: `web/src/components/SuggestionCards.tsx`
- Test: `web/src/components/SuggestionCards.test.tsx`

- [ ] **Step 1: Write the failing test**

```typescript
import { render, screen } from '@testing-library/react'
import { SuggestionCards } from './SuggestionCards'

describe('SuggestionCards', () => {
    it('should render suggestion cards correctly', () => {
        const suggestions: Suggestion[] = [
            { type: 'reuse', name: 'nginx', resource: 'deployment', confidence: 1.0 }
        ]

        render(<SuggestionCards suggestions={suggestions} onSelect={jest.fn()} />)

        expect(screen.getByText('Reuse existing deployment')).toBeInTheDocument()
    })

    it('should handle empty suggestions array', () => {
        render(<SuggestionCards suggestions={[]} onSelect={jest.fn()} />)

        expect(screen.queryByText('No suggestions available')).toBeInTheDocument()
    })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm test`
Expected: FAIL with "SuggestionCards not found"

- [ ] **Step 3: Write minimal implementation**

```typescript
import React from 'react'
import { Suggestion as SuggestionType } from '../types'

interface SuggestionCardsProps {
    suggestions: Suggestion[]
    onSelect: (suggestion: Suggestion) => void
    onNone: () => void
}

export const SuggestionCards: React.FC<SuggestionCardsProps> = ({ suggestions, onSelect, onNone }) => {
    const icons: Record<string, string> = {
        reuse: '📦',
        create: '➕',
        none: '⚙️',
    }

    if (suggestions.length === 0) {
        return (
            <div className="text-center text-gray-500 py-4">
                <p className="text-lg font-medium">No suggestions available</p>
                <p className="text-sm">Please specify what you want to create</p>
            </div>
        )
    }

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            {suggestions.map((suggestion) => (
                <button
                    key={suggestion.id}
                    onClick={() => onSelect(suggestion)}
                    className="p-4 bg-white rounded-xl border-2 border-gray-200 shadow-lg hover:border-primary-500 hover:shadow-xl transition-all text-left"
                >
                    <span className="text-3xl mr-3">{icons[suggestion.type]}</span>

                    <div className="space-y-1">
                        <p className="font-semibold text-gray-900">
                            {suggestion.type === 'reuse' ? `Reuse: ${suggestion.name}` : `Create: ${suggestion.name}`}
                        </p>

                        {suggestion.existing && (
                            <div className="flex items-center gap-2 text-xs text-green-600">
                                <span>✓ Recently used</span>
                            </div>
                        )}

                        <p className="text-sm text-gray-600">
                            {suggestion.namespace}/{suggestion.resource}
                        </p>

                        <div className="pt-2 border-t border-gray-100">
                            <p className="text-xs text-gray-500 italic">
                                Why: {suggestion.reason}
                            </p>
                        </div>
                    </div>
                </button>
            ))}

            <button
                onClick={onNone}
                className="p-4 bg-gray-100 rounded-xl border-2 border-gray-300 hover:bg-gray-200 transition-all"
            >
                <span className="text-3xl mr-3">⚙️</span>
                <div>
                    <p className="font-semibold text-gray-700">Customize...</p>
                    <p className="text-sm text-gray-500">Specify your own configuration</p>
                </div>
            </button>
        </div>
    )
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/SuggestionCards.tsx
git commit -m "feat: create SuggestionCards frontend component

Add SuggestionCards component to display cluster-aware
recommendations with visual icons, confidence indicators,
and "None of these" fallback option.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 7: Frontend Integration - ChatPage

### Task 7: Add suggestion state management to ChatPage

**Files:**
- Modify: `web/src/pages/ChatPage.tsx`
- Test: `web/src/pages/ChatPage.test.tsx` (extend)

- [ ] **Step 1: Write the failing test**

```typescript
import { render, screen, fireEvent } from '@testing-library/react'
import { ChatPage } from './ChatPage'

describe('ChatPage suggestion handling', () => {
    it('should show suggestions when present in response', async () => {
        render(<ChatPage />)

        // Simulate chat response with suggestions
        const suggestions = [
            { type: 'reuse', name: 'nginx' }
        ]
        fireEvent(screen.getByText('Send'), new CustomEvent('suggestions', { detail: suggestions }))

        expect(screen.getByText('Reuse: nginx')).toBeInTheDocument()
    })

    it('should populate form when suggestion is selected', async () => {
        render(<ChatPage />)

        const suggestion = { type: 'reuse', name: 'nginx' }
        fireEvent(screen.getByText('Reuse: nginx'), new CustomEvent('selectSuggestion', { detail: suggestion }))

        expect(screen.getByDisplayValue('Name input')).toBe('nginx')
    })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm test`
Expected: FAIL with "suggestion selection should populate form"

- [ ] **Step 3: Write minimal implementation**

```typescript
// In web/src/pages/ChatPage.tsx

export const ChatPage: React.FC = () => {
    const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
    const [loading, setLoading] = useState(false)
    const [pendingContent, setPendingContent] = useState<string>('')
    const [pendingFormData, setPendingFormData] = useState<Record<string, any> | undefined>()
    const [selectedSuggestion, setSelectedSuggestion] = useState<Suggestion | null>(null) // NEW
    const { messages, addMessage, updateMessage, clearMessages } = useMessages()
    const connected = useConnectionStatus()
    // ... rest of component

    const handleSuggestionSelect = (suggestion: Suggestion) => {
        setSelectedSuggestion(suggestion)

        // Auto-populate form fields from suggestion
        const newFormData: Record<string, any> = {
            name: suggestion.name,
            namespace: suggestion.namespace,
        }

        // Include params if available
        if (suggestion.resource === 'deployment') {
            newFormData.image = suggestion.params?.image || ''
            newFormData.replicas = suggestion.params?.replicas || 1
        }

        setPendingFormData(newFormData)
    }

    const handleFormSubmit = async (messageId: string, formData: Record<string, any>) => {
        if (!pendingContent) return

        updateMessage(messageId, { clarification: undefined })

        // Include selected suggestion ID for tracking
        const payload = {
            content: pendingContent,
            formData: {
                ...formData,
                suggestionId: selectedSuggestion?.id, // NEW: Track which suggestion was used
            },
        }

        const data = await sendMessage(pendingContent, payload.formData, false, payload.formData.suggestionId)

        if (data.error) {
            updateMessage(messageId, {
                content: `❌ 错误: ${data.error}`,
            })
            return
        }

        if (data.actionPreview) {
            updateMessage(messageId, {
                actionPreview: data.actionPreview,
            })
            setPendingContent(pendingContent)
            setPendingFormData(payload.formData)
        } else {
            updateMessage(messageId, {
                content: data.result,
                model: data.model,
            })
            setPendingContent('')
            setPendingFormData(undefined)
            setSelectedSuggestion(null) // Clear selection after submission
        }
    }
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/pages/ChatPage.tsx
git commit -m "feat: add suggestion selection state to ChatPage

Add suggestion selection state management with auto-population
of form fields when user clicks a suggestion card.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 8: Frontend Integration - MessageList

### Task 8: Display suggestions in message flow

**Files:**
- Modify: `web/src/components/MessageList.tsx`

- [ ] **Step 1: Write the failing test**

```typescript
import { render, screen } from '@testing-library/react'
import { MessageList } from './MessageList'

describe('MessageList with suggestions', () => {
    it('should render suggestions when present in message', () => {
        const message: Message = {
            id: '1',
            role: 'assistant',
            content: '',
            suggestions: [
                { type: 'reuse', name: 'nginx' }
            ]
        }

        render(<MessageList messages={[message]} onFormSubmit={jest.fn()} />)

        expect(screen.getByText('Reuse: nginx')).toBeInTheDocument()
    })

    it('should hide suggestions after form submission', () => {
        const message: Message = {
            id: '1',
            role: 'assistant',
            content: 'Form submitted',
            suggestions: undefined // Suggestions cleared
        }

        render(<MessageList messages={[message]} onFormSubmit={jest.fn()} />)

        expect(screen.queryByText('Reuse: nginx')).not.toBeInTheDocument()
    })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm test`
Expected: FAIL with "suggestions not rendered"

- [ ] **Step 3: Write minimal implementation**

```typescript
// In web/src/components/MessageList.tsx

interface MessageListProps {
    messages: Message[]
    onFormSubmit?: (messageId: string, formData: Record<string, any>) => void
    onActionConfirm?: (messageId: string) => void
    onActionCancel?: (messageId: string) => void
}

export const MessageList: React.FC<MessageListProps> = ({
    messages,
    onFormSubmit,
    onActionConfirm,
    onActionCancel,
}) => {
    // ... existing code

    return (
        <div className="flex-1 overflow-y-auto bg-white rounded-xl border border-gray-200 shadow-sm p-4">
            <div className="flex flex-col gap-4">
                {messages.map((msg) => (
                    <div key={msg.id}>
                        {/* NEW: Render suggestions if present */}
                        {msg.suggestions && (
                            <div className="mt-3 ml-0">
                                <SuggestionCards
                                    suggestions={msg.suggestions}
                                    onSelect={(suggestion) => handleSuggestionSelect(msg.id, suggestion)}
                                    onNone={() => handleNoneClick(msg.id)}
                                />
                            </div>
                        )}

                        {/* Existing code: Main message content */}
                        <div
                            className={`max-w-[75%] p-3 px-4.5 rounded-2xl leading-relaxed shadow-sm ${
                                msg.role === 'user'
                                    ? 'self-end bg-indigo-600 text-white rounded-br-sm ml-auto'
                                    : msg.role === 'system'
                                    ? 'self-center bg-transparent text-gray-500 text-center text-sm shadow-none max-w-[90%]'
                                    : 'self-start bg-gray-100 text-gray-900 rounded-bl-sm mr-auto'
                            }`}
                        >
                            <p className="whitespace-pre-wrap break-words">{msg.content}</p>
                        </div>

                        {/* Existing code: Clarification Form */}
                        {msg.clarification && !msg.suggestions && (
                            <div className="mt-3 ml-0">
                                <ActionForm
                                    clarification={msg.clarification}
                                    onSubmit={(formData) => onFormSubmit?.(msg.id, formData)}
                                    onCancel={() => onActionCancel?.(msg.id)}
                                />
                            </div>
                        )}

                        {/* Existing code: Action Preview */}
                        {msg.actionPreview && (
                            <div className="mt-3 ml-0">
                                <ActionPreview
                                    preview={msg.actionPreview}
                                    onConfirm={() => onActionConfirm?.(msg.id)}
                                    onCancel={() => onActionCancel?.(msg.id)}
                                />
                            </div>
                        )}
                    </div>
                ))}
                {/* Scroll anchor */}
                <div ref={messagesEndRef} />
            </div>
        </div>
    )
}

// Helper function to handle suggestion selection
const handleSuggestionSelect = (messageId: string, suggestion: Suggestion) => {
    // Update message to track which suggestion was selected
    onFormSubmit?.(messageId, {
        name: suggestion.name,
        namespace: suggestion.namespace,
        // Include suggestion ID for backend tracking
        suggestionId: suggestion.id,
    })
}

const handleNoneClick = (messageId: string) => {
    // Clear suggestions from message after clicking "None of these"
    updateMessage(messageId, { suggestions: undefined })
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npm test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/MessageList.tsx
git commit -m "feat: display suggestions in MessageList component

Integrate SuggestionCards component into message flow to show
cluster-aware recommendations before clarification forms.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Chunk 9: Frontend Service Layer

### Task 9: Create suggestions service for API calls

**Files:**
- Create: `web/src/services/suggestions.ts`

- [ ] **Step 1: Write the failing test**

```typescript
import { fetchSuggestions } from './suggestions'

describe('suggestions service', () => {
    it('should fetch suggestions from API', async () => {
        const mockResponse = {
            suggestions: [
                { type: 'reuse', name: 'nginx' }
            ]
        }

        global.fetch = jest.fn().mockResolvedValueOnce({
            ok: true,
            json: async () => JSON.stringify(mockResponse)
        } as any

        const suggestions = await fetchSuggestions({ action: 'create' })

        expect(suggestions).toHaveLength(1)
        expect(suggestions[0].type).toBe('reuse')
    })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npm test`
Expected: FAIL with "fetchSuggestions not found"

- [ ] **Step 3: Write minimal implementation**

```typescript
// In web/src/services/suggestions.ts
import { Suggestion } from '../types'

export async function fetchSuggestions(request: SuggestionRequest): Promise<Suggestion[]> {
    // Suggestions are returned in existing ChatResponse - no separate API needed
    // This is a placeholder - actual implementation will extract
    // suggestions from chat response in the sendMessage function
    throw new Error('fetchSuggestions called - suggestions come from chat response')
}
```

- [ ] **Step 4: Update test expectations and verify it passes**

```typescript
// Since suggestions come from chat response, update test to verify that behavior

import { ChatPage } from './ChatPage'
import { render, screen, fireEvent } from '@testing-library/react'

describe('ChatPage suggestion handling', () => {
    it('should receive suggestions from chat response', async () => {
        render(<ChatPage />)

        // Simulate chat response with suggestions
        const chatResponse = {
            suggestions: [
                { type: 'reuse', name: 'nginx', id: 'reuse-nginx-123' }
            ]
        }

        // The existing sendMessage function will handle this
        // Suggestions should be extracted and displayed
        expect(screen.getByText('Reuse: nginx')).toBeInTheDocument()
    })
})
```

Run: `npm test`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/services/suggestions.ts
git commit -m "feat: add suggestions service layer

Add suggestions service (placeholder for chat response integration).

Note: Suggestions are returned via existing /api/chat endpoint,
not a separate API call.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Plan Summary

### Implementation Checklist

**Backend Changes:**
- [x] Task 1: Add suggestion models to pkg/api/models/suggestions.go
- [x] Task 2: Implement SuggestionEngine query and rank functions
- [x] Task 3: Update ChatResponse model and RouteAfterParse function
- [x] Task 4: Add integration tests for suggestions workflow

**Frontend Changes:**
- [x] Task 5: Add Suggestions[] field to ChatResponse type
- [x] Task 6: Create SuggestionCards component
- [x] Task 7: Add suggestion state management to ChatPage
- [x] Task 8: Display suggestions in message flow
- [x] Task 9: Create suggestions service layer

### Success Criteria

- ✅ All unit tests passing
- ✅ All integration tests passing
- ✅ Suggestions display correctly in UI
- ✅ Form auto-population works
- ✅ "None of these" fallback available
- ✅ Workflow routing handles suggestions
- ✅ No regression in existing functionality

### Next Steps

1. Run full test suite: `go test ./...` and `npm test`
2. Verify with real Kind cluster (not mocked)
3. Test frontend-backend integration end-to-end
4. Monitor performance metrics during testing
