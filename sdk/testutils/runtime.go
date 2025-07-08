package testutils

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"testing"

	valuespb "github.com/smartcontractkit/chainlink-common/pkg/values/pb"
	"github.com/smartcontractkit/chainlink-common/pkg/workflows/sdk/v2/pb"
	"github.com/smartcontractkit/cre-sdk-go/internal/sdkimpl"
	consensusmock "github.com/smartcontractkit/cre-sdk-go/internal_testing/capabilities/consensus/mock"
	"github.com/smartcontractkit/cre-sdk-go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/sdk/testutils/registry"
	"google.golang.org/protobuf/proto"
)

func NewRuntime(tb testing.TB, secrets map[string]string) *TestRuntime {
	defaultConsensus, err := consensusmock.NewConsensusCapability(tb)

	// Do not override if the user provided their own consensus method
	if err == nil {
		defaultConsensus.Simple = defaultSimpleConsensus
	}

	return &TestRuntime{
		Runtime: sdkimpl.Runtime{
			RuntimeBase: sdkimpl.RuntimeBase{
				Mode:            pb.Mode_MODE_DON,
				MaxResponseSize: sdk.DefaultMaxResponseSizeBytes,
				RuntimeHelpers:  &runtimeHelpers{tb: tb, calls: map[int32]chan *pb.CapabilityResponse{}, secretsCalls: map[int32][]*pb.SecretResponse{}, secrets: secrets},
			},
		},
	}
}

type TestRuntime struct {
	sdkimpl.Runtime
	*testWriter
}

func NewEnvironment[C any](config C, runtime *TestRuntime) *sdk.Environment[C] {
	return &sdk.Environment[C]{
		NodeEnvironment: sdk.NodeEnvironment[C]{
			Config:    config,
			LogWriter: runtime.testWriter,
			Logger:    slog.New(slog.NewTextHandler(runtime.testWriter, nil)),
		},
		SecretsProvider: runtime,
	}
}

func (t *TestRuntime) SetRandomSource(source rand.Source) {
	t.RuntimeHelpers.(*runtimeHelpers).donSrc = source
}

func (t *TestRuntime) SetNodeRandomSource(source rand.Source) {
	t.RuntimeHelpers.(*runtimeHelpers).nodeSrc = source
}

func defaultSimpleConsensus(_ context.Context, input *pb.SimpleConsensusInputs) (*valuespb.Value, error) {
	switch o := input.Observation.(type) {
	case *pb.SimpleConsensusInputs_Value:
		return reportFromValue(o.Value), nil
	case *pb.SimpleConsensusInputs_Error:
		if input.Default == nil || input.Default.Value == nil {
			return nil, errors.New(o.Error)
		}

		return reportFromValue(input.Default), nil
	default:
		return nil, fmt.Errorf("unknown observation type %T", o)
	}
}

// reportFromValue will go away once the real consensus method is implemented.
func reportFromValue(result *valuespb.Value) *valuespb.Value {
	return &valuespb.Value{
		Value: &valuespb.Value_MapValue{
			MapValue: &valuespb.Map{
				Fields: map[string]*valuespb.Value{
					sdk.ConsensusResponseMapKeyMetadata: {Value: &valuespb.Value_StringValue{StringValue: "test_metadata"}},
					sdk.ConsensusResponseMapKeyPayload: {
						Value: result.Value,
					},
				},
			},
		},
	}
}

type runtimeHelpers struct {
	tb      testing.TB
	calls   map[int32]chan *pb.CapabilityResponse
	donSrc  rand.Source
	nodeSrc rand.Source

	secretsCalls map[int32][]*pb.SecretResponse
	secrets      map[string]string
}

func (rh *runtimeHelpers) GetSource(mode pb.Mode) rand.Source {
	if mode == pb.Mode_MODE_DON {
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

func (rh *runtimeHelpers) Call(request *pb.CapabilityRequest) error {
	reg := registry.GetRegistry(rh.tb)
	capability, err := reg.GetCapability(request.Id)
	if err != nil {
		return err
	}

	respCh := make(chan *pb.CapabilityResponse, 1)
	rh.calls[request.CallbackId] = respCh
	go func() {
		respCh <- capability.Invoke(rh.tb.Context(), request)
	}()
	return nil
}

func (rh *runtimeHelpers) Await(request *pb.AwaitCapabilitiesRequest, maxResponseSize uint64) (*pb.AwaitCapabilitiesResponse, error) {
	response := &pb.AwaitCapabilitiesResponse{Responses: map[int32]*pb.CapabilityResponse{}}

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
		return nil, errors.New(sdk.ResponseBufferTooSmall)
	}

	return response, errors.Join(errs...)
}

func (rh *runtimeHelpers) GetSecrets(req *pb.GetSecretsRequest, _ uint64) error {
	resp := []*pb.SecretResponse{}
	for _, secret := range req.Requests {
		key := secret.Namespace + "/" + secret.Id
		sec, ok := rh.secrets[key]
		if !ok {
			resp = append(resp, &pb.SecretResponse{
				Response: &pb.SecretResponse_Error{
					Error: &pb.SecretError{
						Id:        secret.Id,
						Namespace: secret.Namespace,
						Error:     "could not find secret " + key,
					},
				},
			})
		} else {
			resp = append(resp, &pb.SecretResponse{
				Response: &pb.SecretResponse_Secret{
					Secret: &pb.Secret{
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

func (rh *runtimeHelpers) AwaitSecrets(req *pb.AwaitSecretsRequest, _ uint64) (*pb.AwaitSecretsResponse, error) {
	response := &pb.AwaitSecretsResponse{Responses: map[int32]*pb.SecretResponses{}}

	for _, id := range req.Ids {
		resp, ok := rh.secretsCalls[id]
		if !ok {
			return nil, fmt.Errorf("could not find call with id: %d", id)
		}

		response.Responses[id] = &pb.SecretResponses{
			Responses: resp,
		}
	}

	return response, nil
}

func (rh *runtimeHelpers) SwitchModes(_ pb.Mode) {}
