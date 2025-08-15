package testutils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	valuespb "github.com/smartcontractkit/chainlink-protos/cre/go/values/pb"
	"github.com/smartcontractkit/cre-sdk-go/cre"
	"github.com/smartcontractkit/cre-sdk-go/cre/testutils/registry"
	"github.com/smartcontractkit/cre-sdk-go/internal/sdkimpl"
	consensusmock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/consensus/mock"

	"google.golang.org/protobuf/proto"
)

func NewRuntime(tb testing.TB, secrets map[string]string) *TestRuntime {
	defaultConsensus, err := consensusmock.NewConsensusCapability(tb)

	// Do not override if the user provided their own consensus method
	if err == nil {
		defaultConsensus.Simple = defaultSimpleConsensus
		defaultConsensus.Report = defaultReport
	}

	if secrets == nil {
		secrets = map[string]string{}
	}

	tw := &testWriter{}

	return &TestRuntime{
		testWriter: tw,
		Runtime: sdkimpl.Runtime{
			RuntimeBase: sdkimpl.RuntimeBase{
				Mode:            sdk.Mode_MODE_DON,
				MaxResponseSize: cre.DefaultMaxResponseSizeBytes,
				RuntimeHelpers:  &runtimeHelpers{tb: tb, calls: map[int32]chan *sdk.CapabilityResponse{}, secretsCalls: map[int32][]*sdk.SecretResponse{}, secrets: secrets},
				Lggr:            slog.New(slog.NewTextHandler(tw, &slog.HandlerOptions{})),
			},
		},
	}
}

type TestRuntime struct {
	sdkimpl.Runtime
	testWriter *testWriter
}

var _ cre.Runtime = (*TestRuntime)(nil)

func (t *TestRuntime) GetLogs() [][]byte {
	logs := make([][]byte, len(t.testWriter.logs))
	for i, log := range t.testWriter.logs {
		logs[i] = make([]byte, len(log))
		copy(logs[i], log)
	}
	return logs
}

func (t *TestRuntime) SetRandomSource(source rand.Source) {
	t.RuntimeHelpers.(*runtimeHelpers).donSrc = source
}

func (t *TestRuntime) SetNodeRandomSource(source rand.Source) {
	t.RuntimeHelpers.(*runtimeHelpers).nodeSrc = source
}

func defaultSimpleConsensus(_ context.Context, input *sdk.SimpleConsensusInputs) (*valuespb.Value, error) {
	switch o := input.Observation.(type) {
	case *sdk.SimpleConsensusInputs_Value:
		return reportFromValue(o.Value), nil
	case *sdk.SimpleConsensusInputs_Error:
		if input.Default == nil || input.Default.Value == nil {
			return nil, errors.New(o.Error)
		}

		return reportFromValue(input.Default), nil
	default:
		return nil, fmt.Errorf("unknown observation type %T", o)
	}
}

func defaultReport(_ context.Context, input *sdk.ReportRequest) (*sdk.ReportResponse, error) {
	metadata := createTestReportMetadata()
	rawReportBytes := append(metadata, input.EncodedPayload...)
	defaultSigs := [][]byte{
		[]byte("default_signature_1"),
		[]byte("default_signature_2"),
	}
	return &sdk.ReportResponse{
		RawReport: rawReportBytes,
		Sigs: []*sdk.AttributedSignature{
			{
				Signature: defaultSigs[0],
				SignerId:  0,
			},
			{
				Signature: defaultSigs[1],
				SignerId:  1,
			},
		},
	}, nil
}

// createTestReportMetadata generates a byte slice for metadata
// that is sdk.ReportMetadataHeaderLength long and has an assertable pattern.
func createTestReportMetadata() []byte {
	metadata := make([]byte, cre.ReportMetadataHeaderLength)
	for i := range cre.ReportMetadataHeaderLength {
		metadata[i] = byte(i % 256)
	}
	return metadata
}

// reportFromValue will go away once the real consensus method is implemented.
func reportFromValue(result *valuespb.Value) *valuespb.Value {
	return &valuespb.Value{Value: result.Value}
}

type runtimeHelpers struct {
	tb      testing.TB
	calls   map[int32]chan *sdk.CapabilityResponse
	donSrc  rand.Source
	nodeSrc rand.Source

	secretsCalls map[int32][]*sdk.SecretResponse
	secrets      map[string]string
}

func (rh *runtimeHelpers) GetSource(mode sdk.Mode) rand.Source {
	if mode == sdk.Mode_MODE_DON {
		if rh.donSrc == nil {
			rh.donSrc = rand.NewSource(123)
		}
		return rh.donSrc
	}

	if rh.nodeSrc == nil {
		rh.nodeSrc = rand.NewSource(456)
	}
	return rh.nodeSrc
}

func (rh *runtimeHelpers) Call(request *sdk.CapabilityRequest) error {
	reg := registry.GetRegistry(rh.tb)
	capability, err := reg.GetCapability(request.Id)
	if err != nil {
		return err
	}

	respCh := make(chan *sdk.CapabilityResponse, 1)
	rh.calls[request.CallbackId] = respCh
	go func() {
		respCh <- capability.Invoke(rh.tb.Context(), request)
	}()
	return nil
}

func (rh *runtimeHelpers) Await(request *sdk.AwaitCapabilitiesRequest, maxResponseSize uint64) (*sdk.AwaitCapabilitiesResponse, error) {
	response := &sdk.AwaitCapabilitiesResponse{Responses: map[int32]*sdk.CapabilityResponse{}}

	var errs []error
	for _, id := range request.Ids {
		ch, ok := rh.calls[id]
		if !ok {
			errs = append(errs, fmt.Errorf("no call found for %d", id))
			continue
		}
		select {
		case resp := <-ch:
			response.Responses[id] = resp
		case <-rh.tb.Context().Done():
			return nil, rh.tb.Context().Err()
		}
	}

	bytes, _ := proto.Marshal(response)
	if len(bytes) > int(maxResponseSize) {
		return nil, errors.New(cre.ResponseBufferTooSmall)
	}

	return response, errors.Join(errs...)
}

func (rh *runtimeHelpers) GetSecrets(req *sdk.GetSecretsRequest, _ uint64) error {
	var resp []*sdk.SecretResponse
	for _, secret := range req.Requests {
		key := secret.Namespace + "/" + secret.Id
		sec, ok := rh.secrets[key]
		if !ok {
			resp = append(resp, &sdk.SecretResponse{
				Response: &sdk.SecretResponse_Error{
					Error: &sdk.SecretError{
						Id:        secret.Id,
						Namespace: secret.Namespace,
						Error:     "could not find secret " + key,
					},
				},
			})
		} else {
			resp = append(resp, &sdk.SecretResponse{
				Response: &sdk.SecretResponse_Secret{
					Secret: &sdk.Secret{
						Id:        secret.Id,
						Namespace: secret.Namespace,
						Value:     sec,
					},
				},
			})
		}
	}

	rh.secretsCalls[req.CallbackId] = resp
	return nil
}

func (rh *runtimeHelpers) AwaitSecrets(req *sdk.AwaitSecretsRequest, _ uint64) (*sdk.AwaitSecretsResponse, error) {
	response := &sdk.AwaitSecretsResponse{Responses: map[int32]*sdk.SecretResponses{}}

	for _, id := range req.Ids {
		resp, ok := rh.secretsCalls[id]
		if !ok {
			return nil, fmt.Errorf("could not find call with id: %d", id)
		}

		response.Responses[id] = &sdk.SecretResponses{
			Responses: resp,
		}
	}

	return response, nil
}

func (rh *runtimeHelpers) SwitchModes(_ sdk.Mode) {}

func (rh *runtimeHelpers) Now() time.Time {
	return time.Time{}
}
