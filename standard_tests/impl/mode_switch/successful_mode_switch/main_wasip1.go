package main

import (
	"fmt"
	"time"

	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
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
			basictrigger.Trigger(&basictrigger.Config{}),
			changeModes,
		),
	}, nil
}

type resultType struct {
	OutputThing int32 `consensus_aggregation:"median"`
}

func changeModes(env *sdk.Environment[[]byte], rt sdk.Runtime, _ *basictrigger.Outputs) (string, error) {
	ignoreTimeCall()
	dinput := &basicaction.Inputs{InputThing: true}
	doutput, err := (&basicaction.BasicAction{}).PerformAction(rt, dinput).Await()
	if err != nil {
		return "", err
	}

	defaultValue := &resultType{OutputThing: 123}
	coutput, err := sdk.RunInNodeMode(
		env,
		rt,
		nodeMode,
		sdk.ConsensusAggregationFromTags[*resultType]().WithDefault(defaultValue),
	).Await()

	ignoreTimeCall()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d", doutput.AdaptedThing, coutput.OutputThing), nil
}

func nodeMode(_ *sdk.NodeEnvironment[[]byte], nrt sdk.NodeRuntime) (*resultType, error) {
	ignoreTimeCall()
	ninput := &nodeaction.NodeInputs{InputThing: true}
	result, err := (&nodeaction.BasicAction{}).PerformAction(nrt, ninput).Await()
	if err != nil {
		return nil, err
	}

	return &resultType{OutputThing: result.OutputThing}, nil
}

// ignoreTimeCall makes a time now call and forces the compiler not to optimize it away.
func ignoreTimeCall() {
	fmt.Println(time.Now())
}
