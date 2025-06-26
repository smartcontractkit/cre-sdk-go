package main

import "github.com/smartcontractkit/cre-sdk-go/generator/protos"

func main() {
	if err := protos.Generate(&protos.CapabilityConfig{
		Category:     "scheduler",
		Pkg:          "cron",
		MajorVersion: 1,
		Files:        []string{"trigger.proto"},
	}); err != nil {
		panic(err)
	}
}
