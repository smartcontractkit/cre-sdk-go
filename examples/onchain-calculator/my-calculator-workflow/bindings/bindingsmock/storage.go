package bindingsmock

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	evmmock "github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm/mock"
	"my-calculator-workflow/bindings"
)

// TODO replace with actual contract binding generator

type StorageMock struct {
	Get func() (*big.Int, error)
}

// NewStorageMock creates a new StorageMock.
func NewStorageMock(address common.Address, clientMock *evmmock.ClientCapability) *StorageMock {
	storage := &StorageMock{}
	a := bindings.NewStorageAbi()
	get := a.Methods["get"]
	funcMap := map[string]func([]byte) ([]byte, error){
		string(get.ID): func(payload []byte) ([]byte, error) {
			if (storage.Get) == nil {
				// TODO better if we can match the EVM's error
				return nil, errors.New("method not found on the contract")
			}

			result, err := storage.Get()
			if err != nil {
				return nil, err
			}
			return get.Outputs.Pack(result)
		},
	}
	evmmock.AddContractMock(address, clientMock, funcMap, nil)
	return storage
}
