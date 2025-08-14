package cre

type Workflow[C any] []ExecutionHandler[C, Runtime]

type TEEWorkflow[C any] []ExecutionHandler[C, TEERuntime]

// TEERetriableError is an error that can be retried on another TEE to avoid censorship concerns.
// returning an error wrapped in this struct will cause the TEE to retry the execution on another TEE up to F+1 times.
// This will automatically be returned by the TEERuntime in the event that a call can be potentially censored by the host.
type TEERetriableError struct {
	error
}
