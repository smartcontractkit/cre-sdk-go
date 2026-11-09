package sdkimpl_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/testutils"
	"github.com/smartcontractkit/cre-sdk-go/internal/sdkimpl"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// Synthetic identifiers use production encodings; execution IDs follow the
// engine hash for the workflow ID, a cron event timestamp, and trigger index 0.
const (
	metadataWorkflowIDFixture      = "0020350fb4f89856a031268826d6e9b9340996637aa27859baa7b07db58bb020"
	metadataExecutionIDFixture     = "9f38bdece8892b3452c401a7a2d5a170f078ab9b2c98c591e24028ce94b33ed2"
	metadataOwnerFixture           = "5366890c26e8737b89d9084b4c632f7de73d1674"
	metadataOtherWorkflowIDFixture = "0052457bc477e58cad894372b8a80d1f0b03322c8a070d93bcf9bbd1f21cbd09"
)

func TestRuntime_ExecutionMetadataReturnsCopyWithoutHostCalls(t *testing.T) {
	want := metadataValueFixture()
	// Any accidental host operation would panic because RuntimeHelpers is nil.
	rt := &sdkimpl.RuntimeBase{ExecutionMeta: want}
	got := rt.ExecutionMetadata()
	require.Equal(t, want, got)

	got.WorkflowID = metadataOtherWorkflowIDFixture
	require.Equal(t, want, rt.ExecutionMetadata())
}

func TestTeeRuntime_ExecutionMetadataMatchesDON(t *testing.T) {
	want := metadataValueFixture()
	don := &sdkimpl.Runtime{RuntimeBase: sdkimpl.RuntimeBase{ExecutionMeta: want}}
	tee := sdkimpl.NewTeeRuntime(don)

	require.Equal(t, want, tee.ExecutionMetadata())
	require.Same(t, don, tee.UsingTheDons())
}

func TestRuntime_ExecutionMetadataDefaultsToZero(t *testing.T) {
	var rt sdkimpl.RuntimeBase
	require.Zero(t, rt.ExecutionMetadata())
}

func TestRuntime_SetExecutionMetadata(t *testing.T) {
	rt := testutils.NewRuntime(t, nil)
	want := metadataValueFixture()
	rt.SetExecutionMetadata(want)
	require.Equal(t, want, rt.ExecutionMetadata())
}

func TestNodeRuntime_ExecutionMetadataMatchesDON(t *testing.T) {
	rt := testutils.NewRuntime(t, nil)
	want := metadataValueFixture()
	rt.SetExecutionMetadata(want)
	var observed cre.WorkflowExecutionMetadata

	// The node callback runs synchronously; this test does not await consensus.
	rt.RunInNodeMode(func(node cre.NodeRuntime) *sdk.SimpleConsensusInputs {
		observed = node.ExecutionMetadata()
		return &sdk.SimpleConsensusInputs{}
	})

	require.Equal(t, want, observed)
	require.Equal(t, want, rt.ExecutionMetadata())
}

func TestRuntime_SetExecutionRequestMapsMetadata(t *testing.T) {
	var rt sdkimpl.RuntimeBase
	rt.SetExecutionRequest(metadataRequestFixture())

	require.Equal(t, metadataValueFixture(), rt.ExecutionMetadata())
	require.Equal(t, uint64(2048), rt.MaxResponseSize)
}

func TestRuntime_SetExecutionRequestMissingMetadataClearsPreviousValues(t *testing.T) {
	for _, tc := range []struct {
		name                string
		request             *sdk.ExecuteRequest
		wantMaxResponseSize uint64
	}{
		{name: "nil request"},
		{name: "old host", request: &sdk.ExecuteRequest{MaxResponseSize: 4096}, wantMaxResponseSize: 4096},
		{name: "empty message", request: &sdk.ExecuteRequest{MaxResponseSize: 4096, ExecutionMetadata: &sdk.WorkflowExecutionMetadata{}}, wantMaxResponseSize: 4096},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rt sdkimpl.RuntimeBase
			rt.SetExecutionRequest(metadataRequestFixture())
			rt.SetExecutionRequest(tc.request)

			require.Zero(t, rt.ExecutionMetadata())
			require.Equal(t, tc.wantMaxResponseSize, rt.MaxResponseSize)
		})
	}
}

func TestRuntime_SetExecutionRequestPreservesZeroF(t *testing.T) {
	request := metadataRequestFixture()
	request.ExecutionMetadata.DonF = 0
	want := metadataValueFixture()
	want.DonF = 0
	var rt sdkimpl.RuntimeBase
	rt.SetExecutionRequest(request)

	require.Equal(t, want, rt.ExecutionMetadata())
}

func TestRuntime_SetExecutionRequestCopiesSourceMetadata(t *testing.T) {
	request := metadataRequestFixture()
	var rt sdkimpl.RuntimeBase
	rt.SetExecutionRequest(request)
	proto.Reset(request.ExecutionMetadata)
	request.MaxResponseSize = 0

	require.Equal(t, metadataValueFixture(), rt.ExecutionMetadata())
	require.Equal(t, uint64(2048), rt.MaxResponseSize)
}

func TestRuntime_SetExecutionRequestReplacesPreviousMetadata(t *testing.T) {
	var rt sdkimpl.RuntimeBase
	rt.SetExecutionRequest(metadataRequestFixture())
	request := &sdk.ExecuteRequest{
		MaxResponseSize: 4096,
		ExecutionMetadata: &sdk.WorkflowExecutionMetadata{
			WorkflowId: metadataOtherWorkflowIDFixture,
		},
	}

	rt.SetExecutionRequest(request)

	require.Equal(t, cre.WorkflowExecutionMetadata{WorkflowID: metadataOtherWorkflowIDFixture}, rt.ExecutionMetadata())
	require.Equal(t, uint64(4096), rt.MaxResponseSize)
}

func metadataValueFixture() cre.WorkflowExecutionMetadata {
	return cre.WorkflowExecutionMetadata{
		WorkflowID:    metadataWorkflowIDFixture,
		ExecutionID:   metadataExecutionIDFixture,
		WorkflowOwner: metadataOwnerFixture,
		DonID:         17,
		DonF:          3,
	}
}

func metadataRequestFixture() *sdk.ExecuteRequest {
	return &sdk.ExecuteRequest{
		MaxResponseSize: 2048,
		ExecutionMetadata: &sdk.WorkflowExecutionMetadata{
			WorkflowId:    metadataWorkflowIDFixture,
			ExecutionId:   metadataExecutionIDFixture,
			WorkflowOwner: metadataOwnerFixture,
			DonId:         17,
			DonF:          3,
		},
	}
}
