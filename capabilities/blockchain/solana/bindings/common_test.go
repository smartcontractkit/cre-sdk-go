package bindings

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"testing"

	bin "github.com/gagliardetto/binary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/solana"
)

func TestForwarderReport_Marshal(t *testing.T) {
	t.Run("encodes to expected Borsh format", func(t *testing.T) {
		hash := [32]byte{
			0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
			0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
			0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
			0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20,
		}
		payload := []byte("hello solana")

		report := ForwarderReport{
			AccountHash: hash,
			Payload:     payload,
		}

		data, err := report.Marshal()
		require.NoError(t, err)

		// Borsh format: account_hash(32) | payload_len(u32 LE) | payload
		expectedLen := 32 + 4 + len(payload)
		require.Len(t, data, expectedLen)

		// First 32 bytes: account hash
		assert.Equal(t, hash[:], data[:32])

		// Next 4 bytes: little-endian u32 payload length
		payloadLen := binary.LittleEndian.Uint32(data[32:36])
		assert.Equal(t, uint32(len(payload)), payloadLen)

		// Remaining bytes: payload
		assert.Equal(t, payload, data[36:])
	})
}

func TestCalculateAccountsHash(t *testing.T) {
	t.Run("single account", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}

		accs := []*solana.AccountMeta{
			{PublicKey: key},
		}

		got := CalculateAccountsHash(accs)
		expected := sha256.Sum256(key)
		assert.Equal(t, expected, got)
	})

	t.Run("multiple accounts", func(t *testing.T) {
		key1 := make([]byte, 32)
		key2 := make([]byte, 32)
		for i := range key1 {
			key1[i] = byte(i)
			key2[i] = byte(i + 32)
		}

		accs := []*solana.AccountMeta{
			{PublicKey: key1},
			{PublicKey: key2},
		}

		got := CalculateAccountsHash(accs)

		// Hash should be SHA-256 of concatenated keys
		concat := append(key1, key2...)
		expected := sha256.Sum256(concat)
		assert.Equal(t, expected, got)
	})

	t.Run("empty slice", func(t *testing.T) {
		got := CalculateAccountsHash([]*solana.AccountMeta{})
		expected := sha256.Sum256([]byte{})
		assert.Equal(t, expected, got)
	})

	t.Run("nil slice", func(t *testing.T) {
		got := CalculateAccountsHash(nil)
		expected := sha256.Sum256([]byte{})
		assert.Equal(t, expected, got)
	})

	t.Run("skips nil entries", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i + 100)
		}

		accs := []*solana.AccountMeta{
			nil,
			{PublicKey: key},
			nil,
		}

		got := CalculateAccountsHash(accs)
		expected := sha256.Sum256(key)
		assert.Equal(t, expected, got)
	})

	t.Run("order matters", func(t *testing.T) {
		key1 := make([]byte, 32)
		key2 := make([]byte, 32)
		for i := range key1 {
			key1[i] = byte(i)
			key2[i] = byte(i + 32)
		}

		hash12 := CalculateAccountsHash([]*solana.AccountMeta{
			{PublicKey: key1},
			{PublicKey: key2},
		})
		hash21 := CalculateAccountsHash([]*solana.AccountMeta{
			{PublicKey: key2},
			{PublicKey: key1},
		})

		assert.NotEqual(t, hash12, hash21, "hash should depend on account order")
	})
}

func TestPrepareSubkeyValue(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		got, err := PrepareSubkeyValue("hello")
		require.NoError(t, err)
		assert.Equal(t, []byte("hello"), got)
	})

	t.Run("bytes", func(t *testing.T) {
		input := []byte{1, 2, 3}
		got, err := PrepareSubkeyValue(input)
		require.NoError(t, err)
		assert.Equal(t, input, got)
	})

	t.Run("uint64 big endian", func(t *testing.T) {
		got, err := PrepareSubkeyValue(uint64(1000))
		require.NoError(t, err)
		expected := make([]byte, 8)
		binary.BigEndian.PutUint64(expected, 1000)
		assert.Equal(t, expected, got)
	})

	t.Run("int64 two's complement", func(t *testing.T) {
		got, err := PrepareSubkeyValue(int64(-1))
		require.NoError(t, err)
		expected := make([]byte, 8)
		binary.BigEndian.PutUint64(expected, uint64(math.MaxUint64)) //nolint:gosec
		assert.Equal(t, expected, got)
	})

	t.Run("byte array", func(t *testing.T) {
		var arr [32]byte
		for i := range arr {
			arr[i] = byte(i)
		}
		got, err := PrepareSubkeyValue(arr)
		require.NoError(t, err)
		assert.Equal(t, arr[:], got)
	})

	t.Run("pointer dereference", func(t *testing.T) {
		val := uint64(42)
		got, err := PrepareSubkeyValue(&val)
		require.NoError(t, err)
		expected := make([]byte, 8)
		binary.BigEndian.PutUint64(expected, 42)
		assert.Equal(t, expected, got)
	})

	t.Run("uint8 widened to 8 bytes", func(t *testing.T) {
		got, err := PrepareSubkeyValue(uint8(5))
		require.NoError(t, err)
		expected := make([]byte, 8)
		binary.BigEndian.PutUint64(expected, 5)
		assert.Equal(t, expected, got)
	})

	t.Run("float32 widened to float64 encoding", func(t *testing.T) {
		got, err := PrepareSubkeyValue(float32(1.5))
		require.NoError(t, err)
		expected, err := PrepareSubkeyValue(float64(1.5))
		require.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("nil error", func(t *testing.T) {
		_, err := PrepareSubkeyValue(nil)
		require.Error(t, err)
	})

	t.Run("unsupported type", func(t *testing.T) {
		_, err := PrepareSubkeyValue([]string{"a"})
		require.Error(t, err)
	})

	t.Run("bool unsupported", func(t *testing.T) {
		_, err := PrepareSubkeyValue(true)
		require.Error(t, err)
	})

	t.Run("uint128 unsupported", func(t *testing.T) {
		_, err := PrepareSubkeyValue(bin.Uint128{Lo: 1000, Hi: 0})
		require.Error(t, err)
	})
}

func TestEncodeIndexedValueMatchesPrepareSubkeyValue(t *testing.T) {
	values := []any{
		"hello",
		[]byte{1, 2, 3},
		uint64(1000),
		int64(-1),
		uint8(5),
		float32(1.5),
	}
	for _, value := range values {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			prepared, err := PrepareSubkeyValue(value)
			require.NoError(t, err)
			encoded, err := EncodeIndexedValue(value)
			require.NoError(t, err)
			assert.Equal(t, prepared, encoded)
		})
	}
}
