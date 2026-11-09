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
