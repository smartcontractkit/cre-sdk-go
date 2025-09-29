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

// NewRunner creates a new cre.Runner instance with the provided function to parse config.
func NewRunner[C Config](parse func(configBytes []byte) (C, error)) cre.Runner[C] {
	return newRunner[C](parse, runnerInternalsImpl{}, runtimeInternalsImpl{})
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
