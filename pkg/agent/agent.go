package agent

import (
	"context"
	"fmt"
	"os"
	"sync"

	"k8s-wizard/api/models"
	"k8s-wizard/pkg/config"
	"k8s-wizard/pkg/k8s"
	"k8s-wizard/pkg/llm"
	"k8s-wizard/pkg/tools"
	"k8s-wizard/pkg/workflow"

	lgg "github.com/smallnest/langgraphgo/graph"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ============================================================================
// Interface Definition
// ============================================================================

// AgentInterface defines the interface for K8s Wizard agents.
// GraphAgent and GraphAgentWithCheckpointer implement this interface.
type AgentInterface interface {
	// ProcessCommandWithClarification processes a command with the clarification flow.
	ProcessCommandWithClarification(ctx context.Context, userMsg string, formData map[string]interface{}, confirm *bool) (result string, clarification *models.ClarificationRequest, actionPreview *models.ActionPreview, err error)

	// ProcessCommand processes a simple command without clarification flow.
	ProcessCommand(ctx context.Context, userMsg string) (string, error)

	// GetModelName returns the current model name.
	GetModelName() string

	// SetModel switches to a different model.
	SetModel(modelString string) error
}

// ============================================================================
// GraphAgent (without checkpointing)
// ============================================================================

// GraphAgent wraps the langgraphgo graph and provides a compatible interface
// with the existing Agent API.
type GraphAgent struct {
	graph     *lgg.StateRunnable[workflow.AgentState]
	deps      *workflow.Dependencies
	mu        sync.RWMutex
	modelName string
}

// NewGraphAgent creates a new GraphAgent.
func NewGraphAgent(k8sClient k8s.Client, llmClient llm.Client, modelName string) (*GraphAgent, error) {
	// Create suggestion engine
	suggestionEngine := workflow.NewSuggestionEngine(k8sClient)

	deps := &workflow.Dependencies{
		K8sClient:    k8sClient,
		LLM:          llmClient,
		ModelName:    modelName,
		ToolRegistry: tools.NewRegistry(),
		// PromptLoader, SubGraphMgr, ContextMgr will be initialized in later phases
		PromptLoader: nil,
		SubGraphMgr:  nil,
		ContextMgr:   nil,
		SuggestionEngine: suggestionEngine,
	}

	compiledGraph, err := workflow.NewK8sWizardGraph(deps)
	if err != nil {
		return nil, err
	}

	return &GraphAgent{
		graph:     compiledGraph,
		deps:      deps,
		modelName: modelName,
	}, nil
}

// ProcessCommandWithClarification processes a command with the clarification flow.
// This maintains API compatibility with the existing Agent interface.
func (a *GraphAgent) ProcessCommandWithClarification(
	ctx context.Context,
	userMsg string,
	formData map[string]interface{},
	confirm *bool,
) (result string, clarification *models.ClarificationRequest, actionPreview *models.ActionPreview, err error) {
	a.mu.RLock()
	modelName := a.modelName
	a.mu.RUnlock()

	// Build initial state
	initialState := workflow.AgentState{
		UserMessage: userMsg,
		FormData:    formData,
		Confirm:     confirm,
		Status:      workflow.StatusPending,
	}
	_ = modelName // modelName is currently unused but kept for future use

	// Execute the graph
	finalState, execErr := a.graph.Invoke(ctx, initialState)
	if execErr != nil {
		return "", nil, nil, execErr
	}

	// Handle final state - check fields in addition to status since routing
	// function state modifications don't persist in langgraphgo

	// Check for clarification needs first
	if finalState.NeedsClarification && finalState.ClarificationRequest != nil {
		return "", finalState.ClarificationRequest, nil, nil
	}

	// Check for confirmation needs
	if finalState.ActionPreview != nil && (finalState.Confirm == nil || !*finalState.Confirm) {
		return "请确认以下操作：", nil, finalState.ActionPreview, nil
	}

	switch finalState.Status {
	case workflow.StatusChat:
		// Non-K8s chat response
		return finalState.Reply, nil, nil, nil

	case workflow.StatusNeedsInfo:
		// Need clarification
		return "", finalState.ClarificationRequest, nil, nil

	case workflow.StatusNeedsConfirm:
		// Need confirmation
		return "请确认以下操作：", nil, finalState.ActionPreview, nil

	case workflow.StatusExecuted:
		// Successfully executed
		return finalState.Result, nil, nil, nil

	case workflow.StatusError:
		// Error occurred
		if finalState.Error != nil {
			return "", nil, nil, finalState.Error
		}
		return finalState.Result, nil, nil, nil

	default:
		// Check for any error or result
		if finalState.Error != nil {
			return "", nil, nil, finalState.Error
		}
		return finalState.Result, nil, nil, nil
	}
}

// SetModel updates the model name.
// Note: This is a simplified implementation that only updates the model name string.
// For full model switching functionality, the graph and LLM client need to be recreated.
func (a *GraphAgent) SetModel(modelName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.modelName = modelName
	return nil
}

// GetModelName returns the current model name.
// This is an alias for GetModel() for API compatibility.
func (a *GraphAgent) GetModelName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.modelName
}

// GetModel returns the current model name.
func (a *GraphAgent) GetModel() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.modelName
}

// ProcessCommand processes a simple command without clarification flow.
// This is for API compatibility with the existing Agent interface.
func (a *GraphAgent) ProcessCommand(ctx context.Context, userMsg string) (string, error) {
	result, _, _, err := a.ProcessCommandWithClarification(ctx, userMsg, nil, nil)
	return result, err
}

// GetGraph returns the underlying compiled graph for advanced operations.
func (a *GraphAgent) GetGraph() *lgg.StateRunnable[workflow.AgentState] {
	return a.graph
}

// Ensure GraphAgent implements AgentInterface
var _ AgentInterface = (*GraphAgent)(nil)

// ============================================================================
// GraphAgentWithCheckpointer (with session persistence)
// ============================================================================

// GraphAgentWithCheckpointer wraps the langgraphgo graph with checkpointing support.
// It provides session persistence for multi-turn conversations.
type GraphAgentWithCheckpointer struct {
	graph        *lgg.CheckpointableRunnable[workflow.AgentState]
	deps         *workflow.Dependencies
	checkpointer *workflow.CheckpointerManager
	mu           sync.RWMutex
	modelName    string
}

// NewGraphAgentWithCheckpointer creates a new GraphAgent with session persistence.
func NewGraphAgentWithCheckpointer(k8sClient k8s.Client, llmClient llm.Client, modelName string, checkpointer *workflow.CheckpointerManager) (*GraphAgentWithCheckpointer, error) {
	// Create suggestion engine
	suggestionEngine := workflow.NewSuggestionEngine(k8sClient)

	deps := &workflow.Dependencies{
		K8sClient:    k8sClient,
		LLM:          llmClient,
		ModelName:    modelName,
		ToolRegistry: tools.NewRegistry(),
		// PromptLoader, SubGraphMgr, ContextMgr will be initialized in later phases
		PromptLoader: nil,
		SubGraphMgr:  nil,
		ContextMgr:   nil,
		SuggestionEngine: suggestionEngine,
	}

	compiledGraph, err := workflow.NewK8sWizardGraphWithCheckpointer(deps, checkpointer.GetStore())
	if err != nil {
		return nil, err
	}

	return &GraphAgentWithCheckpointer{
		graph:        compiledGraph,
		deps:         deps,
		checkpointer: checkpointer,
		modelName:    modelName,
	}, nil
}

// ProcessCommandWithClarification processes a command with the clarification flow.
// The threadID parameter is used for session persistence.
func (a *GraphAgentWithCheckpointer) ProcessCommandWithClarification(
	ctx context.Context,
	userMsg string,
	formData map[string]interface{},
	confirm *bool,
) (result string, clarification *models.ClarificationRequest, actionPreview *models.ActionPreview, err error) {
	return a.ProcessCommandWithClarificationAndThread(ctx, userMsg, formData, confirm, "")
}

// ProcessCommandWithClarificationAndThread processes a command with the clarification flow and thread ID.
// The threadID is used for session persistence - same threadID will resume from previous state.
func (a *GraphAgentWithCheckpointer) ProcessCommandWithClarificationAndThread(
	ctx context.Context,
	userMsg string,
	formData map[string]interface{},
	confirm *bool,
	threadID string,
) (result string, clarification *models.ClarificationRequest, actionPreview *models.ActionPreview, err error) {
	a.mu.RLock()
	modelName := a.modelName
	a.mu.RUnlock()

	// Build initial state
	initialState := workflow.AgentState{
		UserMessage: userMsg,
		FormData:    formData,
		Confirm:     confirm,
		ThreadID:    threadID,
		Status:      workflow.StatusPending,
	}
	_ = modelName // modelName is currently unused but kept for future use

	var finalState workflow.AgentState
	var execErr error

	if threadID != "" {
		// Use thread-aware configuration
		config := lgg.WithThreadID(threadID)
		finalState, execErr = a.graph.InvokeWithConfig(ctx, initialState, config)
	} else {
		finalState, execErr = a.graph.Invoke(ctx, initialState)
	}

	if execErr != nil {
		return "", nil, nil, execErr
	}

	// Handle final state - check fields in addition to status since routing
	// function state modifications don't persist in langgraphgo

	// Check for clarification needs first
	if finalState.NeedsClarification && finalState.ClarificationRequest != nil {
		return "", finalState.ClarificationRequest, nil, nil
	}

	// Check for confirmation needs
	if finalState.ActionPreview != nil && (finalState.Confirm == nil || !*finalState.Confirm) {
		return "请确认以下操作：", nil, finalState.ActionPreview, nil
	}

	switch finalState.Status {
	case workflow.StatusChat:
		return finalState.Reply, nil, nil, nil

	case workflow.StatusNeedsInfo:
		return "", finalState.ClarificationRequest, nil, nil

	case workflow.StatusNeedsConfirm:
		return "请确认以下操作：", nil, finalState.ActionPreview, nil

	case workflow.StatusExecuted:
		return finalState.Result, nil, nil, nil

	case workflow.StatusError:
		if finalState.Error != nil {
			return "", nil, nil, finalState.Error
		}
		return finalState.Result, nil, nil, nil

	default:
		if finalState.Error != nil {
			return "", nil, nil, finalState.Error
		}
		return finalState.Result, nil, nil, nil
	}
}

// SetModel updates the model name.
func (a *GraphAgentWithCheckpointer) SetModel(modelName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.modelName = modelName
	return nil
}

// GetModelName returns the current model name.
func (a *GraphAgentWithCheckpointer) GetModelName() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.modelName
}

// GetModel returns the current model name.
func (a *GraphAgentWithCheckpointer) GetModel() string {
	return a.GetModelName()
}

// ProcessCommand processes a simple command without clarification flow.
func (a *GraphAgentWithCheckpointer) ProcessCommand(ctx context.Context, userMsg string) (string, error) {
	result, _, _, err := a.ProcessCommandWithClarification(ctx, userMsg, nil, nil)
	return result, err
}

// ClearSession clears the session state for a given thread ID.
func (a *GraphAgentWithCheckpointer) ClearSession(ctx context.Context, threadID string) error {
	return a.checkpointer.ClearSession(ctx, threadID)
}

// GetGraph returns the underlying compiled graph for advanced operations.
func (a *GraphAgentWithCheckpointer) GetGraph() *lgg.CheckpointableRunnable[workflow.AgentState] {
	return a.graph
}

// Close closes the checkpointer store.
func (a *GraphAgentWithCheckpointer) Close() error {
	if a.checkpointer != nil {
		return a.checkpointer.Close()
	}
	return nil
}

// Ensure GraphAgentWithCheckpointer implements AgentInterface
var _ AgentInterface = (*GraphAgentWithCheckpointer)(nil)

// ============================================================================
// Factory Functions
// ============================================================================

// NewGraphAgentFromConfig creates a new GraphAgent with all dependencies initialized from config.
func NewGraphAgentFromConfig() (*GraphAgent, error) {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize K8s client
	k8sConfig, err := getK8sConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Create LLM client
	llmClient, modelName, err := createLLMClient(cfg)
	if err != nil {
		return nil, err
	}

	return NewGraphAgent(k8s.NewClient(clientset, k8sConfig), llmClient, modelName)
}

// NewGraphAgentWithCheckpointerFromConfig creates a new GraphAgent with session persistence.
// The dataDir parameter specifies where to store checkpoints (empty string uses default).
func NewGraphAgentWithCheckpointerFromConfig(dataDir string) (*GraphAgentWithCheckpointer, error) {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize K8s client
	k8sConfig, err := getK8sConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	// Create LLM client
	llmClient, modelName, err := createLLMClient(cfg)
	if err != nil {
		return nil, err
	}

	// Create checkpointer
	checkpointer, err := workflow.NewCheckpointerManager(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkpointer: %w", err)
	}

	return NewGraphAgentWithCheckpointer(k8s.NewClient(clientset, k8sConfig), llmClient, modelName, checkpointer)
}

// getK8sConfig initializes and returns the Kubernetes configuration.
func getK8sConfig() (*rest.Config, error) {
	var k8sConfig *rest.Config
	var err error

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		k8sConfig, err = rest.InClusterConfig()
	}
	if err != nil {
		// If both fail, try default location
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", os.Getenv("HOME")+"/.kube/config")
		if err != nil {
			return nil, fmt.Errorf("failed to create k8s config: %w", err)
		}
	}
	return k8sConfig, nil
}

// createLLMClient creates an LLM client from the configuration.
func createLLMClient(cfg *config.Config) (llm.Client, string, error) {
	modelString := cfg.Agents.Defaults.Model.Primary
	if envModel := os.Getenv("K8S_WIZARD_MODEL"); envModel != "" {
		modelString = envModel
	}

	provider, modelID, err := cfg.GetModelProvider(modelString)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse model config: %w", err)
	}

	providerConfig, ok := cfg.Models.Providers[provider]
	if !ok {
		return nil, "", fmt.Errorf("provider not configured: %s", provider)
	}

	llmClient, err := llm.NewClient(provider, modelID, providerConfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create LLM client: %w", err)
	}

	modelName := llmClient.GetModel()
	fmt.Printf("🤖 Using model: %s\n", modelName)

	return llmClient, modelName, nil
}
