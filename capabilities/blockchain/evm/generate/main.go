package main

import "github.com/smartcontractkit/cre-sdk-go/generator/protos"

func main() {
	if err := protos.Generate(&protos.CapabilityConfig{
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
