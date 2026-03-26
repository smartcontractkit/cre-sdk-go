package cre

import "fmt"

// CapabilityError is an error returned when a capability call fails.
// It carries the capability ID so that callers can attribute the failure
// to a specific capability.
type CapabilityError struct {
	Message      string
	CapabilityID string
}

func NewCapabilityError(message, capabilityID string) *CapabilityError {
	return &CapabilityError{Message: message, CapabilityID: capabilityID}
}

func (e *CapabilityError) Error() string {
	return fmt.Sprintf("capability %s error: %s", e.CapabilityID, e.Message)
}
