package main

import (
	"fmt"
	"strconv"
	"strings"
)

func parseCapabilityFlags(args []string) (map[string]*CapabilityConfig, error) {
	configs := map[string]*CapabilityConfig{
		"": {},
	}
	seen := map[string]bool{}

	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("invalid argument: %s", arg)
		}
		raw := strings.TrimPrefix(arg, "--")
		parts := strings.SplitN(raw, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid key=value pair: %s", raw)
		}
		name, value := parts[0], parts[1]

		if seen[name] {
			return nil, fmt.Errorf("duplicate flag: %s", name)
		}
		seen[name] = true

		var prefix, field string
		switch {
		case name == "category", name == "pkg", name == "major-version", name == "pre-release-tag", name == "files":
			field = name
		case strings.Contains(name, "-"):
			segments := strings.SplitN(name, "-", 2)
			prefix, field = segments[0], segments[1]
		default:
			return nil, fmt.Errorf("malformed flag: %s", name)
		}

		conf, ok := configs[prefix]
		if !ok {
			conf = &CapabilityConfig{}
			configs[prefix] = conf
		}

		switch field {
		case "category":
			conf.Category = value
		case "pkg":
			conf.Pkg = value
		case "major-version":
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid major-version: %s", value)
			}
			conf.MajorVersion = n
		case "pre-release-tag":
			conf.PreReleaseTag = value
		case "files":
			if value != "" {
				conf.Files = strings.Split(value, ",")
			}
		default:
			return nil, fmt.Errorf("unknown field: %s", field)
		}
	}

	for name, conf := range configs {
		if err := conf.Validate(); err != nil {
			return nil, fmt.Errorf("invalid config for capability %q: %w", name, err)
		}
	}

	return configs, nil
}
