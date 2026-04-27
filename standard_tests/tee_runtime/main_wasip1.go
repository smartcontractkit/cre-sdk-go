package main

import (
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
)

var teeRequirements = []cre.TeeAndRegions{{Type: cre.TeeType_TEE_TYPE_AWS_NITRO, Regions: []string{"us-west-2"}}}

func subscribe(_ []byte, _ *slog.Logger, _ cre.SecretsProvider) (cre.Workflow[[]byte], error) {
	return cre.Workflow[[]byte]{
		cre.HandlerInTee(
			basictrigger.Trigger(&basictrigger.Config{Name: "first-trigger", Number: 100}),
			teeTrigger,
			teeRequirements,
		),
		cre.Handler(
			basictrigger.Trigger(&basictrigger.Config{Name: "second-trigger", Number: 200}),
			regularTrigger,
		),
	}, nil
}

func teeTrigger(_ []byte, _ cre.TeeRuntime, _ *basictrigger.Outputs) (int32, error) {
	return 0, nil
}

func regularTrigger(_ []byte, _ cre.Runtime, _ *basictrigger.Outputs) (int32, error) {
	return 0, nil
}

func main() {
	runner := wasm.NewRunner(func(configBytes []byte) ([]byte, error) { return configBytes, nil })
	runner.Run(subscribe)
}
