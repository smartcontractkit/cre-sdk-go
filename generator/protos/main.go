package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"
)

func main() {
	// TODO be smarter, just take the files, imply the config from it...
	// remap capabilities/internal to internal/capabilities
	if err := installProtocGen(); err != nil {
		panic(err)
	}
	gen := &pkg.ProtocGen{Plugins: []pkg.Plugin{{Name: "cre", Path: ".tools"}}}

	capabilityInfos, err := parseCapabilityFlags(os.Args[1:])
	if err != nil {
		panic(err)
	}

	mainCapability, ok := capabilityInfos[""]
	if !ok {
		panic("no main capability specified")
	}

	setupGenerator(gen, capabilityInfos)

	generateCapability(mainCapability, gen)

	if err = os.RemoveAll("capabilities"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error removing capabilities directory:\n%v\n", err)
		os.Exit(1)
	}
}

func generateCapability(capability *CapabilityConfig, gen *pkg.ProtocGen) {
	errors := generateFiles(capability, gen)
	if len(errors) > 0 {
		for file, fileErr := range errors {
			_, _ = fmt.Fprintf(os.Stderr, "Error generating file %s:\n%v\n", file, fileErr)
		}
		os.Exit(1)
	}
}
func generateFiles(capability *CapabilityConfig, gen *pkg.ProtocGen) map[string]error {
	errors := map[string]error{}
	for _, file := range capability.Files {
		if err := gen.Generate(file, "."); err == nil {
			pbName := strings.Replace(file, ".proto", ".pb.go", 1)
			if err = os.Rename(pbName, filepath.Base(pbName)); err != nil {
				errors[file] = err
			}
		} else {
			errors[file] = err
		}
	}
	return errors
}

func setupGenerator(gen *pkg.ProtocGen, capabilityInfos map[string]*CapabilityConfig) {
	// Note the second directory is for chain-capabilities/evm.
	// Once the dependencies are inverted, it will be removed.
	gen.AddSourceDirectories(".")
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre/pb", Proto: "tools/generator/v1alpha/cre_metadata.proto"})
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/sdk/pb", Proto: "sdk/v1alpha/sdk.proto"})
	for _, info := range capabilityInfos {
		for _, file := range info.Files {
			gen.LinkPackage(pkg.Packages{Go: info.FullGoPackageName(), Proto: file})
		}
	}
}

func installProtocGen() error {
	// TODO have something ensure the two of them have the same version of values
	// Maybe use debug.ReadBuildInfo() to get the right generator

	// Running in capabilities/*/*
	// install the proto-gen-cre from the same version this tool is in

	cmd := exec.Command("go", "build", ".")
	cmd.Dir = "../../../generator/protoc-gen-cre"
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build protoc-gen-cre: %w\nOutput: %s", err, out)
	}
	if err = os.MkdirAll(".tools", os.ModePerm); err != nil {
		return fmt.Errorf("failed to create tools directory: %w", err)
	}

	if err = os.Rename(cmd.Dir+"/protoc-gen-cre", ".tools/protoc-gen-cre"); err != nil {
		return fmt.Errorf("failed to move protoc-gen-cre: %w", err)
	}
	return nil
}
