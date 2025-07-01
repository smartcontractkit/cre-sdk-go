package protos

import (
	"fmt"
	"github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"
	"path/filepath"
	"strings"
)

type ProtocHelper struct{}

var _ pkg.ProtocHelper = ProtocHelper{}

func (g ProtocHelper) SdkPgk() string {
	return "github.com/smartcontractkit/cre-sdk-go/sdk/pb"
}

func (g ProtocHelper) PluginName() string {
	return "github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre"
}

func (g ProtocHelper) HelperName() string {
	return "github.com/smartcontractkit/cre-sdk-go/generator/protos"
}

func (g ProtocHelper) FullGoPackageName(c *pkg.CapabilityConfig) string {
	base := "github.com/smartcontractkit/cre-sdk-go/capabilities/" + c.Category + "/" + c.Pkg

	if strings.Split(c.Category, string(filepath.Separator))[0] == "internal" {
		base = strings.Replace(base, "capabilities/internal", "internal/capabilities", 1)
	}

	if c.MajorVersion == 1 {
		return base
	}
	return fmt.Sprintf("%s/v%d", base, c.MajorVersion)
}
