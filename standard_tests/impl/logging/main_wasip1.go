package main

import (
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
			doLog,
		),
	}, nil
}

func doLog(env *sdk.Environment[[]byte], _ sdk.Runtime, _ *basictrigger.Outputs) ([]byte, error) {
	env.Logger.Info("log from wasm!")
	return []byte{}, nil
}
