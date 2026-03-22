package workflow

import (
	"context"
	"testing"

	"k8s-wizard/api/models"
)

func TestRouteAfterParse(t *testing.T) {
	tests := []struct {
		name     string
		state    AgentState
		expected string
	}{
		{
			name: "chat response - non K8s operation",
			state: AgentState{
				Status:         StatusChat,
				IsK8sOperation: false,
			},
			expected: "END",
		},
		{
			name: "error status",
			state: AgentState{
				Status: StatusError,
				Error:  &testError{},
			},
			expected: "END",
		},
		{
			name: "K8s operation with form data - show_suggestions",
			state: AgentState{
				Status:         StatusPending,
				IsK8sOperation: true,
				FormData:       map[string]interface{}{"name": "nginx"},
				Action: &K8sAction{
					Action:   "create",
					Resource: "deployment",
				},
			},
			expected: "show_suggestions",
		},
		{
			name: "K8s operation without form data - show_suggestions",
			state: AgentState{
				Status:         StatusPending,
				IsK8sOperation: true,
				FormData:       nil,
				Action: &K8sAction{
					Action:   "create",
					Resource: "deployment",
				},
			},
			expected: "show_suggestions",
		},
		{
			name: "get action - show_suggestions",
			state: AgentState{
				Status:         StatusPending,
				IsK8sOperation: true,
				Action: &K8sAction{
					Action:   "get",
					Resource: "pod",
				},
			},
			expected: "show_suggestions",
		},
		{
			name: "K8s operation with suggestions - show_suggestions",
			state: AgentState{
				Status:         StatusPending,
				IsK8sOperation: true,
				Suggestions: []models.Suggestion{
					{Type: "reuse", Name: "nginx", ID: "reuse-nginx-123"},
				},
				NeedsClarification: false,
				Action: &K8sAction{
					Action:   "create",
					Resource: "deployment",
				},
			},
			expected: "show_suggestions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RouteAfterParse(context.Background(), tt.state)
			if result != tt.expected {
				t.Errorf("RouteAfterParse() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRouteAfterClarify(t *testing.T) {
	tests := []struct {
		name     string
		state    AgentState
		expected string
	}{
		{
			name: "needs clarification",
			state: AgentState{
				NeedsClarification: true,
				ClarificationRequest: &models.ClarificationRequest{
					Type:  "form",
					Title: "Need info",
				},
			},
			expected: "END",
		},
		{
			name: "no clarification needed - generate_preview",
			state: AgentState{
				NeedsClarification: false,
				Action: &K8sAction{
					Action: "delete",
				},
			},
			expected: "generate_preview",
		},
		{
			name: "create action - generate_preview",
			state: AgentState{
				NeedsClarification: false,
				Action: &K8sAction{
					Action: "create",
				},
			},
			expected: "generate_preview",
		},
		{
			name: "scale action - generate_preview",
			state: AgentState{
				NeedsClarification: false,
				Action: &K8sAction{
					Action: "scale",
				},
			},
			expected: "generate_preview",
		},
		{
			name: "get action - generate_preview",
			state: AgentState{
				NeedsClarification: false,
				Action: &K8sAction{
					Action: "get",
				},
			},
			expected: "generate_preview",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RouteAfterClarify(context.Background(), tt.state)
			if result != tt.expected {
				t.Errorf("RouteAfterClarify() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestRouteAfterPreview(t *testing.T) {
	tests := []struct {
		name     string
		state    AgentState
		expected string
	}{
		{
			name: "no preview - unsupported operation",
			state: AgentState{
				ActionPreview: nil,
			},
			expected: "END",
		},
		{
			name: "get action - execute directly",
			state: AgentState{
				Action: &K8sAction{
					Action: "get",
				},
				ActionPreview: &models.ActionPreview{
					Type: "get",
				},
			},
			expected: "execute",
		},
		{
			name: "list action - execute directly",
			state: AgentState{
				Action: &K8sAction{
					Action: "list",
				},
				ActionPreview: &models.ActionPreview{
					Type: "get",
				},
			},
			expected: "execute",
		},
		{
			name: "user confirmed - execute",
			state: AgentState{
				Confirm: boolPtr(true),
				Action: &K8sAction{
					Action: "create",
				},
				ActionPreview: &models.ActionPreview{
					Type: "create",
				},
			},
			expected: "execute",
		},
		{
			name: "user rejected - END",
			state: AgentState{
				Confirm: boolPtr(false),
				Action: &K8sAction{
					Action: "create",
				},
				ActionPreview: &models.ActionPreview{
					Type: "create",
				},
			},
			expected: "END",
		},
		{
			name: "no confirmation yet - END",
			state: AgentState{
				Confirm: nil,
				Action: &K8sAction{
					Action: "create",
				},
				ActionPreview: &models.ActionPreview{
					Type: "create",
				},
			},
			expected: "END",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RouteAfterPreview(context.Background(), tt.state)
			if result != tt.expected {
				t.Errorf("RouteAfterPreview() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// Test error type
type testError struct{}

func (e *testError) Error() string {
	return "test error"
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
