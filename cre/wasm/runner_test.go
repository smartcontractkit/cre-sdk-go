package wasm

import (
	"encoding/base64"
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/values"
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
	"github.com/smartcontractkit/cre-sdk-go/cre/testutils"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basictrigger"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/cre-sdk-go/cre"
)

var (
	anyConfig          = []byte("config")
	anyMaxResponseSize = uint64(2048)

	defaultBasicTrigger = basictrigger.Trigger(&basictrigger.Config{})
	triggerIndex        = int(0)
	capID               = defaultBasicTrigger.CapabilityID()

	subscribeRequest = &pb.ExecuteRequest{
		Config:          anyConfig,
		MaxResponseSize: anyMaxResponseSize,
		Request:         &pb.ExecuteRequest_Subscribe{Subscribe: &emptypb.Empty{}},
	}

	anyExecuteRequest = &pb.ExecuteRequest{
		Config:          anyConfig,
		MaxResponseSize: anyMaxResponseSize,
		Request: &pb.ExecuteRequest_Trigger{
			Trigger: &pb.Trigger{
				Id:      uint64(triggerIndex),
				Payload: mustAny(testutils.TestWorkflowTrigger()),
			},
		},
	}
)

func TestRunner_CreateWorkflows(t *testing.T) {
	assertEnv(t, getTestRunner(t, anyExecuteRequest))
	assertEnv(t, getTestRunner(t, subscribeRequest))
}

func TestRunner_GetSecrets_PassesMaxResponseSize(t *testing.T) {
	dr := getTestRunner(t, subscribeRequest)
	dr.Run(func(env *cre.Environment[string]) (cre.Workflow[string], error) {
		_, err := env.GetSecret(&pb.SecretRequest{Namespace: "Foo", Id: "Bar"}).Await()
		// This will fail with "buffer cannot be empty" if we fail to pass the maxResponseSize from the
		// runner to the runtime.
		assert.ErrorContains(t, err, "secret Foo.Bar not found")

		return cre.Workflow[string]{
			cre.Handler(
				basictrigger.Trigger(testutils.TestWorkflowTriggerConfig()),
				func(env *cre.Environment[string], _ cre.Runtime, _ *basictrigger.Outputs) (int, error) {
					return 0, nil
				}),
		}, nil
	})
}

func TestRunner_Run(t *testing.T) {
	t.Run("runner gathers subscriptions", func(t *testing.T) {
		dr := getTestRunner(t, subscribeRequest)
		dr.Run(func(_ *cre.Environment[string]) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.Handler(
					basictrigger.Trigger(testutils.TestWorkflowTriggerConfig()),
					func(_ *cre.Environment[string], _ cre.Runtime, _ *basictrigger.Outputs) (int, error) {
						require.Fail(t, "Must not be called during registration to tiggers")
						return 0, nil
					}),
			}, nil
		})

		actual := &pb.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*subscriber[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		switch result := actual.Result.(type) {
		case *pb.ExecutionResult_TriggerSubscriptions:
			subscriptions := result.TriggerSubscriptions.Subscriptions
			require.Len(t, subscriptions, 1)
			subscription := subscriptions[triggerIndex]
			assert.Equal(t, capID, subscription.Id)
			assert.Equal(t, "Trigger", subscription.Method)
			payload := &basictrigger.Config{}
			require.NoError(t, subscription.Payload.UnmarshalTo(payload))
			assert.True(t, proto.Equal(testutils.TestWorkflowTriggerConfig(), payload))
		default:
			assert.Fail(t, "unexpected result type", result)
		}
	})

	t.Run("makes callback with correct runner", func(t *testing.T) {
		testutils.SetupExpectedCalls(t)
		dr := getTestRunner(t, anyExecuteRequest)
		testutils.RunTestWorkflow(dr)

		actual := &pb.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*runner[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		switch result := actual.Result.(type) {
		case *pb.ExecutionResult_Value:
			v, err := values.FromProto(result.Value)
			require.NoError(t, err)
			returnedValue, err := v.Unwrap()
			require.NoError(t, err)
			assert.Equal(t, testutils.TestWorkflowExpectedResult(), returnedValue)
		default:
			assert.Fail(t, "unexpected result type", result)
		}
	})

	t.Run("makes callback with correct runner and multiple handlers", func(t *testing.T) {
		secondTriggerReq := &pb.ExecuteRequest{
			Config:          anyConfig,
			MaxResponseSize: anyMaxResponseSize,
			Request: &pb.ExecuteRequest_Trigger{
				Trigger: &pb.Trigger{
					Id:      uint64(triggerIndex + 1),
					Payload: mustAny(testutils.TestWorkflowTrigger()),
				},
			},
		}
		testutils.SetupExpectedCalls(t)
		dr := getTestRunner(t, secondTriggerReq)
		testutils.RunIdenticalTriggersWorkflow(dr)

		actual := &pb.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*runner[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		switch result := actual.Result.(type) {
		case *pb.ExecutionResult_Value:
			v, err := values.FromProto(result.Value)
			require.NoError(t, err)
			returnedValue, err := v.Unwrap()
			require.NoError(t, err)
			assert.Equal(t, testutils.TestWorkflowExpectedResult()+"true", returnedValue)
		default:
			assert.Fail(t, "unexpected result type", result)
		}
	})
}

func assertEnv(t *testing.T, r cre.Runner[string]) {
	ran := false
	verifyEnv := func(env *cre.Environment[string]) (cre.Workflow[string], error) {
		ran = true
		assert.Equal(t, string(anyConfig), env.Config)
		assert.IsType(t, &writer{}, env.LogWriter)
		return cre.Workflow[string]{}, nil

	}
	r.Run(verifyEnv)
	assert.True(t, ran, "Workflow should have been run")
}

func getTestRunner(tb testing.TB, request *pb.ExecuteRequest) cre.Runner[string] {
	return newRunner(func(b []byte) (string, error) { return string(b), nil }, testRunnerInternals(tb, request), testRuntimeInternals(tb))
}

func testRunnerInternals(tb testing.TB, request *pb.ExecuteRequest) *runnerInternalsTestHook {
	serialzied, err := proto.Marshal(request)
	require.NoError(tb, err)
	encoded := base64.StdEncoding.EncodeToString(serialzied)

	return &runnerInternalsTestHook{
		testTb:    tb,
		arguments: []string{"wasm", encoded},
	}
}

func testRuntimeInternals(tb testing.TB) *runtimeInternalsTestHook {
	return &runtimeInternalsTestHook{
		testTb:                  tb,
		outstandingCalls:        map[int32]cre.Promise[*pb.CapabilityResponse]{},
		outstandingSecretsCalls: map[int32]cre.Promise[[]*pb.SecretResponse]{},
		secrets:                 map[string]*pb.Secret{},
	}
}

func mustAny(msg proto.Message) *anypb.Any {
	a, err := anypb.New(msg)
	if err != nil {
		panic(err)
	}
	return a
}
