package main

import (
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/sdk/wasm"
)

func main() {
	runner := wasm.NewRunner(func(configBytes []byte) ([]byte, error) {
		return configBytes, nil
	})
	runner.Run(initFn)
}

func initFn(env *sdk.Environment[[]byte]) (sdk.Workflow[[]byte], error) {
	return sdk.Workflow[[]byte]{
		sdk.Handler(
			basictrigger.Trigger(&basictrigger.Config{}),
			returnConfig,
		),
	}, nil
}

func returnConfig(env *sdk.Environment[[]byte], _ sdk.Runtime, _ *basictrigger.Outputs) ([]byte, error) {
	return env.Config, nil
}
