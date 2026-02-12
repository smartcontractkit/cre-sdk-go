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
	runner.Run(initFn)
}

func initFn([]byte, *slog.Logger, cre.SecretsProvider) (cre.Workflow[[]byte], error) {
	return cre.Workflow[[]byte]{
		cre.Handler(
			basictrigger.Trigger(&basictrigger.Config{}),
			timeInterpretationHandler,
		),
	}, nil
}

func timeInterpretationHandler(config []byte, rt cre.Runtime, _ *basictrigger.Outputs) (string, error) {
	// Get the host time
	t := rt.Now()
	
	// Format as ISO 8601 UTC with no fractional seconds
	// RFC3339 format in UTC is exactly what we need: "2006-01-02T15:04:05Z07:00"
	// For UTC timezone, this becomes: "2006-01-02T15:04:05Z"
	isoString := t.UTC().Format("2006-01-02T15:04:05Z07:00")
	
	return isoString, nil
}
