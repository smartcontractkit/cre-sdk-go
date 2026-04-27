package main

import (
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
)

func subscribe(_ []byte, _ *slog.Logger, _ cre.SecretsProvider) (cre.Workflow[[]byte], error) {
	teeRequiements := []cre.TeeAndRegions{{Type: cre.TeeType_TEE_TYPE_AWS_NITRO, Regions: []string{"us-west-2"}}}
	return cre.Workflow[[]byte]{
		cre.HandlerInTee(
			basictrigger.Trigger(&basictrigger.Config{Name: "first-trigger", Number: 100}),
			trigger,
			teeRequiements),
	}, nil
}

func trigger(config []byte, runtime cre.TeeRuntime, payload *basictrigger.Outputs) (int32, error) {
	return 0, nil
}

func main() {
	runner := wasm.NewRunner(func(configBytes []byte) ([]byte, error) { return configBytes, nil })
	runner.Run(subscribe)
}
