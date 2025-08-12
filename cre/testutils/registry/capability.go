package registry

import (
	"context"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
)

type Capability interface {
	Invoke(ctx context.Context, request *sdk.CapabilityRequest) *sdk.CapabilityResponse
	ID() string
}

type ErrNoTriggerStub string

func (n ErrNoTriggerStub) Error() string {
	return "Stub not implemented for trigger: " + string(n)
}

var _ error = ErrNoTriggerStub("")
