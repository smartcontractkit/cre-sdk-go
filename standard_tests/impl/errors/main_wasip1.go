package main

import (
	"errors"

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
			returnConfig,
		),
	}, nil
}

func returnConfig(_ *sdk.Environment[[]byte], _ sdk.Runtime, _ *basictrigger.Outputs) ([]byte, error) {
	return nil, errors.New("workflow execution failure")
}
