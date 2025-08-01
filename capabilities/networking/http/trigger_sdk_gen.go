// Code generated by github.com/smartcontractkit/chainlink-common/pkg/capabilities/v2/protoc, DO NOT EDIT.

package http

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/smartcontractkit/cre-sdk-go/cre"
)

type HTTP struct {
	// TODO: https://smartcontract-it.atlassian.net/browse/CAPPL-799 allow defaults for capabilities
}

func Trigger(config *Config) cre.Trigger[*Payload, *Payload] {
	configAny := &anypb.Any{}
	_ = anypb.MarshalFrom(configAny, config, proto.MarshalOptions{Deterministic: true})
	return &hTTPTrigger{

		config: configAny,
	}
}

type hTTPTrigger struct {
	config *anypb.Any
}

func (*hTTPTrigger) IsTrigger() {}

func (*hTTPTrigger) NewT() *Payload {
	return &Payload{}
}

func (c *hTTPTrigger) CapabilityID() string {
	return "http-trigger@1.0.0-alpha"
}

func (*hTTPTrigger) Method() string {
	return "Trigger"
}

func (t *hTTPTrigger) ConfigAsAny() *anypb.Any {
	return t.config
}

func (t *hTTPTrigger) Adapt(trigger *Payload) (*Payload, error) {
	return trigger, nil
}
