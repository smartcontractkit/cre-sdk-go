package protos

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/chainlink-protos/cre/go/installer/pkg"
)

const plugin = "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre"
const sdk = "github.com/smartcontractkit/cre-sdk-go"

func NewGeneratorAndInstallToolsForCapability() (*pkg.ProtocGen, error) {
	return newGeneratorAndInstallTools(installFromMod)
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
		Plugins:      []pkg.Plugin{pkg.GoPlugin, {Name: "cre", Path: ".tools"}},
	}, nil
}

// installFromMod installs the protoc-gen-cre plugin from the same commit as the SDK you're using
func installFromMod() error {
	fmt.Printf("Finding version to use for %s\n.", sdk)
	pluginVersion, err := getVersion(sdk, ".")
	if err != nil {
		return err
	}

	pluginDir, err := downloadPlugin(plugin, pluginVersion)
	if err != nil {
		return err
	}

	return buildPlugin(pluginDir)
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

func getVersion(of, dir string) (string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Version}}", of)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("Failed to get version of %s in directory %s: %w\nOutput: %s", of, dir, err, out)
	}

	return strings.TrimSpace(string(out)), nil
}

func downloadPlugin(pkgName, version string) (string, error) {
	fmt.Printf("Downloading plugin version %s\n", version)
	cmd := exec.Command("go", "mod", "download", "-json", fmt.Sprintf("%s@%s", pkgName, version))
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to download module: %w\nOutput: %s", err, out)
	}

	var mod struct{ Dir string }
	if err = json.Unmarshal(out, &mod); err != nil {
		return "", fmt.Errorf("failed to parse go mod download output: %w", err)
	}

	return mod.Dir, nil
}

func buildPlugin(pluginDir string) error {
	toolsDir, err := filepath.Abs(".tools")
	if err != nil {
		return fmt.Errorf("failed to get absolute path for .tools: %w", err)
	}

	if err = os.MkdirAll(toolsDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create .tools directory: %w", err)
	}

	fmt.Println("Building plugin")
	cmd := exec.Command("go", "build", "-o", toolsDir, ".")
	cmd.Dir = pluginDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build plugin: %w\nOutput: %s", err, out)
	}

	return nil
}
