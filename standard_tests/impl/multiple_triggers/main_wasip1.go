package main

import (
	"fmt"

	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/actionandtrigger"
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
			basictrigger.Trigger(&basictrigger.Config{
				Name:   "first-trigger",
				Number: 100,
			}),
			prove0,
		),
		sdk.Handler(
			actionandtrigger.Trigger(&actionandtrigger.Config{
				Name:   "second-trigger",
				Number: 150,
			}),
			prove1,
		),
		sdk.Handler(
			basictrigger.Trigger(&basictrigger.Config{
				Name:   "third-trigger",
				Number: 200,
			}),
			prove2,
		),
	}, nil
}

type resultType struct {
	OutputThing int32 `consensus_aggregation:"median"`
}

func prove0(_ *sdk.Environment[[]byte], _ sdk.Runtime, t *basictrigger.Outputs) (string, error) {
	return returnMsg(0, t.CoolOutput), nil
}

func prove1(_ *sdk.Environment[[]byte], _ sdk.Runtime, t *actionandtrigger.TriggerEvent) (string, error) {
	return returnMsg(1, t.CoolOutput), nil
}

func prove2(_ *sdk.Environment[[]byte], _ sdk.Runtime, t *basictrigger.Outputs) (string, error) {
	return returnMsg(2, t.CoolOutput), nil
}

func returnMsg(id int, value string) string {
	return fmt.Sprintf("called %v with %v", id, value)
}
