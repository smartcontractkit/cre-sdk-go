package main

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"
)

func main() {
	gen := &pkg.ProtocGen{}
	// Make a local copy of sdk proto so that we can use it without a circular dependency on sdk.
	// This is safe because the SDK will have its values package version enforced by the version of this library it uses.
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre/pb", Proto: "sdk/v1alpha/sdk.proto"})
	if err := gen.Generate("sdk/v1alpha/sdk.proto", "."); err != nil {
		panic(fmt.Errorf("failed to generate sdk proto: %w", err))
	}

	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre/pb", Proto: "tools/generator/v1alpha/cre_metadata.proto"})
	if err := gen.Generate("tools/generator/v1alpha/cre_metadata.proto", "."); err != nil {
		panic(fmt.Errorf("failed to generate protobuf metadata: %w", err))
	}
}
