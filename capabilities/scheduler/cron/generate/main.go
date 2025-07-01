package main

import (
	"github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"
	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
)

func main() {
    gen := protos.ProtocGen{}
	if err := gen.Generate(&pkg.CapabilityConfig{
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
