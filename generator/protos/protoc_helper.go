package protos

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/smartcontractkit/chainlink-protos/cre/go/installer/pkg"
)

type ProtocHelper struct{}

var _ pkg.ProtocHelper = ProtocHelper{}

func (g ProtocHelper) FullGoPackageName(c *pkg.CapabilityConfig) string {
	base := "github.com/smartcontractkit/cre-sdk-go/capabilities/" + c.Category + "/" + c.Pkg

	if strings.Split(c.Category, string(filepath.Separator))[0] == "internal" {
		base = strings.Replace(base, "capabilities/internal", "internal_testing/capabilities", 1)
	}

	if c.MajorVersion == 1 {
		return base
	}
	return fmt.Sprintf("%s/v%d", base, c.MajorVersion)
}
