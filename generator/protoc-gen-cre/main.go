package main

import (
	"log"
	"os"

	"github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre/pkg"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"
)

const (
	toolName    = "github.com/smartcontractkit/chainlink-common/pkg/capabilities/v2/protoc"
	localPrefix = "http://github.com/smartcontractkit/cre-sdk-go"
)

func main() {
	protogen.Options{}.Run(func(plugin *protogen.Plugin) error {
		plugin.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		for _, file := range plugin.Files {
			if !file.Generate {
				continue
			}
			if err := pkg.GenerateClient(plugin, file, toolName, localPrefix); err != nil {
				log.Printf("failed to generate for %s: %v", file.Desc.Path(), err)
				os.Exit(1)
			}
		}
		return nil
	})
}
