package main

import (
	"fmt"
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/nodeaction"
)

func main() {
	runner := wasm.NewRunner(func(configBytes []byte) ([]byte, error) {
		return configBytes, nil
	})
	runner.Run(initFn)
}

func initFn([]byte, *slog.Logger, cre.SecretsProvider) (cre.Workflow[[]byte], error) {
	return cre.Workflow[[]byte]{
		cre.Handler(
			basictrigger.Trigger(&basictrigger.Config{}),
			changeModes,
		),
	}, nil
}

type resultType struct {
	OutputThing int32 `consensus_aggregation:"median"`
}

func changeModes(config []byte, rt cre.Runtime, _ *basictrigger.Outputs) (string, error) {
	ignoreTimeCall(rt)
	dinput := &basicaction.Inputs{InputThing: true}
	doutput, err := (&basicaction.BasicAction{}).PerformAction(rt, dinput).Await()
	if err != nil {
		return "", err
	}

	defaultValue := &resultType{OutputThing: 123}
	coutput, err := cre.RunInNodeMode(
		config,
		rt,
		nodeMode,
		cre.ConsensusAggregationFromTags[*resultType]().WithDefault(defaultValue),
	).Await()

	ignoreTimeCall(rt)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%d", doutput.AdaptedThing, coutput.OutputThing), nil
}

func nodeMode(_ []byte, nrt cre.NodeRuntime) (*resultType, error) {
	ignoreTimeCall(nrt)
	ninput := &nodeaction.NodeInputs{InputThing: true}
	result, err := (&nodeaction.BasicAction{}).PerformAction(nrt, ninput).Await()
	if err != nil {
		return nil, err
	}

	return &resultType{OutputThing: result.OutputThing}, nil
}

// ignoreTimeCall makes a time now call and forces the compiler not to optimize it away.
func ignoreTimeCall(rt cre.RuntimeBase) {
	fmt.Println(rt.Now())
}
