package protos

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/chainlink-common/pkg/capabilities/v2/protoc/installer"
)

type GeneratorHelper struct{}

var _ installer.GeneratorHelper = GeneratorHelper{}

func (g GeneratorHelper) SdkPgk() string {
	return "github.com/smartcontractkit/cre-sdk-go/sdk/pb"
}

func (g GeneratorHelper) PluginName() string {
	return "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre"
}

func (g GeneratorHelper) HelperName() string {
	return "github.com/smartcontractkit/cre-sdk-go/generator/protos"
}

func (g GeneratorHelper) FullGoPackageName(c *installer.CapabilityConfig) string {
	base := "github.com/smartcontractkit/cre-sdk-go/capabilities/" + c.Category + "/" + c.Pkg

	if strings.Split(c.Category, string(filepath.Separator))[0] == "internal" {
		base = strings.Replace(base, "capabilities/internal", "internal/capabilities", 1)
	}

	if c.MajorVersion == 1 {
		return base
	}
	return fmt.Sprintf("%s/v%d", base, c.MajorVersion)
}
