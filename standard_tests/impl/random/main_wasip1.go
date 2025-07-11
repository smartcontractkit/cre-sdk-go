package main

import (
	"strconv"

	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/nodeaction"
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
			basictrigger.Trigger(&basictrigger.Config{
				Name:   "first-trigger",
				Number: 100,
			}),
			proveRand,
		),
	}, nil
}

type resultType struct {
	OutputThing int32 `consensus_aggregation:"median"`
}

func proveRand(env *sdk.Environment[[]byte], r sdk.Runtime, _ *basictrigger.Outputs) (uint64, error) {
	dr, err := r.Rand()
	if err != nil {
		return 0, err
	}

	total := dr.Uint64()
	_, err = sdk.RunInNodeMode(
		env,
		r,
		nodeMode,
		sdk.ConsensusAggregationFromTags[*resultType]().WithDefault(&resultType{OutputThing: 123}),
	).Await()
	if err != nil {
		return 0, err
	}

	total += dr.Uint64()
	return total, nil
}

func nodeMode(env *sdk.NodeEnvironment[[]byte], nrt sdk.NodeRuntime) (*resultType, error) {
	input := &nodeaction.NodeInputs{InputThing: true}
	result, err := (&nodeaction.BasicAction{}).PerformAction(nrt, input).Await()
	if err != nil {
		return nil, err
	}

	if result.OutputThing < 100 {
		nr, err := nrt.Rand()
		if err != nil {
			return nil, err
		}

		env.Logger.Info("***" + strconv.FormatUint(nr.Uint64(), 10) + "***")
	}

	return &resultType{OutputThing: result.OutputThing}, nil
}
