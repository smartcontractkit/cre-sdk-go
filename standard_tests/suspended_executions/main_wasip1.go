package main

import (
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
)

// This workflow dispatches a single async capability call and awaits it. When
// suspension is enabled the host has no response at the await, so it suspends
// the guest; the handler runs again from the top once the host resumes it with
// the response. A log line is emitted on every run so the test can observe how
// many times the guest executed (once without suspension, twice with).
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
	rt.Logger().Info("suspended_executions:run")

	result, err := (&basicaction.BasicAction{}).PerformAction(rt, &basicaction.Inputs{InputThing: true}).Await()
	if err != nil {
		return "", err
	}
	return result.AdaptedThing, nil
}
