package tools

import (
	"context"
	"testing"
)

func BenchmarkToolRegistryRegister(b *testing.B) {
	registry := NewRegistry()
	tool := &mockTool{name: "benchmark_tool"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Register(tool)
	}
}

func BenchmarkToolRegistryExecute(b *testing.B) {
	registry := NewRegistry()
	tool := &mockTool{name: "benchmark_tool"}
	registry.Register(tool)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Execute(context.Background(), "benchmark_tool", nil)
	}
}

func BenchmarkToolRegistryGet(b *testing.B) {
	registry := NewRegistry()
	tool := &mockTool{name: "benchmark_tool"}
	registry.Register(tool)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Get("benchmark_tool")
	}
}
