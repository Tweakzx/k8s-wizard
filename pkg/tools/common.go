package tools

// DangerLevel represents the risk level of an operation.
type DangerLevel int

const (
	DangerLow DangerLevel = iota
	DangerMedium
	DangerHigh
)

// Parameter describes a tool input parameter.
type Parameter struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     interface{}
}

// Result represents the output of a tool execution.
type Result struct {
	Success bool
	Message string
	Data    map[string]interface{}
	Preview string
}
