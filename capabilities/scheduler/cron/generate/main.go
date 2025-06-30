package main

import (
	"github.com/smartcontractkit/chainlink-common/pkg/capabilities/v2/protoc/installer"
	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
)

func main() {
    gen := installer.Generator{GeneratorHelper: protos.GeneratorHelper{}}
	if err := gen.Generate(&installer.CapabilityConfig{
		Category:     "scheduler",
		Pkg:          "cron",
		MajorVersion: 1,
		Files: []string{
			"trigger.proto",
		},
	}); err != nil {
		panic(err)
	}
}
