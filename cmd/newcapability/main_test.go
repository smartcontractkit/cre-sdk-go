package main

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
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

// TestTemplates checks the templates are valid Go and generate the capability from its development protos.
func TestTemplates(t *testing.T) {
	for _, path := range []string{"templates/generate.go.tmpl", "templates/generate_main.go.tmpl"} {
		_, err := format.Source([]byte(render(t, path, cron)))
		assert.NoError(t, err, path)
	}

	main := render(t, "templates/generate_main.go.tmpl", cron)
	assert.Contains(t, main, `Fetched:   ".",`)
	assert.Contains(t, main, `GoPackage: "github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron",`)
	assert.Contains(t, main, `feature.Path: featurePkg,`, "the development protos import feature.proto, which no content root fetches")

	fetch := render(t, "templates/protofetch.toml.tmpl", cron)
	assert.Contains(t, fetch, `content_roots = ["/scheduler/cron/protos"]`)
	assert.Equal(t, 1, strings.Count(fetch, `revision = ""`), "generating pins the revision to what generate/go.mod requires")
	assert.NotContains(t, fetch, "[chainlink-protos]", "the base protos come with the capability's")

	// A re-run keeps the revision generating pinned the toml to, so the lock and the toml stay in step.
	pinned := cron
	pinned.Revision = "9777ffcac0b966d2488cb1e63e155da8b98bd8d6"
	assert.Contains(t, render(t, "templates/protofetch.toml.tmpl", pinned), `revision = "9777ffcac0b966d2488cb1e63e155da8b98bd8d6"`)
}

func TestPinnedRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "protofetch.toml")
	assert.Empty(t, pinnedRevision(path), "no protofetch.toml pins no revision")

	require.NoError(t, os.WriteFile(path, []byte(`[capabilities]
url = "github.com/smartcontractkit/capabilities"
revision = "9777ffcac0b966d2488cb1e63e155da8b98bd8d6"
content_roots = ["/scheduler/cron/protos"]
`), 0o600))
	assert.Equal(t, "9777ffcac0b966d2488cb1e63e155da8b98bd8d6", pinnedRevision(path))
}
