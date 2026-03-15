package workflow

import (
	"context"
	"testing"

	lgg "github.com/smallnest/langgraphgo/graph"
)

// mockSubGraph implements SubGraph interface for testing
type mockSubGraph struct{}

func (m *mockSubGraph) Name() string {
	return "test_subgraph"
}

func (m *mockSubGraph) Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error) {
	g := lgg.NewStateGraph[AgentState]()
	g.AddNode("entry_node", "Test entry node", func(ctx context.Context, state AgentState) (AgentState, error) {
		return state, nil
	})
	g.AddEdge("entry_node", lgg.END)
	return g.Compile()
}

func (m *mockSubGraph) Entry() string {
	return "entry_node"
}

func (m *mockSubGraph) Exit() []string {
	return []string{lgg.END}
}

func TestSubGraphInterface(t *testing.T) {
	subgraph := &mockSubGraph{}

	if subgraph.Name() != "test_subgraph" {
		t.Errorf("expected subgraph name to be 'test_subgraph'")
	}

	if subgraph.Entry() != "entry_node" {
		t.Errorf("expected entry node to be 'entry_node'")
	}

	exits := subgraph.Exit()
	if len(exits) != 1 || exits[0] != lgg.END {
		t.Errorf("expected subgraph to have single exit to END")
	}
}
