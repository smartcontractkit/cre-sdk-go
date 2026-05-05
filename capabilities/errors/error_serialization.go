package errors

import (
	"errors"
	"strings"
)

const errorMessageSeparator = ":"

func PrePendPrivateVisibilityIdentifier(errorMessage string) string {
	return VisibilityPrivate.String() + errorMessageSeparator + errorMessage
}

// IsSerializedCapabilityErrorString reports whether s uses the capability error
// wire format (Visibility:Origin:ErrorCode:detail) with known enum/code names.
// Arbitrary text that happens to contain three colons is not treated as
// serialized unless the first three segments are valid.
func IsSerializedCapabilityErrorString(s string) bool {
	parts := strings.SplitN(s, errorMessageSeparator, 4)
	return len(parts) == 4 && serializedCapabilityErrorMetadataValid(parts[0], parts[1], parts[2])
}

func serializedCapabilityErrorMetadataValid(visibility, origin, errorCode string) bool {
	switch FromVisibilityString(visibility) {
	case VisibilityPublic, VisibilityPrivate:
	default:
		return false
	}
	switch FromOriginString(origin) {
	case OriginSystem, OriginUser:
	default:
		return false
	}
	return IsKnownErrorCodeString(errorCode)
}

// DeserializeErrorFromString parses errorMsg in the capability error wire format.
// If errorMsg is not a serialized capability error, behavior depends on wrapUndeserializableAsCapabilityError:
// when true, the full string is wrapped as a private system error with code Unknown (backwards compatible with
// older nodes); when false, errors.New(errorMsg) is returned and the result is not a capability Error.
func DeserializeErrorFromString(errorMsg string, wrapUndeserializableAsCapabilityError bool) error {
	parts := strings.SplitN(errorMsg, errorMessageSeparator, 4)

	if len(parts) < 4 || !serializedCapabilityErrorMetadataValid(parts[0], parts[1], parts[2]) {
		if wrapUndeserializableAsCapabilityError {
			// To maintain backwards compatibility with messages from remote nodes on an older code version, create an error
			// with the full message and default to private system error with an unknown error code.
			return NewError(errors.New(errorMsg), VisibilityPrivate, OriginSystem, Unknown)
		}
		return errors.New(errorMsg)
	}

	visibility := FromVisibilityString(parts[0])
	origin := FromOriginString(parts[1])
	errorCode := FromErrorCodeString(parts[2])
	detail := parts[3]

	return NewError(errors.New(detail), visibility, origin, errorCode)
}

func (e capabilityError) SerializeToString() string {
	return e.serializeToString(e.err.Error())
}

func (e capabilityError) serializeToString(errMsg string) string {
	return e.visibility.String() + errorMessageSeparator + e.origin.String() + errorMessageSeparator + e.Code().String() + errorMessageSeparator + errMsg
}

// SerializeToRemoteString serializes the error for sending to remote nodes.
// If the error is private, the actual error message is replaced with a generic message.
func (e capabilityError) SerializeToRemoteString() string {
	if e.Visibility() == VisibilityPublic {
		return e.serializeToString(e.err.Error())
	}

	return e.serializeToString("error whilst executing capability - the error message is not publicly reportable")
}
