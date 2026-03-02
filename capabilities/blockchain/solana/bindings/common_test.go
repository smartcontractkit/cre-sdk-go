package bindings

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

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
