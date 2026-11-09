package main

import (
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
)

func main() {
	runner := wasm.NewRunner(func(configBytes []byte) ([]byte, error) {
		return configBytes, nil
	})
	runner.Run(initWorkflow)
}

func initWorkflow([]byte, *slog.Logger, cre.SecretsProvider) (cre.Workflow[[]byte], error) {
	return cre.Workflow[[]byte]{
		cre.Handler(
			basictrigger.Trigger(&basictrigger.Config{}),
			readExecutionMetadata,
		),
	}, nil
}

func readExecutionMetadata(_ []byte, runtime cre.Runtime, _ *basictrigger.Outputs) (cre.WorkflowExecutionMetadata, error) {
	return cre.ExecutionMetadata(runtime), nil
}
