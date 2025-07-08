package main

import (
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
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

func initFn(_ *sdk.Environment[[]byte]) (sdk.Workflow[[]byte], error) {
	return sdk.Workflow[[]byte]{
		sdk.Handler(
			basictrigger.Trigger(&basictrigger.Config{}),
			secrets,
		),
	}, nil
}

func secrets(env *sdk.Environment[[]byte], _ sdk.Runtime, _ *basictrigger.Outputs) (string, error) {
	s, err := env.GetSecret(&pb.SecretRequest{Id: "Foo"}).Await()
	if err != nil {
		return "", err
	}
	return s.Value, nil
}
