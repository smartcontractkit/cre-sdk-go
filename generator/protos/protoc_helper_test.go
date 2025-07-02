package protos_test

import (
	"testing"

	"github.com/smartcontractkit/chainlink-common/pkg/values/installer/pkg"
	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
	"github.com/stretchr/testify/assert"
)

func TestFullGoPackageName(t *testing.T) {
	t.Parallel()
	gh := protos.ProtocHelper{}
	t.Run("version 1", func(t *testing.T) {
		assert.Equal(t,
			"github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron",
			gh.FullGoPackageName(&pkg.CapabilityConfig{
				Category:     "scheduler",
				Pkg:          "cron",
				MajorVersion: 1,
			}),
		)
	})

	t.Run("version 1 nested", func(t *testing.T) {
		assert.Equal(t,
			"github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/something/cron",
			gh.FullGoPackageName(&pkg.CapabilityConfig{
				Category:     "scheduler/something",
				Pkg:          "cron",
				MajorVersion: 1,
			}),
		)
	})

	t.Run("Not version 1", func(t *testing.T) {
		assert.Equal(t,
			"github.com/smartcontractkit/cre-sdk-go/capabilities/stream/price/v2",
			gh.FullGoPackageName(&pkg.CapabilityConfig{
				Category:     "stream",
				Pkg:          "price",
				MajorVersion: 2,
			}),
		)
	})

	t.Run("internal category", func(t *testing.T) {
		assert.Equal(t,
			"github.com/smartcontractkit/cre-sdk-go/internal/capabilities/cron",
			gh.FullGoPackageName(&pkg.CapabilityConfig{
				Category:     "internal",
				Pkg:          "cron",
				MajorVersion: 1,
			}),
		)
	})

	t.Run("internal category nested", func(t *testing.T) {
		assert.Equal(t,
			"github.com/smartcontractkit/cre-sdk-go/internal/capabilities/something/cron",
			gh.FullGoPackageName(&pkg.CapabilityConfig{
				Category:     "internal/something",
				Pkg:          "cron",
				MajorVersion: 1,
			}),
		)
	})
}
