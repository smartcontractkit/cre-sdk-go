package testutils

import (
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
)

func RunTestWorkflow(runner cre.Runner[string]) {
	runner.Run(func(config string, _ *slog.Logger, _ cre.SecretsProvider) (cre.Workflow[string], error) {
		return cre.Workflow[string]{
			cre.Handler(
				basictrigger.Trigger(TestWorkflowTriggerConfig()),
				onTrigger),
		}, nil
	})
}

func RunIdenticalTriggersWorkflow(runner cre.Runner[string]) {
	runner.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
		return cre.Workflow[string]{
			cre.Handler(
				basictrigger.Trigger(TestWorkflowTriggerConfig()),
				onTrigger,
			),
			cre.Handler(
				basictrigger.Trigger(&basictrigger.Config{
					Name:   "second-trigger",
					Number: 200,
				}),
				func(config string, rt cre.Runtime, outputs *basictrigger.Outputs) (string, error) {
					res, err := onTrigger(config, rt, outputs)
					if err != nil {
						return "", err
					}
					return res + "true", nil
				},
			),
		}, nil
	})
}

func onTrigger(_ string, runtime cre.Runtime, outputs *basictrigger.Outputs) (string, error) {
	runtime.Logger().Info("Hi")
	action := basicaction.BasicAction{ /* TODO config */ }
	first := action.PerformAction(runtime, &basicaction.Inputs{InputThing: false})
	firstResult, err := first.Await()
	if err != nil {
		return "", err
	}

	second := action.PerformAction(runtime, &basicaction.Inputs{InputThing: true})
	secondResult, err := second.Await()
	if err != nil {
		return "", err
	}

	return outputs.CoolOutput + firstResult.AdaptedThing + secondResult.AdaptedThing, nil
}
