package cre

import caperrs "github.com/smartcontractkit/cre-sdk-go/capabilities/errors"

// ErrorFromCapabilityResponse converts the CapabilityResponse_Error message string
// into an error using the capability error serialization format from chainlink-common.
// Plain messages that are not in that format are returned as a standard Go error.
func ErrorFromCapabilityResponse(message string) error {
	return caperrs.DeserializeErrorFromString(message, false)
}
