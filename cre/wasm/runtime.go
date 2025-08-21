package wasm

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"time"
	"unsafe"

	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"github.com/smartcontractkit/cre-sdk-go/internal/sdkimpl"
	"google.golang.org/protobuf/proto"
)

const (
	ErrnoSuccess = 0
)

type runtimeInternals interface {
	callCapability(req unsafe.Pointer, reqLen int32) int64
	awaitCapabilities(awaitRequest unsafe.Pointer, awaitRequestLen int32, responseBuffer unsafe.Pointer, maxResponseLen int32) int64
	getSecrets(req unsafe.Pointer, reqLen int32, responseBuffer unsafe.Pointer, maxResponseLen int32) int64
	awaitSecrets(awaitRequest unsafe.Pointer, awaitRequestLen int32, responseBuffer unsafe.Pointer, maxResponseLen int32) int64
	switchModes(mode int32)
	getSeed(mode int32) int64
	now(response unsafe.Pointer) int32
}

func newRuntime(internals runtimeInternals, mode sdk.Mode) sdkimpl.RuntimeBase {
	return sdkimpl.RuntimeBase{
		Mode:           mode,
		RuntimeHelpers: &runtimeHelper{runtimeInternals: internals},
		Lggr:           newSlogger(),
	}
}

type runtimeHelper struct {
	runtimeInternals
	donSource  rand.Source
	nodeSource rand.Source
}

func (r *runtimeHelper) GetSource(mode sdk.Mode) rand.Source {
	switch mode {
	case sdk.Mode_MODE_DON:
		if r.donSource == nil {
			seed := r.getSeed(int32(mode))
			r.donSource = rand.NewSource(seed)
		}
		return r.donSource
	default:
		if r.nodeSource == nil {
			seed := r.getSeed(int32(mode))
			r.nodeSource = rand.NewSource(seed)
		}
		return r.nodeSource
	}
}

func (r *runtimeHelper) GetSecrets(request *sdk.GetSecretsRequest, maxResponseSize uint64) error {
	marshalled, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	marshalledPtr, marshalledLen, err := bufferToPointerLen(marshalled)
	if err != nil {
		return err
	}

	response := make([]byte, maxResponseSize)
	responsePtr, responseLen, err := bufferToPointerLen(response)
	if err != nil {
		return err
	}

	bytes := r.getSecrets(marshalledPtr, marshalledLen, responsePtr, responseLen)
	if bytes < 0 {
		return errors.New(string(response[:-bytes]))
	}

	return nil
}

func (r *runtimeHelper) AwaitSecrets(request *sdk.AwaitSecretsRequest, maxResponseSize uint64) (*sdk.AwaitSecretsResponse, error) {
	m, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	mptr, mlen, err := bufferToPointerLen(m)
	if err != nil {
		return nil, err
	}

	response := make([]byte, maxResponseSize)
	responsePtr, responseLen, err := bufferToPointerLen(response)
	if err != nil {
		return nil, err
	}

	bytes := r.awaitSecrets(mptr, mlen, responsePtr, responseLen)
	if bytes < 0 {
		return nil, errors.New(string(response[:-bytes]))
	}

	awaitResponse := &sdk.AwaitSecretsResponse{}
	err = proto.Unmarshal(response[:bytes], awaitResponse)
	if err != nil {
		return nil, err
	}

	return awaitResponse, nil
}

func (r *runtimeHelper) Call(request *sdk.CapabilityRequest) error {
	marshalled, err := proto.Marshal(request)
	if err != nil {
		return err
	}

	marshalledPtr, marshalledLen, err := bufferToPointerLen(marshalled)
	if err != nil {
		return err
	}

	// TODO (CAPPL-846): callCapability should also have a response pointer and response pointer buffer
	result := r.callCapability(marshalledPtr, marshalledLen)
	if result < 0 {
		return errors.New("cannot find capability " + request.Id)
	}

	return nil
}

func (r *runtimeHelper) Await(request *sdk.AwaitCapabilitiesRequest, maxResponseSize uint64) (*sdk.AwaitCapabilitiesResponse, error) {
	m, err := proto.Marshal(request)
	if err != nil {
		return nil, err
	}

	mptr, mlen, err := bufferToPointerLen(m)
	if err != nil {
		return nil, err
	}

	response := make([]byte, maxResponseSize)
	responsePtr, responseLen, err := bufferToPointerLen(response)
	if err != nil {
		return nil, err
	}

	bytes := r.awaitCapabilities(mptr, mlen, responsePtr, responseLen)
	if bytes < 0 {
		return nil, errors.New(string(response[:-bytes]))
	}

	awaitResponse := &sdk.AwaitCapabilitiesResponse{}
	err = proto.Unmarshal(response[:bytes], awaitResponse)
	if err != nil {
		return nil, err
	}

	return awaitResponse, nil
}

func (r *runtimeHelper) SwitchModes(mode sdk.Mode) {
	r.switchModes(int32(mode))
}

func (r *runtimeHelper) Now() time.Time {
	var buf [8]byte // host writes UnixNano as little-endian uint64
	rc := r.now(unsafe.Pointer(&buf[0]))
	if rc != ErrnoSuccess {
		panic(fmt.Errorf("failed to fetch time from host: now() returned errno %d", rc))
	}
	ns := int64(binary.LittleEndian.Uint64(buf[:]))
	return time.Unix(0, ns)
}
