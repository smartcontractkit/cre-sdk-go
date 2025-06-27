package main

import (
	"github.com/smartcontractkit/cre-sdk-go/internal/capabilities/basictrigger"
	"github.com/smartcontractkit/cre-sdk-go/internal/capabilities/nodeaction"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/sdk/wasm"
)

func main() {
	runner := wasm.NewRunner(func(configBytes []byte) ([]byte, error) {
		return configBytes, nil
	})
	runner.Run(initFn)
}

func initFn(_ *sdk.Environment[[]byte]) (sdk.Workflow[[]byte], error) {
	return sdk.Workflow[[]byte]{
		sdk.Handler(
			basictrigger.Trigger(&basictrigger.Config{}),
			breakClosure,
		),
	}, nil
}

func breakClosure(env *sdk.Environment[[]byte], rt sdk.Runtime, _ *basictrigger.Outputs) (int32, error) {
	var nrt sdk.NodeRuntime
	_, err := sdk.RunInNodeMode(
		env,
		rt,
		func(_ *sdk.NodeEnvironment[[]byte], r sdk.NodeRuntime) (string, error) {
			nrt = r
			return "hi", nil
		},
		sdk.ConsensusIdenticalAggregation[string](),
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
