package wasm

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"unsafe"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/internal/sdkimpl"
	"google.golang.org/protobuf/proto"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/chainlink-protos/cre/go/values"
)

type Config any

type runnerInternals interface {
	args() []string
	sendResponse(response unsafe.Pointer, responseLen int32) int32
	versionV2()
	switchModes(mode int32)
	exit()
}

func newRunner[C Config](parse func(configBytes []byte) (C, error), runnerInternals runnerInternals, runtimeInternals runtimeInternals) cre.Runner[C] {
	runnerInternals.versionV2()
	runnerInternals.switchModes(int32(sdk.Mode_MODE_DON))
	drt := &sdkimpl.Runtime{RuntimeBase: newRuntime(runtimeInternals, sdk.Mode_MODE_DON)}
	return runnerWrapper[C]{
		baseRunner: getRunner(
			parse,
			&subscriber[C, cre.Runtime]{
				sp:              drt,
				runnerInternals: runnerInternals,
				setRuntime: func(maxResponseSize uint64) {
					drt.MaxResponseSize = maxResponseSize
				},
			},
			&runner[C, cre.Runtime]{
				sp:              drt,
				runtime:         drt,
				runnerInternals: runnerInternals,
				setRuntime: func(maxResponseSize uint64) {
					drt.MaxResponseSize = maxResponseSize
				},
			}),
		runnerInternals: runnerInternals,
	}
}

type runner[C, T any] struct {
	runnerInternals
	trigger    *sdk.Trigger
	id         string
	runtime    T
	setRuntime func(maxResponseSize uint64)
	config     C
	sp         cre.SecretsProvider
}

var _ baseRunner[any, cre.Runtime] = (*runner[any, cre.Runtime])(nil)

func (r *runner[C, T]) cfg() C {
	return r.config
}

func (r *runner[C, T]) secretsProvider() cre.SecretsProvider {
	return r.sp
}

func (r *runner[C, T]) run(wfs []cre.ExecutionHandler[C, T]) {
	for idx, handler := range wfs {
		if uint64(idx) == r.trigger.Id {
			response, err := handler.Callback()(r.config, r.runtime, r.trigger.Payload)
			if err == nil {
				wrapped, err := values.Wrap(response)
				if err != nil {
					exit(r.runnerInternals, &sdk.ExecutionResult{Result: &sdk.ExecutionResult_Error{Error: err.Error()}})
				} else {
					exit(r.runnerInternals, &sdk.ExecutionResult{Result: &sdk.ExecutionResult_Value{Value: values.Proto(wrapped)}})
				}
			} else {
				exit(r.runnerInternals, &sdk.ExecutionResult{Result: &sdk.ExecutionResult_Error{Error: err.Error()}})
			}
		}
	}
}

type subscriber[C, T any] struct {
	runnerInternals
	id         string
	config     C
	sp         cre.SecretsProvider
	setRuntime func(maxResponseSize uint64)
}

var _ baseRunner[any, cre.Runtime] = &subscriber[any, cre.Runtime]{}

func (s *subscriber[C, T]) cfg() C {
	return s.config
}

func (s *subscriber[C, T]) secretsProvider() cre.SecretsProvider {
	return s.sp
}

func (s *subscriber[C, T]) run(wfs []cre.ExecutionHandler[C, T]) {
	subscriptions := make([]*sdk.TriggerSubscription, len(wfs))
	for i, handler := range wfs {
		subscriptions[i] = &sdk.TriggerSubscription{
			Id:      handler.CapabilityID(),
			Payload: handler.TriggerCfg(),
			Method:  handler.Method(),
		}
	}
	triggerSubscription := &sdk.TriggerSubscriptionRequest{Subscriptions: subscriptions}

	execResponse := &sdk.ExecutionResult{
		Result: &sdk.ExecutionResult_TriggerSubscriptions{TriggerSubscriptions: triggerSubscription},
	}

	exit(s.runnerInternals, execResponse)
}

func (r runnerWrapper[C]) getWorkflows(config C, secretsProvider cre.SecretsProvider, initFn func(C, *slog.Logger, cre.SecretsProvider) (cre.Workflow[C], error)) cre.Workflow[C] {
	wfs, err := initFn(config, newSlogger(), secretsProvider)
	if err != nil {
		exitErr(r.runnerInternals, err.Error())
	}
	return wfs
}

func getRunner[C, T any](parse func(configBytes []byte) (C, error), subscribe *subscriber[C, T], run *runner[C, T]) baseRunner[C, T] {
	args := run.args()

	// We expect exactly 2 args, i.e. `wasm <blob>`,
	// where <blob> is a base64 encoded protobuf message.
	if len(args) != 2 {
		exitErr(subscribe.runnerInternals, "invalid request: request must contain a payload")
	}

	request := args[1]
	if request == "" {
		exitErr(subscribe.runnerInternals, "invalid request: request cannot be empty")
	}

	b, err := base64.StdEncoding.DecodeString(request)
	if err != nil {
		exitErr(subscribe.runnerInternals, "invalid request: could not decode request into bytes")
	}

	execRequest := &sdk.ExecuteRequest{}
	if err = proto.Unmarshal(b, execRequest); err != nil {
		exitErr(subscribe.runnerInternals, "invalid request: could not unmarshal request into ExecuteRequest")
	}

	c, err := parse(execRequest.Config)
	if err != nil {
		exitErr(subscribe.runnerInternals, err.Error())
	}

	switch req := execRequest.Request.(type) {
	case *sdk.ExecuteRequest_Subscribe:
		subscribe.config = c
		subscribe.setRuntime(execRequest.MaxResponseSize)
		return subscribe
	case *sdk.ExecuteRequest_Trigger:
		run.trigger = req.Trigger
		run.config = c
		run.setRuntime(execRequest.MaxResponseSize)
		return run
	}

	exitErr(subscribe.runnerInternals, fmt.Sprintf("invalid request: unknown request type %T", execRequest.Request))
	return nil
}

func exitErr(r runnerInternals, err string) {
	exit(r, &sdk.ExecutionResult{Result: &sdk.ExecutionResult_Error{Error: err}})
}

func exit(r runnerInternals, result *sdk.ExecutionResult) {
	marshalled, _ := proto.Marshal(result)
	marshalledPtr, marshalledLen, _ := bufferToPointerLen(marshalled)
	r.sendResponse(marshalledPtr, marshalledLen)
	r.exit()
}

type baseRunner[C, T any] interface {
	secretsProvider() cre.SecretsProvider
	cfg() C
	run([]cre.ExecutionHandler[C, T])
}

type runnerWrapper[C any] struct {
	baseRunner[C, cre.Runtime]
	runnerInternals runnerInternals
}

func (r runnerWrapper[C]) Run(initFn func(config C, logger *slog.Logger, secretsProvider cre.SecretsProvider) (cre.Workflow[C], error)) {
	wfs := r.getWorkflows(r.baseRunner.cfg(), r.secretsProvider(), initFn)
	r.baseRunner.run(wfs)
}
