package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/smartcontractkit/capabilities/libs/capability/feature"
	"github.com/smartcontractkit/capabilities/libs/capability/protofetch"

	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
)

// featurePkg is the Go package feature.proto is generated to.
const featurePkg = "github.com/smartcontractkit/capabilities/libs/capability/feature"

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
	// The development protos tag their fields and methods with the feature they belong to, importing feature.proto
	// for it, which no content root fetches: the libs embed it, for the protoc runs that import it.
	include, err := os.MkdirTemp("", "protoc-include-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(include)

	featureProto := filepath.Join(include, filepath.FromSlash(feature.Path))
	if err = os.MkdirAll(filepath.Dir(featureProto), 0o755); err != nil {
		return err
	}
	if err = os.WriteFile(featureProto, feature.Proto, 0o600); err != nil {
		return err
	}

	return protofetch.Generate(ctx, protofetch.Options{
		Module: ".",
		Modules: []string{generator},
		Dir:    ".",
		// The development protos are at the root of what is fetched, unlike their releases, which the production
		// directory has at their versioned path.
		Fetched:   ".",
		GoPackage: "github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron",
		Plugins:   []string{"google.golang.org/protobuf/cmd/protoc-gen-go"},
		Binaries:  []string{plugin},
		Includes:  []string{include},
		GoPackages: map[string]string{
			feature.Path: featurePkg,
		},
	})
}
