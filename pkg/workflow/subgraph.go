package workflow

import lgg "github.com/smallnest/langgraphgo/graph"

// SubGraph represents a reusable workflow fragment.
type SubGraph interface {
	// Name returns the unique identifier for this sub-graph.
	Name() string

	// Build constructs and compiles the workflow graph with the given dependencies.
	Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error)

	// Entry returns the name of the entry node for this sub-graph.
	Entry() string

	// Exit returns the list of exit node names for this sub-graph.
	Exit() []string
}
