package main

import (
	"errors"

	"github.com/smartcontractkit/chainlink-common/pkg/workflows/wasm/host/internal/rawsdk"
	"github.com/smartcontractkit/cre-sdk-go/sdk/pb"
)

func main() {
	// The real SDKs do something to capture the runtime.
	// This is to mimic the mode switch calls they would make
	rawsdk.SwitchModes(int32(pb.Mode_MODE_NODE))
	rawsdk.SwitchModes(int32(pb.Mode_MODE_DON))
	rawsdk.SendError(errors.New("cannot use NodeRuntime outside RunInNodeMode"))
}
