package handlers

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s-wizard/pkg/tools"
)

func TestCreateDeployment(t *testing.T) {
	// Setup fake clientset
	clientset := fake.NewSimpleClientset()

	handler := NewDeploymentHandler(clientset)

	args := map[string]interface{}{
		"name":      "test-deployment",
		"namespace": "default",
		"image":     "nginx:1.25",
		"replicas":  1,
	}

	result, err := handler.Create(context.Background(), args)

	if err != nil {
		t.Errorf("unexpected error creating deployment: %v", err)
	}

	if !result.Success {
		t.Errorf("expected deployment creation to succeed")
	}

	if result.Preview == "" {
		t.Errorf("expected YAML preview to be generated")
	}

	// Verify deployment was created
	created, err := clientset.AppsV1().Deployments("default").Get(context.Background(), "test-deployment", metav1.GetOptions{})
	if err != nil {
		t.Errorf("failed to get created deployment: %v", err)
	}

	if created.Name != "test-deployment" {
		t.Errorf("expected deployment name 'test-deployment', got '%s'", created.Name)
	}
}

func TestGetDeployment(t *testing.T) {
	// Setup fake clientset with existing deployment
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "existing-deployment",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(3),
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:   3,
				UpdatedReplicas: 3,
			},
		},
	)

	handler := NewDeploymentHandler(clientset)

	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := handler.Get(context.Background(), args)

	if err != nil {
		t.Errorf("unexpected error getting deployments: %v", err)
	}

	if !result.Success {
		t.Errorf("expected get operation to succeed")
	}

	// Check that data is returned
	data, ok := result.Data.([]map[string]interface{})
	if !ok {
		t.Errorf("expected result.Data to be []map[string]interface{}, got %T", result.Data)
	}

	if len(data) != 1 {
		t.Errorf("expected 1 deployment, got %d", len(data))
	}

	if data[0]["name"] != "existing-deployment" {
		t.Errorf("expected deployment name 'existing-deployment', got '%v'", data[0]["name"])
	}
}

func TestScaleDeployment(t *testing.T) {
	// Setup fake clientset with existing deployment
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:   1,
				UpdatedReplicas: 1,
			},
		},
	)

	handler := NewDeploymentHandler(clientset)

	args := map[string]interface{}{
		"name":      "test-deployment",
		"namespace": "default",
		"replicas":  5,
	}

	result, err := handler.Scale(context.Background(), args)

	if err != nil {
		t.Errorf("unexpected error scaling deployment: %v", err)
	}

	if !result.Success {
		t.Errorf("expected scale operation to succeed")
	}

	// Verify deployment was scaled
	updated, err := clientset.AppsV1().Deployments("default").Get(context.Background(), "test-deployment", metav1.GetOptions{})
	if err != nil {
		t.Errorf("failed to get scaled deployment: %v", err)
	}

	if *updated.Spec.Replicas != 5 {
		t.Errorf("expected 5 replicas, got %d", *updated.Spec.Replicas)
	}
}

func TestDeleteDeployment(t *testing.T) {
	// Setup fake clientset with existing deployment
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(1),
			},
		},
	)

	handler := NewDeploymentHandler(clientset)

	args := map[string]interface{}{
		"name":      "test-deployment",
		"namespace": "default",
	}

	result, err := handler.Delete(context.Background(), args)

	if err != nil {
		t.Errorf("unexpected error deleting deployment: %v", err)
	}

	if !result.Success {
		t.Errorf("expected delete operation to succeed")
	}

	// Verify deployment was deleted
	_, err = clientset.AppsV1().Deployments("default").Get(context.Background(), "test-deployment", metav1.GetOptions{})
	if err == nil {
		t.Errorf("expected deployment to be deleted, but it still exists")
	}
}

func TestCreateDeploymentMissingRequiredFields(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	handler := NewDeploymentHandler(clientset)

	// Test missing name
	args := map[string]interface{}{
		"namespace": "default",
		"image":     "nginx:1.25",
		"replicas":  1,
	}

	_, err := handler.Create(context.Background(), args)
	if err == nil {
		t.Errorf("expected error when name is missing")
	}

	// Test missing image
	args = map[string]interface{}{
		"name":      "test-deployment",
		"namespace": "default",
		"replicas":  1,
	}

	_, err = handler.Create(context.Background(), args)
	if err == nil {
		t.Errorf("expected error when image is missing")
	}
}

func TestDeploymentHandlerOperations(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	handler := NewDeploymentHandler(clientset)

	ops := handler.Operations()

	if len(ops) != 4 {
		t.Errorf("expected 4 operations, got %d", len(ops))
	}

	// Verify create operation
	createOp := findOperation(ops, "create")
	if createOp == nil {
		t.Errorf("create operation not found")
	} else {
		if createOp.DangerLevel != tools.DangerLow {
			t.Errorf("expected create danger level to be low, got %s", createOp.DangerLevel)
		}
	}

	// Verify delete operation
	deleteOp := findOperation(ops, "delete")
	if deleteOp == nil {
		t.Errorf("delete operation not found")
	} else {
		if deleteOp.DangerLevel != tools.DangerHigh {
			t.Errorf("expected delete danger level to be high, got %s", deleteOp.DangerLevel)
		}
	}
}

// Helper functions
func int32Ptr(i int32) *int32 {
	return &i
}

func findOperation(ops []Operation, name string) *Operation {
	for _, op := range ops {
		if op.Name == name {
			return &op
		}
	}
	return nil
}
