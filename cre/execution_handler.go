package cre

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// ExecutionHandler defines a coupling of a Trigger and a callback function to be used by the SDK.
// The methods on the handler are meant to be used internally by the Runtime.
type ExecutionHandler[C, R any] interface {
	CapabilityID() string
	Method() string
	TriggerCfg() *anypb.Any
	Callback() func(config C, runtime R, payload *anypb.Any) (any, error)
}

// Handler creates a coupling of a Trigger and a callback function to be used by the SDK.
// The coupling ensures that when the Trigger is invoked, the callback function is called with the appropriate parameters.
func Handler[C any, M proto.Message, T any, O any](trigger Trigger[M, T], callback func(config C, runtime Runtime, payload T) (O, error)) ExecutionHandler[C, Runtime] {
	return handler(trigger, callback)
}

func handler[R, C any, M proto.Message, T any, O any](trigger Trigger[M, T], callback func(config C, runtime R, payload T) (O, error)) ExecutionHandler[C, R] {
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
	return &executionHandlerImpl[C, R, M, T]{
		Trigger: trigger,
		fn:      wrapped,
	}
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
