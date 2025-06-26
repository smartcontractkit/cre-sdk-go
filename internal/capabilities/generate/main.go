package main

import "github.com/smartcontractkit/cre-sdk-go/generator/protos"

func main() {
	protos.Generate()
	if err := protos.Generate(); err != nil {
		panic(err)
	}
}
