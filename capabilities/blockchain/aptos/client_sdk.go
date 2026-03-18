
// Capability ID: aptos:ChainSelector:<chainSelector>@1.0.0, method "View".

package aptos

import (
	"errors"
	"fmt"
	"strconv"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	sdkpb "github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/cre"
)

type Client struct {
	ChainSelector uint64
}

func (c *Client) WriteReport(runtime cre.Runtime, input *WriteReportRequest) cre.Promise[*WriteReportReply] {
	if input == nil {
		return cre.PromiseFromResult[*WriteReportReply](nil, errors.New("nil WriteReportRequest"))
	}

	wrapped := &anypb.Any{}
	err := anypb.MarshalFrom(wrapped, input, proto.MarshalOptions{Deterministic: true})
	if err != nil {
		return cre.PromiseFromResult[*WriteReportReply](nil, err)
	}

	capID := "aptos:ChainSelector:" + strconv.FormatUint(c.ChainSelector, 10) + "@1.0.0"
	return cre.Then(runtime.CallCapability(&sdkpb.CapabilityRequest{
		Id:      capID,
		Payload: wrapped,
		Method:  "WriteReport",
	}), func(i *sdkpb.CapabilityResponse) (_ *WriteReportReply, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("aptos write decode panic: %v", r)
			}
		}()
		if i == nil {
			return nil, errors.New("nil capability response")
		}
		switch payload := i.Response.(type) {
		case *sdkpb.CapabilityResponse_Error:
			return nil, errors.New(payload.Error)
		case *sdkpb.CapabilityResponse_Payload:
			if payload.Payload == nil {
				return nil, errors.New("nil capability payload")
			}
			return decodeWriteReportReply(payload.Payload.Value)
		default:
			return nil, errors.New("unexpected response type")
		}
	})
}

func (c *Client) View(runtime cre.Runtime, input *ViewRequest) cre.Promise[*ViewReply] {
	if input == nil {
		return cre.PromiseFromResult[*ViewReply](nil, errors.New("nil ViewRequest"))
	}

	wrapped := &anypb.Any{}
	err := anypb.MarshalFrom(wrapped, input, proto.MarshalOptions{Deterministic: true})
	if err != nil {
		return cre.PromiseFromResult[*ViewReply](nil, err)
	}
	capID := "aptos:ChainSelector:" + strconv.FormatUint(c.ChainSelector, 10) + "@1.0.0"
	capCallResponse := cre.Then(runtime.CallCapability(&sdkpb.CapabilityRequest{
		Id:      capID,
		Payload: wrapped,
		Method:  "View",
	}), func(i *sdkpb.CapabilityResponse) (_ *ViewReply, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("aptos view decode panic: %v", r)
			}
		}()
		if i == nil {
			return nil, errors.New("nil capability response")
		}
		switch payload := i.Response.(type) {
		case *sdkpb.CapabilityResponse_Error:
			return nil, errors.New(payload.Error)
		case *sdkpb.CapabilityResponse_Payload:
			if payload.Payload == nil {
				return nil, errors.New("nil capability payload")
			}
			return decodeViewReply(payload.Payload.Value)
		default:
			return nil, errors.New("unexpected response type")
		}
	})
	return capCallResponse
}

// decodeWriteReportReply decodes capabilities.blockchain.aptos.v1alpha.WriteReportReply
// via protobuf wire parsing to avoid runtime reflection panics under WASM.
func decodeWriteReportReply(b []byte) (*WriteReportReply, error) {
	out := &WriteReportReply{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("decode WriteReportReply tag: %v", protowire.ParseError(n))
		}
		b = b[n:]
		switch num {
		case 1: // tx_status enum (varint)
			if typ != protowire.VarintType {
				return nil, fmt.Errorf("decode WriteReportReply.tx_status: unexpected wire type %d", typ)
			}
			v, m := protowire.ConsumeVarint(b)
			if m < 0 {
				return nil, fmt.Errorf("decode WriteReportReply.tx_status varint: %v", protowire.ParseError(m))
			}
			out.TxStatus = TxStatus(v)
			b = b[m:]
		case 2: // receiver_contract_execution_status enum (varint)
			if typ != protowire.VarintType {
				return nil, fmt.Errorf("decode WriteReportReply.receiver_contract_execution_status: unexpected wire type %d", typ)
			}
			v, m := protowire.ConsumeVarint(b)
			if m < 0 {
				return nil, fmt.Errorf("decode WriteReportReply.receiver_contract_execution_status varint: %v", protowire.ParseError(m))
			}
			status := ReceiverContractExecutionStatus(v)
			out.ReceiverContractExecutionStatus = &status
			b = b[m:]
		case 3: // tx_hash string
			if typ != protowire.BytesType {
				return nil, fmt.Errorf("decode WriteReportReply.tx_hash: unexpected wire type %d", typ)
			}
			v, m := protowire.ConsumeBytes(b)
			if m < 0 {
				return nil, fmt.Errorf("decode WriteReportReply.tx_hash bytes: %v", protowire.ParseError(m))
			}
			txHash := string(v)
			out.TxHash = &txHash
			b = b[m:]
		case 4: // transaction_fee varint
			if typ != protowire.VarintType {
				return nil, fmt.Errorf("decode WriteReportReply.transaction_fee: unexpected wire type %d", typ)
			}
			v, m := protowire.ConsumeVarint(b)
			if m < 0 {
				return nil, fmt.Errorf("decode WriteReportReply.transaction_fee varint: %v", protowire.ParseError(m))
			}
			fee := uint64(v)
			out.TransactionFee = &fee
			b = b[m:]
		case 5: // error_message string
			if typ != protowire.BytesType {
				return nil, fmt.Errorf("decode WriteReportReply.error_message: unexpected wire type %d", typ)
			}
			v, m := protowire.ConsumeBytes(b)
			if m < 0 {
				return nil, fmt.Errorf("decode WriteReportReply.error_message bytes: %v", protowire.ParseError(m))
			}
			msg := string(v)
			out.ErrorMessage = &msg
			b = b[m:]
		default:
			m := protowire.ConsumeFieldValue(num, typ, b)
			if m < 0 {
				return nil, fmt.Errorf("decode WriteReportReply skip field %d: %v", num, protowire.ParseError(m))
			}
			b = b[m:]
		}
	}
	return out, nil
}

// decodeViewReply decodes capabilities.blockchain.aptos.v1alpha.ViewReply using
// protobuf wire parsing to avoid runtime reflection panics seen under WASM.
// ViewReply currently contains a single bytes field: data = 1.
func decodeViewReply(b []byte) (*ViewReply, error) {
	out := &ViewReply{}
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, fmt.Errorf("decode ViewReply tag: %v", protowire.ParseError(n))
		}
		b = b[n:]
		switch num {
		case 1: // data bytes
			if typ != protowire.BytesType {
				return nil, fmt.Errorf("decode ViewReply.data: unexpected wire type %d", typ)
			}
			v, m := protowire.ConsumeBytes(b)
			if m < 0 {
				return nil, fmt.Errorf("decode ViewReply.data bytes: %v", protowire.ParseError(m))
			}
			out.Data = append([]byte(nil), v...)
			b = b[m:]
		default:
			m := protowire.ConsumeFieldValue(num, typ, b)
			if m < 0 {
				return nil, fmt.Errorf("decode ViewReply skip field %d: %v", num, protowire.ParseError(m))
			}
			b = b[m:]
		}
	}
	return out, nil
}
