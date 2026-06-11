package bindings

import (
	"bytes"
	"crypto/sha256"
	encbinary "encoding/binary"
	"fmt"
	"math"
	"reflect"

	binary "github.com/gagliardetto/binary"

	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/solana"
)

// DecodedLog wraps a Solana log with its decoded event data.
type DecodedLog[T any] struct {
	*solana.Log
	Data T
}

// LogTriggerOptions holds optional configuration for log trigger registration.
type LogTriggerOptions struct {
	CpiFilterConfig *solana.CPIFilterConfig
}

// ForwarderReport represents the Borsh-serialized report format expected by
// the Solana keystone-forwarder program's on_report instruction.
type ForwarderReport struct {
	AccountHash [32]byte
	Payload     []byte
}

// MarshalWithEncoder serializes the ForwarderReport using the provided Borsh encoder.
func (obj ForwarderReport) MarshalWithEncoder(encoder *binary.Encoder) (err error) {
	// Serialize `AccountHash`:
	err = encoder.Encode(obj.AccountHash)
	if err != nil {
		return fmt.Errorf("field AccountHash: %w", err)
	}
	// Serialize `Payload`:
	err = encoder.Encode(obj.Payload)
	if err != nil {
		return fmt.Errorf("field Payload: %w", err)
	}
	return nil
}

// Marshal serializes the ForwarderReport into Borsh-encoded bytes.
func (obj ForwarderReport) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	encoder := binary.NewBorshEncoder(buf)
	err := obj.MarshalWithEncoder(encoder)
	if err != nil {
		return nil, fmt.Errorf("error while encoding ForwarderReport: %w", err)
	}
	return buf.Bytes(), nil
}

// CalculateAccountsHash computes the SHA-256 hash of the concatenated public
// keys of the given accounts, matching the on-chain account hash verification.
func CalculateAccountsHash(accs []*solana.AccountMeta) [32]byte {
	accounts := make([]byte, 0, len(accs)*32)
	for _, acc := range accs {
		if acc == nil {
			continue
		}
		accounts = append(accounts, acc.PublicKey...)
	}
	return sha256.Sum256(accounts)
}

// PrepareSubkeyValue encodes a filter value for use in a SubkeyConfig ValueComparator.
// Encoding rules match the Solana log poller IndexedValue format.
func PrepareSubkeyValue(value any) ([]byte, error) {
	if value == nil {
		return nil, fmt.Errorf("subkey value cannot be nil")
	}

	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	case bool:
		if v {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("subkey value cannot be nil")
		}
		return PrepareSubkeyValue(rv.Elem().Interface())
	}

	// byte arrays (e.g. solana.PublicKey / [32]byte)
	if rv.Kind() == reflect.Array && rv.Type().Elem().Kind() == reflect.Uint8 {
		result := make([]byte, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			result[i] = byte(rv.Index(i).Uint())
		}
		return result, nil
	}

	if rv.CanUint() {
		buf := make([]byte, 8)
		encbinary.BigEndian.PutUint64(buf, rv.Uint())
		return buf, nil
	}
	if rv.CanInt() {
		buf := make([]byte, 8)
		encbinary.BigEndian.PutUint64(buf, uint64(rv.Int())) //nolint:gosec // two's complement encoding
		return buf, nil
	}
	if rv.CanFloat() {
		buf := make([]byte, 8)
		f := rv.Float()
		if f > 0 {
			encbinary.BigEndian.PutUint64(buf, math.Float64bits(f)+math.MaxInt64+1)
		} else {
			encbinary.BigEndian.PutUint64(buf, math.MaxInt64+1-math.Float64bits(f))
		}
		return buf, nil
	}

	return nil, fmt.Errorf("unsupported subkey value type %T", value)
}
