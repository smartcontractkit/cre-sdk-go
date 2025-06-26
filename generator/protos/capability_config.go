package main

import (
	"errors"
	"fmt"
)

type CapabilityConfig struct {
	Category      string
	Pkg           string
	MajorVersion  int
	PreReleaseTag string
	Files         []string
}

func (c *CapabilityConfig) FullGoPackageName() string {
	// TODO internal to internal/capabilities or make new generator...?
	base := "github.com/smartcontractkit/cre-sdk-go/capabilities/" + c.Category + "/" + c.Pkg

	if c.Category == "internal" {
		base = "github.com/smartcontractkit/cre-sdk-go/internal/" + c.Category + "/" + c.Pkg
	}

	if c.MajorVersion == 1 {
		return base
	}
	return fmt.Sprintf("%s/v%d", base, c.MajorVersion)
}

func (c *CapabilityConfig) Validate() error {
	if c.Category == "" {
		return errors.New("category must not be empty")
	}
	if c.Pkg == "" {
		return errors.New("pkg must not be empty")
	}
	if c.MajorVersion < 1 {
		return fmt.Errorf("major-version must be >= 1, got %d", c.MajorVersion)
	}
	if len(c.Files) == 0 {
		return errors.New("files must not be empty")
	}
	return nil
}
