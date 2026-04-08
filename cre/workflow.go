package cre

// Workflow is a sequence of ExecutionHandlers that define the logic of a CRE application.
type Workflow[C any] []ExecutionHandler[C, Runtime]

// TeeWorkflow is a sequence of ExecutionHandlers for TEE (Trusted Execution Environment) mode.
type TeeWorkflow[C any] []ExecutionHandler[C, TeeRuntime]
