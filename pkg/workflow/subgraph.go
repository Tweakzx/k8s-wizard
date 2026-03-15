package workflow

import lgg "github.com/smallnest/langgraphgo/graph"

// SubGraph represents a reusable workflow fragment.
type SubGraph interface {
	Name() string
	Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error)
	Entry() string
	Exit() []string
}
