package main

import (
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/nodeaction"
)

func main() {
	runner := wasm.NewRunner(func(configBytes []byte) ([]byte, error) {
		return configBytes, nil
	})
	runner.Run(initFn)
}

func initFn(_ *cre.Environment[[]byte]) (cre.Workflow[[]byte], error) {
	return cre.Workflow[[]byte]{
		cre.Handler(
			basictrigger.Trigger(&basictrigger.Config{}),
			breakClosure,
		),
	}, nil
}

func breakClosure(env *cre.Environment[[]byte], rt cre.Runtime, _ *basictrigger.Outputs) (int32, error) {
	var nrt cre.NodeRuntime
	_, err := cre.RunInNodeMode(
		env,
		rt,
		func(_ *cre.NodeEnvironment[[]byte], r cre.NodeRuntime) (string, error) {
			nrt = r
			return "hi", nil
		},
		cre.ConsensusIdenticalAggregation[string](),
	).Await()
	if err != nil {
		return 0, err
	}

	nodeCap := nodeaction.BasicAction{}
	_, err = nodeCap.PerformAction(nrt, &nodeaction.NodeInputs{InputThing: true}).Await()
	if err != nil {
		return 0, err
	}

	return 1, nil
}
