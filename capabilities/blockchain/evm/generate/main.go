package main

import (
	"github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"
	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
)

func main() {
    gen := protos.ProtocGen{}
	if err := gen.Generate(&pkg.CapabilityConfig{
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
