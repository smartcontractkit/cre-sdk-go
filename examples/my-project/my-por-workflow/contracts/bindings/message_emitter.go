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
// 8. Adjust the unpacking logic to match the function's output parameters: result, ok := unpacked[0].(ResultType)
// 9. Ajust the return type of the function to match the expected output Go type.
// 10. For interpreting emitted events, follow a function example to create a custom implementation for your contract.
//////////////////////////////////////////////////////////////////////////////

// NOTE: This code is not auto-generated. It is a manual binding for the MessageEmitter interface.
// Example of this contract on Ethereum Sepolia: 0x1d598672486ecB50685Da5497390571Ac4E93FDc

// Specify the ABI of the MessageEmitter contract
var messageEmitterABI string = "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"emitter\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"message\",\"type\":\"string\"}],\"name\":\"MessageEmitted\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"message\",\"type\":\"string\"}],\"name\":\"emitMessage\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"emitter\",\"type\":\"address\"}],\"name\":\"getLastMessage\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"emitter\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"name\":\"getMessage\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"typeAndVersion\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]"

type MessageEmitter struct {
	evmClient *evm.Client
	abi       abi.ABI
	address   []byte
}

// Constructor for the MessageEmitter contract binding
func NewMessageEmitter(evmClient *evm.Client, contractAddress []byte) (*MessageEmitter, error) {
	parsedAbi, err := abi.JSON(strings.NewReader(messageEmitterABI))
	if err != nil {
		return nil, err
	}
	return &MessageEmitter{
		evmClient: evmClient,
		abi:       parsedAbi,
		address:   contractAddress,
	}, nil
}

func (br *MessageEmitter) GetMessage(logger *slog.Logger, runtime sdk.Runtime, options *ReadOptions, emitter []byte, timestamp *big.Int) sdk.Promise[string] {
	functionName := "getMessage"

	data, err := br.abi.Pack(functionName, common.BytesToAddress(emitter), timestamp)
	logger.Info("Packed data for getMessage", "data", data)
	if err != nil {
		return sdk.Then(nil, func(callContractReply *evm.CallContractReply) (string, error) {
			return "", fmt.Errorf("ABI packing failed: %v", err)
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

	return sdk.Then(callContractReplyPromise, func(callContractReply *evm.CallContractReply) (string, error) {
		unpacked, err := br.abi.Unpack(functionName, callContractReply.Data)
		if err != nil {
			return "", err
		}

		if len(unpacked) == 0 {
			return "", fmt.Errorf("no data unpacked from contract response")
		}

		result, ok := unpacked[0].(string)
		if !ok {
			return "", fmt.Errorf("unexpected type in unpacked data: expected *string, got %T", unpacked[0])
		}

		return result, nil
	})
}

func (br *MessageEmitter) GetLastMessage(logger *slog.Logger, runtime sdk.Runtime, options *ReadOptions, emitter []byte) sdk.Promise[string] {
	functionName := "getLastMessage"

	data, err := br.abi.Pack(functionName, common.BytesToAddress(emitter))
	logger.Info("Packed data for getLastMessage", "data", data)
	if err != nil {
		return sdk.Then(nil, func(callContractReply *evm.CallContractReply) (string, error) {
			return "", fmt.Errorf("ABI packing failed: %v", err)
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

	return sdk.Then(callContractReplyPromise, func(callContractReply *evm.CallContractReply) (string, error) {
		unpacked, err := br.abi.Unpack(functionName, callContractReply.Data)
		if err != nil {
			return "", err
		}

		if len(unpacked) == 0 {
			return "", fmt.Errorf("no data unpacked from contract response")
		}

		result, ok := unpacked[0].(string)
		if !ok {
			return "", fmt.Errorf("unexpected type in unpacked data: expected *string, got %T", unpacked[0])
		}

		return result, nil
	})
}

func (br *MessageEmitter) ReadEmittedMessage(logger *slog.Logger, topics [][]byte, data []byte) (string, error) {
	eventName := "MessageEmitted"
	unpacked, err := br.abi.Unpack(eventName, data)
	if err != nil {
		logger.Error("Failed to unpack MessageEmitted event", "error", err)
		return "", err
	}

	message, ok := unpacked[0].(string)
	if !ok {
		logger.Error("Failed to interpret MessageEmitted event data")
		return "", fmt.Errorf("unexpected type in unpacked data: expected string, got %T", unpacked[0])
	}

	if len(topics) < 3 {
		logger.Error("Not enough topics for MessageEmitted event", "topics", topics)
		return "", fmt.Errorf("not enough topics for MessageEmitted event, expected at least 3, got %d", len(topics))
	}

	emitter := common.BytesToAddress(topics[1])
	timestamp := new(big.Int).SetBytes(topics[2])

	logger.Info("Reading emitted message",
		"emitter", emitter.Hex(),
		"timestamp", timestamp.String(),
		"message", message)

	return message, nil
}
