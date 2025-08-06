package registry

import (
	"context"

	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
)

// Capability is meant to be implemented by generated code for capability mocks.
type Capability interface {
	Invoke(ctx context.Context, request *pb.CapabilityRequest) *pb.CapabilityResponse
	ID() string
}
