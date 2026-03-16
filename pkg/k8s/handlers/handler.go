package handlers

// Handler provides operations for a specific K8s resource type.
type Handler interface {
	// Resource returns the resource type this handler manages.
	Resource() string

	// Operations returns a list of operations supported.
	Operations() []Operation

	// Validate checks if an operation is valid for this resource.
	Validate(op Operation, args map[string]interface{}) error
}
