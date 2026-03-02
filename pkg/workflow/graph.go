package workflow

import (
	lgg "github.com/smallnest/langgraphgo/graph"
)

// NewK8sWizardGraph creates and compiles the K8s Wizard graph without checkpointing.
// For checkpointing support, use NewK8sWizardGraphWithCheckpointer.
func NewK8sWizardGraph(deps *Dependencies) (*lgg.StateRunnable[AgentState], error) {
	g := lgg.NewStateGraph[AgentState]()

	// Add nodes
	g.AddNode("parse_intent", "Parse user intent using LLM",
		MakeParseIntentNode(deps.LLM))
	g.AddNode("merge_form", "Merge form data into action",
		MakeMergeFormNode())
	g.AddNode("check_clarify", "Check if clarification is needed",
		MakeCheckClarifyNode())
	g.AddNode("generate_preview", "Generate action preview",
		MakeGeneratePreviewNode())
	g.AddNode("execute", "Execute K8s action",
		MakeExecuteNode(deps.K8sClient))

	// Set entry point
	g.SetEntryPoint("parse_intent")

	// Add conditional edges
	g.AddConditionalEdge("parse_intent", RouteAfterParse)
	g.AddConditionalEdge("check_clarify", RouteAfterClarify)
	g.AddConditionalEdge("generate_preview", RouteAfterPreview)

	// Add regular edges
	g.AddEdge("merge_form", "check_clarify")
	g.AddEdge("execute", lgg.END)

	// Compile the graph
	return g.Compile()
}

// NewK8sWizardGraphWithCheckpointer creates and compiles the K8s Wizard graph with checkpointing support.
// This enables session persistence and resume capability.
func NewK8sWizardGraphWithCheckpointer(deps *Dependencies, store lgg.CheckpointStore) (*lgg.CheckpointableRunnable[AgentState], error) {
	g := lgg.NewCheckpointableStateGraph[AgentState]()

	// Add nodes
	g.AddNode("parse_intent", "Parse user intent using LLM",
		MakeParseIntentNode(deps.LLM))
	g.AddNode("merge_form", "Merge form data into action",
		MakeMergeFormNode())
	g.AddNode("check_clarify", "Check if clarification is needed",
		MakeCheckClarifyNode())
	g.AddNode("generate_preview", "Generate action preview",
		MakeGeneratePreviewNode())
	g.AddNode("execute", "Execute K8s action",
		MakeExecuteNode(deps.K8sClient))

	// Set entry point
	g.SetEntryPoint("parse_intent")

	// Add conditional edges
	g.AddConditionalEdge("parse_intent", RouteAfterParse)
	g.AddConditionalEdge("check_clarify", RouteAfterClarify)
	g.AddConditionalEdge("generate_preview", RouteAfterPreview)

	// Add regular edges
	g.AddEdge("merge_form", "check_clarify")
	g.AddEdge("execute", lgg.END)

	// Compile with checkpointing
	config := lgg.DefaultCheckpointConfig()
	config.Store = store
	g.SetCheckpointConfig(config)
	return g.CompileCheckpointable()
}
