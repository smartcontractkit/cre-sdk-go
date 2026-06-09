package errors_test

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/require"

	caperrs "github.com/smartcontractkit/cre-sdk-go/capabilities/errors"
)

func requireCapabilityError(tb testing.TB, err error) caperrs.Error {
	tb.Helper()
	ce, ok := err.(caperrs.Error)
	require.True(tb, ok, "expected capability errors.Error, got %T", err)
	return ce
}

func capabilityErrorsEqual(a, b caperrs.Error) bool {
	return a.Code() == b.Code() &&
		a.Origin() == b.Origin() &&
		a.Visibility() == b.Visibility() &&
		a.Error() == b.Error()
}

// TestDeserializeErrorFromStringInvalidFields verifies that DeserializeErrorFromString can handle invalid visibility,
// origin, and error code fields gracefully, preserving unrecognized tokens as typed sentinel values (-1) and still
// returning a capability error when the format is correct. This ensures backwards compatibility for any new values
// introduced in the future.
func TestDeserializeErrorFromStringInvalidFields(t *testing.T) {
	t.Run("InvalidVisibility", func(t *testing.T) {
		serializedError := "InvalidVisibility:User:Unknown:some error occurred"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serializedError))
		expectedErr := caperrs.NewError(stderrors.New("some error occurred"), caperrs.Visibility(-1), caperrs.OriginUser, caperrs.Unknown)
		require.True(t, capabilityErrorsEqual(deserializedErr, expectedErr))
		require.Equal(t, "UnknownVisibility", deserializedErr.Visibility().String())
	})

	t.Run("InvalidOrigin", func(t *testing.T) {
		serializedError := "Public:InvalidOrigin:Unknown:some error occurred"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serializedError))
		expectedErr := caperrs.NewError(stderrors.New("some error occurred"), caperrs.VisibilityPublic, caperrs.Origin(-1), caperrs.Unknown)
		require.True(t, capabilityErrorsEqual(deserializedErr, expectedErr))
		require.Equal(t, "UnknownOrigin", deserializedErr.Origin().String())
	})

	t.Run("UnrecognisedErrorCode", func(t *testing.T) {
		serializedError := "Public:System:NewUnknownErrorCode:some error occurred"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serializedError))
		expectedErr := caperrs.NewError(stderrors.New("some error occurred"), caperrs.VisibilityPublic, caperrs.OriginSystem, caperrs.UnrecognisedErrorCode)
		require.True(t, capabilityErrorsEqual(deserializedErr, expectedErr))
		require.Equal(t, "UnrecognisedErrorCode", deserializedErr.Code().String())
		require.Equal(t, "some error occurred", deserializedErr.Error())
	})

	t.Run("ColonRichPlainMessageMatchingUnknownCode", func(t *testing.T) {
		// Four segments are always parsed as wire format; invalid visibility/origin tokens are preserved.
		msg := "failed:attempt:Unknown: details here"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(msg))
		expectedErr := caperrs.NewError(stderrors.New(" details here"), caperrs.Visibility(-1), caperrs.Origin(-1), caperrs.Unknown)
		require.True(t, capabilityErrorsEqual(deserializedErr, expectedErr))
	})
}

func TestDeserializeErrorFromStringNonWireFormat(t *testing.T) {
	t.Run("InsufficientFields", func(t *testing.T) {
		msg := "Public:System:Unknown"
		err := caperrs.DeserializeErrorFromString(msg)
		require.Equal(t, msg, err.Error())
		_, ok := err.(caperrs.Error)
		require.False(t, ok)
	})

	t.Run("PlainMessage", func(t *testing.T) {
		msg := "some error has occurred that is not in the serialized capability error format"
		err := caperrs.DeserializeErrorFromString(msg)
		require.Equal(t, msg, err.Error())
		_, ok := err.(caperrs.Error)
		require.False(t, ok)
	})
}

func TestDeserializeErrorFromStringThatIsNotSerialisedCapabilityError(t *testing.T) {
	t.Run("plain message returns stdlib error", func(t *testing.T) {
		msg := "some plain failure"
		err := caperrs.DeserializeErrorFromString(msg)
		require.Equal(t, msg, err.Error())
		_, ok := err.(caperrs.Error)
		require.False(t, ok)
	})

	t.Run("valid serialized still returns capability error", func(t *testing.T) {
		serialized := "Public:User:DeadlineExceeded:detail"
		expected := caperrs.NewError(stderrors.New("detail"), caperrs.VisibilityPublic, caperrs.OriginUser, caperrs.DeadlineExceeded)
		err := caperrs.DeserializeErrorFromString(serialized)
		deserialized := requireCapabilityError(t, err)
		require.True(t, capabilityErrorsEqual(expected, deserialized))
	})
}
