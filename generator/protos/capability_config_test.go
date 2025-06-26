package protos_test

import (
	"testing"

	"github.com/smartcontractkit/cre-sdk-go/generator/protos"
	"github.com/stretchr/testify/assert"
)

func TestFullGoPackageName(t *testing.T) {
	t.Run("version 1", func(t *testing.T) {
		assert.Equal(t,
			"github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron",
			(&protos.CapabilityConfig{
				Category:     "scheduler",
				Pkg:          "cron",
				MajorVersion: 1,
			}).FullGoPackageName(),
		)
	})

	t.Run("Not version 1", func(t *testing.T) {
		assert.Equal(t,
			"github.com/smartcontractkit/cre-sdk-go/capabilities/stream/price/v2",
			(&protos.CapabilityConfig{
				Category:     "stream",
				Pkg:          "price",
				MajorVersion: 2,
			}).FullGoPackageName(),
		)
	})

	t.Run("internal category", func(t *testing.T) {
		assert.Equal(t,
			"github.com/smartcontractkit/cre-sdk-go/internal/capabilities/cron",
			(&protos.CapabilityConfig{
				Category:     "internal",
				Pkg:          "cron",
				MajorVersion: 1,
			}).FullGoPackageName(),
		)
	})
}

func TestCapabilityConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     protos.CapabilityConfig
		wantErr string
	}{
		{
			name: "valid config",
			cfg: protos.CapabilityConfig{
				Category:     "scheduler",
				Pkg:          "cron",
				MajorVersion: 1,
				Files:        []string{"a.proto"},
			},
			wantErr: "",
		},
		{
			name: "missing category",
			cfg: protos.CapabilityConfig{
				Pkg:          "cron",
				MajorVersion: 1,
				Files:        []string{"a.proto"},
			},
			wantErr: "category must not be empty",
		},
		{
			name: "missing pkg",
			cfg: protos.CapabilityConfig{
				Category:     "scheduler",
				MajorVersion: 1,
				Files:        []string{"a.proto"},
			},
			wantErr: "pkg must not be empty",
		},
		{
			name: "invalid major version",
			cfg: protos.CapabilityConfig{
				Category: "scheduler",
				Pkg:      "cron",
				Files:    []string{"a.proto"},
			},
			wantErr: "major-version must be >= 1, got 0",
		},
		{
			name: "missing files",
			cfg: protos.CapabilityConfig{
				Category:     "scheduler",
				Pkg:          "cron",
				MajorVersion: 1,
			},
			wantErr: "files must not be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}
