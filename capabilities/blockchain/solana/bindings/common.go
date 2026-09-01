package bindings

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	binary "github.com/gagliardetto/binary"

	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/solana"
)

// DecodedLog wraps a Solana log with its decoded event data.
type DecodedLog[T any] struct {
	*solana.Log
	Data T
}

const anchorCPIMethodName = "anchor:event"

type LogTriggerOptions struct {
	CPI bool
}

func AnchorCPILogTriggerConfig(programID []byte) *solana.CPIFilterConfig {
	return &solana.CPIFilterConfig{
		DestAddress: programID,
		MethodName:  []byte(anchorCPIMethodName),
	}
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
func PrepareSubkeyValue(value any) ([]byte, error) {
	return EncodeIndexedValue(value)
}
