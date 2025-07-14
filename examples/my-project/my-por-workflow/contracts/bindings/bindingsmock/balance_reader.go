package bindingsmock

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	evmmock "github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm/mock"
	"my-por-workflow/contracts/bindings"
)

// TODO replace with actual contract binding generator

type BalanceReaderMock struct {
	GetNativeBalances func(addresses []common.Address) ([]*big.Int, error)
}

// NewBalanceReaderMock creates a new BalanceReaderMock.
func NewBalanceReaderMock(address common.Address, clientMock *evmmock.ClientCapability) *BalanceReaderMock {
	balanceReader := &BalanceReaderMock{}
	a := bindings.NewBalanceReaderAbi()
	getNativeBalances := a.Methods["getNativeBalances"]
	funcMap := map[string]func([]byte) ([]byte, error){
		string(getNativeBalances.ID): func(payload []byte) ([]byte, error) {
			if (balanceReader.GetNativeBalances) == nil {
				// TODO better if we can match the EVM's error
				return nil, errors.New("method not found on the contract")
			}

			inputs, err := getNativeBalances.Inputs.Unpack(payload)
			if err != nil {
				return nil, err
			}
			addresses := inputs[0].([]common.Address)

			result, err := balanceReader.GetNativeBalances(addresses)
			if err != nil {
				return nil, err
			}
			return getNativeBalances.Outputs.Pack(result)
		},
	}
	evmmock.AddContractMock(address, clientMock, funcMap, nil)
	return balanceReader
}
