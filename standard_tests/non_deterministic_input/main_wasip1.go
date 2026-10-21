package main

import (
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
)

// This workflow derives its capability request input from the DON random source
// (seeded by the host). The suspend/resume integrity check requires a resumed
// (replayed) execution to issue exactly the same capability requests as the
// original run. The accompanying host test changes the seed between the
// suspended run and the resume, so the replay builds a different request for the
// same callback id and the host rejects it as non-deterministic.
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
			onTrigger,
		),
	}, nil
}

func onTrigger(_ []byte, rt cre.Runtime, _ *basictrigger.Outputs) (string, error) {
	dr, err := rt.Rand()
	if err != nil {
		return "", err
	}

	// The request payload depends on the seed, so changing the seed across a
	// replay produces a different request.
	input := &basicaction.Inputs{InputThing: dr.Uint64()%2 == 0}
	result, err := (&basicaction.BasicAction{}).PerformAction(rt, input).Await()
	if err != nil {
		return "", err
	}
	return result.AdaptedThing, nil
}
