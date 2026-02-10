package bindings

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	"github.com/gagliardetto/anchor-go/errors"
	binary "github.com/gagliardetto/binary"

	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/solana"
)

type ForwarderReport struct {
	AccountHash [32]byte
	Payload     []byte
}

func (obj ForwarderReport) MarshalWithEncoder(encoder *binary.Encoder) (err error) {
	// Serialize `AccountHash`:
	err = encoder.Encode(obj.AccountHash)
	if err != nil {
		return errors.NewField("AccountHash", err)
	}
	// Serialize `Payload`:
	err = encoder.Encode(obj.Payload)
	if err != nil {
		return errors.NewField("Payload", err)
	}
	return nil
}

func (obj ForwarderReport) Marshal() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	encoder := binary.NewBorshEncoder(buf)
	err := obj.MarshalWithEncoder(encoder)
	if err != nil {
		return nil, fmt.Errorf("error while encoding ForwarderReport: %w", err)
	}
	return buf.Bytes(), nil
}

func CalculateAccountsHash(accs []*solana.AccountMeta) [32]byte {
	var accounts = make([]byte, 0)
	for _, acc := range accs {
		accounts = append(accounts, acc.PublicKey[:]...)
	}
	return sha256.Sum256(accounts)
}
