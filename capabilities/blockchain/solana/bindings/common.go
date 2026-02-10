package bindings

import (
	"crypto/sha256"

	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/solana"
)

type ForwarderReport struct {
	AccountHash [32]byte
	Payload     []byte
}

func CalculateAccountHash(accs []*solana.AccountMeta) [32]byte {
	var accounts = make([]byte, 0)
	for _, acc := range accs {
		accounts = append(accounts, acc.PublicKey[:]...)
	}
	return sha256.Sum256(accounts)
}
