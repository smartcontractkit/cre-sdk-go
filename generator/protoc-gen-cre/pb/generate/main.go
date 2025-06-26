package main

import "github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"

func main() {
	gen := &pkg.ProtocGen{}
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre/pb", Proto: "tools/generator/v1alpha/cre_metadata.proto"})
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/sdk/pb", Proto: "sdk/v1alpha/sdk.proto"})
	if err := gen.Generate("tools/generator/v1alpha/cre_metadata.proto", "."); err != nil {
		panic(err)
	}
}
