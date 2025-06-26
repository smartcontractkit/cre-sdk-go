package protos

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"
)

func Generate(config *CapabilityConfig) error {
	return GenerateMany(map[string]*CapabilityConfig{".": config})
}

func GenerateMany(dirToConfig map[string]*CapabilityConfig) error {
	if err := installProtocGen(); err != nil {
		return err
	}

	gen := createGenerator()

	fileToFrom := map[string]string{}
	for from, config := range dirToConfig {
		for _, file := range config.FullProtoFiles() {
			fileToFrom[file] = from
		}
		link(gen, config)
	}

	errMap := gen.GenerateMany(fileToFrom)
	if len(errMap) > 0 {
		var errStrings []string
		for file, err := range errMap {
			if err != nil {
				errStrings = append(errStrings, fmt.Sprintf("file %s\n%v\n", file, err))
			}
		}

		return errors.New(strings.Join(errStrings, ""))
	}

	for from, config := range dirToConfig {
		for i, file := range config.FullProtoFiles() {
			file = strings.Replace(file, ".proto", ".pb.go", 1)
			to := strings.Replace(config.Files[i], ".proto", ".pb.go", 1)
			if err := os.Rename(path.Join(from, file), path.Join(from, to)); err != nil {
				return fmt.Errorf("failed to move generated file %s: %w", file, err)
			}
		}

		if err := os.RemoveAll(path.Join(from, "capabilities")); err != nil {
			return fmt.Errorf("failed to remove capabilities directory %w", err)
		}
	}

	return nil
}

func createGenerator() *pkg.ProtocGen {
	gen := &pkg.ProtocGen{Plugins: []pkg.Plugin{{Name: "cre", Path: ".tools"}}}
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre/pb", Proto: "tools/generator/v1alpha/cre_metadata.proto"})
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/sdk/pb", Proto: "sdk/v1alpha/sdk.proto"})
	return gen
}

func link(gen *pkg.ProtocGen, config *CapabilityConfig) {
	for _, file := range config.FullProtoFiles() {
		gen.LinkPackage(pkg.Packages{Go: config.FullGoPackageName(), Proto: file})
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
