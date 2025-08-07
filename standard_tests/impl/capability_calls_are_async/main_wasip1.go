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
	input1 := &basicaction.Inputs{InputThing: true}
	input2 := &basicaction.Inputs{InputThing: false}
	action := basicaction.BasicAction{}

	r1Id := action.PerformAction(rt, input1)
	r1I2 := action.PerformAction(rt, input2)

	results2, err := r1I2.Await()
	if err != nil {
		return "", err
	}

	results1, err := r1Id.Await()
	if err != nil {
		return "", err
	}

	return results1.AdaptedThing + results2.AdaptedThing, nil
}
