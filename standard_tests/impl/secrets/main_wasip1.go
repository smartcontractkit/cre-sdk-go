package main

import (
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
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
			secrets,
		),
	}, nil
}

func secrets(env *cre.Environment[[]byte], _ cre.Runtime, _ *basictrigger.Outputs) (string, error) {
	s, err := env.GetSecret(&pb.SecretRequest{Id: "Foo"}).Await()
	if err != nil {
		return "", err
	}
	return s.Value, nil
}
