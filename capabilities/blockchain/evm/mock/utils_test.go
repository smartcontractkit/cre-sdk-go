package evmmock_test

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm"
	"github.com/smartcontractkit/cre-sdk-go/capabilities/blockchain/evm/mock"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var anyHash = common.Hash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

func TestCallContract_MatchingAddress(t *testing.T) {
	mockAddr := common.HexToAddress("0xabc")
	methodID := []byte{0x01, 0x02, 0x03, 0x04}
	callHit := false

	clientMock := &evmmock.ClientCapability{}
	evmmock.AddContractMock(
		mockAddr,
		clientMock,
		map[string]func([]byte) ([]byte, error){
			string(methodID): func(payload []byte) ([]byte, error) {
				callHit = true
				assert.Equal(t, []byte("hello"), payload)
				return []byte("world"), nil
			},
		},
		func(_ []byte, _ *evm.GasConfig) (*evm.WriteReportReply, error) {
			t.Fatal("should not call WriteReport in CallContract")
			return nil, nil
		},
	)

	resp, err := clientMock.CallContract(context.Background(), &evm.CallContractRequest{
		Call: &evm.CallMsg{
			To:   mockAddr.Bytes(),
			Data: append(methodID, []byte("hello")...),
		},
	})
	require.NoError(t, err)
	assert.True(t, callHit)
	assert.Equal(t, []byte("world"), resp.Data)
}

func TestCallContract_ShortData(t *testing.T) {
	mockAddr := common.HexToAddress("0xabc")

	clientMock := &evmmock.ClientCapability{}
	evmmock.AddContractMock(mockAddr, clientMock, nil, nil)

	_, err := clientMock.CallContract(context.Background(), &evm.CallContractRequest{
		Call: &evm.CallMsg{
			To:   mockAddr.Bytes(),
			Data: []byte{0x01, 0x02, 0x03},
		},
	})
	require.EqualError(t, err, "data too short")
}

func TestCallContract_MethodNotImplemented_NoFallback(t *testing.T) {
	mockAddr := common.HexToAddress("0xabc")

	clientMock := &evmmock.ClientCapability{}
	evmmock.AddContractMock(mockAddr, clientMock, map[string]func([]byte) ([]byte, error){}, nil)

	_, err := clientMock.CallContract(context.Background(), &evm.CallContractRequest{
		Call: &evm.CallMsg{
			To:   mockAddr.Bytes(),
			Data: []byte{0xde, 0xad, 0xbe, 0xef},
		},
	})
	require.EqualError(t, err, "method with ID deadbeef not implemented")
}

func TestCallContract_MismatchedAddress_WithFallback(t *testing.T) {
	mockAddr := common.HexToAddress("0xabc")
	otherAddr := common.HexToAddress("0xdef")
	expected := []byte("fallback")

	clientMock := &evmmock.ClientCapability{
		CallContract: func(_ context.Context, _ *evm.CallContractRequest) (*evm.CallContractReply, error) {
			return &evm.CallContractReply{Data: expected}, nil
		},
	}
	evmmock.AddContractMock(mockAddr, clientMock, nil, nil)

	resp, err := clientMock.CallContract(context.Background(), &evm.CallContractRequest{
		Call: &evm.CallMsg{
			To:   otherAddr.Bytes(),
			Data: []byte{0xde, 0xad, 0xbe, 0xef},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, expected, resp.Data)
}

func TestCallContract_MismatchedAddress_NoFallback(t *testing.T) {
	mockAddr := common.HexToAddress("0xabc")
	otherAddr := common.HexToAddress("0xdef")

	clientMock := &evmmock.ClientCapability{}
	evmmock.AddContractMock(mockAddr, clientMock, nil, nil)

	_, err := clientMock.CallContract(context.Background(), &evm.CallContractRequest{
		Call: &evm.CallMsg{
			To:   otherAddr.Bytes(),
			Data: []byte{0xde, 0xad, 0xbe, 0xef},
		},
	})
	require.EqualError(t, err, "contract 0x0000000000000000000000000000000000000aBc not found")
}

func TestWriteReport_MatchingAddress(t *testing.T) {
	mockAddr := common.HexToAddress("0xabc")
	payload := []byte("context")
	gasCfg := &evm.GasConfig{GasLimit: 12345}
	writeCalled := false

	clientMock := &evmmock.ClientCapability{}
	evmmock.AddContractMock(
		mockAddr,
		clientMock,
		nil,
		func(report []byte, cfg *evm.GasConfig) (*evm.WriteReportReply, error) {
			writeCalled = true
			assert.Equal(t, payload, report)
			assert.Equal(t, gasCfg, cfg)
			return &evm.WriteReportReply{TxStatus: evm.TxStatus_TX_STATUS_SUCCESS, TxHash: anyHash[:]}, nil
		},
	)

	resp, err := clientMock.WriteReport(context.Background(), &evm.WriteReportRequest{
		Receiver:  mockAddr.Bytes(),
		Report:    &evm.SignedReport{RawReport: append(testReportMetadata(), payload...)},
		GasConfig: gasCfg,
	})
	require.NoError(t, err)
	assert.True(t, writeCalled)
	assert.Equal(t, evm.TxStatus_TX_STATUS_SUCCESS, resp.TxStatus)
	assert.ElementsMatch(t, anyHash, resp.TxHash)
}

func TestWriteReport_MismatchedAddress_WithFallback(t *testing.T) {
	mockAddr := common.HexToAddress("0xabc")
	otherAddr := common.HexToAddress("0xdef")

	expected := &evm.WriteReportReply{TxStatus: evm.TxStatus_TX_STATUS_REVERTED, TxHash: anyHash[:]}
	clientMock := &evmmock.ClientCapability{
		WriteReport: func(_ context.Context, _ *evm.WriteReportRequest) (*evm.WriteReportReply, error) {
			return expected, nil
		},
	}
	evmmock.AddContractMock(mockAddr, clientMock, nil, nil)

	resp, err := clientMock.WriteReport(context.Background(), &evm.WriteReportRequest{
		Receiver:  otherAddr.Bytes(),
		Report:    &evm.SignedReport{RawReport: append(testReportMetadata(), 'x')},
		GasConfig: &evm.GasConfig{},
	})
	require.NoError(t, err)
	assert.Equal(t, expected.TxStatus, resp.TxStatus)
	assert.ElementsMatch(t, expected.TxHash, resp.TxHash)
}

func TestWriteReport_MismatchedAddress_NoFallback(t *testing.T) {
	mockAddr := common.HexToAddress("0xabc")
	otherAddr := common.HexToAddress("0xdef")

	clientMock := &evmmock.ClientCapability{}
	evmmock.AddContractMock(mockAddr, clientMock, nil, nil)

	_, err := clientMock.WriteReport(context.Background(), &evm.WriteReportRequest{
		Receiver:  otherAddr.Bytes(),
		Report:    &evm.SignedReport{RawReport: append(testReportMetadata(), 'x')},
		GasConfig: &evm.GasConfig{},
	})
	require.EqualError(t, err, "contract 0x0000000000000000000000000000000000000aBc not found")
}

func testReportMetadata() []byte {
	metadata := make([]byte, sdk.ReportMetadataHeaderLength)
	for i := 0; i < sdk.ReportMetadataHeaderLength; i++ {
		metadata[i] = byte(i % 256)
	}
	return metadata
}
