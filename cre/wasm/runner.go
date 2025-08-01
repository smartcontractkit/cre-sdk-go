package wasm

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"unsafe"

	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/internal/sdkimpl"
	"google.golang.org/protobuf/proto"

	"github.com/smartcontractkit/chainlink-common/pkg/values"
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
)

type Config any

type runnerInternals interface {
	args() []string
	sendResponse(response unsafe.Pointer, responseLen int32) int32
	versionV2()
	switchModes(mode int32)
}

func newRunner[C Config](parse func(configBytes []byte) (C, error), runnerInternals runnerInternals, runtimeInternals runtimeInternals) cre.Runner[C] {
	runnerInternals.versionV2()
	runnerInternals.switchModes(int32(pb.Mode_MODE_DON))
	drt := &sdkimpl.Runtime{RuntimeBase: newRuntime(runtimeInternals, pb.Mode_MODE_DON)}
	return runnerWrapper[C]{baseRunner: getRunner(
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
	}
}

type runner[C, T any] struct {
	runnerInternals
	trigger    *pb.Trigger
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
	env := &cre.Environment[C]{
		NodeEnvironment: cre.NodeEnvironment[C]{
			Config:    r.config,
			LogWriter: &writer{},
			Logger:    slog.New(slog.NewTextHandler(&writer{}, nil)),
		},
		SecretsProvider: r.secretsProvider(),
	}
	for idx, handler := range wfs {
		if uint64(idx) == r.trigger.Id {
			response, err := handler.Callback()(env, r.runtime, r.trigger.Payload)
			execResponse := &pb.ExecutionResult{}
			if err == nil {
				wrapped, err := values.Wrap(response)
				if err != nil {
					execResponse.Result = &pb.ExecutionResult_Error{Error: err.Error()}
				} else {
					execResponse.Result = &pb.ExecutionResult_Value{Value: values.Proto(wrapped)}
				}
			} else {
				execResponse.Result = &pb.ExecutionResult_Error{Error: err.Error()}
			}
			marshalled, _ := proto.Marshal(execResponse)
			marshalledPtr, marshalledLen, _ := bufferToPointerLen(marshalled)
			r.sendResponse(marshalledPtr, marshalledLen)
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
	subscriptions := make([]*pb.TriggerSubscription, len(wfs))
	for i, handler := range wfs {
		subscriptions[i] = &pb.TriggerSubscription{
			Id:      handler.CapabilityID(),
			Payload: handler.TriggerCfg(),
			Method:  handler.Method(),
		}
	}
	triggerSubscription := &pb.TriggerSubscriptionRequest{Subscriptions: subscriptions}

	execResponse := &pb.ExecutionResult{
		Result: &pb.ExecutionResult_TriggerSubscriptions{TriggerSubscriptions: triggerSubscription},
	}

	configBytes, _ := proto.Marshal(execResponse)
	configPtr, configLen, _ := bufferToPointerLen(configBytes)

	result := s.sendResponse(configPtr, configLen)
	if result < 0 {
		exitErr(fmt.Sprintf("could not subscribe to triggers: %s", string(configBytes[:-result])))
	}
}

func getWorkflows[C any](config C, secretsProvider cre.SecretsProvider, initFn func(env *cre.Environment[C]) (cre.Workflow[C], error)) cre.Workflow[C] {
	wfs, err := initFn(&cre.Environment[C]{
		NodeEnvironment: cre.NodeEnvironment[C]{
			Config:    config,
			LogWriter: &writer{},
			Logger:    slog.New(slog.NewTextHandler(&writer{}, nil)),
		},
		SecretsProvider: secretsProvider,
	})
	if err != nil {
		exitErr(err.Error())
	}
	return wfs
}

func getRunner[C, T any](parse func(configBytes []byte) (C, error), subscribe *subscriber[C, T], run *runner[C, T]) baseRunner[C, T] {
	args := run.args()

	// We expect exactly 2 args, i.e. `wasm <blob>`,
	// where <blob> is a base64 encoded protobuf message.
	if len(args) != 2 {
		exitErr("invalid request: request must contain a payload")
	}

	request := args[1]
	if request == "" {
		exitErr("invalid request: request cannot be empty")
	}

	b, err := base64.StdEncoding.DecodeString(request)
	if err != nil {
		exitErr("invalid request: could not decode request into bytes")
	}

	execRequest := &pb.ExecuteRequest{}
	if err = proto.Unmarshal(b, execRequest); err != nil {
		exitErr("invalid request: could not unmarshal request into ExecuteRequest")
	}

	c, err := parse(execRequest.Config)
	if err != nil {
		exitErr(err.Error())
	}

	switch req := execRequest.Request.(type) {
	case *pb.ExecuteRequest_Subscribe:
		subscribe.config = c
		subscribe.setRuntime(execRequest.MaxResponseSize)
		return subscribe
	case *pb.ExecuteRequest_Trigger:
		run.trigger = req.Trigger
		run.config = c
		run.setRuntime(execRequest.MaxResponseSize)
		return run
	}

	exitErr(fmt.Sprintf("invalid request: unknown request type %T", execRequest.Request))
	return nil
}

func exitErr(msg string) {
	_, _ = (&writer{}).Write([]byte(msg))
	os.Exit(1)
}

type baseRunner[C, T any] interface {
	secretsProvider() cre.SecretsProvider
	cfg() C
	run([]cre.ExecutionHandler[C, T])
}

type runnerWrapper[C any] struct {
	baseRunner[C, cre.Runtime]
}

func (r runnerWrapper[C]) Run(initFn func(env *cre.Environment[C]) (cre.Workflow[C], error)) {
	wfs := getWorkflows(r.baseRunner.cfg(), r.baseRunner.secretsProvider(), initFn)
	r.baseRunner.run(wfs)
}
