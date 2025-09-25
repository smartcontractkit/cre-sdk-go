package wasm

import (
	"os"
	"unsafe"

	"github.com/smartcontractkit/cre-sdk-go/cre"
)

//go:wasmimport env send_response
func sendResponse(response unsafe.Pointer, responseLen int32) int32

//go:wasmimport env version_v2_go
func versionV2()

//go:wasmimport env switch_modes
func switchModes(mode int32)

//go:wasmimport env now
func now(response unsafe.Pointer) int32

func NewRunner[C Config](parse func(configBytes []byte) (C, error)) cre.Runner[C] {
	return newRunner[C](parse, runnerInternalsImpl{}, runtimeInternalsImpl{})
}

func NewTEERunner[C Config](parse func(configBytes []byte) (C, error)) cre.TEERunner[C] {
	panic("Only here to show how to call this...")
}

type runnerInternalsImpl struct{}

var _ runnerInternals = runnerInternalsImpl{}

func (r runnerInternalsImpl) args() []string {
	return os.Args
}

func (r runnerInternalsImpl) sendResponse(response unsafe.Pointer, responseLen int32) int32 {
	return sendResponse(response, responseLen)
}

func (r runnerInternalsImpl) versionV2() {
	versionV2()
}

func (r runnerInternalsImpl) switchModes(mode int32) {
	switchModes(mode)
}

func (r runnerInternalsImpl) now(response unsafe.Pointer) int32 {
	return now(response)
}

func (r runnerInternalsImpl) exit() {
	os.Exit(0)
}
