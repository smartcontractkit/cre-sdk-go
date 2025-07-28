package testutils_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/testutils"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	basicactionmock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction/mock"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/nodeaction"
	nodeactionmock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/nodeaction/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntime_CallCapability(t *testing.T) {
	t.Run("response too large", func(t *testing.T) {
		action, err := basicactionmock.NewBasicActionCapability(t)
		require.NoError(t, err)
		action.PerformAction = func(_ context.Context, input *basicaction.Inputs) (*basicaction.Outputs, error) {
			return &basicaction.Outputs{AdaptedThing: strings.Repeat("a", cre.DefaultMaxResponseSizeBytes+1)}, nil
		}

		rt, _ := testutils.NewRuntimeAndEnv(t, "", map[string]string{})
		workflowAction1 := &basicaction.BasicAction{}
		call := workflowAction1.PerformAction(rt, &basicaction.Inputs{InputThing: true})
		_, err = call.Await()

		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), cre.ResponseBufferTooSmall))
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

	consensus := cre.RunInNodeMode(env, rt, func(_ *cre.NodeEnvironment[string], nodeRuntime cre.NodeRuntime) (int32, error) {
		action := &nodeaction.BasicAction{}
		resp, err := action.PerformAction(nodeRuntime, &nodeaction.NodeInputs{InputThing: true}).Await()
		require.NoError(t, err)
		return resp.OutputThing, nil
	}, cre.ConsensusMedianAggregation[int32]())

	consensusResult, err := consensus.Await()

	require.NoError(t, err)
	assert.Equal(t, anyValue, consensusResult)
}

func TestRuntime_ConsensusReturnsTheDefaultValue(t *testing.T) {
	anyValue := int32(100)

	runtime, env := testutils.NewRuntimeAndEnv(t, "anything", map[string]string{})
	consensus := cre.RunInNodeMode(
		env,
		runtime,
		func(_ *cre.NodeEnvironment[string], nodeRuntime cre.NodeRuntime) (int32, error) {
			return 0, errors.New("no consensus")
		},
		cre.ConsensusMedianAggregation[int32]().WithDefault(anyValue))

	consensusResult, err := consensus.Await()
	require.NoError(t, err)
	assert.Equal(t, anyValue, consensusResult)
}

func TestRuntime_ConsensusReturnsErrors(t *testing.T) {
	runtime, env := testutils.NewRuntimeAndEnv(t, "anything", map[string]string{})
	anyErr := errors.New("no consensus")
	consensus := cre.RunInNodeMode(
		env,
		runtime,
		func(_ *cre.NodeEnvironment[string], nodeRuntime cre.NodeRuntime) (int32, error) {
			return 0, anyErr
		},
		cre.ConsensusMedianAggregation[int32]())
	_, err := consensus.Await()
	require.ErrorContains(t, err, anyErr.Error())
}

func TestRuntime_CallsReportMethod(t *testing.T) {
	expectedInputPayload := []byte("some_encoded_report_data")
	expectedMetadata := make([]byte, cre.ReportMetadataHeaderLength)
	for i := range cre.ReportMetadataHeaderLength {
		expectedMetadata[i] = byte(i % 256)
	}
	expectedRawReport := append(expectedMetadata, expectedInputPayload...)

	runtime, _ := testutils.NewRuntimeAndEnv(t, "test_config", nil)
	reportRequest := &pb.ReportRequest{
		EncodedPayload: expectedInputPayload,
		EncoderName:    "my-encoder",
		SigningAlgo:    "my-signer",
		HashingAlgo:    "my-hasher",
	}

	resp, err := runtime.GenerateReport(reportRequest).Await()
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Assert the RawReport matches the expected concatenated bytes
	assert.Equal(t, expectedRawReport, resp.RawReport)
	assert.Equal(t, len(expectedRawReport), len(resp.RawReport))

	assert.Equal(t, []byte("default_signature_1"), resp.Sigs[0].Signature)
	assert.Equal(t, []byte("default_signature_2"), resp.Sigs[1].Signature)
}
