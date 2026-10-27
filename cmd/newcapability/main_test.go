package main

import (
	"bytes"
	"go/format"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// render executes the template at path with cfg.
func render(t *testing.T, path string, cfg config) string {
	t.Helper()
	tmpl, err := template.ParseFS(templates, path)
	require.NoError(t, err)
	var b bytes.Buffer
	require.NoError(t, tmpl.Execute(&b, cfg))
	return b.String()
}

var cron = config{
	Category: "scheduler", Pkg: "cron", MajorVersion: 1, CapabilityDir: "scheduler/cron",
	ModPath: "github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron",
}

// TestTemplates checks the templates are valid Go and generate the capability from its production protos.
func TestTemplates(t *testing.T) {
	for _, path := range []string{"templates/generate.go.tmpl", "templates/generate_main.go.tmpl"} {
		_, err := format.Source([]byte(render(t, path, cron)))
		assert.NoError(t, err, path)
	}

	main := render(t, "templates/generate_main.go.tmpl", cron)
	assert.Contains(t, main, `Fetched:   "capabilities/scheduler/cron/v1",`)
	assert.Contains(t, main, `GoPackage: "github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron",`)

	pre := cron
	pre.MajorVersion, pre.PreReleaseTag = 2, "alpha"
	assert.Contains(t, render(t, "templates/generate_main.go.tmpl", pre), `"capabilities/scheduler/cron/v2alpha"`)

	fetch := render(t, "templates/protofetch.toml.tmpl", cron)
	assert.Contains(t, fetch, `content_roots = ["/scheduler/cron/protos/production"]`)
	assert.Equal(t, 1, strings.Count(fetch, `revision = ""`), "generating pins the revision to what generate/go.mod requires")
	assert.NotContains(t, fetch, "[chainlink-protos]", "the base protos come with the capability's")

}
