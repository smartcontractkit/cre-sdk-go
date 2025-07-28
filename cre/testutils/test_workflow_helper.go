package testutils

import (
	"context"
	"sync"
	"testing"

	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	basicactionmock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction/mock"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowTrigger() *basictrigger.Outputs {
	return &basictrigger.Outputs{CoolOutput: "Hi"}
}

func TestWorkflowTriggerConfig() *basictrigger.Config {
	return &basictrigger.Config{
		Name:   "name",
		Number: 100,
	}
}

func SetupExpectedCalls(t *testing.T) {
	basicAction, err := basicactionmock.NewBasicActionCapability(t)
	require.NoError(t, err)

	firstCall := true
	callLock := &sync.Mutex{}
	basicAction.PerformAction = func(ctx context.Context, input *basicaction.Inputs) (*basicaction.Outputs, error) {
		callLock.Lock()
		defer callLock.Unlock()
		assert.NotEqual(t, firstCall, input.InputThing, "failed first call assertion")
		firstCall = false
		if input.InputThing {
			return &basicaction.Outputs{AdaptedThing: "true"}, nil
		} else {
			return &basicaction.Outputs{AdaptedThing: "false"}, nil
		}
	}
}

func TestWorkflowExpectedResult() string {
	return "Hifalsetrue"
}
