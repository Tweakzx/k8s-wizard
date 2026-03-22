// Package k8s provides Kubernetes client operations for K8s Wizard.
package k8s

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// Client defines the interface for Kubernetes operations.
type Client interface {
	// CreateDeployment creates a new deployment in the specified namespace.
	CreateDeployment(ctx context.Context, namespace, name, image string, replicas int32) (string, error)

	// GetResources lists resources of the specified type in the namespace.
	// If namespace is empty, lists resources across all namespaces.
	GetResources(ctx context.Context, namespace, resourceType string) (string, error)

	// ScaleDeployment scales a deployment to the specified number of replicas.
	ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) (string, error)

	// DeleteResource deletes a resource of the specified type and name.
	DeleteResource(ctx context.Context, namespace, name, resourceType string) (string, error)

	// GetPodLogs fetches pod logs with optional container and tail lines.
	GetPodLogs(ctx context.Context, namespace, pod, container string, tailLines int64) (string, error)

	// ExecPod executes a command in a pod's container.
	ExecPod(ctx context.Context, namespace, pod, container string, command []string) (string, error)
}

// KubernetesClient implements Client using kubernetes.Interface.
type KubernetesClient struct {
	clientset kubernetes.Interface
	config    *rest.Config
}

// NewClient creates a new KubernetesClient from a kubernetes.Interface.
// The config parameter can be nil for read-only operations.
func NewClient(clientset kubernetes.Interface, config *rest.Config) *KubernetesClient {
	return &KubernetesClient{
		clientset: clientset,
		config:    config,
	}
}

// CreateDeployment creates a new deployment.
func (c *KubernetesClient) CreateDeployment(ctx context.Context, namespace, name, image string, replicas int32) (string, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
							Ports: []corev1.ContainerPort{{ContainerPort: 80}},
						},
					},
				},
			},
		},
	}

	created, err := c.clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("create deployment failed: %w", err)
	}

	return fmt.Sprintf("✓ 已创建 Deployment %s/%s，副本数: %d，镜像: %s", namespace, created.Name, *created.Spec.Replicas, image), nil
}

// GetResources lists resources of the specified type.
func (c *KubernetesClient) GetResources(ctx context.Context, namespace, resourceType string) (string, error) {
	allNamespaces := namespace == ""

	switch resourceType {
	case "pod", "pods":
		return c.getPods(ctx, namespace, allNamespaces)
	case "deployment", "deployments", "deploy":
		return c.getDeployments(ctx, namespace, allNamespaces)
	case "service", "services", "svc":
		return c.getServices(ctx, namespace, allNamespaces)
	case "namespace", "namespaces", "ns":
		return c.getNamespaces(ctx)
	case "node", "nodes":
		return c.getNodes(ctx)
	default:
		return "", fmt.Errorf("unsupported resource type: %s. Supported: pod, deployment, service, namespace, node", resourceType)
	}
}

func (c *KubernetesClient) getPods(ctx context.Context, namespace string, allNamespaces bool) (string, error) {
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("get pod list failed: %w", err)
	}

	if len(pods.Items) == 0 {
		if allNamespaces {
			return "集群中没有 Pod", nil
		}
		return fmt.Sprintf("命名空间 %s 中没有 Pod", namespace), nil
	}

	result := fmt.Sprintf("📦 集群中的 Pod (共 %d 个):\n", len(pods.Items))
	for _, pod := range pods.Items {
		age := pod.CreationTimestamp.Format("2006-01-02 15:04")
		if allNamespaces {
			result += fmt.Sprintf("  • [%s] %s (%s) - %s\n", pod.Namespace, pod.Name, pod.Status.Phase, age)
		} else {
			result += fmt.Sprintf("  • %s (%s) - %s\n", pod.Name, pod.Status.Phase, age)
		}
	}
	return result, nil
}

func (c *KubernetesClient) getDeployments(ctx context.Context, namespace string, allNamespaces bool) (string, error) {
	deps, err := c.clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("get deployment list failed: %w", err)
	}

	if len(deps.Items) == 0 {
		if allNamespaces {
			return "集群中没有 Deployment", nil
		}
		return fmt.Sprintf("命名空间 %s 中没有 Deployment", namespace), nil
	}

	result := fmt.Sprintf("🚀 集群中的 Deployment (共 %d 个):\n", len(deps.Items))
	for _, dep := range deps.Items {
		if allNamespaces {
			result += fmt.Sprintf("  • [%s] %s (副本: %d/%d)\n", dep.Namespace, dep.Name, dep.Status.ReadyReplicas, dep.Status.Replicas)
		} else {
			result += fmt.Sprintf("  • %s (副本: %d/%d)\n", dep.Name, dep.Status.ReadyReplicas, dep.Status.Replicas)
		}
	}
	return result, nil
}

func (c *KubernetesClient) getServices(ctx context.Context, namespace string, allNamespaces bool) (string, error) {
	svcs, err := c.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("get service list failed: %w", err)
	}

	if len(svcs.Items) == 0 {
		if allNamespaces {
			return "集群中没有 Service", nil
		}
		return fmt.Sprintf("命名空间 %s 中没有 Service", namespace), nil
	}

	result := fmt.Sprintf("🔗 集群中的 Service (共 %d 个):\n", len(svcs.Items))
	for _, svc := range svcs.Items {
		port := "-"
		if len(svc.Spec.Ports) > 0 {
			port = fmt.Sprintf("%d", svc.Spec.Ports[0].Port)
		}
		if allNamespaces {
			result += fmt.Sprintf("  • [%s] %s (类型: %s, 端口: %s)\n", svc.Namespace, svc.Name, svc.Spec.Type, port)
		} else {
			result += fmt.Sprintf("  • %s (类型: %s, 端口: %s)\n", svc.Name, svc.Spec.Type, port)
		}
	}
	return result, nil
}

func (c *KubernetesClient) getNamespaces(ctx context.Context) (string, error) {
	nss, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("get namespace list failed: %w", err)
	}

	result := fmt.Sprintf("📁 集群中的 Namespace (共 %d 个):\n", len(nss.Items))
	for _, ns := range nss.Items {
		result += fmt.Sprintf("  • %s (状态: %s)\n", ns.Name, ns.Status.Phase)
	}
	return result, nil
}

func (c *KubernetesClient) getNodes(ctx context.Context) (string, error) {
	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("get node list failed: %w", err)
	}

	result := fmt.Sprintf("🖥️ 集群中的 Node (共 %d 个):\n", len(nodes.Items))
	for _, node := range nodes.Items {
		ready := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				ready = "Ready"
				break
			}
		}
		result += fmt.Sprintf("  • %s (%s)\n", node.Name, ready)
	}
	return result, nil
}

// ScaleDeployment scales a deployment to the specified number of replicas.
func (c *KubernetesClient) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) (string, error) {
	dep, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("get deployment failed: %w", err)
	}

	dep.Spec.Replicas = &replicas
	_, err = c.clientset.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	if err != nil {
		return "", fmt.Errorf("scale deployment failed: %w", err)
	}

	return fmt.Sprintf("✓ 已将 Deployment %s/%s 扩缩容到 %d 个副本", namespace, name, replicas), nil
}

// DeleteResource deletes a resource of the specified type and name.
func (c *KubernetesClient) DeleteResource(ctx context.Context, namespace, name, resourceType string) (string, error) {
	switch resourceType {
	case "deployment", "deploy":
		err := c.clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			return "", fmt.Errorf("delete deployment failed: %w", err)
		}
		return fmt.Sprintf("✓ 已删除 Deployment %s/%s", namespace, name), nil

	case "pod", "pods":
		err := c.clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			return "", fmt.Errorf("delete pod failed: %w", err)
		}
		return fmt.Sprintf("✓ 已删除 Pod %s/%s", namespace, name), nil

	case "service", "services", "svc":
		err := c.clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			return "", fmt.Errorf("delete service failed: %w", err)
		}
		return fmt.Sprintf("✓ 已删除 Service %s/%s", namespace, name), nil

	default:
		return "", fmt.Errorf("unsupported resource type for deletion: %s", resourceType)
	}
}

// GetPodLogs fetches pod logs.
func (c *KubernetesClient) GetPodLogs(ctx context.Context, namespace string, pod string, container string, tailLines int64) (string, error) {
	// Validate parameters
	if namespace == "" {
		return "", fmt.Errorf("namespace cannot be empty")
	}
	if pod == "" {
		return "", fmt.Errorf("pod name cannot be empty")
	}
	if tailLines <= 0 {
		return "", fmt.Errorf("tailLines must be positive, got: %d", tailLines)
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
		Container: container,
		TailLines: &tailLines,
	})

	logs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer logs.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(logs); err != nil {
		return "", fmt.Errorf("read logs failed: %w", err)
	}

	return buf.String(), nil
}

var dangerousShellPattern = regexp.MustCompile(`[;&|` + "`" + `$<>]|\$\(|\r|\n`)

// ExecPod executes a command in a pod.
func (c *KubernetesClient) ExecPod(ctx context.Context, namespace string, pod string, container string, command []string) (string, error) {
	// Validate parameters
	if namespace == "" {
		return "", fmt.Errorf("namespace cannot be empty")
	}
	if pod == "" {
		return "", fmt.Errorf("pod name cannot be empty")
	}
	if container == "" {
		return "", fmt.Errorf("container cannot be empty")
	}
	if len(command) == 0 {
		return "", fmt.Errorf("command cannot be empty")
	}
	for i, arg := range command {
		if strings.TrimSpace(arg) == "" {
			return "", fmt.Errorf("command argument at index %d cannot be empty", i)
		}
		if strings.ContainsRune(arg, '\x00') {
			return "", fmt.Errorf("command argument at index %d contains null byte", i)
		}
	}
	if isShellCommand(command) && dangerousShellPattern.MatchString(command[2]) {
		return "", fmt.Errorf("unsafe shell command rejected")
	}

	// Check config is available for SPDY executor
	if c.config == nil {
		return "", fmt.Errorf("client config is nil, cannot execute pod commands")
	}

	// Build exec request
	req := c.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespace).
		Name(pod).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	// Create executor
	executor, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create executor: %w", err)
	}

	// Capture output
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)

	// Execute command
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})
	if err != nil {
		return "", fmt.Errorf("exec command failed: %w: %s", err, errBuf.String())
	}

	return buf.String(), nil
}

func isShellCommand(command []string) bool {
	if len(command) < 3 {
		return false
	}

	switch command[0] {
	case "sh", "bash", "zsh", "dash", "ksh", "ash":
		return command[1] == "-c"
	default:
		return false
	}
}
