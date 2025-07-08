package bindings

import (
	_ "embed"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/smartcontractkit/cre-sdk-go/sdk"

	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"
)

// TODO replace with actual contract binding generator

//go:embed solc/compiled/IReserveManager.abi
var iReserveManagerRaw string

var iReserveManagerApi = NewIReserveManagerAbi()

func NewIReserveManagerAbi() abi.ABI {
	a, _ := abi.JSON(strings.NewReader(iReserveManagerRaw))
	return a
}

type IReserveManagerCodec interface {
}

type iReserveManagerCodec struct {
	abi abi.ABI
}

type IReserverManager struct {
	codec          IReserveManagerCodec
	ContractInputs ContractInputs
}

func NewIReserveManagerCodec() (IReserveManagerCodec, error) {
	return iReserveManagerCodec{abi: NewIReserveManagerAbi()}, nil
}

func NewIReserveManager(contracInputs ContractInputs) IReserverManager {
	codec, _ := NewIReserveManagerCodec()
	reserveManager := IReserverManager{ContractInputs: contracInputs, codec: codec}
	return reserveManager
}

type UpdateReserves struct {
	reserveManager *IReserverManager
}

type UpdateReservesStruct struct {
	TotalMinted  *big.Int
	TotalReserve *big.Int
}

func (irm IReserverManager) WriteReportUpdateReserves(runtime sdk.Runtime, updateReserves UpdateReservesStruct, options *WriteOptions) sdk.Promise[*evm.WriteReportReply] {
	// Pack the complete function call (selector + args)
	body, err := iReserveManagerApi.Pack("updateReserves", updateReserves)
	if err != nil {
		return sdk.PromiseFromResult[*evm.WriteReportReply](nil, err)
	}

	writeReportReplyPromise := irm.ContractInputs.EVM.WriteReport(runtime, &evm.WriteReportRequest{
		Receiver: irm.ContractInputs.Address,
		Report: &evm.SignedReport{
			RawReport:     body,
			ReportContext: []byte{},
			Signatures:    [][]byte{},
			Id:            []byte{},
		},
	})

	return writeReportReplyPromise
}
