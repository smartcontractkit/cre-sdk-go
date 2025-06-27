package protos

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/chainlink-common/pkg/capabilities/v2/protoc/installer"
	"github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"
)

func Generate(config *CapabilityConfig) error {
	return GenerateMany(map[string]*CapabilityConfig{".": config})
}

func GenerateMany(dirToConfig map[string]*CapabilityConfig) error {
	_ = installProtocGen()
	if err := installer.InstallProtocGenToDir("github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre", "github.com/smartcontractkit/cre-sdk-go/generator/protos"); err != nil {
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

	fmt.Println("Generating capabilities")
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

	fmt.Println("Moving generated files to correct locations")
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
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/sdk/pb", Proto: "tools/generator/v1alpha/cre_metadata.proto"})
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/sdk/pb", Proto: "sdk/v1alpha/sdk.proto"})
	return gen
}

func link(gen *pkg.ProtocGen, config *CapabilityConfig) {
	for _, file := range config.FullProtoFiles() {
		gen.LinkPackage(pkg.Packages{Go: config.FullGoPackageName(), Proto: file})
	}
}

func installProtocGen() error {
	fmt.Println("Finding version to use for protoc-gen-cre.")
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Version}}", "github.com/smartcontractkit/cre-sdk-go/generator/protos")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get module version: %w\nOutput: %s", err, out)
	}
	version := strings.TrimSpace(string(out))

	fmt.Printf("Downloading protoc-gen-cre version %s\n", version)
	cmd = exec.Command("go", "mod", "download", "-json", "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre@"+version)
	out, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to download module: %w\nOutput: %s", err, out)
	}

	var mod struct{ Dir string }
	if err = json.Unmarshal(out, &mod); err != nil {
		return fmt.Errorf("failed to parse go mod download output: %w", err)
	}

	absDir, err := filepath.Abs(".tools")
	if err != nil {
		return fmt.Errorf("failed to get absolute path for .tools directory: %w", err)
	}

	fmt.Println("Building protoc-gen-cre")
	cmd = exec.Command("go", "build", "-o", absDir, ".")
	cmd.Dir = mod.Dir
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build protoc-gen-cre: %w\nOutput: %s", err, out)
	}

	return nil
}
