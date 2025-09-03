package main

import (
	"fmt"
	"log/slog"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/wasm"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/actionandtrigger"
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
			basictrigger.Trigger(&basictrigger.Config{
				Name:   "first-trigger",
				Number: 100,
			}),
			prove0,
		),
		cre.Handler(
			actionandtrigger.Trigger(&actionandtrigger.Config{
				Name:   "second-trigger",
				Number: 150,
			}),
			prove1,
		),
		cre.Handler(
			basictrigger.Trigger(&basictrigger.Config{
				Name:   "third-trigger",
				Number: 200,
			}),
			prove2,
		),
	}, nil
}

func prove0(_ []byte, _ cre.Runtime, t *basictrigger.Outputs) (string, error) {
	return returnMsg(0, t.CoolOutput), nil
}

func prove1(_ []byte, _ cre.Runtime, t *actionandtrigger.TriggerEvent) (string, error) {
	return returnMsg(1, t.CoolOutput), nil
}

func prove2(_ []byte, _ cre.Runtime, t *basictrigger.Outputs) (string, error) {
	return returnMsg(2, t.CoolOutput), nil
}

func returnMsg(id int, value string) string {
	return fmt.Sprintf("called %v with %v", id, value)
}
