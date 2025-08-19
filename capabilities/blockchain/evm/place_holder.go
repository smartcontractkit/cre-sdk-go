package evm

import (
	"github.com/smartcontractkit/cre-sdk-go/cre"
)

// This would be part of the generated code

func DynamicLogTrigger() cre.DynamicTrigger[*Log, *Log, *DynamicLogTriggerRef] {
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

func (d dynamicLogTrigger) Adapt(m *Log) (*Log, error) {
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
