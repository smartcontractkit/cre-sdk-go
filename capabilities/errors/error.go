// Package errors defines CRE capability errors and a wire format shared across CRE components.
// It aligns with github.com/smartcontractkit/chainlink-common/pkg/capabilities/errors.
package errors

import "fmt"

type Origin int

const (
	// OriginSystem The error originated from a system issue.
	OriginSystem Origin = 0

	// OriginUser The error originated from user input or action.
	OriginUser Origin = 1
)

func (o Origin) String() string {
	switch o {
	case OriginSystem:
		return "System"
	case OriginUser:
		return "User"
	default:
		return "UnknownOrigin"
	}
}

// FromOriginString converts a string to an Origin value.
func FromOriginString(s string) Origin {
	switch s {
	case "System":
		return OriginSystem
	case "User":
		return OriginUser
	default:
		return Origin(-1)
	}
}

type Visibility int

const (
	// VisibilityPublic The full details of the error can be shared across all nodes in the network.
	VisibilityPublic Visibility = 0

	// VisibilityPrivate The error contains sensitive information that should only be visible to the local node.
	VisibilityPrivate Visibility = 1
)

// String returns the string representation of the Visibility value.
func (v Visibility) String() string {
	switch v {
	case VisibilityPublic:
		return "Public"
	case VisibilityPrivate:
		return "Private"
	default:
		return "UnknownVisibility"
	}
}

// FromVisibilityString converts a string to a Visibility value.
func FromVisibilityString(s string) Visibility {
	switch s {
	case "Public":
		return VisibilityPublic
	case "Private":
		return VisibilityPrivate
	default:
		return Visibility(-1)
	}
}

type Error interface {
	error

	Visibility() Visibility
	Origin() Origin
	Code() ErrorCode
}

type capabilityError struct {
	err        error
	origin     Origin
	visibility Visibility
	errorCode  ErrorCode
}

func NewError(err error, visibility Visibility, origin Origin, errorCode ErrorCode) Error {
	return &capabilityError{
		err:        err,
		origin:     origin,
		visibility: visibility,
		errorCode:  errorCode,
	}
}

func (e capabilityError) Error() string {
	if e.errorCode == UnrecognisedErrorCode {
		return e.err.Error()
	}
	return fmt.Sprintf("[%d]%s: %s", e.errorCode, e.errorCode.String(), e.err.Error())
}

func (e capabilityError) Origin() Origin {
	return e.origin
}

func (e capabilityError) Visibility() Visibility {
	return e.visibility
}

func (e capabilityError) Code() ErrorCode {
	return e.errorCode
}
