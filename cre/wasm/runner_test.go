package wasm

import (
	"encoding/base64"
	"log/slog"
	"testing"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/chainlink-protos/cre/go/values"
	testworkflow "github.com/smartcontractkit/cre-sdk-go/internal/test_workflow"
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
	triggerIndex        = 0
	capID               = defaultBasicTrigger.CapabilityID()

	subscribeRequest = &sdk.ExecuteRequest{
		Config:          anyConfig,
		MaxResponseSize: anyMaxResponseSize,
		Request:         &sdk.ExecuteRequest_Subscribe{Subscribe: &emptypb.Empty{}},
	}

	anyExecuteRequest = &sdk.ExecuteRequest{
		Config:          anyConfig,
		MaxResponseSize: anyMaxResponseSize,
		Request: &sdk.ExecuteRequest_Trigger{
			Trigger: &sdk.Trigger{
				Id:      uint64(triggerIndex),
				Payload: mustAny(testworkflow.TestWorkflowTrigger()),
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
	dr.Run(func(_ string, _ *slog.Logger, secretsProvider cre.SecretsProvider) (cre.Workflow[string], error) {
		_, err := secretsProvider.GetSecret(&sdk.SecretRequest{Namespace: "Foo", Id: "Bar"}).Await()
		// This will fail with "buffer cannot be empty" if we fail to pass the maxResponseSize from the
		// runner to the runtime.
		assert.ErrorContains(t, err, "secret Foo.Bar not found")

		return cre.Workflow[string]{
			cre.Handler(
				basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
				func(string, cre.Runtime, *basictrigger.Outputs) (int, error) {
					return 0, nil
				}),
		}, nil
	})
}

func TestRunner_Run(t *testing.T) {
	t.Run("runner gathers subscriptions", func(t *testing.T) {
		dr := getTestRunner(t, subscribeRequest)
		dr.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.Handler(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(string, cre.Runtime, *basictrigger.Outputs) (int, error) {
						require.Fail(t, "Must not be called during registration to tiggers")
						return 0, nil
					}),
			}, nil
		})

		actual := &sdk.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*subscriber[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		switch result := actual.Result.(type) {
		case *sdk.ExecutionResult_TriggerSubscriptions:
			subscriptions := result.TriggerSubscriptions.Subscriptions
			require.Len(t, subscriptions, 1)
			subscription := subscriptions[triggerIndex]
			assert.Equal(t, capID, subscription.Id)
			assert.Equal(t, "Trigger", subscription.Method)
			payload := &basictrigger.Config{}
			require.NoError(t, subscription.Payload.UnmarshalTo(payload))
			assert.True(t, proto.Equal(testworkflow.TestWorkflowTriggerConfig(), payload))
		default:
			assert.Fail(t, "unexpected result type", result)
		}
	})

	t.Run("makes callback with correct runner", func(t *testing.T) {
		testworkflow.SetupExpectedCalls(t)
		dr := getTestRunner(t, anyExecuteRequest)
		testworkflow.RunTestWorkflow(dr)

		actual := &sdk.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*runner[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		switch result := actual.Result.(type) {
		case *sdk.ExecutionResult_Value:
			v, err := values.FromProto(result.Value)
			require.NoError(t, err)
			returnedValue, err := v.Unwrap()
			require.NoError(t, err)
			assert.Equal(t, testworkflow.TestWorkflowExpectedResult(), returnedValue)
		default:
			assert.Fail(t, "unexpected result type", result)
		}
	})

	t.Run("makes callback with correct runner and multiple handlers", func(t *testing.T) {
		secondTriggerReq := &sdk.ExecuteRequest{
			Config:          anyConfig,
			MaxResponseSize: anyMaxResponseSize,
			Request: &sdk.ExecuteRequest_Trigger{
				Trigger: &sdk.Trigger{
					Id:      uint64(triggerIndex + 1),
					Payload: mustAny(testworkflow.TestWorkflowTrigger()),
				},
			},
		}
		testworkflow.SetupExpectedCalls(t)
		dr := getTestRunner(t, secondTriggerReq)
		testworkflow.RunIdenticalTriggersWorkflow(dr)

		actual := &sdk.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*runner[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		switch result := actual.Result.(type) {
		case *sdk.ExecutionResult_Value:
			v, err := values.FromProto(result.Value)
			require.NoError(t, err)
			returnedValue, err := v.Unwrap()
			require.NoError(t, err)
			assert.Equal(t, testworkflow.TestWorkflowExpectedResult()+"true", returnedValue)
		default:
			assert.Fail(t, "unexpected result type", result)
		}
	})
}

func assertEnv(t *testing.T, r cre.Runner[string]) {
	ran := false
	verifyEnv := func(config string, logger *slog.Logger, secretsProvider cre.SecretsProvider) (cre.Workflow[string], error) {
		ran = true
		assert.Equal(t, string(anyConfig), config)
		return cre.Workflow[string]{}, nil

	}
	r.Run(verifyEnv)
	assert.True(t, ran, "Workflow should have been run")
}

func getTestRunner(tb testing.TB, request *sdk.ExecuteRequest) cre.Runner[string] {
	return newRunner(func(b []byte) (string, error) { return string(b), nil }, testRunnerInternals(tb, request), testRuntimeInternals(tb))
}

func testRunnerInternals(tb testing.TB, request *sdk.ExecuteRequest) *runnerInternalsTestHook {
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
		outstandingCalls:        map[int32]cre.Promise[*sdk.CapabilityResponse]{},
		outstandingSecretsCalls: map[int32]cre.Promise[[]*sdk.SecretResponse]{},
		secrets:                 map[string]*sdk.Secret{},
	}
}

func mustAny(msg proto.Message) *anypb.Any {
	a, err := anypb.New(msg)
	if err != nil {
		panic(err)
	}
	return a
}
