//go:build wasip1

package main

import (
	"github.com/smartcontractkit/cre-sdk-go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/sdk/wasm"
)

func main() {
	wasm.NewRunner(sdk.ParseJSON[Config]).Run(InitWorkflow)
}
