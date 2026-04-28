package cre

import (
	"log/slog"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
)

type InitFn[C any] = func(config C, logger *slog.Logger, secretsProvider SecretsProvider) (Workflow[C], error)

// Runner is the entry point to running a CRE workflow.
type Runner[C any] interface {
	// Run creates the workflow and starts it.
	// Upon registration of a workflow, a run is used to register to `Trigger`s.
	// Upon receiving a trigger, the appropriate handler's callback is invoked.
	Run(initFn InitFn[C])
}

type AnyTee struct{}

type Type = sdk.TeeType

type TeeAndRegions struct {
	Type

	// Regions limits what regions the TEE can run in.
	// If empty or nil, there is no region limitation.
	Regions []string
}

const TeeType_TEE_TYPE_AWS_NITRO = sdk.TeeType_TEE_TYPE_AWS_NITRO

type AcceptedTees interface {
	[]TeeAndRegions | AnyTee
}
