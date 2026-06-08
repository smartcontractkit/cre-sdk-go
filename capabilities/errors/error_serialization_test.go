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

func TestParsingOldErrorFormat(t *testing.T) {
	oldErrorMsgString := "failed to execute capability: some error occurred"
	deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(oldErrorMsgString, true))

	expectedErr := caperrs.NewError(stderrors.New(oldErrorMsgString), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
	if !capabilityErrorsEqual(deserializedErr, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
	}
}

func TestParsingWithInvalidVisibilityOriginAndErrorCodesAndBackwardsCompatibility(t *testing.T) {
	t.Run("InvalidVisibility", func(t *testing.T) {
		serializedError := "InvalidVisibility:User:Unknown:some error occurred"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serializedError, true))
		expectedErr := caperrs.NewError(stderrors.New(serializedError), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		if !capabilityErrorsEqual(deserializedErr, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
		}
	})

	t.Run("InvalidOrigin", func(t *testing.T) {
		serializedError := "Public:InvalidOrigin:Unknown:some error occurred"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serializedError, true))
		expectedErr := caperrs.NewError(stderrors.New(serializedError), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		if !capabilityErrorsEqual(deserializedErr, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
		}
	})

	t.Run("InvalidErrorCode", func(t *testing.T) {
		serializedError := "Public:System:InvalidErrorCode:some error occurred"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serializedError, true))
		expectedErr := caperrs.NewError(stderrors.New(serializedError), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		if !capabilityErrorsEqual(deserializedErr, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
		}
	})

	t.Run("InvalidMessageInsufficientFields", func(t *testing.T) {
		msg := "Public:System:Unknown"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(msg, true))

		expectedErr := caperrs.NewError(stderrors.New(msg), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		if !capabilityErrorsEqual(deserializedErr, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
		}
	})

	t.Run("NotASerializedCapabilityError", func(t *testing.T) {
		msg := "some error has occurred that is not in the serialized capability error format"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(msg, true))

		expectedErr := caperrs.NewError(stderrors.New(msg), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		if !capabilityErrorsEqual(deserializedErr, expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
		}
	})

	t.Run("ColonRichPlainMessageMatchingUnknownCode", func(t *testing.T) {
		// Four segments with a known error code token but invalid visibility — legacy wrap.
		msg := "failed: attempt: Unknown: details here"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(msg, true))
		expectedErr := caperrs.NewError(stderrors.New(msg), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		require.True(t, capabilityErrorsEqual(deserializedErr, expectedErr))
	})
}

func TestDeserializeErrorFromString_withoutCapabilityWrap(t *testing.T) {
	t.Run("plain message returns stdlib error", func(t *testing.T) {
		msg := "some plain failure"
		err := caperrs.DeserializeErrorFromString(msg, false)
		require.Equal(t, msg, err.Error())
		_, ok := err.(caperrs.Error)
		require.False(t, ok)
	})

	t.Run("valid serialized still returns capability error", func(t *testing.T) {
		serialized := "Public:User:DeadlineExceeded:detail"
		expected := caperrs.NewError(stderrors.New("detail"), caperrs.VisibilityPublic, caperrs.OriginUser, caperrs.DeadlineExceeded)
		err := caperrs.DeserializeErrorFromString(serialized, false)
		deserialized := requireCapabilityError(t, err)
		require.True(t, capabilityErrorsEqual(expected, deserialized))
	})
}
