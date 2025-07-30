package evmmock

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"
	"github.com/smartcontractkit/cre-sdk-go/cre"
)

func AddContractMock(
	address common.Address,
	clientMock *ClientCapability,
	callContract map[string]func(payload []byte) ([]byte, error),
	writeReport func(payload []byte, config *evm.GasConfig) (*evm.WriteReportReply, error),
) {
	// copy the mock so that other contract interfaces can be implemented on the same contract
	original := *clientMock

	// We need to do this for all callbacks. Some refactoring might be good...
	clientMock.CallContract = func(ctx context.Context, input *evm.CallContractRequest) (*evm.CallContractReply, error) {
		if !bytes.Equal(address[:], input.Call.To) {
			if original.CallContract == nil {
				return nil, fmt.Errorf("contract %s not found", address.Hex())
			} else {
				return original.CallContract(ctx, input)
			}
		}

		data := input.Call.Data
		if len(data) < 4 {
			return nil, errors.New("data too short")
		}

		methodID := data[:4]
		callback := callContract[string(methodID)]
		if callback == nil {
			if original.CallContract != nil {
				return original.CallContract(ctx, input)
			}
			return nil, fmt.Errorf("method with ID %x not implemented", methodID)
		}

		responsePayload, err := callback(data[4:])
		if err != nil {
			return nil, err
		}

		return &evm.CallContractReply{
			Data: responsePayload,
		}, nil
	}

	clientMock.WriteReport = func(ctx context.Context, input *evm.WriteReportRequest) (*evm.WriteReportReply, error) {
		if !bytes.Equal(address[:], input.Receiver) {
			if original.WriteReport == nil {
				return nil, fmt.Errorf("contract %s not found", address.Hex())
			} else {
				return original.WriteReport(ctx, input)
			}
		}

		return writeReport(input.Report.RawReport[cre.ReportMetadataHeaderLength:], input.GasConfig)
	}
}
