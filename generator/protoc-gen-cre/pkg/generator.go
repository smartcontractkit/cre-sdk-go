package pkg

import (
	_ "embed"

	"github.com/smartcontractkit/chainlink-common/pkg/capabilities/v2/protoc/pkg"
	"google.golang.org/protobuf/compiler/protogen"
)

//go:embed templates/sdk.go.tmpl
var goClientBaseTemplate string

//go:embed templates/sdk_trigger.go.tmpl
var goTriggerMethodTemplate string

//go:embed templates/sdk_action.go.tmpl
var goActionMethodTemplate string

//go:embed templates/mock.go.tmpl
var goMockTemplate string

var clientTemplates = []pkg.TemplateGenerator{
	{
		Name:             "go_sdk",
		Template:         goClientBaseTemplate,
		FileNameTemplate: "{{.}}_sdk_gen.go",
		Partials: map[string]string{
			"trigger_method": goTriggerMethodTemplate,
			"action_method":  goActionMethodTemplate,
		},
	},
	{
		Name:             "go_mock",
		Template:         goMockTemplate,
		FileNameTemplate: "mock/{{.}}_mock_gen.go",
	},
}

func GenerateClient(plugin *protogen.Plugin, file *protogen.File, toolName, localPrefix string) error {
	if len(file.Services) == 0 {
		return nil
	}

	for _, template := range clientTemplates {
		if err := template.GenerateFile(file, plugin, file, toolName, localPrefix); err != nil {
			return err
		}
	}

	return nil
}
