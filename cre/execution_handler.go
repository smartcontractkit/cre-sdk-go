package cre

import (
	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ExecutionHandler defines a coupling of a Trigger and a callback function to be used by the SDK.
// The methods on the handler are meant to be used internally by the Runtime.
type ExecutionHandler[C, R any] interface {
	CapabilityID() string
	Method() string
	TriggerCfg() *anypb.Any
	Callback() func(config C, runtime R, payload *anypb.Any) (any, error)
}

type ExecutionHandlerWithRequirements[C, R any] interface {
	ExecutionHandler[C, R]
	Requirements() *sdk.Requirements
}

// Handler creates a coupling of a Trigger and a callback function to be used by the SDK.
// The coupling ensures that when the Trigger is invoked, the callback function is called with the appropriate parameters.
func Handler[C any, M proto.Message, T any, O any](trigger Trigger[M, T], callback func(config C, runtime Runtime, payload T) (O, error)) ExecutionHandler[C, Runtime] {
	return handler(trigger, callback, nil)
}

// HandlerInTee creates a coupling of a Trigger and a callback function to be used in TEE (Trusted Execution Environment) mode.
// The coupling ensures that when the Trigger is invoked, the callback function is called with a TeeRuntime.
func HandlerInTee[C any, M proto.Message, T any, O any, A AcceptedTees](trigger Trigger[M, T], callback func(config C, runtime TeeRuntime, payload T) (O, error), tees A) ExecutionHandler[C, Runtime] {
	requirements := &sdk.Requirements{Tee: &sdk.Tee{}}
	reqs := &sdk.Requirements{Tee: &sdk.Tee{}}
	switch typedTees := any(tees).(type) {
	case []TeeAndRegions:
		typeRegions := make([]*sdk.TeeTypeAndRegions, len(typedTees))
		for i, tee := range typedTees {
			typeRegions[i] = &sdk.TeeTypeAndRegions{Type: tee.Type, Regions: tee.Regions}
		}
		reqs.Tee.Type = &sdk.Tee_TypeSelection{TypeSelection: &sdk.TeeTypeSelection{Types: typeRegions}}
	case AnyTee:
		reqs.Tee.Type = &sdk.Tee_Any{Any: &emptypb.Empty{}}
	}

	wrapped := func(config C, runtime Runtime, t T) (O, error) {
		// hack to allow it to pass us a teeRuntime
		helper, ok := runtime.(interface{ Tee() TeeRuntime })
		if !ok {
			panic("Runner did not provide an extractable TEERuntime. If you wrapped the runtime, wrap the method Tee() TeeRuntime instead.")
		}

		return callback(config, helper.Tee(), t)
	}
	return handler(trigger, wrapped, requirements)
}

func handler[R, C any, M proto.Message, T any, O any](trigger Trigger[M, T], callback func(config C, runtime R, payload T) (O, error), requirements *sdk.Requirements) ExecutionHandler[C, R] {
	wrapped := func(config C, runtime R, payload *anypb.Any) (any, error) {
		unwrappedTrigger := trigger.NewT()
		if err := payload.UnmarshalTo(unwrappedTrigger); err != nil {
			return nil, err
		}
		input, err := trigger.Adapt(unwrappedTrigger)
		if err != nil {
			return nil, err
		}
		return callback(config, runtime, input)
	}

	eh := &executionHandlerImpl[C, R, M, T]{
		Trigger: trigger,
		fn:      wrapped,
	}

	if requirements == nil {
		return eh
	}

	return executionHandlerWithRequirementsImpl[C, R]{ExecutionHandler: eh, requirements: requirements}
}

type executionHandlerImpl[C, R any, M proto.Message, T any] struct {
	Trigger[M, T]
	fn func(config C, runtime R, trigger *anypb.Any) (any, error)
}

var _ ExecutionHandler[int, any] = (*executionHandlerImpl[int, any, proto.Message, any])(nil)

func (h *executionHandlerImpl[C, R, M, T]) TriggerCfg() *anypb.Any {
	return h.Trigger.ConfigAsAny()
}

func (h *executionHandlerImpl[C, R, M, T]) Callback() func(config C, runtime R, payload *anypb.Any) (any, error) {
	return h.fn
}

type executionHandlerWithRequirementsImpl[C, R any] struct {
	ExecutionHandler[C, R]
	requirements *sdk.Requirements
}

func (h *executionHandlerWithRequirementsImpl[C, R]) Requirements() *sdk.Requirements {
	return h.requirements
}
