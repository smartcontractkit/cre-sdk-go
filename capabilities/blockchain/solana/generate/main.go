package main

import (
	"github.com/smartcontractkit/chainlink-protos/cre/go/installer/pkg"
	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
)

func main() {
	gen, err := protos.NewGeneratorAndInstallToolsForCapability()
	if err != nil {
		panic(err)
	}
	if err := gen.Generate(&pkg.CapabilityConfig{
		Category:      "blockchain",
		Pkg:           "solana",
		MajorVersion:  1,
		PreReleaseTag: "alpha",
		Files: []string{
			"client.proto",
		},
	}); err != nil {
		panic(err)
	}
}
