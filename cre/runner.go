package cre

import (
	"log/slog"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
)

// Runner is the entry point to running a CRE workflow.
type Runner[C any] interface {
	// Run creates the workflow and starts it.
	// Upon registration of a workflow, a run is used to register to `Trigger`s.
	// Upon receiving a trigger, the appropriate handler's callback is invoked.
	Run(initFn func(config C, logger *slog.Logger, secretsProvider SecretsProvider) (Workflow[C], error))
}

type AnyTee struct{}

type TeeType = sdk.TeeType

const TeeType_TEE_TYPE_AWS_NITRO = sdk.TeeType_TEE_TYPE_AWS_NITRO

type AcceptedTees interface {
	[]TeeType | AnyTee
}

// TeeRunner is the entry point to running a CRE workflow in TEE (Trusted Execution Environment) mode.
type TeeRunner[C any] interface {
	// Run creates the TEE workflow and starts it.
	// Upon registration of a workflow, a run is used to register to `Trigger`s.
	// Upon receiving a trigger, the appropriate handler's callback is invoked with a TeeRuntime.
	Run(initFn func(config C, logger *slog.Logger, secretsProvider SecretsProvider) (TeeWorkflow[C], error))
}
