package bindingsmock

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm/mock"
	"my-por-workflow/contracts/bindings"
)

// TODO replace with actual contract binding generator

type MessageEmitterMock struct {
	// Other methods would be generated here as well
	GetLastMessage func(emitter common.Address) (string, error)
}

// The exact signature of the helper is TBD.

func MessageEmitterMessageEmittedTrigger(
	address common.Address,
	timestamp *big.Int,
	message string,
) *evm.Log {
	a := bindings.NewMessageEmitterAbi()
	messageEmitted := a.Events["MessageEmitted"]
	data, err := messageEmitted.Inputs.NonIndexed().Pack(message)
	fmt.Println(err)
	return &evm.Log{
		Address: address.Bytes(),
		Topics:  [][]byte{messageEmitted.ID.Bytes(), address.Bytes(), common.LeftPadBytes(timestamp.Bytes(), 32)},
		Data:    data,
		// TODO, in future, include the tx data so it can be used?
	}
}

// NewMessageEmitterMock creates a new MessageEmitter mock.
func NewMessageEmitterMock(address common.Address, clientMock *evmmock.ClientCapability) *MessageEmitterMock {
	messageEmitter := &MessageEmitterMock{}
	a := bindings.NewMessageEmitterAbi()
	lastMessage := a.Methods["getLastMessage"]
	funcMap := map[string]func([]byte) ([]byte, error){
		string(lastMessage.ID): func(payload []byte) ([]byte, error) {
			if (messageEmitter.GetLastMessage) == nil {
				// TODO better if we can match the EVM's error
				return nil, errors.New("method not found on the contract")
			}

			inputs, err := lastMessage.Inputs.Unpack(payload)
			if err != nil {
				return nil, err
			}
			addresses := inputs[0].(common.Address)

			result, err := messageEmitter.GetLastMessage(addresses)
			if err != nil {
				return nil, err
			}
			return lastMessage.Outputs.Pack(result)
		},
	}
	evmmock.AddContractMock(address, clientMock, funcMap, nil)
	return messageEmitter
}
