package cre

// WorkflowExecutionMetadata provides metadata about a workflow execution.
// Values may change between invocations.
type WorkflowExecutionMetadata struct {
	WorkflowID    string
	ExecutionID   string // empty for subscribe requests
	WorkflowOwner string
	DonID         uint32
	DonF          uint32
}

// ExecutionMetadataProvider is optionally implemented by runtimes that expose
// workflow execution metadata. Implementations return a value copy without
// making host or capability calls.
type ExecutionMetadataProvider interface {
	ExecutionMetadata() WorkflowExecutionMetadata
}

// ExecutionMetadata returns a copy of the runtime's execution metadata.
// It returns zero values if runtime is nil.
func ExecutionMetadata(runtime RuntimeBase) WorkflowExecutionMetadata {
	provider, ok := runtime.(ExecutionMetadataProvider)
	if !ok {
		return WorkflowExecutionMetadata{}
	}
	return provider.ExecutionMetadata()
}
