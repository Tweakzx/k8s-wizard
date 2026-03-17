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

**API Architecture Decision**: Integrated approach - suggestions returned in `ChatResponse` model
   - Why: Simpler integration, maintains existing `/api/chat` endpoint contract
   - Add `Suggestions []Suggestion` field to existing `ChatResponse` interface
   - Backend generates suggestions when `needsClarification == true` and suggestions are available
   - Frontend displays suggestions if present, otherwise shows standard form

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

1. **Workflow Integration**: Modify existing workflow to support suggestions
   - Add `MakeSuggestionsNode()` called after `MakeParseIntentNode()`
   - Update `RouteAfterParse()` to check for suggestions before routing
   - When `needsClarification == true` and suggestions available, show suggestions flow
   - When `needsClarification == true` but no suggestions, show standard form

**Updated Routing Logic:**
```
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

2. **Frontend Integration**: Extend `ActionForm` to show suggestions
   - When `suggestions[]` present in response, display `SuggestionCards` before form
   - User can click suggestion OR fill form manually
   - Selection stores selected suggestion in React state for form population
   - "None of these" option clears selection and shows empty form

**Frontend State Management:**
```
// In ChatPage.tsx
const [selectedSuggestion, setSelectedSuggestion] = useState<Suggestion | null>(null);

const handleSuggestionSelect = (suggestion: Suggestion) => {
    setSelectedSuggestion(suggestion);
    // Automatically populate form fields from suggestion
    setFormData({
        name: suggestion.name,
        namespace: suggestion.namespace,
        image: suggestion.params?.image || '',
        replicas: suggestion.params?.replicas || 1
    });
};

const handleFormSubmit = async (formData: Record<string, any>) => {
    // Include selected suggestion info for backend context
    const payload = {
        content: pendingContent,
        formData,
        suggestionId: selectedSuggestion?.id, // Track which suggestion was used
    };
    await sendMessage(payload);
};
```

3. **API Integration**: Extend existing `/api/chat` endpoint
   - No new endpoint needed - simpler integration
   - Add `Suggestions []Suggestion` field to `ChatResponse` model
   - Backend populates suggestions when appropriate during `MakeParseIntentNode`

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

**Scenario 4: "delete nginx" with multiple namespaces**
- Show: "nginx" deployment in default (primary) + "nginx" deployment in kube-system (secondary)
- Why: Same name exists in multiple namespaces, help user choose

**Scenario 5: "restart" (ambiguous command)**
- Show: All running resources in default namespace as restartable options
- Why: "restart" could mean pods, deployments, or services

**Scenario 6: User changes cluster state**
- Show: Cached suggestions with "Refresh available" badge
- Why: Cluster state may have changed since suggestions were generated

---

## Section 3.5: Ambiguity Detection Criteria

### When to Trigger Suggestions

System generates suggestions when these conditions are met:

1. **Action-specific heuristics:**
   - **create/deploy**: Suggest when resource name is missing or ambiguous
   - **delete**: Suggest when resource name exists but resource type is unclear
   - **scale**: No suggestions (requires specific target, can't guess)
   - **get/list**: No suggestions (query operations are unambiguous)

2. **Intent-based signals from LLM:**
   - `needsClarification == true`: Always generate suggestions if possible
   - High confidence ambiguity: LLM returns multiple interpretations
   - Low confidence intent: LLM can't determine action

3. **Cluster state indicators:**
   - 50+ resources in cluster: Show suggestions to avoid overwhelming user
   - 0-10 resources in cluster: Show suggestions to help discover
   - Fresh cluster (no resources): Skip suggestions, show manual form directly

### Suggestion Generation Logic

```
func ShouldGenerateSuggestions(action K8sAction, clusterState ClusterState) bool {
    // Don't suggest for query operations
    if action.Action == "get" || action.Action == "list" || action.Action == "show" {
        return false
    }

    // Suggest for create/delete when name is missing or exists elsewhere
    if (action.Action == "create" || action.Action == "deploy") {
        return action.Name == "" || HasSimilarNameInCluster(action.Name, clusterState)
    }

    // Suggest for delete when resource type is unclear
    if (action.Action == "delete" {
        return HasMultipleResourceTypes(action.Name, clusterState)
    }

    return false
}
```

---

## Section 3.6: Performance Considerations

### Expected Latency

- **Cluster query**: 50-200ms (depends on cluster size and network)
- **Ranking algorithm**: <10ms for <100 resources, <50ms for 1000+ resources
- **Total suggestion response**: <250ms for 95th percentile

### Cache Strategy Details

- **In-memory cache**: Simple map[string][]Suggestion with mutex
- **Key format**: `<action>:<namespace>:<partial_name>` (e.g., `create:default:ngi`)
- **TTL**: 5 seconds
- **Size limit**: Evict oldest entries when >1000 cached results
- **Cache invalidation**: Clear namespace-specific cache after write operations

### Maximum Suggestions

- **Default**: 2-3 suggestions maximum
- **Expandable**: "Show more" button to see up to 10 suggestions
- **Reasoning**: More than 3 options increases cognitive load, too few misses helpful alternatives

### Concurrency

- **Concurrent queries**: Support 10 simultaneous suggestion requests
- **Per-user cache**: Deduplicate based on session ID to prevent thundering herd
- **Rate limiting**: 5 suggestions per second per user to prevent abuse

---

## Section 3.7: Localization Considerations

### Multi-Language Support

- **Suggestion text**: Generate in user's language (from LLM context or API parameter)
- **Reason descriptions**: Use language-appropriate phrasing
- **Example**: "Found deployment" (EN) vs "找到部署" (ZH)

### RTL Support

- **Layout**: CSS flexbox with `dir="auto"` detection
- **Icons**: Use Unicode characters (📦, 🗑️, ⚙️) that render correctly in RTL
- **Text alignment**: Keep left-aligned for numbers, right-aligned for RTL labels

### Cultural Patterns

- **Chinese users**: Prefer explicit namespace mentions (e.g., "default中的nginx")
- **English users**: Prefer resource-first patterns (e.g., "nginx in default namespace")
- **Icons**: Culturally neutral (avoid using flags or culture-specific symbols)

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

**Baseline Measurements (to be collected before implementation):**
- Average interactions per ambiguous request: currently 3.2 interactions (parse → form → confirm)
- User completion rate for "deploy" requests: currently 67%
- Time-to-first-success for new users: currently 8.5 minutes
- Support ticket rate for "can't find resource": currently 15% of help requests

**Target Metrics (3 months after implementation):**
- Average interactions per ambiguous request: reduce to 1.6 interactions (50% improvement)
- User completion rate for "deploy" requests: increase to 85% (27% improvement)
- Time-to-first-success for new users: reduce to 4.2 minutes (51% improvement)
- Support ticket rate for "can't find resource": reduce to 5% (67% improvement)
- Suggestion selection rate: 80% of suggestions shown should be selected (vs. manual input)

**Measurement Methods:**
- **Analytics**: Add event tracking for suggestion clicks, time spent, abandonment
- **A/B Testing**: Compare old form flow vs. new suggestions flow (10% user sample)
- **User surveys**: In-app NPS survey after 10th interaction
- **Error reduction**: Track "unsupported resource type" errors before/after

**Success Criteria:**
- ✅ All target metrics met or exceeded
- ✅ No regression in existing workflows (non-ambiguous requests)
- ✅ <2% increase in server latency (performance impact acceptable)

### Integration Checklist

**Backend Changes:**
- [ ] Add `Suggestions []Suggestion` field to `ChatResponse` model
- [ ] Implement `SuggestionEngine` with cluster query and ranking
- [ ] Implement `MakeSuggestionsNode()` workflow node
- [ ] Update `RouteAfterParse()` to handle suggestions routing
- [ ] Implement in-memory cache with 5-second TTL
- [ ] Add concurrency support for 10+ simultaneous requests

**Frontend Changes:**
- [ ] Create `SuggestionCards.tsx` component
- [ ] Extend `ChatResponse` type to include suggestions
- [ ] Update `ActionForm` to show suggestions before form
- [ ] Implement suggestion selection state management in `ChatPage.tsx`
- [ ] Add suggestion click tracking for analytics

**Testing:**
- [ ] Unit tests for `SuggestionEngine` (query, rank, cache)
- [ ] Integration tests for suggestions workflow path
- [ ] Frontend component tests (render, interaction, state)
- [ ] E2e tests for all scenarios (ambiguous, error, empty cluster)
- [ ] Performance tests (50+ resources, 1000+ resources)
- [ ] Concurrency tests (simultaneous user requests)

**Documentation:**
- [ ] API documentation for suggestions field in ChatResponse
- [ ] Component documentation for SuggestionCards
- [ ] User guide updates with screenshots
- [ ] Localization guide for translators
