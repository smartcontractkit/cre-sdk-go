package main

import (
	"log/slog"

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

func initFn([]byte, *slog.Logger, cre.SecretsProvider) (cre.Workflow[[]byte], error) {
	return cre.Workflow[[]byte]{
		cre.Handler(
			basictrigger.Trigger(&basictrigger.Config{}),
			secrets,
		),
	}, nil
}

func secrets(_ []byte, rt cre.Runtime, _ *basictrigger.Outputs) (string, error) {
	s, err := rt.GetSecret(&pb.SecretRequest{Id: "Foo"}).Await()
	if err != nil {
		return "", err
	}
	return s.Value, nil
}
