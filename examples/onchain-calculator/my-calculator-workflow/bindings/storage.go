// in onchain-calculator/my-calculator-workflow/bindings/storage.go
package bindings

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
)

// The ABI for the get() function of our Storage contract.
var StorageABI = `[{"inputs":[],"name":"get","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`

// Storage is a Go struct that wraps our contract.
type Storage struct {
	evmClient *evm.Client
	abi       abi.ABI
	address   []byte
}

// NewStorage is a constructor function that initializes our binding.
func NewStorage(contractAddress []byte, evmClient *evm.Client) (*Storage, error) {
	parsedAbi, err := abi.JSON(strings.NewReader(StorageABI))
	if err != nil {
		return nil, err
	}
	return &Storage{
		evmClient: evmClient,
		abi:       parsedAbi,
		address:   contractAddress,
	}, nil
}

// Get is a method on our binding that corresponds to the `get` function
// in the smart contract. It handles the ABI encoding and decoding for us.
func (s *Storage) Get(runtime sdk.Runtime) sdk.Promise[*big.Int] {
	// 1. ABI-encode the call
	packedData, err := s.abi.Pack("get")
	if err != nil {
		return sdk.PromiseFromResult[*big.Int](nil, err)
	}

	// 2. Prepare the request for the generic evm.Client
	req := &evm.CallContractRequest{
		Call: &evm.CallMsg{
			To:   s.address,
			Data: packedData,
		},
	}
	promise := s.evmClient.CallContract(runtime, req)

	// 3. Decode the response and return a type-safe promise
	return sdk.Then(promise, func(reply *evm.CallContractReply) (*big.Int, error) {
		// 4. ABI-decode the raw bytes into a Go type
		unpacked, err := s.abi.Unpack("get", reply.Data)
		if err != nil {
			return nil, err
		}

		result, ok := unpacked[0].(*big.Int)
		if !ok {
			return nil, fmt.Errorf("unexpected type in unpacked data")
		}
		return result, nil
	})
}
