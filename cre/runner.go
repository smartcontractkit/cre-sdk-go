package cre

import "log/slog"

type Runner[C any] interface {
	Run(initFn func(config C, logger *slog.Logger, secretsProvider SecretsProvider) (Workflow[C], error))
}
