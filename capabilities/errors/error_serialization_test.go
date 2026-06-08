package errors_test

import (
	stderrors "errors"
	"strings"
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

func TestErrorSerializationAndDeserialization(t *testing.T) {
	visibilities := []caperrs.Visibility{caperrs.VisibilityPublic, caperrs.VisibilityPrivate}
	origins := []caperrs.Origin{caperrs.OriginUser, caperrs.OriginSystem}
	errorCodes := []caperrs.ErrorCode{caperrs.Unknown, caperrs.ConsensusFailed, caperrs.InvalidArgument}

	for _, v := range visibilities {
		for _, o := range origins {
			for _, c := range errorCodes {
				originalErr := caperrs.NewError(stderrors.New("test error"), v, o, c)
				serialized := originalErr.SerializeToString()
				deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serialized, true))
				if !originalErr.Equals(deserializedErr) {
					t.Errorf("expected %v, got %v", originalErr, deserializedErr)
				}
			}
		}
	}
}

func TestRemoteErrorSerializationAndDeserialization(t *testing.T) {
	visibilities := []caperrs.Visibility{caperrs.VisibilityPublic, caperrs.VisibilityPrivate}
	origins := []caperrs.Origin{caperrs.OriginUser, caperrs.OriginSystem}
	errorCodes := []caperrs.ErrorCode{caperrs.Unknown, caperrs.ConsensusFailed, caperrs.InvalidArgument}

	for _, v := range visibilities {
		for _, o := range origins {
			for _, c := range errorCodes {
				originalErr := caperrs.NewError(stderrors.New("test error"), v, o, c)
				serialized := originalErr.SerializeToRemoteString()
				deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serialized, true))
				if v == caperrs.VisibilityPrivate {
					require.Equal(t, deserializedErr.Visibility(), originalErr.Visibility())
					require.Equal(t, deserializedErr.Origin(), originalErr.Origin())
					require.Equal(t, deserializedErr.Code(), originalErr.Code())
					require.True(t, strings.Contains(deserializedErr.Error(), "error whilst executing capability - the error message is not publicly reportable"))
				} else {
					if !originalErr.Equals(deserializedErr) {
						t.Errorf("expected %v, got %v", originalErr, deserializedErr)
					}
				}
			}
		}
	}
}

func TestParsingOldErrorFormat(t *testing.T) {
	oldErrorMsgString := "failed to execute capability: some error occurred"
	deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(oldErrorMsgString, true))

	expectedErr := caperrs.NewError(stderrors.New(oldErrorMsgString), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
	if !deserializedErr.Equals(expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
	}
}

func TestParsingWithInvalidVisibilityOriginAndErrorCodesAndBackwardsCompatibility(t *testing.T) {
	t.Run("InvalidVisibility", func(t *testing.T) {
		serializedError := "InvalidVisibility:User:Unknown:some error occurred"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serializedError, true))
		require.False(t, caperrs.IsSerializedCapabilityErrorString(serializedError))
		expectedErr := caperrs.NewError(stderrors.New(serializedError), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		if !deserializedErr.Equals(expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
		}
	})

	t.Run("InvalidOrigin", func(t *testing.T) {
		serializedError := "Public:InvalidOrigin:Unknown:some error occurred"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serializedError, true))
		require.False(t, caperrs.IsSerializedCapabilityErrorString(serializedError))
		expectedErr := caperrs.NewError(stderrors.New(serializedError), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		if !deserializedErr.Equals(expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
		}
	})

	t.Run("InvalidErrorCode", func(t *testing.T) {
		serializedError := "Public:System:InvalidErrorCode:some error occurred"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(serializedError, true))
		require.False(t, caperrs.IsSerializedCapabilityErrorString(serializedError))
		expectedErr := caperrs.NewError(stderrors.New(serializedError), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		if !deserializedErr.Equals(expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
		}
	})

	t.Run("InvalidMessageInsufficientFields", func(t *testing.T) {
		msg := "Public:System:Unknown"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(msg, true))

		expectedErr := caperrs.NewError(stderrors.New(msg), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		if !deserializedErr.Equals(expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
		}
	})

	t.Run("NotASerializedCapabilityError", func(t *testing.T) {
		msg := "some error has occurred that is not in the serialized capability error format"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(msg, true))

		expectedErr := caperrs.NewError(stderrors.New(msg), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		if !deserializedErr.Equals(expectedErr) {
			t.Errorf("expected %v, got %v", expectedErr, deserializedErr)
		}
	})

	t.Run("ColonRichPlainMessageMatchingUnknownCode", func(t *testing.T) {
		// Four segments with a known error code token but invalid visibility — legacy wrap.
		msg := "failed: attempt: Unknown: details here"
		deserializedErr := requireCapabilityError(t, caperrs.DeserializeErrorFromString(msg, true))
		require.False(t, caperrs.IsSerializedCapabilityErrorString(msg))
		expectedErr := caperrs.NewError(stderrors.New(msg), caperrs.VisibilityPrivate, caperrs.OriginSystem, caperrs.Unknown)
		require.True(t, deserializedErr.Equals(expectedErr))
	})
}

func TestIsSerializedCapabilityErrorString(t *testing.T) {
	valid := caperrs.NewError(stderrors.New("detail"), caperrs.VisibilityPublic, caperrs.OriginSystem, caperrs.InvalidArgument).SerializeToString()
	require.True(t, caperrs.IsSerializedCapabilityErrorString(valid))

	require.False(t, caperrs.IsSerializedCapabilityErrorString("Public:System"))
	require.False(t, caperrs.IsSerializedCapabilityErrorString("Public:System:Unknown"))
	require.True(t, caperrs.IsSerializedCapabilityErrorString("Public:System:Unknown:"))
	require.True(t, caperrs.IsSerializedCapabilityErrorString("Public:System:Unknown:x"))

	require.False(t, caperrs.IsSerializedCapabilityErrorString("InvalidVisibility:User:Unknown:msg"))
	require.False(t, caperrs.IsSerializedCapabilityErrorString("Public:InvalidOrigin:Unknown:msg"))
	require.False(t, caperrs.IsSerializedCapabilityErrorString("Public:System:InvalidErrorCode:msg"))
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
		original := caperrs.NewError(stderrors.New("detail"), caperrs.VisibilityPublic, caperrs.OriginUser, caperrs.DeadlineExceeded)
		serialized := original.SerializeToString()
		err := caperrs.DeserializeErrorFromString(serialized, false)
		deserialized := requireCapabilityError(t, err)
		require.True(t, original.Equals(deserialized))
	})
}
