package main

import (
	"os"

	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
)

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
		if err := os.MkdirAll(proto.Pkg, os.ModePerm); err != nil {
			panic(err)
		}
	}

	if err := protos.GenerateMany(internalProtosToDir); err != nil {
		panic(err)
	}
}
