package aptos

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

func TestDecodeWriteReportReply_NewWireShape(t *testing.T) {
	t.Parallel()

	txHash := "0xabc123"
	txFee := uint64(42)
	errMsg := "receiver execution failed"
	replyBytes, err := proto.Marshal(&WriteReportReply{
		TxStatus:                        TxStatus_TX_STATUS_ABORTED,
		ReceiverContractExecutionStatus: ReceiverContractExecutionStatus_RECEIVER_CONTRACT_EXECUTION_STATUS_REVERTED.Enum(),
		TxHash:                          &txHash,
		TransactionFee:                  &txFee,
		ErrorMessage:                    &errMsg,
	})
	require.NoError(t, err)

	reply, err := decodeWriteReportReply(replyBytes)
	require.NoError(t, err)
	require.Equal(t, TxStatus_TX_STATUS_ABORTED, reply.TxStatus)
	require.NotNil(t, reply.ReceiverContractExecutionStatus)
	require.Equal(t, ReceiverContractExecutionStatus_RECEIVER_CONTRACT_EXECUTION_STATUS_REVERTED, *reply.ReceiverContractExecutionStatus)
	require.NotNil(t, reply.TxHash)
	require.Equal(t, txHash, *reply.TxHash)
	require.NotNil(t, reply.TransactionFee)
	require.Equal(t, txFee, *reply.TransactionFee)
	require.NotNil(t, reply.ErrorMessage)
	require.Equal(t, errMsg, *reply.ErrorMessage)
}

func TestDecodeWriteReportReply_DeployedWireShape(t *testing.T) {
	t.Parallel()

	var replyBytes []byte
	replyBytes = protowire.AppendTag(replyBytes, 1, protowire.VarintType)
	replyBytes = protowire.AppendVarint(replyBytes, uint64(TxStatus_TX_STATUS_ABORTED))
	replyBytes = protowire.AppendTag(replyBytes, 2, protowire.BytesType)
	replyBytes = protowire.AppendString(replyBytes, "0xabc123")
	replyBytes = protowire.AppendTag(replyBytes, 3, protowire.VarintType)
	replyBytes = protowire.AppendVarint(replyBytes, 42)
	replyBytes = protowire.AppendTag(replyBytes, 4, protowire.BytesType)
	replyBytes = protowire.AppendString(replyBytes, "receiver execution failed")
	replyBytes = protowire.AppendTag(replyBytes, 5, protowire.VarintType)
	replyBytes = protowire.AppendVarint(replyBytes, uint64(ReceiverContractExecutionStatus_RECEIVER_CONTRACT_EXECUTION_STATUS_REVERTED))

	reply, err := decodeWriteReportReply(replyBytes)
	require.NoError(t, err)
	require.Equal(t, TxStatus_TX_STATUS_ABORTED, reply.TxStatus)
	require.NotNil(t, reply.ReceiverContractExecutionStatus)
	require.Equal(t, ReceiverContractExecutionStatus_RECEIVER_CONTRACT_EXECUTION_STATUS_REVERTED, *reply.ReceiverContractExecutionStatus)
	require.NotNil(t, reply.TxHash)
	require.Equal(t, "0xabc123", *reply.TxHash)
	require.NotNil(t, reply.TransactionFee)
	require.Equal(t, uint64(42), *reply.TransactionFee)
	require.NotNil(t, reply.ErrorMessage)
	require.Equal(t, "receiver execution failed", *reply.ErrorMessage)
}
