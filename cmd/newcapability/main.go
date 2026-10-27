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

const (
	// capabilitiesLibs is the module capabilities are generated with.
	capabilitiesLibs = "github.com/smartcontractkit/capabilities/libs"
	// creGo is the module generated from the base protos.
	creGo = "github.com/smartcontractkit/chainlink-protos/cre/go"
	// protobuf is the module of protoc-gen-go and what the Go it generates links.
	protobuf = "google.golang.org/protobuf"
)

type config struct {
	Category      string
	Pkg           string
	MajorVersion  int
	PreReleaseTag string
	Commit        string
	// CapabilityDir is the capability's directory in the capabilities repo, which has its production protos.
	CapabilityDir string
	// Libs is the version of the capabilities repo's libs to generate with.
	Libs string

	ModPath string
}

func main() {
	cfg := config{}
	flag.StringVar(&cfg.Category, "category", "", "Capability category (e.g. scheduler)")
	flag.StringVar(&cfg.Pkg, "pkg", "", "Capability package name (e.g. cron)")
	flag.IntVar(&cfg.MajorVersion, "major", 1, "Major version number")
	flag.StringVar(&cfg.PreReleaseTag, "pre", "", "Optional pre-release tag (e.g. alpha1)")
	flag.StringVar(&cfg.Commit, "commit", "", "Override commit hash (default: HEAD, which must be pushed)")
	flag.StringVar(&cfg.CapabilityDir, "capability-dir", "", "The capability's directory in the capabilities repo (default: <category>/<pkg>)")
	flag.StringVar(&cfg.Libs, "libs", "latest", "The version of the capabilities repo's libs to generate with")
	flag.Parse()

	if cfg.Category == "" || cfg.Pkg == "" {
		log.Fatal("--category and --pkg are required")
	}
	if cfg.CapabilityDir == "" {
		cfg.CapabilityDir = cfg.Category + "/" + cfg.Pkg
	}

	// The capability is generated with the SDK it is in, so it requires the commit that has it.
	if cfg.Commit == "" {
		out, err := exec.Command("git", "rev-parse", "HEAD").Output()
		if err != nil {
			log.Fatalf("failed to get the HEAD commit: %v", err)
		}
		cfg.Commit = strings.TrimSpace(string(out))
	}

	capDir := filepath.Join("capabilities", cfg.Category, cfg.Pkg)
	if cfg.MajorVersion != 1 {
		capDir = filepath.Join(capDir, fmt.Sprintf("v%d", cfg.MajorVersion))
	}
	genDir := filepath.Join(capDir, "generate")
	mustMkdirAll(genDir)

	cfg.ModPath = "github.com/smartcontractkit/cre-sdk-go/" + filepath.ToSlash(capDir)

	// The generator is a module of its own, so the capabilities repo's libs it generates with is not in the
	// capability's module graph, where it would move what users of the capability require. The capability's protos are
	// fetched at the commit of the libs it requires, with the base protos at the commit the capabilities repo pins,
	// which generating keeps the capability's chainlink-protos/cre/go at.
	writeTemplate(filepath.Join(capDir, "generate.go"), "templates/generate.go.tmpl", cfg)
	writeTemplate(filepath.Join(capDir, "protofetch.toml"), "templates/protofetch.toml.tmpl", cfg)
	writeTemplate(filepath.Join(genDir, "main.go"), "templates/generate_main.go.tmpl", cfg)

	for dir, mod := range map[string]string{capDir: cfg.ModPath, genDir: cfg.ModPath + "/generate"} {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); os.IsNotExist(err) {
			execCmd(dir, "go", "mod", "init", mod)
		}
	}
	// The capability needs only the go version the SDK does, so its users do too; the generator is its own module.
	execCmd(capDir, "go", "mod", "edit", "-go="+goVersion("."))
	sdk := "github.com/smartcontractkit/cre-sdk-go@" + cfg.Commit
	execCmd(capDir, "go", "get", sdk)
	// The capability requires what its generated code links, at the SDK's versions: protobuf, whose protoc-gen-go
	// generates it, and chainlink-protos/cre/go, which generating then moves to the base protos the capability is at.
	for _, mod := range []string{protobuf, creGo} {
		execCmd(capDir, "go", "get", mod+"@"+goList(capDir, "-m", "-f", "{{.Version}}", mod))
	}
	// The generator is the SDK's, at the version the capability uses.
	execCmd(genDir, "go", "get", sdk, capabilitiesLibs+"@"+cfg.Libs)
	execCmd(genDir, "go", "mod", "tidy")
	execCmd(capDir, "go", "generate", "./...")
	execCmd(capDir, "go", "mod", "tidy")
}

// goVersion is the go version of the module at dir.
func goVersion(dir string) string {
	return goList(dir, "-m", "-f", "{{.GoVersion}}")
}

// goList runs go list in dir with args, returning what it prints.
func goList(dir string, args ...string) string {
	cmd := exec.Command("go", append([]string{"list"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		log.Fatalf("go list %v in %s failed: %v", args, dir, err)
	}
	return strings.TrimSpace(string(out))
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
