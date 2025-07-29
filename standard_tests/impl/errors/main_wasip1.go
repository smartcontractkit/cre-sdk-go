package main

import (
	"errors"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
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
			basictrigger.Trigger(&basictrigger.Config{}),
			returnConfig,
		),
	}, nil
}

func returnConfig(_ *cre.Environment[[]byte], _ cre.Runtime, _ *basictrigger.Outputs) ([]byte, error) {
	return nil, errors.New("workflow execution failure")
}
