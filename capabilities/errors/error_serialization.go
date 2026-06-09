package errors

import (
	"errors"
	"strings"
)

const errorMessageSeparator = ":"

// DeserializeErrorFromString parses errorMsg in the capability error wire format.
// If errorMsg is not a serialized capability error errors.New(errorMsg) is returned and the result is not a capability Error.
func DeserializeErrorFromString(errorMsg string) error {
	parts := strings.SplitN(errorMsg, errorMessageSeparator, 4)

	if len(parts) < 4 {
		return errors.New(errorMsg)
	}

	visibility := FromVisibilityString(parts[0])
	origin := FromOriginString(parts[1])
	errorCode := FromErrorCodeString(parts[2])
	detail := parts[3]

	return NewError(errors.New(detail), visibility, origin, errorCode)
}
