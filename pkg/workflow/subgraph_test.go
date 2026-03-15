package workflow

import (
	"context"
	"fmt"
	"testing"

	lgg "github.com/smallnest/langgraphgo/graph"
)

// mockSubGraph implements SubGraph interface for testing
type mockSubGraph struct{}

func (m *mockSubGraph) Name() string {
	return "test_subgraph"
}

func (m *mockSubGraph) Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error) {
	if deps == nil {
		return nil, fmt.Errorf("dependencies cannot be nil")
	}
	g := lgg.NewStateGraph[AgentState]()
	g.AddNode("entry_node", "Test entry node", func(ctx context.Context, state AgentState) (AgentState, error) {
		return state, nil
	})
	g.AddEdge("entry_node", lgg.END)
	g.SetEntryPoint("entry_node")
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

func TestSubGraphBuild(t *testing.T) {
	subgraph := &mockSubGraph{}
	deps := &Dependencies{}

	runnable, err := subgraph.Build(deps)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if runnable == nil {
		t.Fatal("Build returned nil runnable")
	}
}

func TestSubGraphBuildNilDependencies(t *testing.T) {
	subgraph := &mockSubGraph{}

	runnable, err := subgraph.Build(nil)
	if err == nil {
		t.Fatal("Build with nil dependencies should return error")
	}
	if err.Error() != "dependencies cannot be nil" {
		t.Errorf("expected error message 'dependencies cannot be nil', got '%s'", err.Error())
	}
	if runnable != nil {
		t.Fatal("Build with nil dependencies should return nil runnable")
	}
}

type multiExitSubGraph struct{}

func (m *multiExitSubGraph) Name() string {
	return "multi_exit_subgraph"
}

func (m *multiExitSubGraph) Build(deps *Dependencies) (*lgg.StateRunnable[AgentState], error) {
	if deps == nil {
		return nil, fmt.Errorf("dependencies cannot be nil")
	}
	g := lgg.NewStateGraph[AgentState]()
	g.AddNode("entry_node", "Test entry node", func(ctx context.Context, state AgentState) (AgentState, error) {
		return state, nil
	})
	g.AddNode("success_node", "Success path", func(ctx context.Context, state AgentState) (AgentState, error) {
		return state, nil
	})
	g.AddNode("failure_node", "Failure path", func(ctx context.Context, state AgentState) (AgentState, error) {
		return state, nil
	})
	g.AddEdge("entry_node", lgg.END)
	g.AddEdge("success_node", lgg.END)
	g.AddEdge("failure_node", lgg.END)
	g.SetEntryPoint("entry_node")
	return g.Compile()
}

func (m *multiExitSubGraph) Entry() string {
	return "entry_node"
}

func (m *multiExitSubGraph) Exit() []string {
	return []string{"success_node", "failure_node", lgg.END}
}

func TestSubGraphMultipleExits(t *testing.T) {
	subgraph := &multiExitSubGraph{}
	exits := subgraph.Exit()
	if len(exits) != 3 {
		t.Errorf("expected 3 exits, got %d", len(exits))
	}

	expectedExits := []string{"success_node", "failure_node", lgg.END}
	for i, exit := range exits {
		if exit != expectedExits[i] {
			t.Errorf("expected exit %d to be '%s', got '%s'", i, expectedExits[i], exit)
		}
	}
}
