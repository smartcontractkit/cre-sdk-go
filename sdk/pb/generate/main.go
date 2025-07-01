package main

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"
)

func main() {
	gen := &pkg.ProtocGen{}
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/pb", Proto: "sdk/v1alpha/sdk.proto"})
	if err := gen.GenerateFile("sdk/v1alpha/sdk.proto", "."); err != nil {
		panic(fmt.Errorf("failed to generate sdk proto: %w", err))
	}

	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/pb", Proto: "tools/generator/v1alpha/cre_metadata.proto"})
	if err := gen.GenerateFile("tools/generator/v1alpha/cre_metadata.proto", "."); err != nil {
		panic(fmt.Errorf("failed to generate protobuf metadata: %w", err))
	}
}
