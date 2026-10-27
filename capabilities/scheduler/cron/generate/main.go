package main

import (
	"context"
	"log"
	"os"

	"github.com/smartcontractkit/capabilities/libs/capability/protofetch"

	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
)

func main() {
	if err := generate(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func generate(ctx context.Context) error {
	// It runs in this module, which has what it is built from, and generates in the capability's, which has the
	// protos it fetches and the SDK version the protoc-gen-cre it generates with is.
	generator, err := os.Getwd()
	if err != nil {
		return err
	}
	if err = os.Chdir(".."); err != nil {
		return err
	}
	plugin, err := protos.InstallPluginForCapability()
	if err != nil {
		return err
	}
	return protofetch.Generate(ctx, protofetch.Options{
		Module:    ".",
		Modules:   []string{generator},
		Dir:       ".",
		Fetched:   "capabilities/scheduler/cron/v1",
		GoPackage: "github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron",
		Plugins:   []string{"google.golang.org/protobuf/cmd/protoc-gen-go"},
		Binaries:  []string{plugin},
	})
}
