package tools

// DangerLevel represents the risk level of an operation.
type DangerLevel string

const (
	DangerLow    DangerLevel = "low"
	DangerMedium DangerLevel = "medium"
	DangerHigh   DangerLevel = "high"
)

// Parameter describes a tool input parameter.
type Parameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default"`
}

// Result represents the output of a tool execution.
type Result struct {
	Success     bool
	Message     string
	Data        interface{}
	Preview     string
	NeedsConfirm bool
}
