package bindings

import (
	_ "embed"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
	"math/big"
)

// TODO replace with actual contract binding generator

//go:embed solc/compiled/IERC20.abi
var iErc20Raw string

var iErc20Api = NewIERC20Abi()

type IERC20 struct {
	Methods        Methods
	ContractInputs ContractInputs
}

func NewIERC20(contractInputs ContractInputs) IERC20 {
	ierc20 := IERC20{ContractInputs: contractInputs}
	ierc20.Methods = Methods{
		TotalSupply: TotalSupply{
			IERC20: &ierc20,
		},
	}

	return ierc20
}

type Methods struct {
	TotalSupply
}

type TotalSupply struct {
	IERC20 *IERC20
}

func (ts TotalSupply) Call(runtime sdk.Runtime, options *ReadOptions) sdk.Promise[*big.Int] {
	method := iErc20Api.Methods["totalSupply"]
	data := make([]byte, 4)
	copy(data, method.ID)
	callContractReplyPromise := ts.IERC20.ContractInputs.EVM.CallContract(runtime, &evm.CallContractRequest{
		Call: &evm.CallMsg{
			To:   ts.IERC20.ContractInputs.Address,
			Data: data,
		},
	})

	return sdk.Then(callContractReplyPromise, func(callContractReply *evm.CallContractReply) (*big.Int, error) {
		unpacked, err := method.Outputs.Unpack(callContractReply.Data)
		if err != nil {
			return nil, err
		}

		return unpacked[0].(*big.Int), nil
	})
}
