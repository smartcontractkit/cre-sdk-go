package evm

import (
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// This would be part of the generated code

// This would be a proto message, I'm just lazy

type LogAndChain struct {
	ChainSelector uint64
	Log           *Log
}

func (l LogAndChain) ProtoReflect() protoreflect.Message {
	//TODO implement me
	panic("implement me")
}

func DynamicLogTrigger() cre.DynamicTrigger[*LogAndChain, *LogAndChain, *DynamicLogTriggerRef] {
	return &dynamicLogTrigger{}
}

type dynamicLogTrigger struct {
}

func (d dynamicLogTrigger) Method() string {
	return "DynamicLogTrigger"
}

func (d dynamicLogTrigger) Ref(id string) *DynamicLogTriggerRef {
	return &DynamicLogTriggerRef{id: id}

}

func (d dynamicLogTrigger) Adapt(m *LogAndChain) (*LogAndChain, error) {
	return m, nil
}

type DynamicLogTriggerRef struct {
	id string
}

func (d DynamicLogTriggerRef) AddTrigger(runtime cre.Runtime, chainSelector uint64, t *FilterLogTriggerRequest) cre.Promise[struct{}] {
	// Call to the runtime to add the handler
	// it would wrap the request using the chan selector and filter
	panic("Not done yet")
}

func (d DynamicLogTriggerRef) RemoveTrigger(runtime cre.Runtime, chainSelector uint64, t *FilterLogTriggerRequest) cre.Promise[struct{}] {
	// Call to the runtime to add the handler
	// it would wrap the request using the chan selector and filter
	panic("Not done yet")
}
