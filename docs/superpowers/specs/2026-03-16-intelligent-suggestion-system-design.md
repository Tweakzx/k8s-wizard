# Intelligent Suggestion System Design

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an intelligent suggestion system that detects ambiguous user requests and offers cluster-aware alternatives to improve both clarity and workflow efficiency in the k8s-wizard frontend.

**Architecture:** Frontend-backend collaboration where backend queries cluster state and ranks suggestions based on relevance, while frontend displays them as selectable cards with explanations.

**Tech Stack:** Go (backend), React + TypeScript (frontend), Kubernetes API (cluster queries)

---

## Section 1: Architecture and Data Flow

### High-Level Flow

```
User Input "deploy nginx"
    ↓
LLM Parse Intent (existing)
    ↓
Detected Ambiguity?
    ↓ Yes
Cluster Query Engine
    ↓ GET /api/resources
    ↓ Existing Deployments/Pods/Namespaces
    ↓
Suggestion Ranker
    ↓ [Rank by recency, health, name match]
    ↓
Frontend Display: Suggestions Component
    ↓
User Selects or Provides Manual Input
    ↓
Proceed with Workflow (existing ActionPreview)
```

### Key Design Decisions

1. **Backend Extension**: Add `/api/suggestions` endpoint that:
   - Takes parsed intent from LLM (action, partial info)
   - Returns ranked suggestions based on cluster state
   - Caches results for performance (5-second TTL)

2. **Frontend Integration**: New `Suggestions` component that:
   - Receives suggestions from backend
   - Displays as selectable cards with explanations
   - Integrates with existing `ActionForm` as "None of these" fallback
   - Maintains consistent styling with current `ActionPreview`

3. **Ranking Algorithm**: Simple but effective:
   - **Primary**: Name match score (exact > partial > none)
   - **Secondary**: Recency (last accessed/deployed)
   - **Tertiary**: Resource health (running > pending > error)
   - **Fallback**: Namespace default order (default > kube-system > others)

---

## Section 2: Components and Interfaces

### New Backend Components

```
pkg/workflow/suggestions.go
├── SuggestionEngine
│   ├── QueryCluster()      - Fetches resources from k8s client
│   ├── RankSuggestions()  - Applies ranking algorithm
│   └── CacheStrategy()    - Manages in-memory cache
│
└── SuggestionsHandler
    └── MakeSuggestionsNode() - New workflow node

pkg/api/models/suggestions.go
├── Suggestion (new model)
│   ├── Type: string        - "reuse", "create", "none"
│   ├── Action: string      - "create", "delete", "scale"
│   ├── Resource: string    - "deployment", "pod", "service"
│   ├── Name: string
│   ├── Namespace: string
│   ├── Reason: string       - Why this suggestion
│   ├── Confidence: float    - 0.0 to 1.0
│   └── Existing: boolean    - Is this already in cluster?
```

### New Frontend Components

```
web/src/components/SuggestionCards.tsx
├── SuggestionCard
│   ├── Icon (📦, ➕, 🗑️, ⚙️)
│   ├── Title (ex: "Reuse existing deployment")
│   ├── Description (ex: "nginx in default namespace")
│   ├── WhyThis section
│   └── Select button
│
└── SuggestionsContainer
    ├── Suggestion cards grid
    ├── "None of these" option
    └── Manual input trigger

web/src/services/suggestions.ts
├── fetchSuggestions(intent)
└── POST /api/suggestions

web/src/types/index.ts (extended)
├── Suggestion (new interface)
└── SuggestionRequest (new interface)
```

### Integration Points

1. **Workflow Integration**: Add `MakeSuggestionsNode()` after `MakeParseIntentNode()`
   - If `needsClarification == true`, query suggestions first
   - Pass suggestions to frontend in `ChatResponse`

2. **Frontend Integration**: Extend `ActionForm` to show suggestions
   - When `suggestions[]` present, display before form
   - User can click suggestion OR fill form manually
   - Selection auto-populates form with suggestion data

3. **API Integration**: Add new endpoint `POST /api/suggestions`
   - Endpoint signature mirrors `/api/chat` structure
   - Reuses existing LLM intent parsing result

---

## Section 3: User Experience Flow

### Complete Interaction Flow

```
User Types: "deploy nginx"
    ↓
[Loading... 2s]
    ↓
System Detects: Ambiguous Request
    ↓
┌─────────────────────────────────────┐
│ 🤔 What would you like to do? │
├─────────────────────────────────────┤
│                               │
│ 📦 Reuse: nginx               │
│    (exists in default)            │
│    ✓ Recently used               │
│    ✗ Health: Pending            │
│    Why: Found deployment with    │
│         same name                │
│                               │
│ ➕ Create: nginx deployment      │
│    (new with defaults)            │
│    ✓ Recommended pattern          │
│    Why: Common naming for         │
│         web apps                │
│                               │
│ ⚙️ Customize...                │
│    (specify your own)            │
│    Why: None match above          │
└─────────────────────────────────────┘
    ↓
User Clicks Suggestion Card
    ↓
[Animation: Card slides into form]
    ↓
┌─────────────────────────────────────┐
│ 📦 Create Deployment           │
│                               │
│ Name:          [nginx         ] │
│ Image:         [nginx:latest  ] │
│ Replicas:      [1] [-][+]    │
│ Namespace:      [default       ] │
│                               │
│ [⏪ Change suggestion]         │
│ [✓ Confirm] [✗ Cancel]       │
└─────────────────────────────────────┘
    ↓
User Confirms → Action Preview (existing)
    ↓
User Confirms → Execute (existing)
```

### Key UX Principles

1. **Progressive Disclosure**: Start with 2-3 options, expand if needed
2. **Confidence Indicators**: Show green checkmarks for strong matches, yellow for partial
3. **Fast Recovery**: "Change suggestion" button always visible, one click back to cards
4. **Smart Defaults**: Selected suggestion pre-fills form, user can still edit
5. **Visual Hierarchy**: Most likely suggestion first, clear visual separation

### Example Scenarios

**Scenario 1: "delete nginx"**
- Show: Deployment "nginx" (primary) + Pod "nginx" (secondary if exists)
- Why: Multiple resources possible, deployment most likely

**Scenario 2: "deploy something"**
- Show: "Specify name" (manual) as primary option
- Suggest: Recent deployment names for inspiration
- Why: Can't guess without name, provide helpful context

**Scenario 3: "scale"**
- Show: Scale form directly (no suggestions needed)
- Why: Single interpretation, current form handles well

---

## Section 4: Error Handling

### Failure Mode: Cluster Query Fails

```
User Types: "deploy nginx"
    ↓
[Querying cluster...]
    ↓ Error!
┌─────────────────────────────────────┐
│ ⚠️  Couldn't check cluster     │
│                               │
│ Error: Failed to connect to       │
│        Kubernetes API             │
│                               │
│ You can still specify manually:        │
│ ──────────────────────          │
│ │ Name:        [      ]       │
│ │ Image:       [      ]       │
│ │ Replicas:    [1]           │
│ ──────────────────────          │
│                               │
│ [Try again]  [Specify manually] │
└─────────────────────────────────────┘
```

### Handling Strategies

1. **Graceful Degradation**: Always offer manual form fallback
   - Never block user completely
   - Show what failed and why (connection, permissions, timeout)
   - Provide retry button with exponential backoff

2. **Cache Management**: Handle stale cached data
   - If cluster query succeeds, invalidate suggestions cache
   - If query fails, show "Last known" with timestamp
   - 5-second cache TTL reduces retry burden

3. **Suggestion Validation**: Don't suggest things that don't exist
   - Backend validates all suggestions against cluster
   - If suggestion fails between display and execution, refresh
   - Show "This was available before" with confidence warning

### Edge Cases

- **Empty cluster**: Show "No existing resources, create from scratch" message
- **Permission denied**: Show "You don't have access to namespace X, available: Y, Z"
- **Timeout**: Show "Cluster is slow, using last known resources (from 2m ago)"

---

## Section 5: Testing Strategy

### Backend Tests

```
pkg/workflow/suggestions_test.go
├── TestSuggestionEngine_QueryCluster()
│   └── Mock k8s client with known resources
│   └── Verify correct JSON output
│
├── TestSuggestionEngine_RankSuggestions()
│   ├── Test exact name match (highest score)
│   ├── Test partial name match (medium score)
│   ├── Test recency weighting
│   └── Test health status weighting
│
├── TestSuggestionsHandler_CacheBehavior()
│   ├── Test cache hit (return cached, no API call)
│   ├── Test cache miss (query cluster, store result)
│   └── Test cache expiration (5-second TTL)
│
└── TestSuggestionsHandler_Integration()
    └── End-to-end: request → LLM → suggestions → response

pkg/workflow/e2e_test.go (new)
├── TestWorkflow_SuggestionPath()
│   └── Verify: ambiguous request → suggestions → form submit
│
└── TestWorkflow_DelegationPath()
    └── Verify: manual input → no suggestions → proceed
```

### Frontend Tests

```
web/src/components/SuggestionCards.test.tsx
├── Renders suggestion cards correctly
├── Handles empty suggestions array
├── Triggers callback on card click
└── Shows "None of these" option

web/src/pages/ChatPage.test.tsx (extended)
├── Test suggestion display in message flow
├── Test suggestion selection populates form
├── Test error fallback to manual input
└── Test change suggestion button functionality
```

### Integration Tests

```
test/e2e/suggestions_flow_test.go
├── Scenario1: "deploy nginx" with existing deployment
│   └── Verify: Reuse suggestion shows first, create shows second
│
├── Scenario2: "delete nginx" with both pod and deployment
│   └── Verify: Both suggestions appear with confidence indicators
│
├── Scenario3: Ambiguous request with cluster connection failure
│   └── Verify: Manual form appears with clear error message
│
└── Scenario4: No cluster resources (fresh install)
    └── Verify: Message explains situation, manual form works
```

### Manual Testing Checklist

- [ ] Test with real Kind cluster (not mocked)
- [ ] Test with connection failures (network disabled)
- [ ] Test with 50+ resources (performance check)
- [ ] Test Chinese language (support for user base)
- [ ] Test slow LLM responses (loading states)

---

## Section 6: Implementation Summary

### What We're Building

An intelligent suggestion system that detects ambiguous user requests and offers cluster-aware alternatives, improving both clarity and efficiency.

### Key Benefits

1. **Reduces Ambiguity**: Users don't guess what to type - system shows what's possible
2. **Faster Workflows**: Reuse existing resources instead of creating duplicates
3. **Better Learning**: Users see cluster state and learn from examples
4. **Transparent**: System explains "Why this?" for every suggestion

### Success Metrics

- Users complete tasks in 50% fewer interactions (fewer clarification rounds)
- 80% of ambiguous requests get resolved by suggestion (not manual form)
- User satisfaction increases from current baseline (to be measured)

### Integration Checklist

- [ ] Backend `/api/suggestions` endpoint
- [ ] SuggestionEngine with cluster query and ranking
- [ ] Frontend `SuggestionCards` component
- [ ] Cache strategy for performance
- [ ] Error handling for cluster failures
- [ ] Full test coverage (unit + integration + e2e)
- [ ] Documentation update
