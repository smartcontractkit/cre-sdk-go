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

func Generate(category, pkgName, version string, files ...string) error {
	if err := installProtocGen(); err != nil {
		return err
	}

	fullPkgName := fullGoName(category, pkgName)
	protoDir := path.Join("capabilities", category, pkgName, version)

	gen := generator(fullPkgName, protoDir, files)
	fileToFrom := map[string]string{}
	for _, file := range files {
		fileToFrom[path.Join(protoDir, file)] = "."
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

	for _, file := range files {
		file = strings.Replace(file, ".proto", ".pb.go", 1)
		if err := os.Rename(path.Join(protoDir, file), file); err != nil {
			return fmt.Errorf("failed to move generated file %s: %w", file, err)
		}
	}

	if err := os.RemoveAll("capabilities"); err != nil {
		return fmt.Errorf("failed to remove category directory %s: %w", category, err)
	}

	return nil
}

func fullGoName(category string, name string) string {
	switch category {
	case "internal":
		return fmt.Sprintf("github.com/smartcontractkit/cre-sdk-go/internal/capabilities/%s", name)
	default:
		return fmt.Sprintf("github.com/smartcontractkit/cre-sdk-go/capabilities/%s/%s", category, name)
	}
}

func generator(pkgName, protoDir string, files []string) *pkg.ProtocGen {
	gen := &pkg.ProtocGen{Plugins: []pkg.Plugin{{Name: "cre", Path: ".tools"}}}
	gen.AddSourceDirectories(".")
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre/pb", Proto: "tools/generator/v1alpha/cre_metadata.proto"})
	gen.LinkPackage(pkg.Packages{Go: "github.com/smartcontractkit/cre-sdk-go/sdk/pb", Proto: "sdk/v1alpha/sdk.proto"})
	for _, file := range files {
		gen.LinkPackage(pkg.Packages{Go: pkgName, Proto: path.Join(protoDir, file)})
	}
	return gen
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
