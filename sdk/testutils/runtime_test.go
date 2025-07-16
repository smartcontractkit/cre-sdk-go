package testutils_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	valuespb "github.com/smartcontractkit/chainlink-common/pkg/values/pb"
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	basicactionmock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction/mock"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/nodeaction"
	nodeactionmock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/nodeaction/mock"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/sdk/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/proto"
)

func TestRuntime_CallCapability(t *testing.T) {
	t.Run("response too large", func(t *testing.T) {
		action, err := basicactionmock.NewBasicActionCapability(t)
		require.NoError(t, err)
		action.PerformAction = func(_ context.Context, input *basicaction.Inputs) (*basicaction.Outputs, error) {
			return &basicaction.Outputs{AdaptedThing: strings.Repeat("a", sdk.DefaultMaxResponseSizeBytes+1)}, nil
		}

		rt, _ := testutils.NewRuntimeAndEnv(t, "", map[string]string{})
		workflowAction1 := &basicaction.BasicAction{}
		call := workflowAction1.PerformAction(rt, &basicaction.Inputs{InputThing: true})
		_, err = call.Await()

		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), sdk.ResponseBufferTooSmall))
	})
}

func TestRuntime_ReturnsErrorsFromCapabilitiesThatDoNotExist(t *testing.T) {
	rt, _ := testutils.NewRuntimeAndEnv(t, "", map[string]string{})
	workflowAction1 := &basicaction.BasicAction{}
	call := workflowAction1.PerformAction(rt, &basicaction.Inputs{InputThing: true})
	_, err := call.Await()

	require.Error(t, err)
}

func TestRuntime_ConsensusReturnsTheObservation(t *testing.T) {
	anyValue := int32(100)
	nodeCapability, err := nodeactionmock.NewBasicActionCapability(t)
	require.NoError(t, err)
	nodeCapability.PerformAction = func(_ context.Context, _ *nodeaction.NodeInputs) (*nodeaction.NodeOutputs, error) {
		return &nodeaction.NodeOutputs{OutputThing: anyValue}, nil
	}

	rt, env := testutils.NewRuntimeAndEnv(t, "anything", map[string]string{})
	require.NoError(t, err)

	consensus := sdk.RunInNodeMode(env, rt, func(_ *sdk.NodeEnvironment[string], nodeRuntime sdk.NodeRuntime) (int32, error) {
		action := &nodeaction.BasicAction{}
		resp, err := action.PerformAction(nodeRuntime, &nodeaction.NodeInputs{InputThing: true}).Await()
		require.NoError(t, err)
		return resp.OutputThing, nil
	}, sdk.ConsensusMedianAggregation[int32]())

	consensusResult, err := consensus.Await()

	require.NoError(t, err)
	assert.Equal(t, anyValue, consensusResult)
}

func TestRuntime_ConsensusReturnsTheDefaultValue(t *testing.T) {
	anyValue := int32(100)

	runtime, env := testutils.NewRuntimeAndEnv(t, "anything", map[string]string{})
	consensus := sdk.RunInNodeMode(
		env,
		runtime,
		func(_ *sdk.NodeEnvironment[string], nodeRuntime sdk.NodeRuntime) (int32, error) {
			return 0, errors.New("no consensus")
		},
		sdk.ConsensusMedianAggregation[int32]().WithDefault(anyValue))

	consensusResult, err := consensus.Await()
	require.NoError(t, err)
	assert.Equal(t, anyValue, consensusResult)
}

func TestRuntime_ConsensusReturnsErrors(t *testing.T) {
	runtime, env := testutils.NewRuntimeAndEnv(t, "anything", map[string]string{})
	anyErr := errors.New("no consensus")
	consensus := sdk.RunInNodeMode(
		env,
		runtime,
		func(_ *sdk.NodeEnvironment[string], nodeRuntime sdk.NodeRuntime) (int32, error) {
			return 0, anyErr
		},
		sdk.ConsensusMedianAggregation[int32]())
	_, err := consensus.Await()
	require.ErrorContains(t, err, anyErr.Error())
}

func TestRuntime_CallsReportMethod(t *testing.T) {
	expectedInputPayload := []byte("some_encoded_report_data")
	runtime, _ := testutils.NewRuntimeAndEnv(t, "test_config", nil)
	reportRequest := &pb.ReportRequest{
		EncodedPayload: expectedInputPayload,
		EncoderName:    "my-encoder",
		SigningAlgo:    "my-signer",
		HashingAlgo:    "my-hasher",
	}

	reportPromise := runtime.GenerateReport(reportRequest)
	reportResponse, err := reportPromise.Await()
	require.NoError(t, err)
	require.NotNil(t, reportResponse)

	var actualReport valuespb.Value
	err = proto.Unmarshal(reportResponse.RawReport, &actualReport)
	require.NoError(t, err)

	actualReportMap := actualReport.GetMapValue()

	actualPayloadValue, ok := actualReportMap.Fields[sdk.ConsensusResponseMapKeyPayload]
	assert.True(t, ok)

	actualMetadataValue, ok := actualReportMap.Fields[sdk.ConsensusResponseMapKeyMetadata]
	assert.True(t, ok)

	// Assert on expected data
	assert.Equal(t, &valuespb.Value_BytesValue{BytesValue: expectedInputPayload}, actualPayloadValue.Value)
	assert.Equal(t, &valuespb.Value_StringValue{StringValue: "test_metadata"}, actualMetadataValue.Value)
}
