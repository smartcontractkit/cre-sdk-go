// protos/cmd/newcapability/main.go
package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templates embed.FS

type config struct {
	Category      string
	Pkg           string
	MajorVersion  int
	PreReleaseTag string
	Files         []string
	Commit        string
}

func main() {
	var files string
	cfg := config{}
	flag.StringVar(&cfg.Category, "category", "", "Capability category (e.g. scheduler)")
	flag.StringVar(&cfg.Pkg, "pkg", "", "Capability package name (e.g. cron)")
	flag.IntVar(&cfg.MajorVersion, "major", 1, "Major version number")
	flag.StringVar(&cfg.PreReleaseTag, "pre", "", "Optional pre-release tag (e.g. alpha1)")
	flag.StringVar(&files, "files", "trigger.proto", "Comma-separated list of proto files")
	flag.StringVar(&cfg.Commit, "commit", "", "Override commit hash (default: origin/main)")
	flag.Parse()

	if cfg.Category == "" || cfg.Pkg == "" {
		log.Fatal("--category and --pkg are required")
	}

	cfg.Files = strings.Split(files, ",")

	if cfg.Commit == "" {
		out, err := exec.Command("git", "rev-parse", "origin/main").Output()
		if err != nil {
			log.Fatalf("failed to get origin/main commit: %v", err)
		}
		cfg.Commit = strings.TrimSpace(string(out))
	}

	capDir := filepath.Join("capabilities", cfg.Category, cfg.Pkg)
	if cfg.MajorVersion != 1 {
		capDir = filepath.Join(capDir, fmt.Sprintf("v%d", cfg.MajorVersion))
	}
	genDir := filepath.Join(capDir, "generate")
	mustMkdirAll(genDir)

	writeTemplate(filepath.Join(capDir, "generate.go"), "templates/generate.go.tmpl", cfg)
	writeTemplate(filepath.Join(genDir, "main.go"), "templates/generate_main.go.tmpl", cfg)

	modPath := fmt.Sprintf("github.com/smartcontractkit/cre-sdk-go/capabilities/%s/%s", cfg.Category, cfg.Pkg)
	execCmd(capDir, "go", "mod", "init", modPath)

	// Required dependencies
	execCmd(capDir, "go", "get", fmt.Sprintf("github.com/smartcontractkit/cre-sdk-go/generator/protos@%s", cfg.Commit))

	execCmd(capDir, "go", "mod", "tidy")
	execCmd(capDir, "go", "generate", "./...")

	execCmd(capDir, "go", "get", fmt.Sprintf("github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre@%s", cfg.Commit))
	execCmd(capDir, "go", "get", fmt.Sprintf("github.com/smartcontractkit/cre-sdk-go@%s", cfg.Commit))
	execCmd(capDir, "go", "get", "google.golang.org/protobuf@v1.36.6")
	execCmd(capDir, "go", "mod", "tidy")
}

func mustMkdirAll(path string) {
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalf("failed to create %s: %v", path, err)
	}
}

func writeTemplate(dest, tmplPath string, data any) {
	tmpl := template.Must(template.New(filepath.Base(tmplPath)).ParseFS(templates, tmplPath))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Fatalf("failed to execute template %s: %v", tmplPath, err)
	}

	if err := os.WriteFile(dest, buf.Bytes(), 0644); err != nil {
		log.Fatalf("failed to write %s: %v", dest, err)
	}
}

func execCmd(dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("%s %v failed: %v", name, args, err)
	}
}
