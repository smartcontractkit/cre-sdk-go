package main

import (
	"os"
	"path/filepath"

	"github.com/smartcontractkit/chainlink-protos/cre/go/installer/pkg"
	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
)

func main() {
	cwd, _ := os.Getwd()
	root := cwd
	for {
		goMod := filepath.Join(root, "go.mod")
		aptosDir := filepath.Join(root, "capabilities", "blockchain", "aptos")
		if _, err := os.Stat(goMod); err == nil {
			if _, err := os.Stat(aptosDir); err == nil {
				break
			}
		}
		parent := filepath.Dir(root)
		if parent == root {
			panic("run from cre-sdk-go repo root or from capabilities/blockchain/aptos")
		}
		root = parent
	}
	_ = os.Chdir(root)
	gen, err := protos.NewGeneratorAndInstallToolsForCapability()
	if err != nil {
		panic(err)
	}
	config := &pkg.CapabilityConfig{
		Category:      "blockchain",
		Pkg:           "aptos",
		MajorVersion:  1,
		PreReleaseTag: "alpha",
		Files:         []string{"client.proto"},
	}
	if err := gen.GenerateMany(map[string]*pkg.CapabilityConfig{"capabilities/blockchain/aptos": config}); err != nil {
		panic(err)
	}
}
