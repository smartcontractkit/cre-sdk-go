package main

import "github.com/smartcontractkit/cre-sdk-go/generator/protos"

func main() {
	internalProtos := []*protos.CapabilityConfig{
		{
			Category:      "internal",
			Pkg:           "consensus",
			MajorVersion:  1,
			PreReleaseTag: "alpha",
			Files:         []string{"consensus.proto"},
		},
	}

	internalProtosToDir := map[string]*protos.CapabilityConfig{}

	for _, proto := range internalProtos {
		internalProtosToDir[proto.Pkg] = proto
	}

	if err := protos.GenerateMany(internalProtosToDir); err != nil {
		panic(err)
	}
}
