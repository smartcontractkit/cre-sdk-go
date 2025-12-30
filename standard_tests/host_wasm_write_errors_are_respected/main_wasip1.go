package main

import (
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
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
			basictrigger.Trigger(&basictrigger.Config{
				Name:   "first-trigger",
				Number: 100,
			}),
			asyncCalls,
		),
	}, nil
}

func asyncCalls(_ []byte, rt cre.Runtime, _ *basictrigger.Outputs) (string, error) {
	input := &basicaction.Inputs{InputThing: true}
	action := basicaction.BasicAction{}
	rId := action.PerformAction(rt, input)

	_, err := rId.Await()
	if err != nil {
		return "", err
	}

	return "Should not get here, an error is expected", nil
}
