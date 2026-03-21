package workflow

import (
	"container/list"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s-wizard/api/models"
	"k8s-wizard/pkg/k8s"
	"k8s-wizard/pkg/logger"
)

const suggestionCacheMaxSize = 256

type suggestionCacheEntry struct {
	suggestions []models.Suggestion
	cachedAt    time.Time
}

type SuggestionEngine struct {
	client     k8s.Client
	cache      map[string]suggestionCacheEntry
	cacheOrder *list.List
	cacheIndex map[string]*list.Element
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
	cacheLimit int
}

func NewSuggestionEngine(client k8s.Client) *SuggestionEngine {
	return &SuggestionEngine{
		client:     client,
		cache:      make(map[string]suggestionCacheEntry),
		cacheOrder: list.New(),
		cacheIndex: make(map[string]*list.Element),
		cacheTTL:   5 * time.Second,
		cacheLimit: suggestionCacheMaxSize,
	}
}

func (e *SuggestionEngine) QueryCluster(ctx context.Context, req models.SuggestionRequest) ([]models.Suggestion, error) {
	// Check cache first
	cacheKey := buildCacheKey(req)
	if cached, found := e.getCachedSuggestions(cacheKey); found {
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
		matches := findNameMatches(req.Name, deployments)
		suggestions = append(suggestions, matches...)
	} else {
		// Name not specified, suggest "Specify name" option
		suggestions = append(suggestions, models.Suggestion{
			Type:       "none",
			Action:     req.Action,
			Resource:   req.Resource,
			Name:       "",
			Namespace:  req.Namespace,
			Reason:     "Please specify the deployment name you want to create",
			Confidence: 1.0,
			Existing:   false,
		})
	}

	// Store in cache
	e.setCachedSuggestions(cacheKey, suggestions)

	return suggestions, nil
}

func buildCacheKey(req models.SuggestionRequest) string {
	return fmt.Sprintf("%s:%s:%s", req.Action, req.Namespace, req.Name)
}

func (e *SuggestionEngine) getCachedSuggestions(key string) ([]models.Suggestion, bool) {
	now := time.Now()

	e.cacheMutex.Lock()
	defer e.cacheMutex.Unlock()

	entry, found := e.cache[key]
	if !found {
		return nil, false
	}

	if now.Sub(entry.cachedAt) > e.cacheTTL {
		e.removeCacheEntryLocked(key)
		return nil, false
	}

	e.markCacheAsRecentLocked(key)
	return entry.suggestions, true
}

func (e *SuggestionEngine) setCachedSuggestions(key string, suggestions []models.Suggestion) {
	e.cacheMutex.Lock()
	defer e.cacheMutex.Unlock()

	e.cache[key] = suggestionCacheEntry{
		suggestions: suggestions,
		cachedAt:    time.Now(),
	}
	e.markCacheAsRecentLocked(key)

	for len(e.cache) > e.cacheLimit {
		oldest := e.cacheOrder.Back()
		if oldest == nil {
			break
		}

		oldestKey, ok := oldest.Value.(string)
		if !ok {
			e.cacheOrder.Remove(oldest)
			continue
		}
		e.removeCacheEntryLocked(oldestKey)
	}
}

func (e *SuggestionEngine) markCacheAsRecentLocked(key string) {
	if elem, exists := e.cacheIndex[key]; exists {
		e.cacheOrder.MoveToFront(elem)
		return
	}

	e.cacheIndex[key] = e.cacheOrder.PushFront(key)
}

func (e *SuggestionEngine) removeCacheEntryLocked(key string) {
	delete(e.cache, key)
	if elem, exists := e.cacheIndex[key]; exists {
		e.cacheOrder.Remove(elem)
		delete(e.cacheIndex, key)
	}
}

// MakeSuggestionsNode creates a workflow node that generates suggestions based on parsed intent
func MakeSuggestionsNode(engine *SuggestionEngine) NodeFunc {
	return func(ctx context.Context, state AgentState) (AgentState, error) {
		// Generate suggestions based on parsed action
		if state.Action == nil {
			return state, fmt.Errorf("no action found in state")
		}

		req := models.SuggestionRequest{
			Action:    state.Action.Action,
			Resource:  state.Action.Resource,
			Name:      state.Action.Name,
			Namespace: state.Action.Namespace,
		}

		suggestions, err := engine.QueryCluster(ctx, req)
		if err != nil {
			logger.Error("failed to generate suggestions", "error", err)
			// Continue without suggestions - will show standard form
			return state, nil
		}

		state.Suggestions = suggestions
		logger.Debug("generated suggestions", "count", len(suggestions))
		return state, nil
	}
}

func findNameMatches(name string, deployments string) []models.Suggestion {
	var matches []models.Suggestion

	// Parse deployment names from the formatted output
	lines := strings.Split(deployments, "\n")
	for _, line := range lines {
		if strings.Contains(line, "•") {
			// Extract deployment name from line format:
			// "  • nginx-deployment (副本: 3/3)"
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				depName := strings.TrimPrefix(parts[1], "[default]")
				depName = strings.TrimSpace(depName)

				// Calculate confidence based on match type
				confidence := 0.0
				isExisting := true

				if depName == name {
					// Exact match
					confidence = 1.0
				} else if strings.Contains(strings.ToLower(depName), strings.ToLower(name)) ||
					strings.Contains(strings.ToLower(name), strings.ToLower(depName)) {
					// Partial match
					confidence = 0.6
				} else {
					// No match
					isExisting = false
				}

				if confidence > 0 {
					matches = append(matches, models.Suggestion{
						Type:       "reuse",
						Action:     "create",
						Resource:   "deployment",
						Name:       depName,
						Namespace:  "default",
						Reason:     fmt.Sprintf("Found existing deployment with similar name"),
						Confidence: confidence,
						Existing:   isExisting,
					})
				}
			}
		}
	}

	// If no existing matches, add create suggestion
	if len(matches) == 0 {
		matches = append(matches, models.Suggestion{
			Type:       "create",
			Action:     "create",
			Resource:   "deployment",
			Name:       name,
			Namespace:  "default",
			Reason:     fmt.Sprintf("Create new deployment named '%s'", name),
			Confidence: 1.0,
			Existing:   false,
		})
	}

	return matches
}
