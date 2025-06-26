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
		{
			Category:     "internal",
			Pkg:          "actionandtrigger",
			MajorVersion: 1,
			Files:        []string{"action_and_trigger.proto"},
		},
		{
			Category:     "internal",
			Pkg:          "basicaction",
			MajorVersion: 1,
			Files:        []string{"basic_action.proto"},
		},
		{
			Category:     "internal",
			Pkg:          "basictrigger",
			MajorVersion: 1,
			Files:        []string{"basic_trigger.proto"},
		},
		{
			Category:     "internal",
			Pkg:          "nodeaction",
			MajorVersion: 1,
			Files:        []string{"node_action.proto"},
		},
		{
			Category:     "internal",
			Pkg:          "importclash",
			MajorVersion: 1,
			Files:        []string{"clash.proto"},
		},
		{
			Category:     "internal/importclash",
			Pkg:          "p1",
			MajorVersion: 1,
			Files:        []string{"import.proto"},
		},
		{
			Category:     "internal/importclash",
			Pkg:          "p2",
			MajorVersion: 1,
			Files:        []string{"import.proto"},
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
