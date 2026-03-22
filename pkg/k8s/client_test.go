package k8s

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewClient(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)
	assert.NotNil(t, client)
}

func TestCreateDeployment(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	tests := []struct {
		name      string
		namespace string
		name_     string
		image     string
		replicas  int32
		wantErr   bool
	}{
		{
			name:      "create nginx deployment",
			namespace: "default",
			name_:     "nginx",
			image:     "nginx:latest",
			replicas:  3,
			wantErr:   false,
		},
		{
			name:      "create redis deployment",
			namespace: "test-ns",
			name_:     "redis",
			image:     "redis:alpine",
			replicas:  1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.CreateDeployment(ctx, tt.namespace, tt.name_, tt.image, tt.replicas)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, result, "✓ 已创建 Deployment")
			assert.Contains(t, result, tt.name_)

			// Verify deployment was created
			dep, err := fakeClient.AppsV1().Deployments(tt.namespace).Get(ctx, tt.name_, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, tt.name_, dep.Name)
			assert.Equal(t, tt.replicas, *dep.Spec.Replicas)
		})
	}
}

func TestGetResources_Pods(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	// Create test pods
	fakeClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod1", Namespace: "default"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
	}, metav1.CreateOptions{})
	fakeClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod2", Namespace: "default"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c2"}}},
	}, metav1.CreateOptions{})

	tests := []struct {
		name         string
		namespace    string
		resourceType string
		wantCount    int
		wantErr      bool
	}{
		{
			name:         "list pods in default namespace",
			namespace:    "default",
			resourceType: "pod",
			wantCount:    2,
			wantErr:      false,
		},
		{
			name:         "list pods with alias",
			namespace:    "default",
			resourceType: "pods",
			wantCount:    2,
			wantErr:      false,
		},
		{
			name:         "empty namespace lists all",
			namespace:    "",
			resourceType: "pod",
			wantCount:    2,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.GetResources(ctx, tt.namespace, tt.resourceType)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, result, "📦 集群中的 Pod")
			assert.Contains(t, result, "共 2 个")
		})
	}
}

func TestGetResources_Deployments(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	replicas := int32(2)
	fakeClient.AppsV1().Deployments("default").Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "dep1"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "dep1"}},
		},
	}, metav1.CreateOptions{})

	result, err := client.GetResources(ctx, "default", "deployment")
	require.NoError(t, err)
	assert.Contains(t, result, "🚀 集群中的 Deployment")
	assert.Contains(t, result, "dep1")
}

func TestGetResources_Services(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	fakeClient.CoreV1().Services("default").Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc1"},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{Port: 80}},
		},
	}, metav1.CreateOptions{})

	result, err := client.GetResources(ctx, "default", "service")
	require.NoError(t, err)
	assert.Contains(t, result, "🔗 集群中的 Service")
	assert.Contains(t, result, "svc1")
}

func TestGetResources_Namespaces(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	fakeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "ns1"},
	}, metav1.CreateOptions{})

	result, err := client.GetResources(ctx, "", "namespace")
	require.NoError(t, err)
	assert.Contains(t, result, "📁 集群中的 Namespace")
	assert.Contains(t, result, "ns1")
}

func TestGetResources_Nodes(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	fakeClient.CoreV1().Nodes().Create(ctx, &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node1"},
	}, metav1.CreateOptions{})

	result, err := client.GetResources(ctx, "", "node")
	require.NoError(t, err)
	assert.Contains(t, result, "🖥️ 集群中的 Node")
	assert.Contains(t, result, "node1")
}

func TestGetResources_UnsupportedType(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	_, err := client.GetResources(ctx, "default", "unsupported")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource type")
}

func TestScaleDeployment(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	// Create initial deployment
	initialReplicas := int32(1)
	fakeClient.AppsV1().Deployments("default").Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &initialReplicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
		},
	}, metav1.CreateOptions{})

	tests := []struct {
		name       string
		namespace  string
		deployName string
		replicas   int32
		wantErr    bool
	}{
		{
			name:       "scale up",
			namespace:  "default",
			deployName: "nginx",
			replicas:   5,
			wantErr:    false,
		},
		{
			name:       "scale down",
			namespace:  "default",
			deployName: "nginx",
			replicas:   1,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.ScaleDeployment(ctx, tt.namespace, tt.deployName, tt.replicas)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, result, "✓ 已将 Deployment")
			assert.Contains(t, result, "扩缩容到")
			assert.Contains(t, result, fmt.Sprintf("%d 个副本", tt.replicas))

			// Verify deployment was scaled
			dep, err := fakeClient.AppsV1().Deployments(tt.namespace).Get(ctx, tt.deployName, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, tt.replicas, *dep.Spec.Replicas)
		})
	}
}

func TestScaleDeployment_NotFound(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	_, err := client.ScaleDeployment(ctx, "default", "nonexistent", 3)
	assert.Error(t, err)
}

func TestDeleteResource(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	tests := []struct {
		name         string
		resourceType string
		setup        func()
	}{
		{
			name:         "delete deployment",
			resourceType: "deployment",
			setup: func() {
				replicas := int32(1)
				fakeClient.AppsV1().Deployments("default").Create(ctx, &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
					Spec: appsv1.DeploymentSpec{
						Replicas: &replicas,
						Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "nginx"}},
					},
				}, metav1.CreateOptions{})
			},
		},
		{
			name:         "delete pod",
			resourceType: "pod",
			setup: func() {
				fakeClient.CoreV1().Pods("default").Create(ctx, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod1"},
					Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c1"}}},
				}, metav1.CreateOptions{})
			},
		},
		{
			name:         "delete service",
			resourceType: "service",
			setup: func() {
				fakeClient.CoreV1().Services("default").Create(ctx, &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "svc1"},
					Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}},
				}, metav1.CreateOptions{})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup resource
			tt.setup()

			// Get resource name based on type
			var name string
			switch tt.resourceType {
			case "deployment":
				name = "nginx"
			case "pod":
				name = "pod1"
			case "service":
				name = "svc1"
			}

			result, err := client.DeleteResource(ctx, "default", name, tt.resourceType)

			require.NoError(t, err)
			assert.Contains(t, result, "✓ 已删除")

			// Verify resource was deleted
			switch tt.resourceType {
			case "deployment":
				_, err = fakeClient.AppsV1().Deployments("default").Get(ctx, name, metav1.GetOptions{})
			case "pod":
				_, err = fakeClient.CoreV1().Pods("default").Get(ctx, name, metav1.GetOptions{})
			case "service":
				_, err = fakeClient.CoreV1().Services("default").Get(ctx, name, metav1.GetOptions{})
			}
			assert.Error(t, err, "Resource should have been deleted")
		})
	}
}

func TestDeleteResource_UnsupportedType(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	_, err := client.DeleteResource(ctx, "default", "name", "unsupported")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported resource type")
}

// TestKubernetesClient_ImplementsInterface verifies interface compliance
func TestKubernetesClient_ImplementsInterface(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	// This is a compile-time check
	var _ Client = client
}

func TestClientInterface(t *testing.T) {
	// Tests would require a fake clientset
	t.Skip("client interface tests require k8s cluster setup")
}

func TestGetPodLogs(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	// Create a test pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-ns",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "test-container"},
			},
		},
	}

	_, err := fakeClient.CoreV1().Pods("test-ns").Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)

	logs, err := client.GetPodLogs(ctx, "test-ns", "test-pod", "", 100)
	if err != nil {
		t.Fatalf("unexpected error getting pod logs: %v", err)
	}

	// With fake client, logs will be empty string
	_ = logs
}

func TestGetPodLogs_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	tests := []struct {
		name      string
		namespace string
		pod       string
		tailLines int64
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "empty namespace",
			namespace: "",
			pod:       "test-pod",
			tailLines: 100,
			wantErr:   true,
			errMsg:    "namespace cannot be empty",
		},
		{
			name:      "empty pod name",
			namespace: "test-ns",
			pod:       "",
			tailLines: 100,
			wantErr:   true,
			errMsg:    "pod name cannot be empty",
		},
		{
			name:      "negative tailLines",
			namespace: "test-ns",
			pod:       "test-pod",
			tailLines: -1,
			wantErr:   true,
			errMsg:    "tailLines must be positive",
		},
		{
			name:      "zero tailLines",
			namespace: "test-ns",
			pod:       "test-pod",
			tailLines: 0,
			wantErr:   true,
			errMsg:    "tailLines must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetPodLogs(ctx, tt.namespace, tt.pod, "", tt.tailLines)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecPod(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	// Note: fake client doesn't support exec, so this will fail
	// The implementation should handle this gracefully
	_, err := client.ExecPod(ctx, "test-ns", "test-pod", "", []string{"echo", "test"})
	// We expect this to fail with fake client
	if err == nil {
		t.Error("expected error with fake client for exec")
	}
}

func TestExecPod_NilConfig(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	// Test that ExecPod with nil config returns proper error
	_, err := client.ExecPod(ctx, "test-ns", "test-pod", "test-container", []string{"echo", "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config is nil")
}

func TestExecPod_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	client := NewClient(fakeClient, nil)

	tests := []struct {
		name      string
		namespace string
		pod       string
		container string
		command   []string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "empty namespace",
			namespace: "",
			pod:       "test-pod",
			container: "test-container",
			command:   []string{"echo", "test"},
			wantErr:   true,
			errMsg:    "namespace cannot be empty",
		},
		{
			name:      "empty pod name",
			namespace: "test-ns",
			pod:       "",
			container: "test-container",
			command:   []string{"echo", "test"},
			wantErr:   true,
			errMsg:    "pod name cannot be empty",
		},
		{
			name:      "empty container",
			namespace: "test-ns",
			pod:       "test-pod",
			container: "",
			command:   []string{"echo", "test"},
			wantErr:   true,
			errMsg:    "container cannot be empty",
		},
		{
			name:      "empty command",
			namespace: "test-ns",
			pod:       "test-pod",
			container: "test-container",
			command:   []string{},
			wantErr:   true,
			errMsg:    "command cannot be empty",
		},
		{
			name:      "unsafe shell command",
			namespace: "test-ns",
			pod:       "test-pod",
			container: "test-container",
			command:   []string{"sh", "-c", "echo ok; rm -rf /"},
			wantErr:   true,
			errMsg:    "unsafe shell command rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ExecPod(ctx, tt.namespace, tt.pod, tt.container, tt.command)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
