package cre

import "log/slog"

type Runner[C any] interface {
	Run(initFn func(config C, logger *slog.Logger, secretsProvider SecretsProvider) (Workflow[C], error))
}

type TEERunner[C any] interface {
	Run(initFn func(config C, logger *slog.Logger, secretsProvider SecretsProvider) (TEEWorkflow[C], error))
}
