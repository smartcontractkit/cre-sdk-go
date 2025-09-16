package creclient

import "encoding/json"

type capabilityRequest struct {
	CapabilityId string
	Payload      string
	Method       string
}

type httpRequest struct {
	Body     json.RawMessage
	Workflow string
}

type result struct {
	Result  string
	IsError bool
}
