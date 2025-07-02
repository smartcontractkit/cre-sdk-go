package protos

import "github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"

func NewGeneratorAndInstallTools() (*pkg.ProtocGen, error) {
	plugin := "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre"

	if err := pkg.InstallProtocGenFromThisMod(plugin); err != nil {
		return nil, err
	}

	return &pkg.ProtocGen{
		ProtocHelper: ProtocHelper{},
		Plugins:      []pkg.Plugin{{Name: "cre", Path: ".tools"}},
	}, nil
}
