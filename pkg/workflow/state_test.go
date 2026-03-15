package workflow

import (
	"testing"

	"k8s-wizard/pkg/tools"
)

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"StatusPending", StatusPending, "pending"},
		{"StatusNeedsInfo", StatusNeedsInfo, "needs_info"},
		{"StatusNeedsConfirm", StatusNeedsConfirm, "needs_confirm"},
		{"StatusExecuted", StatusExecuted, "executed"},
		{"StatusError", StatusError, "error"},
		{"StatusChat", StatusChat, "chat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestAgentStateDefaults(t *testing.T) {
	state := AgentState{}

	if state.UserMessage != "" {
		t.Errorf("UserMessage default = %q, want empty", state.UserMessage)
	}
	if state.Status != "" {
		t.Errorf("Status default = %q, want empty", state.Status)
	}
	if state.FormData != nil {
		t.Errorf("FormData default = %v, want nil", state.FormData)
	}
	if state.Action != nil {
		t.Errorf("Action default = %v, want nil", state.Action)
	}
}

func TestAgentStateWithValues(t *testing.T) {
	state := AgentState{
		UserMessage: "create a nginx deployment",
		FormData:    map[string]interface{}{"replicas": 3},
		Confirm:     boolPtr(true),
		ThreadID:    "thread-123",
		Status:      StatusPending,
	}

	if state.UserMessage != "create a nginx deployment" {
		t.Errorf("UserMessage = %q, want %q", state.UserMessage, "create a nginx deployment")
	}
	if state.FormData["replicas"] != 3 {
		t.Errorf("FormData[replicas] = %v, want 3", state.FormData["replicas"])
	}
	if *state.Confirm != true {
		t.Errorf("Confirm = %v, want true", *state.Confirm)
	}
	if state.ThreadID != "thread-123" {
		t.Errorf("ThreadID = %q, want %q", state.ThreadID, "thread-123")
	}
	if state.Status != StatusPending {
		t.Errorf("Status = %q, want %q", state.Status, StatusPending)
	}
}

func TestK8sAction(t *testing.T) {
	action := K8sAction{
		Action:         "create",
		Resource:       "deployment",
		Name:           "nginx",
		Namespace:      "default",
		Params:         map[string]interface{}{"image": "nginx:latest", "replicas": 3},
		IsK8sOperation: true,
		Reply:          "",
	}

	if action.Action != "create" {
		t.Errorf("Action = %q, want %q", action.Action, "create")
	}
	if action.Resource != "deployment" {
		t.Errorf("Resource = %q, want %q", action.Resource, "deployment")
	}
	if action.Name != "nginx" {
		t.Errorf("Name = %q, want %q", action.Name, "nginx")
	}
	if action.Namespace != "default" {
		t.Errorf("Namespace = %q, want %q", action.Namespace, "default")
	}
	if action.Params["image"] != "nginx:latest" {
		t.Errorf("Params[image] = %v, want %q", action.Params["image"], "nginx:latest")
	}
	if !action.IsK8sOperation {
		t.Errorf("IsK8sOperation = %v, want true", action.IsK8sOperation)
	}
}

func TestK8sActionNonK8s(t *testing.T) {
	action := K8sAction{
		Action:         "",
		Resource:       "",
		Name:           "",
		Namespace:      "",
		Params:         nil,
		IsK8sOperation: false,
		Reply:          "你好！我是 K8s Wizard。",
	}

	if action.IsK8sOperation {
		t.Errorf("IsK8sOperation = %v, want false", action.IsK8sOperation)
	}
	if action.Reply != "你好！我是 K8s Wizard。" {
		t.Errorf("Reply = %q, want %q", action.Reply, "你好！我是 K8s Wizard。")
	}
}

func TestDependencies(t *testing.T) {
	deps := Dependencies{
		ModelName: "gpt-4",
	}

	if deps.ModelName != "gpt-4" {
		t.Errorf("ModelName = %q, want %q", deps.ModelName, "gpt-4")
	}
	if deps.K8sClient != nil {
		t.Errorf("K8sClient should be nil by default")
	}
	if deps.LLM != nil {
		t.Errorf("LLM should be nil by default")
	}
}

func TestDependenciesWithToolRegistry(t *testing.T) {
	deps := Dependencies{
		ToolRegistry: tools.NewRegistry(),
		// other fields omitted for brevity
	}

	if deps.ToolRegistry == nil {
		t.Errorf("ToolRegistry field should not be nil in Dependencies")
	}
}

