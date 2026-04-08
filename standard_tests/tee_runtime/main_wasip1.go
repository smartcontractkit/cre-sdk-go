package main

import (
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
)

func subscribe(_ []byte, _ *slog.Logger, _ cre.SecretsProvider) (cre.TeeWorkflow[[]byte], error) {
	return cre.TeeWorkflow[[]byte]{
		cre.HandlerInTee(basictrigger.Trigger(&basictrigger.Config{Name: "first-trigger", Number: 100}), trigger),
	}, nil
}

func trigger(config []byte, runtime cre.TeeRuntime, payload *basictrigger.Outputs) (int32, error) {
	return 0, nil
}

func main() {

	runner := wasm.NewTeeRunner(
		[]cre.TeeType{cre.TeeType_TEE_TYPE_AWS_NITRO},
		func(configBytes []byte) ([]byte, error) { return configBytes, nil },
	)

	runner.Run(subscribe)
}
