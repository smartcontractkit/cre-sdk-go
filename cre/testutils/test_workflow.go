package testutils

import (
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
)

func RunTestWorkflow(runner cre.Runner[string]) {
	runner.Run(func(env *cre.Environment[string]) (cre.Workflow[string], error) {
		return cre.Workflow[string]{
			cre.Handler(
				basictrigger.Trigger(TestWorkflowTriggerConfig()),
				onTrigger),
		}, nil
	})
}

func RunIdenticalTriggersWorkflow(runner cre.Runner[string]) {
	runner.Run(func(env *cre.Environment[string]) (cre.Workflow[string], error) {
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
				func(env *cre.Environment[string], rt cre.Runtime, outputs *basictrigger.Outputs) (string, error) {
					res, err := onTrigger(env, rt, outputs)
					if err != nil {
						return "", err
					}
					return res + "true", nil
				},
			),
		}, nil
	})
}

func onTrigger(env *cre.Environment[string], runtime cre.Runtime, outputs *basictrigger.Outputs) (string, error) {
	env.Logger.Info("Hi")
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

func RunTestSecretsWorkflow(runner cre.Runner[string]) {
	runner.Run(func(env *cre.Environment[string]) (cre.Workflow[string], error) {
		_, err := env.GetSecret(&pb.SecretRequest{Id: "Foo"}).Await()
		if err != nil {
			return nil, err
		}
		return cre.Workflow[string]{
			cre.Handler(
				basictrigger.Trigger(TestWorkflowTriggerConfig()),
				func(env *cre.Environment[string], rt cre.Runtime, outputs *basictrigger.Outputs) (string, error) {
					secret, err := env.GetSecret(&pb.SecretRequest{Id: "Foo"}).Await()
					if err != nil {
						return "", err
					}
					return secret.Value, nil
				}),
		}, nil
	})
}
