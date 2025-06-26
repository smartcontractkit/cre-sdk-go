package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCapabilityFlags_HappyPath(t *testing.T) {
	args := []string{
		"--category=scheduler", "--pkg=cron", "--major-version=1", "--pre-release-tag=alpha", "--files=a.proto,b.proto",
		"--logger-category=logging", "--logger-pkg=zap", "--logger-major-version=2", "--logger-files=log.proto",
	}
	got, err := parseCapabilityFlags(args)
	assert.NoError(t, err)

	assert.Equal(t, &CapabilityConfig{
		Category:      "scheduler",
		Pkg:           "cron",
		MajorVersion:  1,
		PreReleaseTag: "alpha",
		Files:         []string{"a.proto", "b.proto"},
	}, got[""])

	assert.Equal(t, &CapabilityConfig{
		Category:     "logging",
		Pkg:          "zap",
		MajorVersion: 2,
		Files:        []string{"log.proto"},
	}, got["logger"])
}

func TestParseCapabilityFlags_DuplicateField(t *testing.T) {
	args := []string{
		"--category=scheduler", "--category=again", "--pkg=cron", "--major-version=1", "--files=a.proto",
	}
	_, err := parseCapabilityFlags(args)
	assert.EqualError(t, err, `duplicate flag: category`)
}

func TestParseCapabilityFlags_ValidationErrors(t *testing.T) {
	args := []string{
		"--category=scheduler", "--pkg=cron", "--major-version=0", "--files=a.proto",
	}
	_, err := parseCapabilityFlags(args)
	assert.EqualError(t, err, `invalid config for capability "": major-version must be >= 1, got 0`)

	args = []string{"--category=", "--pkg=cron", "--major-version=1", "--files=a.proto"}
	_, err = parseCapabilityFlags(args)
	assert.EqualError(t, err, `invalid config for capability "": category must not be empty`)

	args = []string{"--category=scheduler", "--pkg=", "--major-version=1", "--files=a.proto"}
	_, err = parseCapabilityFlags(args)
	assert.EqualError(t, err, `invalid config for capability "": pkg must not be empty`)

	args = []string{"--category=scheduler", "--pkg=cron", "--major-version=1", "--files="}
	_, err = parseCapabilityFlags(args)
	assert.EqualError(t, err, `invalid config for capability "": files must not be empty`)
}

func TestParseCapabilityFlags_MalformedOrUnknown(t *testing.T) {
	args := []string{"--major-version=abc"}
	_, err := parseCapabilityFlags(args)
	assert.EqualError(t, err, "invalid major-version: abc")

	args = []string{"--logger-unknownfield=value"}
	_, err = parseCapabilityFlags(args)
	assert.EqualError(t, err, "unknown field: unknownfield")

	args = []string{"--loggerfoo=bar"}
	_, err = parseCapabilityFlags(args)
	assert.EqualError(t, err, "malformed flag: loggerfoo")

	args = []string{"--category"}
	_, err = parseCapabilityFlags(args)
	assert.EqualError(t, err, "invalid key=value pair: category")

	args = []string{"category=scheduler"}
	_, err = parseCapabilityFlags(args)
	assert.EqualError(t, err, "invalid argument: category=scheduler")
}
