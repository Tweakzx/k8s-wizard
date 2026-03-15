package handlers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s-wizard/pkg/tools"
	"sigs.k8s.io/yaml"
)

// DeploymentHandler manages Deployment operations.
type DeploymentHandler struct {
	*BaseHandler
}

// NewDeploymentHandler creates a new deployment handler.
func NewDeploymentHandler(clientset kubernetes.Interface) *DeploymentHandler {
	base := NewBaseHandler(clientset, "deployment")

	// Define deployment operations as per spec
	base.ops = []Operation{
		{
			Name:        "create",
			Method:      "create",
			DangerLevel: tools.DangerLow,
			Description: "Create a new deployment",
			Parameters: []tools.Parameter{
				{Name: "name", Type: "string", Description: "Deployment name", Required: true},
				{Name: "namespace", Type: "string", Description: "Namespace", Default: "default"},
				{Name: "image", Type: "string", Description: "Container image", Required: true},
				{Name: "replicas", Type: "number", Description: "Replica count", Default: 1},
			},
		},
		{
			Name:        "get",
			Method:      "get",
			DangerLevel: tools.DangerLow,
			Description: "List deployments",
			Parameters: []tools.Parameter{
				{Name: "namespace", Type: "string", Description: "Namespace (empty for all)"},
			},
		},
		{
			Name:        "scale",
			Method:      "scale",
			DangerLevel: tools.DangerMedium,
			Description: "Scale a deployment",
			Parameters: []tools.Parameter{
				{Name: "name", Type: "string", Description: "Deployment name", Required: true},
				{Name: "replicas", Type: "number", Description: "Target replica count", Required: true},
			},
		},
		{
			Name:        "delete",
			Method:      "delete",
			DangerLevel: tools.DangerHigh,
			Description: "Delete a deployment",
			Parameters: []tools.Parameter{
				{Name: "name", Type: "string", Description: "Deployment name", Required: true},
				{Name: "namespace", Type: "string", Description: "Namespace", Default: "default"},
			},
		},
	}

	return &DeploymentHandler{BaseHandler: base}
}

// Create creates a new deployment.
func (h *DeploymentHandler) Create(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
	// Extract and validate parameters
	ns, _ := args["namespace"].(string)
	if ns == "" {
		ns = "default"
	}

	name, _ := args["name"].(string)
	if name == "" {
		return tools.Result{}, fmt.Errorf("name is required")
	}

	image, _ := args["image"].(string)
	if image == "" {
		return tools.Result{}, fmt.Errorf("image is required")
	}

	var replicas int32 = 1
	if r, ok := args["replicas"].(float64); ok {
		replicas = int32(r)
	}

	// Create deployment object
	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"app": "k8s-wizard",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "k8s-wizard"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "k8s-wizard"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: image,
						},
					},
				},
			},
		},
	}

	// Type cast clientset to kubernetes.Interface
	clientset, ok := h.clientset.(kubernetes.Interface)
	if !ok {
		return tools.Result{}, fmt.Errorf("invalid clientset type")
	}

	// Create deployment using clientset
	_, err := clientset.AppsV1().Deployments(ns).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Generate YAML preview for create operations
	yamlData, err := yaml.Marshal(deployment)
	if err != nil {
		yamlData = []byte(fmt.Sprintf("# Failed to generate YAML: %v", err))
	}

	return tools.Result{
		Success: true,
		Message: fmt.Sprintf("Deployment %s/%s created", ns, name),
		Preview: string(yamlData),
		Data: map[string]interface{}{
			"name":      name,
			"namespace": ns,
			"image":     image,
			"replicas":  replicas,
		},
	}, nil
}

// Get lists deployments.
func (h *DeploymentHandler) Get(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
	ns, _ := args["namespace"].(string)

	listOpts := metav1.ListOptions{}

	// Type cast clientset to kubernetes.Interface
	clientset, ok := h.clientset.(kubernetes.Interface)
	if !ok {
		return tools.Result{}, fmt.Errorf("invalid clientset type")
	}

	deployments, err := clientset.AppsV1().Deployments(ns).List(ctx, listOpts)
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to list deployments: %w", err)
	}

	// Return as data slice
	resultData := make([]map[string]interface{}, 0)
	for _, d := range deployments.Items {
		replicas := int32(0)
		if d.Spec.Replicas != nil {
			replicas = *d.Spec.Replicas
		}

		resultData = append(resultData, map[string]interface{}{
			"name":            d.Name,
			"namespace":       d.Namespace,
			"replicas":        replicas,
			"readyReplicas":   d.Status.ReadyReplicas,
			"updatedReplicas": d.Status.UpdatedReplicas,
		})
	}

	return tools.Result{
		Success: true,
		Data:    resultData,
	}, nil
}

// Scale scales a deployment.
func (h *DeploymentHandler) Scale(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return tools.Result{}, fmt.Errorf("name is required")
	}

	var replicas int32
	switch v := args["replicas"].(type) {
	case float64:
		replicas = int32(v)
	case int:
		replicas = int32(v)
	case int32:
		replicas = v
	case int64:
		replicas = int32(v)
	default:
		return tools.Result{}, fmt.Errorf("replicas is required and must be a number")
	}

	// Get namespace from args or default
	ns, _ := args["namespace"].(string)
	if ns == "" {
		ns = "default"
	}

	// Type cast clientset to kubernetes.Interface
	clientset, ok := h.clientset.(kubernetes.Interface)
	if !ok {
		return tools.Result{}, fmt.Errorf("invalid clientset type")
	}

	// Get current deployment
	deployment, err := clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Scale deployment
	deployment.Spec.Replicas = &replicas
	_, err = clientset.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to scale deployment: %w", err)
	}

	return tools.Result{
		Success:  true,
		Message:  fmt.Sprintf("Deployment %s/%s scaled to %d replicas", ns, name, replicas),
		Data: map[string]interface{}{
			"name":            name,
			"namespace":       ns,
			"replicas":        replicas,
			"readyReplicas":   deployment.Status.ReadyReplicas,
			"updatedReplicas": deployment.Status.UpdatedReplicas,
		},
	}, nil
}

// Delete deletes a deployment.
func (h *DeploymentHandler) Delete(ctx context.Context, args map[string]interface{}) (tools.Result, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return tools.Result{}, fmt.Errorf("name is required")
	}

	ns, _ := args["namespace"].(string)
	if ns == "" {
		ns = "default"
	}

	// Type cast clientset to kubernetes.Interface
	clientset, ok := h.clientset.(kubernetes.Interface)
	if !ok {
		return tools.Result{}, fmt.Errorf("invalid clientset type")
	}

	// Get deployment to confirm existence
	_, err := clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Delete deployment
	err = clientset.AppsV1().Deployments(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return tools.Result{}, fmt.Errorf("failed to delete deployment: %w", err)
	}

	return tools.Result{
		Success: true,
		Message: fmt.Sprintf("Deployment %s/%s deleted", ns, name),
		Data: map[string]interface{}{
			"name":      name,
			"namespace": ns,
		},
	}, nil
}
