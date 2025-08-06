package main

import (
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
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
			breakClosure,
		),
	}, nil
}

func breakClosure(config []byte, rt cre.Runtime, _ *basictrigger.Outputs) (int32, error) {
	return cre.RunInNodeMode(
		config,
		rt,
		func([]byte, cre.NodeRuntime) (int32, error) {
			ba := basicaction.BasicAction{}
			_, err := ba.PerformAction(rt, &basicaction.Inputs{}).Await()
			return 123, err
		},
		cre.ConsensusMedianAggregation[int32](),
	).Await()
}
