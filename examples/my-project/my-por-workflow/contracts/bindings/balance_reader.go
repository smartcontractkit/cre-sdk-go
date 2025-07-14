package bindings

import (
	_ "embed"
	"fmt"
	"log/slog"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-common/pkg/values/pb"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
)

//////////////// HOW TO CREATE A CONTRACT BINDING /////////////////////////
// 1. Create a new file in the `contracts/bindings` directory, e.g., `my_contract.go`.
// 2. Copy the template code below to modify it to suit your contract.
// 3. Compile the contact to retrieve the contract's ABI as a string.
// 4. Copy the ABI string into the `contractNameABI` variable.
// 5. Create a new functions that implements one of the contract's view functions.
// 6. Adjust the functionName to the name of the function you want to call (from the ABI).
// 7. Adjust the abi.Pack(functionName, args...) to match the function's input parameters.
// 8. Adjust the unpacking logic to match the function's output parameters: result, ok := unpacked[0].(ReturnType)
// 9. Ajust the return type of the function to match the expected output Go type.
//////////////////////////////////////////////////////////////////////////////

// NOTE: This code is not auto-generated. It is a manual binding for the BalanceReader interface.
// Example of this contract on Ethereum Sepolia: 0x4b0739c94C1389B55481cb7506c62430cA7211Cf

// Specify the ABI of the BalanceReader contract
var balanceReaderABI string = "[{\"type\":\"function\",\"name\":\"getNativeBalances\",\"inputs\":[{\"name\":\"addresses\",\"type\":\"address[]\",\"internalType\":\"address[]\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"typeAndVersion\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"}]"

type BalanceReader struct {
	evmClient *evm.Client
	abi       abi.ABI
	address   []byte
}

// Constructor for the BalanceReader contract binding
func NewBalanceReader(evmClient *evm.Client, contractAddress []byte) (*BalanceReader, error) {
	parsedAbi, err := abi.JSON(strings.NewReader(balanceReaderABI))
	if err != nil {
		return nil, err
	}
	return &BalanceReader{
		evmClient: evmClient,
		abi:       parsedAbi,
		address:   contractAddress,
	}, nil
}

// GetNativeBalances retrieves the native balances for a list of addresses, it implements the getNativeBalances function from the BalanceReader contract.
func (br *BalanceReader) GetNativeBalances(logger *slog.Logger, runtime sdk.Runtime, options *ReadOptions, addresses [][]byte) sdk.Promise[[]*big.Int] {
	functionName := "getNativeBalances"

	// Convert [][]byte to []common.Address for ABI packing
	addressList := make([]common.Address, len(addresses))
	for i, addr := range addresses {
		if len(addr) != 20 {
			return sdk.Then(nil, func(callContractReply *evm.CallContractReply) ([]*big.Int, error) {
				return nil, fmt.Errorf("invalid address length at index %d: expected 20 bytes, got %d", i, len(addr))
			})
		}
		addressList[i] = common.BytesToAddress(addr)
	}

	data, err := br.abi.Pack(functionName, addressList)
	logger.Info("Packed data for GetNativeBalances", "data", data)
	if err != nil {
		return sdk.Then(nil, func(callContractReply *evm.CallContractReply) ([]*big.Int, error) {
			return nil, fmt.Errorf("ABI packing failed: %v", err)
		})
	}

	callContractRequest := &evm.CallContractRequest{
		Call: &evm.CallMsg{
			To:   br.address,
			Data: data,
		},
	}

	// Add BlockNumber if provided in options, if not provided, it will use the latest block
	if options != nil && options.BlockNumber != nil {
		callContractRequest.BlockNumber = &pb.BigInt{
			AbsVal: options.BlockNumber.Bytes(),
		}
	}

	callContractReplyPromise := br.evmClient.CallContract(runtime, callContractRequest)

	return sdk.Then(callContractReplyPromise, func(callContractReply *evm.CallContractReply) ([]*big.Int, error) {
		unpacked, err := br.abi.Unpack(functionName, callContractReply.Data)
		if err != nil {
			return nil, err
		}

		if len(unpacked) == 0 {
			return nil, fmt.Errorf("no data unpacked from contract response")
		}

		result, ok := unpacked[0].([]*big.Int)
		if !ok {
			return nil, fmt.Errorf("unexpected type in unpacked data: expected []*big.Int, got %T", unpacked[0])
		}

		return result, nil
	})
}
