package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"k8s-wizard/pkg/agent"
)

// mockE2EClient implements llm.Client for e2e testing
type mockE2EClient struct {
	// responses maps user message patterns to LLM responses
	responses map[string]string
}

func (m *mockE2EClient) Chat(ctx context.Context, prompt string) (string, error) {
	// Extract user message from prompt (it's after "用户指令:")
	for pattern, resp := range m.responses {
		if strings.Contains(prompt, pattern) {
			return resp, nil
		}
	}
	// Default response
	return `{"action":"create","resource":"deployment","name":"test-app","namespace":"default","params":{"replicas":1},"is_k8s_operation":true,"reply":""}`, nil
}

func (m *mockE2EClient) GetModel() string {
	return "mock-model"
}

// TestE2ECreateDeployment tests creating a deployment through the agent
func TestE2ECreateDeployment(t *testing.T) {
	tests := []struct {
		name      string
		userInput string
		formData  map[string]interface{}
		wantErr   bool
	}{
		{
			name:      "create nginx deployment with form",
			userInput: "部署一个 nginx",
			formData: map[string]interface{}{
				"name":      "nginx",
				"image":     "nginx:latest",
				"replicas":  3,
				"namespace": "default",
			},
			wantErr: false,
		},
		{
			name:      "create redis deployment",
			userInput: "创建一个 redis 应用",
			formData: map[string]interface{}{
				"name":      "redis",
				"image":     "redis:alpine",
				"replicas":  1,
				"namespace": "default",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock LLM client
			mockLLM := &mockE2EClient{
				responses: map[string]string{
					tt.userInput: fmt.Sprintf(`{"action":"create","resource":"deployment","name":"%s","namespace":"default","params":{},"is_k8s_operation":true,"reply":""}`, tt.formData["name"]),
				},
			}

			// Create agent with nil K8s client (workflow testing)
			ag, err := agent.NewGraphAgent(nil, mockLLM, "test-model")
			if err != nil {
				t.Fatalf("Failed to create agent: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// First request - may need clarification
			result, clarification, preview, err := ag.ProcessCommandWithClarification(ctx, tt.userInput, nil, nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Submit form data if clarification needed
			if clarification != nil {
				result, clarification, preview, err = ag.ProcessCommandWithClarification(ctx, tt.userInput, tt.formData, nil)
				if err != nil {
					t.Fatalf("Unexpected error submitting form: %v", err)
				}
			}

			// Should show preview for confirmation
			if preview != nil {
				t.Logf("Preview: type=%s, resource=%s, namespace=%s", preview.Type, preview.Resource, preview.Namespace)

				// Verify preview content
				if preview.Type != "create" {
					t.Errorf("Preview type = %s, want create", preview.Type)
				}
				if preview.Namespace != "default" {
					t.Errorf("Preview namespace = %s, want default", preview.Namespace)
				}

				// Confirm and execute (will fail without K8s client)
				confirm := true
				result, _, _, err = ag.ProcessCommandWithClarification(ctx, tt.userInput, tt.formData, &confirm)
				if err != nil {
					t.Logf("Expected error (no K8s client): %v", err)
					return
				}
			}

			t.Logf("Result: %s", result)
		})
	}
}

// TestE2EGetResources tests querying resources through the agent
func TestE2EGetResources(t *testing.T) {
	tests := []struct {
		name         string
		userInput    string
		resourceType string
		namespace    string
		wantErr      bool
	}{
		{
			name:         "list pods in default namespace",
			userInput:    "查看所有 pod",
			resourceType: "pod",
			namespace:    "default",
			wantErr:      false,
		},
		{
			name:         "list deployments",
			userInput:    "查看所有 deployment",
			resourceType: "deployment",
			namespace:    "default",
			wantErr:      false,
		},
		{
			name:         "list services",
			userInput:    "查看所有 service",
			resourceType: "service",
			namespace:    "default",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock LLM client
			mockLLM := &mockE2EClient{
				responses: map[string]string{
					tt.userInput: fmt.Sprintf(`{"action":"get","resource":"%s","name":"","namespace":"","params":{},"is_k8s_operation":true,"reply":""}`, tt.resourceType),
				},
			}

			// Create agent (nil K8s client for workflow testing)
			ag, err := agent.NewGraphAgent(nil, mockLLM, "test-model")
			if err != nil {
				t.Fatalf("Failed to create agent: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Process get command (should generate preview for read-only ops)
			result, clarification, preview, err := ag.ProcessCommandWithClarification(ctx, tt.userInput, nil, nil)
			if err != nil {
				if tt.wantErr {
					return
				}
				t.Fatalf("Unexpected error: %v", err)
			}

			// Get operations should generate preview
			if preview != nil {
				t.Logf("Preview generated: %s", preview.Summary)

				// Verify preview content
				if preview.Type != "get" {
					t.Errorf("Preview type = %s, want get", preview.Type)
				}
				if preview.Resource != tt.resourceType {
					t.Errorf("Preview resource = %s, want %s", preview.Resource, tt.resourceType)
				}

				// Confirm execution (will fail without K8s client)
				confirm := true
				result, _, _, err = ag.ProcessCommandWithClarification(ctx, tt.userInput, nil, &confirm)
				if err != nil {
					t.Logf("Expected error (no K8s client): %v", err)
				}
			}

			if clarification != nil {
				t.Logf("Clarification: %s", clarification.Title)
			}

			t.Logf("Result: %s", result)
		})
	}
}

// TestE2EDeleteResource tests deleting resources through the agent
func TestE2EDeleteResource(t *testing.T) {
	tests := []struct {
		name         string
		userInput    string
		resourceType string
		resourceName string
		namespace    string
		formData     map[string]interface{}
		wantErr      bool
	}{
		{
			name:         "delete deployment",
			userInput:    "删除 nginx deployment",
			resourceType: "deployment",
			resourceName: "nginx",
			namespace:    "default",
			formData: map[string]interface{}{
				"name":      "nginx",
				"namespace": "default",
			},
			wantErr: false,
		},
		{
			name:         "delete pod",
			userInput:    "删除 test-pod",
			resourceType: "pod",
			resourceName: "test-pod",
			namespace:    "default",
			formData: map[string]interface{}{
				"name":      "test-pod",
				"namespace": "default",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock LLM client
			mockLLM := &mockE2EClient{
				responses: map[string]string{
					tt.userInput: fmt.Sprintf(`{"action":"delete","resource":"%s","name":"%s","namespace":"%s","params":{},"is_k8s_operation":true,"reply":""}`, tt.resourceType, tt.resourceName, tt.namespace),
				},
			}

			// Create agent
			ag, err := agent.NewGraphAgent(nil, mockLLM, "test-model")
			if err != nil {
				t.Fatalf("Failed to create agent: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Process delete command
			result, clarification, preview, err := ag.ProcessCommandWithClarification(ctx, tt.userInput, nil, nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// If clarification needed, provide form data
			if clarification != nil {
				result, clarification, preview, err = ag.ProcessCommandWithClarification(ctx, tt.userInput, tt.formData, nil)
				if err != nil {
					t.Fatalf("Unexpected error after form: %v", err)
				}
			}

			// Should show preview for confirmation
			if preview != nil {
				t.Logf("Preview: %s", preview.Summary)

				// Verify preview content
				if preview.Type != "delete" {
					t.Errorf("Preview type = %s, want delete", preview.Type)
				}
				if preview.DangerLevel != "high" {
					t.Errorf("Preview dangerLevel = %s, want high", preview.DangerLevel)
				}

				// Confirm deletion (will fail without K8s client)
				confirm := true
				result, _, _, err = ag.ProcessCommandWithClarification(ctx, tt.userInput, tt.formData, &confirm)
				if err != nil {
					t.Logf("Expected error (no K8s client): %v", err)
					return
				}
			}

			t.Logf("Result: %s", result)
		})
	}
}

// TestE2EScaleDeployment tests scaling deployments
func TestE2EScaleDeployment(t *testing.T) {
	tests := []struct {
		name        string
		userInput   string
		deployment  string
		namespace   string
		targetReps  int
		formData    map[string]interface{}
		wantErr     bool
	}{
		{
			name:       "scale up deployment",
			userInput:  "把 nginx 扩容到 5 个副本",
			deployment: "nginx",
			namespace:  "default",
			targetReps: 5,
			formData: map[string]interface{}{
				"name":      "nginx",
				"replicas":  5,
				"namespace": "default",
			},
			wantErr: false,
		},
		{
			name:       "scale down deployment",
			userInput:  "把 web 缩容到 2 个副本",
			deployment: "web",
			namespace:  "default",
			targetReps: 2,
			formData: map[string]interface{}{
				"name":      "web",
				"replicas":  2,
				"namespace": "default",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock LLM client
			mockLLM := &mockE2EClient{
				responses: map[string]string{
					tt.userInput: fmt.Sprintf(`{"action":"scale","resource":"deployment","name":"%s","namespace":"%s","params":{"replicas":%d},"is_k8s_operation":true,"reply":""}`, tt.deployment, tt.namespace, tt.targetReps),
				},
			}

			// Create agent
			ag, err := agent.NewGraphAgent(nil, mockLLM, "test-model")
			if err != nil {
				t.Fatalf("Failed to create agent: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Process scale command
			result, clarification, preview, err := ag.ProcessCommandWithClarification(ctx, tt.userInput, nil, nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// If clarification needed
			if clarification != nil {
				result, clarification, preview, err = ag.ProcessCommandWithClarification(ctx, tt.userInput, tt.formData, nil)
				if err != nil {
					t.Fatalf("Unexpected error after form: %v", err)
				}
			}

			// Confirm and execute
			if preview != nil {
				t.Logf("Preview: %s", preview.Summary)

				// Verify preview content
				if preview.Type != "scale" {
					t.Errorf("Preview type = %s, want scale", preview.Type)
				}
				if preview.DangerLevel != "medium" {
					t.Errorf("Preview dangerLevel = %s, want medium", preview.DangerLevel)
				}

				confirm := true
				result, _, _, err = ag.ProcessCommandWithClarification(ctx, tt.userInput, tt.formData, &confirm)
				if err != nil {
					t.Logf("Expected error (no K8s client): %v", err)
					return
				}
			}

			t.Logf("Result: %s", result)
		})
	}
}

// TestE2EChat tests non-K8s chat responses
func TestE2EChat(t *testing.T) {
	tests := []struct {
		name      string
		userInput string
		wantReply bool
	}{
		{
			name:      "greeting",
			userInput: "你好",
			wantReply: true,
		},
		{
			name:      "question about k8s",
			userInput: "什么是 Kubernetes？",
			wantReply: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockE2EClient{
				responses: map[string]string{
					"你好":                     `{"action":"","resource":"","name":"","namespace":"","params":{},"is_k8s_operation":false,"reply":"你好！我是 K8s Wizard，可以通过自然语言帮你管理 Kubernetes 集群。"}`,
					"什么是 Kubernetes": `{"action":"","resource":"","name":"","namespace":"","params":{},"is_k8s_operation":false,"reply":"Kubernetes 是一个开源的容器编排平台，用于自动化部署、扩展和管理容器化应用程序。"}`,
				},
			}

			ag, err := agent.NewGraphAgent(nil, mockLLM, "test-model")
			if err != nil {
				t.Fatalf("Failed to create agent: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			result, clarification, preview, err := ag.ProcessCommandWithClarification(ctx, tt.userInput, nil, nil)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Chat messages should not need clarification or preview
			if clarification != nil {
				t.Errorf("Unexpected clarification: %s", clarification.Title)
			}

			if preview != nil {
				t.Errorf("Unexpected preview: %s", preview.Summary)
			}

			if tt.wantReply && result == "" {
				t.Error("Expected chat response")
			}

			t.Logf("Chat response: %s", result)
		})
	}
}

// TestE2EWorkflowStates tests different workflow states
func TestE2EWorkflowStates(t *testing.T) {
	t.Run("create_needs_clarification", func(t *testing.T) {
		mockLLM := &mockE2EClient{
			responses: map[string]string{
				"部署应用": `{"action":"create","resource":"deployment","name":"","namespace":"default","params":{},"is_k8s_operation":true,"reply":""}`,
			},
		}

		ag, err := agent.NewGraphAgent(nil, mockLLM, "test-model")
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		ctx := context.Background()
		_, clarification, _, err := ag.ProcessCommandWithClarification(ctx, "部署应用", nil, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if clarification == nil {
			t.Error("Expected clarification request for incomplete create")
		} else {
			t.Logf("Clarification title: %s", clarification.Title)
			t.Logf("Number of fields: %d", len(clarification.Fields))
		}
	})

	t.Run("create_with_complete_info", func(t *testing.T) {
		mockLLM := &mockE2EClient{
			responses: map[string]string{
				"部署 nginx": `{"action":"create","resource":"deployment","name":"nginx","namespace":"default","params":{"image":"nginx:latest","replicas":3},"is_k8s_operation":true,"reply":""}`,
			},
		}

		ag, err := agent.NewGraphAgent(nil, mockLLM, "test-model")
		if err != nil {
			t.Fatalf("Failed to create agent: %v", err)
		}

		ctx := context.Background()
		_, clarification, preview, err := ag.ProcessCommandWithClarification(ctx, "部署 nginx", nil, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// With complete info, should go directly to preview
		if clarification != nil {
			t.Logf("Got clarification (unexpected): %s", clarification.Title)
		}

		if preview == nil {
			t.Error("Expected preview for complete create request")
		} else {
			t.Logf("Preview: %s", preview.Summary)
		}
	})
}
