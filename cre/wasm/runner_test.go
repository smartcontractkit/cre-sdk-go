package wasm

import (
	"encoding/base64"
	"errors"
	"log/slog"
	"testing"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/chainlink-protos/cre/go/values"
	testworkflow "github.com/smartcontractkit/cre-sdk-go/internal/test_workflow"
	"github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/basicaction"
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

	anyPreHookRequest = &sdk.ExecuteRequest{
		Config:          anyConfig,
		MaxResponseSize: anyMaxResponseSize,
		Request: &sdk.ExecuteRequest_PreHook{
			PreHook: &sdk.Trigger{
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

func TestHandlerInTee(t *testing.T) {
	t.Run("specified list sets requirements on subscription", func(t *testing.T) {
		acceptedTees := []cre.TeeAndRegions{{Type: cre.TeeType_TEE_TYPE_AWS_NITRO, Regions: []string{"us-west-2"}}}

		internals := testRunnerInternals(t, subscribeRequest)
		dr := newRunner(func(b []byte) (string, error) { return string(b), nil }, internals, testRuntimeInternals(t))

		dr.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.HandlerInTee(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(_ string, _ cre.TeeRuntime, _ *basictrigger.Outputs) (string, error) {
						return "tee-result", nil
					},
					acceptedTees,
				),
			}, nil
		})

		actual := &sdk.ExecutionResult{}
		require.NoError(t, proto.Unmarshal(internals.sentResponse, actual))
		switch result := actual.Result.(type) {
		case *sdk.ExecutionResult_TriggerSubscriptions:
			subs := result.TriggerSubscriptions.Subscriptions
			require.Len(t, subs, 1)
			expected := &sdk.Requirements{
				Tee: &sdk.Tee{Item: &sdk.Tee_TeeTypesAndRegions{TeeTypesAndRegions: &sdk.TeeTypesAndRegions{TeeTypeAndRegions: []*sdk.TeeTypeAndRegions{{Type: sdk.TeeType_TEE_TYPE_AWS_NITRO, Regions: []string{"us-west-2"}}}}}},
			}
			assert.True(t, proto.Equal(expected, subs[0].Requirements))
		default:
			assert.Fail(t, "unexpected result type", result)
		}
	})

	t.Run("any tee sets requirements on subscription", func(t *testing.T) {
		internals := testRunnerInternals(t, subscribeRequest)
		dr := newRunner(func(b []byte) (string, error) { return string(b), nil }, internals, testRuntimeInternals(t))

		dr.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.HandlerInTee(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(_ string, _ cre.TeeRuntime, _ *basictrigger.Outputs) (string, error) {
						return "tee-result", nil
					},
					cre.AnyTee{Regions: []string{"us-west-2"}},
				),
			}, nil
		})

		actual := &sdk.ExecutionResult{}
		require.NoError(t, proto.Unmarshal(internals.sentResponse, actual))
		switch result := actual.Result.(type) {
		case *sdk.ExecutionResult_TriggerSubscriptions:
			subs := result.TriggerSubscriptions.Subscriptions
			require.Len(t, subs, 1)
			expected := &sdk.Requirements{
				Tee: &sdk.Tee{Item: &sdk.Tee_AnyRegions{AnyRegions: &sdk.Regions{Regions: []string{"us-west-2"}}}},
			}
			assert.True(t, proto.Equal(expected, subs[0].Requirements))
		default:
			assert.Fail(t, "unexpected result type", result)
		}
	})

	t.Run("regular handler has no requirements on subscription", func(t *testing.T) {
		internals := testRunnerInternals(t, subscribeRequest)
		dr := newRunner(func(b []byte) (string, error) { return string(b), nil }, internals, testRuntimeInternals(t))

		dr.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.Handler(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(_ string, _ cre.Runtime, _ *basictrigger.Outputs) (string, error) {
						return "no-tee", nil
					},
				),
			}, nil
		})

		actual := &sdk.ExecutionResult{}
		require.NoError(t, proto.Unmarshal(internals.sentResponse, actual))
		switch result := actual.Result.(type) {
		case *sdk.ExecutionResult_TriggerSubscriptions:
			subs := result.TriggerSubscriptions.Subscriptions
			require.Len(t, subs, 1)
			assert.Nil(t, subs[0].Requirements)
		default:
			assert.Fail(t, "unexpected result type", result)
		}
	})

	t.Run("tee handler callback receives TeeRuntime", func(t *testing.T) {
		acceptedTees := []cre.TeeAndRegions{{Type: cre.TeeType_TEE_TYPE_AWS_NITRO, Regions: []string{"us-west-2"}}}

		triggerReq := &sdk.ExecuteRequest{
			Config:          anyConfig,
			MaxResponseSize: anyMaxResponseSize,
			Request: &sdk.ExecuteRequest_Trigger{
				Trigger: &sdk.Trigger{
					Id:      0,
					Payload: mustAny(testworkflow.TestWorkflowTrigger()),
				},
			},
		}

		internals := testRunnerInternals(t, triggerReq)
		dr := newRunner(func(b []byte) (string, error) { return string(b), nil }, internals, testRuntimeInternals(t))

		callbackInvoked := false
		dr.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.HandlerInTee(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(_ string, rt cre.TeeRuntime, _ *basictrigger.Outputs) (string, error) {
						callbackInvoked = true
						assert.NotNil(t, rt, "TeeRuntime should not be nil")
						return "done", nil
					},
					acceptedTees,
				),
			}, nil
		})

		assert.True(t, callbackInvoked, "tee callback should have been invoked")
	})
}

func TestHandlerWithPreHook(t *testing.T) {
	basicActionRestrictor := &basicaction.BasicActionRestrictor{}

	t.Run("subscribe sets PreHook on subscription", func(t *testing.T) {
		dr := getTestRunner(t, subscribeRequest)
		dr.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.HandlerWithPreHook(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(string, cre.Runtime, *basictrigger.Outputs) (int, error) {
						return 0, nil
					},
					func(string, *basictrigger.Outputs) (*sdk.Restrictions, error) {
						return nil, nil
					},
				),
			}, nil
		})

		actual := &sdk.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*subscriber[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		result, ok := actual.Result.(*sdk.ExecutionResult_TriggerSubscriptions)
		require.True(t, ok, "expected TriggerSubscriptions result")
		require.Len(t, result.TriggerSubscriptions.Subscriptions, 1)
		assert.True(t, result.TriggerSubscriptions.Subscriptions[0].PreHook)
	})

	t.Run("returns restrictions from preHook", func(t *testing.T) {
		dr := getTestRunner(t, anyPreHookRequest)
		dr.Run(func(config string, _ *slog.Logger, _ cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.HandlerWithPreHook(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(string, cre.Runtime, *basictrigger.Outputs) (int, error) {
						return 0, nil
					},
					func(c string, payload *basictrigger.Outputs) (*sdk.Restrictions, error) {
						assert.Equal(t, string(anyConfig), c)
						assert.Equal(t, "Hi", payload.CoolOutput)
						return &sdk.Restrictions{
							Capabilities: &sdk.CapabilityRestrictions{
								Restrictions:  []*sdk.CapabilityRestriction{basicActionRestrictor.LimitPerformAction(2)},
								MaxTotalCalls: 5,
								Type:          sdk.CapabilityRestrictionType_CAPABILITY_RESTRICTION_TYPE_CLOSED,
							},
						}, nil
					},
				),
			}, nil
		})

		actual := &sdk.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*preHookRunner[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		result, ok := actual.Result.(*sdk.ExecutionResult_Restrictions)
		require.True(t, ok, "expected Restrictions result, got %T", actual.Result)
		require.NotNil(t, result.Restrictions.Capabilities)
		assert.Equal(t, int32(5), result.Restrictions.Capabilities.MaxTotalCalls)
		assert.Equal(t, sdk.CapabilityRestrictionType_CAPABILITY_RESTRICTION_TYPE_CLOSED, result.Restrictions.Capabilities.Type)
		require.Len(t, result.Restrictions.Capabilities.Restrictions, 1)

		method := result.Restrictions.Capabilities.Restrictions[0].GetMethod()
		require.NotNil(t, method)
		assert.Equal(t, "basic-test-action@1.0.0", method.Id)
		assert.Equal(t, "PerformAction", method.Method)
		assert.Equal(t, int32(2), method.MaxCalls)
	})

	t.Run("returns error when no preHook is registered", func(t *testing.T) {
		dr := getTestRunner(t, anyPreHookRequest)
		dr.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.Handler(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(string, cre.Runtime, *basictrigger.Outputs) (int, error) { return 0, nil },
				),
			}, nil
		})

		actual := &sdk.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*preHookRunner[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		errResult, ok := actual.Result.(*sdk.ExecutionResult_Error)
		require.True(t, ok, "expected error result, got %T", actual.Result)
		assert.Contains(t, errResult.Error, "no preHook registered")
	})

	t.Run("returns error for out of bounds trigger id", func(t *testing.T) {
		outOfBounds := &sdk.ExecuteRequest{
			Config:          anyConfig,
			MaxResponseSize: anyMaxResponseSize,
			Request: &sdk.ExecuteRequest_PreHook{
				PreHook: &sdk.Trigger{
					Id:      99,
					Payload: mustAny(testworkflow.TestWorkflowTrigger()),
				},
			},
		}
		dr := getTestRunner(t, outOfBounds)
		dr.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.HandlerWithPreHook(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(string, cre.Runtime, *basictrigger.Outputs) (int, error) { return 0, nil },
					func(string, *basictrigger.Outputs) (*sdk.Restrictions, error) { return nil, nil },
				),
			}, nil
		})

		actual := &sdk.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*preHookRunner[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		errResult, ok := actual.Result.(*sdk.ExecutionResult_Error)
		require.True(t, ok, "expected error result, got %T", actual.Result)
		assert.Contains(t, errResult.Error, "trigger not found")
	})

	t.Run("preHook error is reported", func(t *testing.T) {
		dr := getTestRunner(t, anyPreHookRequest)
		dr.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.HandlerWithPreHook(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(string, cre.Runtime, *basictrigger.Outputs) (int, error) { return 0, nil },
					func(string, *basictrigger.Outputs) (*sdk.Restrictions, error) {
						return nil, errors.New("boom")
					},
				),
			}, nil
		})

		actual := &sdk.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*preHookRunner[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		errResult, ok := actual.Result.(*sdk.ExecutionResult_Error)
		require.True(t, ok, "expected error result, got %T", actual.Result)
		assert.Equal(t, "boom", errResult.Error)
	})

	t.Run("HandlerInTeeWithPreHook sets both Requirements and PreHook on subscription", func(t *testing.T) {
		acceptedTees := []cre.TeeAndRegions{{Type: cre.TeeType_TEE_TYPE_AWS_NITRO, Regions: []string{"us-west-2"}}}
		dr := getTestRunner(t, subscribeRequest)
		dr.Run(func(string, *slog.Logger, cre.SecretsProvider) (cre.Workflow[string], error) {
			return cre.Workflow[string]{
				cre.HandlerInTeeWithPreHook(
					basictrigger.Trigger(testworkflow.TestWorkflowTriggerConfig()),
					func(string, cre.TeeRuntime, *basictrigger.Outputs) (string, error) { return "", nil },
					acceptedTees,
					func(string, *basictrigger.Outputs) (*sdk.Restrictions, error) { return nil, nil },
				),
			}, nil
		})

		actual := &sdk.ExecutionResult{}
		sentResponse := dr.(runnerWrapper[string]).baseRunner.(*subscriber[string, cre.Runtime]).runnerInternals.(*runnerInternalsTestHook).sentResponse
		require.NoError(t, proto.Unmarshal(sentResponse, actual))

		subs := actual.Result.(*sdk.ExecutionResult_TriggerSubscriptions).TriggerSubscriptions.Subscriptions
		require.Len(t, subs, 1)
		assert.True(t, subs[0].PreHook)
		require.NotNil(t, subs[0].Requirements)
		require.NotNil(t, subs[0].Requirements.Tee)
		typesAndRegions := subs[0].Requirements.Tee.GetTeeTypesAndRegions()
		require.NotNil(t, typesAndRegions)
		require.Len(t, typesAndRegions.TeeTypeAndRegions, 1)
		assert.Equal(t, sdk.TeeType_TEE_TYPE_AWS_NITRO, typesAndRegions.TeeTypeAndRegions[0].Type)
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
