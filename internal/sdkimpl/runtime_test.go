package sdkimpl_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"

	valuespb "github.com/smartcontractkit/chainlink-common/pkg/values/pb"
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
	"github.com/smartcontractkit/cre-sdk-go/internal/sdkimpl"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/actionandtrigger"
	actionandtriggermock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/actionandtrigger/mock"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
	basicactionmock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction/mock"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
	consensusmock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/consensus/mock"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/nodeaction"
	nodeactionmock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/nodeaction/mock"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/sdk/testutils"
	"google.golang.org/protobuf/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	anyTrigger = &basictrigger.Outputs{CoolOutput: "cool"}
	anyConfig  = &basictrigger.Config{Name: "name", Number: 123}
)

const anyEnvConfig = "env_config"

func TestRuntime_CallCapability(t *testing.T) {
	t.Run("runs async", func(t *testing.T) {
		ch := make(chan struct{}, 1)
		anyResult1 := "ok1"
		action1, err := basicactionmock.NewBasicActionCapability(t)
		require.NoError(t, err)
		action1.PerformAction = func(_ context.Context, input *basicaction.Inputs) (*basicaction.Outputs, error) {
			<-ch
			return &basicaction.Outputs{AdaptedThing: anyResult1}, nil
		}

		anyResult2 := "ok2"
		action2, err := actionandtriggermock.NewBasicCapability(t)
		action2.Action = func(ctx context.Context, input *actionandtrigger.Input) (*actionandtrigger.Output, error) {
			return &actionandtrigger.Output{Welcome: anyResult2}, nil
		}

		test := func(_ *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (string, error) {
			workflowAction1 := &basicaction.BasicAction{}
			call1 := workflowAction1.PerformAction(rt, &basicaction.Inputs{InputThing: true})

			workflowAction2 := &actionandtrigger.Basic{}
			call2 := workflowAction2.Action(rt, &actionandtrigger.Input{Name: "input"})
			result2, err := call2.Await()
			require.NoError(t, err)
			ch <- struct{}{}
			result1, err := call1.Await()
			require.NoError(t, err)
			return result1.AdaptedThing + result2.Welcome, nil
		}

		result, err := testRuntime(t, test)
		require.NoError(t, err)
		assert.Equal(t, anyResult1+anyResult2, result)
	})

	t.Run("call capability errors", func(t *testing.T) {
		// The capability is not registered, so the call will fail.
		test := func(_ *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (string, error) {
			workflowAction1 := &basicaction.BasicAction{}
			call := workflowAction1.PerformAction(rt, &basicaction.Inputs{InputThing: true})
			_, err := call.Await()
			return "", err
		}
		_, err := testRuntime(t, test)
		assert.Error(t, err)
	})

	t.Run("capability errors are returned to the caller", func(t *testing.T) {
		action, err := basicactionmock.NewBasicActionCapability(t)
		require.NoError(t, err)

		expectedErr := errors.New("error")
		action.PerformAction = func(ctx context.Context, input *basicaction.Inputs) (*basicaction.Outputs, error) {
			return nil, expectedErr
		}

		capability := &basicaction.BasicAction{}

		test := func(_ *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (string, error) {
			_, err := capability.PerformAction(rt, &basicaction.Inputs{InputThing: true}).Await()
			return "", err
		}

		_, err = testRuntime(t, test)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("await errors", func(t *testing.T) {
		action, err := basicactionmock.NewBasicActionCapability(t)
		require.NoError(t, err)
		expectedErr := errors.New("error")

		action.PerformAction = func(ctx context.Context, input *basicaction.Inputs) (*basicaction.Outputs, error) {
			return &basicaction.Outputs{AdaptedThing: "ok"}, nil
		}

		capability := &basicaction.BasicAction{}

		test := func(_ *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (string, error) {
			drt := rt.(*testutils.TestRuntime)
			drt.RuntimeHelpers = &awaitOverride{
				RuntimeHelpers: drt.RuntimeHelpers,
				await: func(request *pb.AwaitCapabilitiesRequest, maxResponseSize uint64) (*pb.AwaitCapabilitiesResponse, error) {
					return nil, expectedErr
				},
			}
			_, err := capability.PerformAction(rt, &basicaction.Inputs{InputThing: true}).Await()
			return "", err
		}

		_, err = testRuntime(t, test)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("await missing response", func(t *testing.T) {
		action, err := basicactionmock.NewBasicActionCapability(t)
		require.NoError(t, err)

		action.PerformAction = func(ctx context.Context, input *basicaction.Inputs) (*basicaction.Outputs, error) {
			return &basicaction.Outputs{AdaptedThing: "ok"}, nil
		}

		capability := &basicaction.BasicAction{}

		test := func(_ *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (string, error) {
			drt := rt.(*testutils.TestRuntime)
			drt.RuntimeHelpers = &awaitOverride{
				RuntimeHelpers: drt.RuntimeHelpers,
				await: func(request *pb.AwaitCapabilitiesRequest, maxResponseSize uint64) (*pb.AwaitCapabilitiesResponse, error) {
					return &pb.AwaitCapabilitiesResponse{Responses: map[int32]*pb.CapabilityResponse{}}, nil
				},
			}
			_, err := capability.PerformAction(rt, &basicaction.Inputs{InputThing: true}).Await()
			return "", err
		}

		_, err = testRuntime(t, test)
		assert.Error(t, err)
	})
}

func TestRuntime_Rand(t *testing.T) {
	t.Run("random delegates", func(t *testing.T) {
		runtime := testutils.NewRuntime(t, map[string]string{})
		runtime.SetRandomSource(rand.NewSource(1))
		r, err := runtime.Rand()
		require.NoError(t, err)
		result := r.Uint64()
		assert.Equal(t, rand.New(rand.NewSource(1)).Uint64(), result)
	})

	t.Run("random does not allow use in the wrong mode", func(t *testing.T) {
		test := func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (uint64, error) {
			return sdk.RunInNodeMode(env, rt, func(_ *sdk.NodeEnvironment[string], _ sdk.NodeRuntime) (uint64, error) {
				if _, err := rt.Rand(); err != nil {
					return 0, err
				}

				return 0, fmt.Errorf("should not be called in node mode")
			}, sdk.ConsensusMedianAggregation[uint64]()).Await()
		}

		_, err := testRuntime(t, test)
		require.Error(t, err)
	})

	t.Run("returned random panics if you use it in the wrong mode ", func(t *testing.T) {
		assert.Panics(t, func() {
			test := func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (uint64, error) {
				r, err := rt.Rand()
				if err != nil {
					return 0, err
				}
				return sdk.RunInNodeMode(env, rt, func(_ *sdk.NodeEnvironment[string], _ sdk.NodeRuntime) (uint64, error) {
					r.Uint64()
					return 0, fmt.Errorf("should not be called in node mode")
				}, sdk.ConsensusMedianAggregation[uint64]()).Await()
			}

			_, _ = testRuntime(t, test)
		})
	})
}

func TestDonRuntime_RunInNodeMode(t *testing.T) {
	t.Run("Successful consensus", func(t *testing.T) {
		nodeMock, err := nodeactionmock.NewBasicActionCapability(t)
		require.NoError(t, err)
		anyObservation := int32(10)
		anyMedian := int64(11)
		nodeMock.PerformAction = func(ctx context.Context, input *nodeaction.NodeInputs) (*nodeaction.NodeOutputs, error) {
			return &nodeaction.NodeOutputs{OutputThing: anyObservation}, nil
		}

		mockSimpleConsensus(t, &consensusValues[int64]{GiveObservation: int64(anyObservation), WantResponse: anyMedian})

		test := func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (int64, error) {
			result, err := sdk.RunInNodeMode(env, rt, func(_ *sdk.NodeEnvironment[string], runtime sdk.NodeRuntime) (int64, error) {
				capability := &nodeaction.BasicAction{}
				value, err := capability.PerformAction(runtime, &nodeaction.NodeInputs{InputThing: true}).Await()
				require.NoError(t, err)
				return int64(value.OutputThing), nil
			}, sdk.ConsensusMedianAggregation[int64]()).Await()
			return result, err
		}

		result, err := testRuntime(t, test)
		require.NoError(t, err)
		assert.Equal(t, anyMedian, result)
	})

	t.Run("Failed consensus", func(t *testing.T) {
		anyError := errors.New("error")

		mockSimpleConsensus(t, &consensusValues[int64]{GiveErr: anyError})

		test := func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (int64, error) {
			return sdk.RunInNodeMode(env, rt, func(_ *sdk.NodeEnvironment[string], _ sdk.NodeRuntime) (int64, error) {
				return int64(0), anyError
			}, sdk.ConsensusMedianAggregation[int64]()).Await()
		}

		_, err := testRuntime(t, test)
		require.ErrorContains(t, err, anyError.Error())
	})

	t.Run("Node runtime in Don mode fails", func(t *testing.T) {
		nodeCapability, err := nodeactionmock.NewBasicActionCapability(t)
		require.NoError(t, err)
		nodeCapability.PerformAction = func(_ context.Context, _ *nodeaction.NodeInputs) (*nodeaction.NodeOutputs, error) {
			assert.Fail(t, "node capability should not be called")
			return nil, fmt.Errorf("should not be called")
		}

		test := func(env *sdk.Environment[string], rt sdk.Runtime, input *basictrigger.Outputs) (*nodeaction.NodeOutputs, error) {
			var nrt sdk.NodeRuntime
			sdk.RunInNodeMode(env, rt, func(_ *sdk.NodeEnvironment[string], nodeRuntime sdk.NodeRuntime) (int32, error) {
				nrt = nodeRuntime
				return 0, err
			}, sdk.ConsensusMedianAggregation[int32]())
			na := nodeaction.BasicAction{}
			return na.PerformAction(nrt, &nodeaction.NodeInputs{InputThing: true}).Await()
		}

		_, err = testRuntime(t, test)
		assert.Equal(t, sdk.NodeModeCallInDonMode(), err)
	})

	t.Run("Don runtime in Node mode fails", func(t *testing.T) {
		capability, err := basicactionmock.NewBasicActionCapability(t)
		require.NoError(t, err)
		capability.PerformAction = func(_ context.Context, _ *basicaction.Inputs) (*basicaction.Outputs, error) {
			assert.Fail(t, "should not be called")
			return nil, errors.New("should not be called")
		}

		test := func(env *sdk.Environment[string], rt sdk.Runtime, input *basictrigger.Outputs) (int32, error) {
			consensus := sdk.RunInNodeMode(env, rt, func(_ *sdk.NodeEnvironment[string], nodeRuntime sdk.NodeRuntime) (int32, error) {
				action := basicaction.BasicAction{}
				_, err := action.PerformAction(rt, &basicaction.Inputs{InputThing: true}).Await()
				return 0, err
			}, sdk.ConsensusMedianAggregation[int32]())

			return consensus.Await()
		}
		_, err = testRuntime(t, test)
		assert.Equal(t, sdk.DonModeCallInNodeMode(), err)
	})
}

func TestNewEnvironment_ReturnsConfig(t *testing.T) {
	runtime := testutils.NewRuntime(t, map[string]string{})
	env := testutils.NewEnvironment(anyEnvConfig, runtime)
	assert.Equal(t, anyEnvConfig, env.Config)
}

func TestRuntime_GenerateReport(t *testing.T) {
	var (
		encodedPayload = []byte(`{"price": 42}`)
		anyMedian      = []byte(`{"price": 43}`)
		encoderName    = "some-encoder"
		signingAlgo    = "some-signer"
		hashingAlgo    = "some-hasher"
	)

	mockReportConsensus(t,
		&consensusValues[[]byte]{
			WantResponse: anyMedian,
		},
	)

	testFn := func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (*pb.ReportResponse, error) {
		return rt.GenerateReport(&pb.ReportRequest{
			EncodedPayload: encodedPayload,
			EncoderName:    encoderName,
			SigningAlgo:    signingAlgo,
			HashingAlgo:    hashingAlgo,
		}).Await()
	}

	output, err := testRuntime(t, testFn)
	require.NoError(t, err)

	result, ok := output.(*pb.ReportResponse)
	require.True(t, ok)

	expectedMap := &valuespb.Map{
		Fields: map[string]*valuespb.Value{
			sdk.ConsensusResponseMapKeyMetadata: {Value: &valuespb.Value_StringValue{StringValue: "test_metadata"}},
			sdk.ConsensusResponseMapKeyPayload:  {Value: &valuespb.Value_BytesValue{BytesValue: anyMedian}},
		},
	}

	var gotMap valuespb.Map
	require.NoError(t, proto.Unmarshal(result.RawReport, &gotMap))
	gotFields := gotMap.GetFields()

	require.Equal(t, 2, len(gotFields))
	require.True(t, proto.Equal(gotFields[sdk.ConsensusResponseMapKeyMetadata], expectedMap.GetFields()[sdk.ConsensusResponseMapKeyMetadata]), "metadata mismatch on report")
	require.True(t, proto.Equal(gotFields[sdk.ConsensusResponseMapKeyPayload], expectedMap.GetFields()[sdk.ConsensusResponseMapKeyPayload]), "payload mismatch on report")
}

func testRuntime[T any](t *testing.T, testFn func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (T, error)) (any, error) {
	runtime := testutils.NewRuntime(t, map[string]string{})
	env := testutils.NewEnvironment(anyEnvConfig, runtime)
	return testFn(env, runtime, anyTrigger)
}

type consensusValues[T any] struct {
	GiveObservation T
	GiveErr         error
	WantResponse    T
}

func mockSimpleConsensus[T any](t *testing.T, values *consensusValues[T]) {
	consensus, err := consensusmock.NewConsensusCapability(t)
	require.NoError(t, err)

	consensus.Simple = func(ctx context.Context, input *pb.SimpleConsensusInputs) (*valuespb.Value, error) {
		return handleSimpleConsensusRequest(t, values, input)
	}
}

// handleSimpleConsensusRequest is a private helper to process the gRPC request
// It extracts and validates inputs, and constructs the response based on generic types.
func handleSimpleConsensusRequest[T any](
	t *testing.T,
	values *consensusValues[T],
	input *pb.SimpleConsensusInputs,
) (*valuespb.Value, error) {
	// 1. Initial Validation: Default input value
	assert.Nil(t, input.Default.Value, "Default input value should be nil") // Added custom message

	// 2. Validate Descriptor Type
	switch d := input.Descriptors.Descriptor_.(type) {
	case *pb.ConsensusDescriptor_Aggregation:
		assert.Equal(t, pb.AggregationType_AGGREGATION_TYPE_MEDIAN, d.Aggregation, "Descriptor aggregation type mismatch") // Added custom message
	default:
		assert.Fail(t, "unexpected descriptor type: %T", d)
		return nil, errors.New("unsupported descriptor type") // Return early on fail
	}

	// 3. Handle Observation Type
	switch o := input.Observation.(type) {
	case *pb.SimpleConsensusInputs_Value:
		// Handle value observation
		return handleSimpleConsensusValueObservation(t, values, o.Value)
	case *pb.SimpleConsensusInputs_Error:
		// Handle error observation
		assert.Equal(t, values.GiveErr.Error(), o.Error, "Error observation message mismatch")
		return nil, values.GiveErr
	default:
		// Unexpected top-level observation type
		require.Fail(t, fmt.Sprintf("unexpected observation type: %T", o))
		return nil, errors.New("unsupported observation type")
	}
}

// handleSimpleConsensusValueObservation processes the value observation part of the input.
func handleSimpleConsensusValueObservation[T any](
	t *testing.T,
	values *consensusValues[T],
	obsValue *valuespb.Value, // The actual *valuespb.Value from the observation
) (*valuespb.Value, error) {
	assert.Nil(t, values.GiveErr, "Expected no error from consensusValues, but GiveErr is not nil")

	// Determine the type of the observed value
	switch v := obsValue.Value.(type) {
	case *valuespb.Value_Int64Value:
		// Validate observed int64 value
		assert.Equal(t, values.GiveObservation, v.Int64Value, "Observed Int64Value mismatch")

		// Determine and return the response based on the generic T type
		return buildConsensusResponseValue(t, values.WantResponse)
	// Add other protobuf value types here if your T can match them (e.g., BytesValue)
	default:
		assert.Fail(t, "unexpected observation value type: %T", v)
		return nil, errors.New("unsupported observation value type")
	}
}

// buildConsensusResponseValue constructs the *valuespb.Value for the response based on the generic T.
func buildConsensusResponseValue[T any](t *testing.T, responseVal T) (*valuespb.Value, error) {
	// Use type switch to determine the actual concrete type of T
	switch resp := any(responseVal).(type) {
	case int64:
		// If T is int64, construct an Int64Value protobuf
		return &valuespb.Value{
			Value: &valuespb.Value_Int64Value{
				Int64Value: resp,
			},
		}, nil
	// Add other concrete types for T that can be returned as a response
	default:
		assert.Fail(t, "unexpected response generic type %T, not handled for protobuf conversion", responseVal)
		return nil, fmt.Errorf("unsupported generic type %T for protobuf response", responseVal)
	}
}

// mockReportConsensus overrides the Report method on consensus.
// It creates a fake RawReport from the WantResponse based on the mock's logic.
func mockReportConsensus[T any](t *testing.T, values *consensusValues[T]) {
	consensus, err := consensusmock.NewConsensusCapability(t)
	require.NoError(t, err, "Failed to create consensus mock capability")

	// Assign the mock implementation to the Report method
	consensus.Report = func(ctx context.Context, input *pb.ReportRequest) (*pb.ReportResponse, error) {
		// Handle the error case first (early exit)
		if values.GiveErr != nil {
			return nil, values.GiveErr
		}

		// If no error is expected, build the raw report
		rawValue := buildRawReportFromResponse(t, values.WantResponse)

		// Construct and return the successful response
		return &pb.ReportResponse{
			RawReport: rawValue,
		}, nil
	}
}

// buildRawReportFromResponse creates the serialized RawReport bytes from the generic response.
// It uses require.Fail to stop the test immediately if the generic type is unexpected.
func buildRawReportFromResponse[T any](t *testing.T, response T) []byte {
	// Cast T to any to use type switch for runtime type checking
	switch resp := any(response).(type) {
	case []byte:
		mapProto := &valuespb.Map{
			Fields: map[string]*valuespb.Value{
				sdk.ConsensusResponseMapKeyMetadata: {Value: &valuespb.Value_StringValue{StringValue: "test_metadata"}},
				sdk.ConsensusResponseMapKeyPayload:  {Value: &valuespb.Value_BytesValue{BytesValue: resp}},
			},
		}
		rawValue, err := proto.Marshal(mapProto)
		require.NoError(t, err, "failed to marshal mapProto to RawReport bytes in mock")
		return rawValue
	default:
		// If the generic type T is not []byte, this is an unexpected scenario for this mock.
		require.Fail(t, fmt.Sprintf("unsupported generic type for RawReport: %T, expected []byte", resp))
		return nil
	}
}

type awaitOverride struct {
	sdkimpl.RuntimeHelpers
	await func(request *pb.AwaitCapabilitiesRequest, maxResponseSize uint64) (*pb.AwaitCapabilitiesResponse, error)
}

func (a *awaitOverride) Await(request *pb.AwaitCapabilitiesRequest, maxResponseSize uint64) (*pb.AwaitCapabilitiesResponse, error) {
	return a.await(request, maxResponseSize)
}
