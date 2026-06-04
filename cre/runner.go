package cre

import "log/slog"

type InitFn[C any] = func(config C, logger *slog.Logger, secretsProvider SecretsProvider) (Workflow[C], error)

// Runner is the entry point to running a CRE workflow.
type Runner[C any] interface {
	// Run creates the workflow and starts it.
	// Upon registration of a workflow, a run is used to register to `Trigger`s.
	// Upon receiving a trigger, the appropriate handler's callback is invoked.
	Run(initFn InitFn[C])
}
