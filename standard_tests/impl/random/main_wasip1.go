package main

import (
	"strconv"

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

func proveRand(env *cre.Environment[[]byte], r cre.Runtime, _ *basictrigger.Outputs) (uint64, error) {
	dr, err := r.Rand()
	if err != nil {
		return 0, err
	}

	total := dr.Uint64()
	_, err = cre.RunInNodeMode(
		env,
		r,
		nodeMode,
		cre.ConsensusAggregationFromTags[*resultType]().WithDefault(&resultType{OutputThing: 123}),
	).Await()
	if err != nil {
		return 0, err
	}

	total += dr.Uint64()
	return total, nil
}

func nodeMode(env *cre.NodeEnvironment[[]byte], nrt cre.NodeRuntime) (*resultType, error) {
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
