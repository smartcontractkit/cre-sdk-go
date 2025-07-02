package protos

import "github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"

const plugin = "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre"
const sdk = "github.com/smartcontractkit/cre-sdk-go"

func NewGeneratorAndInstallToolsForCapability() (*pkg.ProtocGen, error) {
	return newGeneratorAndInstallTools(func() error { return pkg.InstallProtocGenToDir(plugin, sdk) })
}

func NewGeneratorAndInstallToolsForSdk() (*pkg.ProtocGen, error) {
	return newGeneratorAndInstallTools(func() error { return pkg.InstallProtocGenFromThisMod(plugin) })
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
