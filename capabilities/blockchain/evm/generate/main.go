package main

import (
	"github.com/smartcontractkit/chainlink-common/pkg/capabilities/v2/protoc/installer"
	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
)

func main() {
    gen := installer.Generator{GeneratorHelper: protos.GeneratorHelper{}}
	if err := gen.Generate(&installer.CapabilityConfig{
		Category:     "blockchain",
		Pkg:          "evm",
		MajorVersion: 1,
		PreReleaseTag: "alpha",
		Files: []string{
			"client.proto",
		},
	}); err != nil {
		panic(err)
	}
}
