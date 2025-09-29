package cre

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type baseTrigger[T proto.Message] interface {
	NewT() T
	CapabilityID() string
	ConfigAsAny() *anypb.Any
	Method() string
}

// Trigger is a capability that initiates a workflow execution.
// This interface is meant for internal use by the Runner
// To obtain an implementation, use the SDKs for the capability you want to utilize.
type Trigger[M proto.Message, T any] interface {
	baseTrigger[M]
	Adapt(m M) (T, error)
}
