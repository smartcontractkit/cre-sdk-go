package protos

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"
)

const plugin = "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre"
const sdk = "github.com/smartcontractkit/cre-sdk-go"

func NewGeneratorAndInstallToolsForCapability() (*pkg.ProtocGen, error) {
	return newGeneratorAndInstallTools(func() error { return pkg.InstallProtocGenToDir(plugin, sdk) })
}

func NewGeneratorAndInstallToolsForSdk() (*pkg.ProtocGen, error) {
	return newGeneratorAndInstallTools(buildProtocGenLocally)
}

func newGeneratorAndInstallTools(install func() error) (*pkg.ProtocGen, error) {
	if err := install(); err != nil {
		return nil, err
	}

	return &pkg.ProtocGen{
		ProtocHelper: ProtocHelper{},
		Plugins:      []pkg.Plugin{{Name: "cre", Path: ".tools"}},
	}, nil
}

// buildProtocGenLocally builds against the local protoc-gen-cre source code.
// Installing it from the same commit won't work well if you modify the protor generator,
// since it will first generate without the changes, then on commit, generate with them.
// building locally allows you to see the changes immediately without needing to commit, push and redo.
func buildProtocGenLocally() error {
	if err := os.MkdirAll(".tools", 0755); err != nil {
		return fmt.Errorf("failed to create .tools directory: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", filepath.Join("..", "..", "internal_testing", "capabilities", ".tools"))
	cmd.Dir = filepath.Join("..", "..", "generator", "protoc-gen-cre")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run protoc: %v\n%s", err, out)
	}

	return nil
}
