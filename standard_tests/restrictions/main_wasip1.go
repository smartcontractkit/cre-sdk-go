package main

import (
	"log/slog"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
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
		cre.HandlerWithPreHook(
			basictrigger.Trigger(&basictrigger.Config{Name: "first-trigger", Number: 100}),
			triggerFn,
			preHook,
		),
	}, nil
}

func triggerFn(_ []byte, rt cre.Runtime, _ *basictrigger.Outputs) (string, error) {
	client := &basicaction.BasicAction{}
	result, err := client.PerformAction(rt, &basicaction.Inputs{InputThing: true}).Await()
	if err != nil {
		return "", err
	}
	return result.AdaptedThing, nil
}

func preHook(_ []byte, _ *basictrigger.Outputs) (*sdk.Restrictions, error) {
	return &sdk.Restrictions{
		Capabilities: &sdk.CapabilityRestrictions{
			MaxTotalCalls: 0,
		},
	}, nil
}
