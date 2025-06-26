package main

import "github.com/smartcontractkit/cre-sdk-go/generator/protos"

func main() {
	if err := protos.Generate("scheduler", "cron", "v1", "trigger.proto"); err != nil {
		panic(err)
	}
}
