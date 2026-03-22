package workflow

import (
	"context"
	"testing"

	"k8s-wizard/api/models"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MockClient for testing
type MockClient struct {
	deployments []appsv1.Deployment
}

func (m *MockClient) CreateDeployment(ctx context.Context, namespace, name, image string, replicas int32) (string, error) {
	return "", nil
}

func (m *MockClient) GetResources(ctx context.Context, namespace, resourceType string) (string, error) {
	// Return mock deployment list
	return `🚀 集群中的 Deployment (共 2 个):
  • nginx-deployment (副本: 3/3)
  • nginx-ingress (副本: 2/2)`, nil
}

func (m *MockClient) ListDeployments(ctx context.Context, namespace string) (*appsv1.DeploymentList, error) {
	if len(m.deployments) == 0 {
		m.deployments = []appsv1.Deployment{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nginx-deployment",
					Namespace: "default",
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nginx-ingress",
					Namespace: "default",
				},
			},
		}
	}
	return &appsv1.DeploymentList{Items: m.deployments}, nil
}

func (m *MockClient) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) (string, error) {
	return "", nil
}

func (m *MockClient) DeleteResource(ctx context.Context, namespace, name, resourceType string) (string, error) {
	return "", nil
}

func (m *MockClient) GetPodLogs(ctx context.Context, namespace, pod, container string, tailLines int64) (string, error) {
	return "", nil
}

func (m *MockClient) ExecPod(ctx context.Context, namespace, pod, container string, command []string) (string, error) {
	return "", nil
}

func TestNewSuggestionEngine(t *testing.T) {
	client := &MockClient{}
	engine := NewSuggestionEngine(client)

	if engine == nil {
		t.Fatal("expected engine to be non-nil")
	}

	if engine.client != client {
		t.Error("expected engine client to match provided client")
	}

	if engine.cache == nil {
		t.Error("expected engine cache to be initialized")
	}

	if engine.cacheTTL != 5*1000000000 { // 5 seconds in nanoseconds
		t.Error("expected cache TTL to be 5 seconds")
	}
}

func TestQueryClusterCompile(t *testing.T) {
	// This test verifies that SuggestionEngine can be created and has QueryCluster method
	client := &MockClient{}
	engine := NewSuggestionEngine(client)

	if engine == nil {
		t.Fatalf("expected engine to be non-nil")
	}

	ctx := context.Background()
	req := models.SuggestionRequest{
		Action:    "create",
		Resource:  "deployment",
		Name:      "nginx",
		Namespace: "default",
	}

	// This will fail until QueryCluster is implemented
	suggestions, err := engine.QueryCluster(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if suggestions == nil {
		t.Fatal("expected suggestions to be non-nil")
	}
}

func TestQueryClusterSuccess(t *testing.T) {
	client := &MockClient{}
	engine := NewSuggestionEngine(client)

	ctx := context.Background()
	req := models.SuggestionRequest{
		Action:    "create",
		Resource:  "deployment",
		Name:      "nginx",
		Namespace: "default",
	}

	suggestions, err := engine.QueryCluster(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if suggestions == nil {
		t.Fatal("expected suggestions to be non-nil")
	}

	if len(suggestions) == 0 {
		t.Fatal("expected at least one suggestion")
	}
}

func TestQueryClusterCache(t *testing.T) {
	client := &MockClient{}
	engine := NewSuggestionEngine(client)

	ctx := context.Background()
	req := models.SuggestionRequest{
		Action:    "create",
		Resource:  "deployment",
		Name:      "nginx",
		Namespace: "default",
	}

	// First call - should hit cluster
	suggestions1, err1 := engine.QueryCluster(ctx, req)
	if err1 != nil {
		t.Fatalf("expected no error on first call, got: %v", err1)
	}

	// Second call - should hit cache
	suggestions2, err2 := engine.QueryCluster(ctx, req)
	if err2 != nil {
		t.Fatalf("expected no error on second call, got: %v", err2)
	}

	if len(suggestions1) != len(suggestions2) {
		t.Errorf("expected same number of suggestions from cache, got %d vs %d",
			len(suggestions1), len(suggestions2))
	}
}

func TestBuildCacheKey(t *testing.T) {
	req := models.SuggestionRequest{
		Action:    "create",
		Resource:  "deployment",
		Name:      "nginx",
		Namespace: "default",
	}

	key := buildCacheKey(req)
	expectedKey := "create:default:nginx"

	if key != expectedKey {
		t.Errorf("expected cache key %s, got %s", expectedKey, key)
	}
}
