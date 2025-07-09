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
	basictriggermock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger/mock"
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

		ran, result, err := testRuntime(t, test)
		require.NoError(t, err)
		assert.True(t, ran)
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
		_, _, err := testRuntime(t, test)
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

		_, _, err = testRuntime(t, test)
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
			drt := rt.(*sdkimpl.Runtime)
			drt.RuntimeHelpers = &awaitOverride{
				RuntimeHelpers: drt.RuntimeHelpers,
				await: func(request *pb.AwaitCapabilitiesRequest, maxResponseSize uint64) (*pb.AwaitCapabilitiesResponse, error) {
					return nil, expectedErr
				},
			}
			_, err := capability.PerformAction(rt, &basicaction.Inputs{InputThing: true}).Await()
			return "", err
		}

		_, _, err = testRuntime(t, test)
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
			drt := rt.(*sdkimpl.Runtime)
			drt.RuntimeHelpers = &awaitOverride{
				RuntimeHelpers: drt.RuntimeHelpers,
				await: func(request *pb.AwaitCapabilitiesRequest, maxResponseSize uint64) (*pb.AwaitCapabilitiesResponse, error) {
					return &pb.AwaitCapabilitiesResponse{Responses: map[int32]*pb.CapabilityResponse{}}, nil
				},
			}
			_, err := capability.PerformAction(rt, &basicaction.Inputs{InputThing: true}).Await()
			return "", err
		}

		_, _, err = testRuntime(t, test)
		assert.Error(t, err)
	})
}

func TestRuntime_Rand(t *testing.T) {
	t.Run("random delegates", func(t *testing.T) {
		test := func(_ *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (uint64, error) {
			r, err := rt.Rand()
			if err != nil {
				return 0, err
			}
			return r.Uint64(), nil
		}

		ran, result, err := testRuntime(t, test)
		require.NoError(t, err)
		assert.True(t, ran)
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

		_, _, err := testRuntime(t, test)
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

			_, _, _ = testRuntime(t, test)
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

		setupSimpleConsensus(t, &consensusValues[int64]{GiveObservation: int64(anyObservation), WantResponse: anyMedian})

		test := func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (int64, error) {
			result, err := sdk.RunInNodeMode(env, rt, func(_ *sdk.NodeEnvironment[string], runtime sdk.NodeRuntime) (int64, error) {
				capability := &nodeaction.BasicAction{}
				value, err := capability.PerformAction(runtime, &nodeaction.NodeInputs{InputThing: true}).Await()
				require.NoError(t, err)
				return int64(value.OutputThing), nil
			}, sdk.ConsensusMedianAggregation[int64]()).Await()
			return result, err
		}

		ran, result, err := testRuntime(t, test)
		require.NoError(t, err)
		assert.True(t, ran)
		assert.Equal(t, anyMedian, result)
	})

	t.Run("Failed consensus", func(t *testing.T) {
		anyError := errors.New("error")

		setupSimpleConsensus(t, &consensusValues[int64]{GiveErr: anyError})

		test := func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (int64, error) {
			return sdk.RunInNodeMode(env, rt, func(_ *sdk.NodeEnvironment[string], _ sdk.NodeRuntime) (int64, error) {
				return int64(0), anyError
			}, sdk.ConsensusMedianAggregation[int64]()).Await()
		}

		_, _, err := testRuntime(t, test)
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

		_, _, err = testRuntime(t, test)
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
		_, _, err = testRuntime(t, test)
		assert.Equal(t, sdk.DonModeCallInNodeMode(), err)
	})
}

func TestRuntime_ReturnsConfig(t *testing.T) {
	trigger, err := basictriggermock.NewBasicCapability(t)
	require.NoError(t, err)
	trigger.Trigger = func(_ context.Context, config *basictrigger.Config) (*basictrigger.Outputs, error) {
		return &basictrigger.Outputs{CoolOutput: "cool"}, nil
	}

	anyConfig := "config"
	runner := testutils.NewRunner(t, anyConfig)

	runner.Run(func(env *sdk.Environment[string]) (sdk.Workflow[string], error) {
		return sdk.Workflow[string]{
			sdk.Handler(
				basictrigger.Trigger(&basictrigger.Config{Name: "name", Number: 123}),
				func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (string, error) {
					return env.Config, nil
				}),
		}, nil
	})

	ran, result, err := runner.Result()
	require.NoError(t, err)
	assert.True(t, ran)
	assert.Equal(t, anyConfig, result)
}

func TestRuntime_GenerateReport(t *testing.T) {
	var (
		encodedPayload = []byte(`{"price": 42}`)
		anyMedian      = []byte(`{"price": 43}`)
		encoderName    = "some-encoder"
		signingAlgo    = "some-signer"
		hashingAlgo    = "some-hasher"
	)

	setupReportConsensus(t,
		&consensusValues[[]byte]{
			WantResponse: anyMedian,
		},
	)

	testFn := func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (*pb.ReportResponse, error) {
		return env.GenerateReport(&pb.ReportRequest{
			EncodedPayload: encodedPayload,
			EncoderName:    encoderName,
			SigningAlgo:    signingAlgo,
			HashingAlgo:    hashingAlgo,
		}).Await()
	}

	ran, output, err := testRuntime(t, testFn)
	assert.True(t, ran)
	require.NoError(t, err)

	result, ok := any(output).(*pb.ReportResponse)
	assert.True(t, ok)

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

func testRuntime[T any](t *testing.T, testFn func(env *sdk.Environment[string], rt sdk.Runtime, _ *basictrigger.Outputs) (T, error)) (bool, any, error) {
	trigger, err := basictriggermock.NewBasicCapability(t)
	require.NoError(t, err)
	trigger.Trigger = func(_ context.Context, config *basictrigger.Config) (*basictrigger.Outputs, error) {
		assert.True(t, proto.Equal(anyConfig, config))
		return anyTrigger, nil
	}

	runner := testutils.NewRunner(t, "unused")
	require.NoError(t, err)

	runner.Run(func(workflowContext *sdk.Environment[string]) (sdk.Workflow[string], error) {
		return sdk.Workflow[string]{sdk.Handler(
			basictrigger.Trigger(anyConfig), testFn,
		)}, nil
	})

	return runner.Result()
}

type consensusValues[T any] struct {
	GiveObservation T
	GiveErr         error
	WantResponse    T
}

func setupSimpleConsensus[T any](t *testing.T, values *consensusValues[T]) {
	consensus, err := consensusmock.NewConsensusCapability(t)
	require.NoError(t, err)

	consensus.Simple = func(ctx context.Context, input *pb.SimpleConsensusInputs) (*valuespb.Value, error) {
		assert.Nil(t, input.Default.Value)

		switch d := input.Descriptors.Descriptor_.(type) {
		case *pb.ConsensusDescriptor_Aggregation:
			assert.Equal(t, pb.AggregationType_AGGREGATION_TYPE_MEDIAN, d.Aggregation)
		default:
			assert.Fail(t, "unexpected descriptor type")
		}

		switch o := input.Observation.(type) {
		case *pb.SimpleConsensusInputs_Value:
			assert.Nil(t, values.GiveErr)
			var (
				rawValue []byte
				err      error
			)
			switch v := o.Value.Value.(type) {
			case *valuespb.Value_Int64Value:
				assert.Equal(t, values.GiveObservation, v.Int64Value)
				switch resp := any(values.WantResponse).(type) {
				case int64:
					mapProto := &valuespb.Map{
						Fields: map[string]*valuespb.Value{
							sdk.ConsensusResponseMapKeyMetadata: {Value: &valuespb.Value_StringValue{StringValue: "test_metadata"}},
							sdk.ConsensusResponseMapKeyPayload:  {Value: &valuespb.Value_Int64Value{Int64Value: resp}},
						},
					}
					rawValue, err = proto.Marshal(mapProto)
					require.NoError(t, err)
				default:
					assert.Fail(t, "unexpected response value type %T, wanted int64", resp)
				}
			case *valuespb.Value_BytesValue:
				assert.Equal(t, values.GiveObservation, v.BytesValue)
				switch resp := any(values.WantResponse).(type) {
				case []byte:
					mapProto := &valuespb.Map{
						Fields: map[string]*valuespb.Value{
							sdk.ConsensusResponseMapKeyMetadata: {Value: &valuespb.Value_StringValue{StringValue: "test_metadata"}},
							sdk.ConsensusResponseMapKeyPayload:  {Value: &valuespb.Value_BytesValue{BytesValue: resp}},
						},
					}
					rawValue, err = proto.Marshal(mapProto)
					require.NoError(t, err)
				default:
					assert.Fail(t, "unexpected response value type %T, wanted []byte", resp)
				}
			default:
				assert.Fail(t, "unexpected observation value type")
			}
			return &valuespb.Value{
				Value: &valuespb.Value_BytesValue{
					BytesValue: rawValue,
				},
			}, nil
		case *pb.SimpleConsensusInputs_Error:
			assert.Equal(t, values.GiveErr.Error(), o.Error)
			return nil, values.GiveErr
		default:
			require.Fail(t, "unexpected observation type")
			return nil, errors.New("should not get here")
		}
	}
}

// setupReportConsensus overrides the Report method on consensus.  Creates a fake
// RawReport from the WantResponse.
func setupReportConsensus[T any](t *testing.T, values *consensusValues[T]) {
	consensus, err := consensusmock.NewConsensusCapability(t)
	require.NoError(t, err)

	consensus.Report = func(ctx context.Context, input *pb.ReportRequest) (*pb.ReportResponse, error) {
		switch resp := any(values.WantResponse).(type) {
		case []byte:
			assert.Nil(t, values.GiveErr)
			mapProto := &valuespb.Map{
				Fields: map[string]*valuespb.Value{
					sdk.ConsensusResponseMapKeyMetadata: {Value: &valuespb.Value_StringValue{StringValue: "test_metadata"}},
					sdk.ConsensusResponseMapKeyPayload:  {Value: &valuespb.Value_BytesValue{BytesValue: resp}},
				},
			}
			rawValue, err := proto.Marshal(mapProto)
			require.NoError(t, err)
			return &pb.ReportResponse{
				RawReport: rawValue,
			}, nil
		default:
			assert.Fail(t, "unexpected response value type %T, wanted []byte", resp)
		}
		return nil, errors.New("unsupported type for consensus report mock")
	}
}

type awaitOverride struct {
	sdkimpl.RuntimeHelpers
	await func(request *pb.AwaitCapabilitiesRequest, maxResponseSize uint64) (*pb.AwaitCapabilitiesResponse, error)
}

func (a *awaitOverride) Await(request *pb.AwaitCapabilitiesRequest, maxResponseSize uint64) (*pb.AwaitCapabilitiesResponse, error) {
	return a.await(request, maxResponseSize)
}
