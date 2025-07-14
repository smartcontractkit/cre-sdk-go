//go:build wasip1

package main

import (
	"fmt"
	"math/big"
	"strconv"

	"my-calculator-workflow/bindings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"

	"github.com/smartcontractkit/cre-sdk-go/capabilities/networking/http"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/sdk/wasm"
)

func main() {
	wasm.NewRunner(sdk.ParseJSON[Config]).Run(InitWorkflow)
}
