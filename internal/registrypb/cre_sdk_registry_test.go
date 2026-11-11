package registrypb_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/smartcontractkit/cre-sdk-go/internal/registrypb"
)

// chainlink-common's pkg/capabilities/pb registers "registry.proto". Binaries linking both packages (e.g. the
// chainlink node) panic at init if this package registers the same file path or full names.
func TestRegistryProtoDoesNotCollideWithChainlinkCommon(t *testing.T) {
	assert.NotEqual(t, "registry.proto", registrypb.File_cre_sdk_registry_proto.Path())

	_, err := protoregistry.GlobalFiles.FindFileByPath("registry.proto")
	require.ErrorIs(t, err, protoregistry.NotFound)

	for _, name := range []string{"loop.CapabilityConfig", "loop.OCR3Config"} {
		_, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
		require.ErrorIs(t, err, protoregistry.NotFound, name)
	}
}
