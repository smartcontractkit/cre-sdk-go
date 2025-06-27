package main

import (
	"github.com/smartcontractkit/cre-sdk-go/internal/capabilities/basicaction"
	"github.com/smartcontractkit/cre-sdk-go/internal/capabilities/basictrigger"
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
	return sdk.RunInNodeMode(
		env,
		rt,
		func(*sdk.NodeEnvironment[[]byte], sdk.NodeRuntime) (int32, error) {
			ba := basicaction.BasicAction{}
			_, err := ba.PerformAction(rt, &basicaction.Inputs{}).Await()
			return 123, err
		},
		sdk.ConsensusMedianAggregation[int32](),
	).Await()
}
