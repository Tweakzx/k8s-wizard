package models

// Suggestion represents a cluster-aware recommendation for ambiguous requests
type Suggestion struct {
	Type       string  `json:"type"`       // "reuse", "create", "none"
	Action     string  `json:"action"`     // "create", "delete", "scale"
	Resource   string  `json:"resource"`   // "deployment", "pod", "service"
	Name       string  `json:"name"`       // Resource name
	Namespace  string  `json:"namespace"`  // Resource namespace
	Reason     string  `json:"reason"`     // Why this suggestion
	Confidence float64 `json:"confidence"` // 0.0 to 1.0
	Existing   bool    `json:"existing"`   // Is this already in cluster?
	ID         string  `json:"id,omitempty"` // Unique identifier for tracking
}

// SuggestionRequest represents a request for suggestions
type SuggestionRequest struct {
	Action    string         `json:"action"`    // Parsed action from LLM
	Resource  string         `json:"resource"`  // Parsed resource type
	Name      string         `json:"name"`      // Parsed resource name (may be empty)
	Namespace string         `json:"namespace"` // Parsed namespace
	Params    map[string]any `json:"params,omitempty"` // Additional parameters
}
