package registry

import (
	"context"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
)

// Capability is meant to be implemented by generated code for capability mocks.
type Capability interface {
	Invoke(ctx context.Context, request *sdk.CapabilityRequest) *sdk.CapabilityResponse
	ID() string
}
