package protos

import "github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"

func NewGenerator() *pkg.ProtocGen {
	return &pkg.ProtocGen{
		ProtocHelper: ProtocHelper{},
		Plugins:      []pkg.Plugin{{Name: "cre", Path: ".tools"}},
	}
}
