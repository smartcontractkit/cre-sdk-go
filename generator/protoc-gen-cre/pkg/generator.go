package pkg

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/smartcontractkit/chainlink-common/pkg/capabilities/v2/protoc/pkg"
	"github.com/smartcontractkit/chainlink-protos/cre/go/sdk"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

//go:embed templates/sdk.go.tmpl
var goClientBaseTemplate string

//go:embed templates/sdk_trigger.go.tmpl
var triggerMethodTemplate string

//go:embed templates/sdk_action.go.tmpl
var actionMethodTemplate string

//go:embed templates/wrap_report.go.tmpl
var wrapReportTemplate string

//go:embed templates/mock.go.tmpl
var mockTemplate string

var reportResponseType = (&sdk.ReportResponse{}).ProtoReflect().Descriptor().FullName()
var reportIdent = protogen.GoIdent{
	GoName:       "Report",
	GoImportPath: "github.com/smartcontractkit/cre-sdk-go/cre",
}

var clientTemplates = []*pkg.TemplateGenerator{
	{
		Name:               "go_sdk",
		Template:           goClientBaseTemplate,
		FileNameTemplate:   "{{.}}_sdk_gen.go",
		PbLabelTLangLabels: pkg.PbLabelToGoLabels,
		StringLblValue:     pkg.StringLblValue(false),
		Partials: map[string]string{
			"trigger_method": triggerMethodTemplate,
			"action_method":  actionMethodTemplate,
			"wrap_report":    wrapReportTemplate,
		},
		ExtraFns: map[string]any{
			"SafeGoName":  strcase.ToCamel,
			"WrapMessage": wrapMessage,
			"IsFieldReport": func(field *protogen.Field) bool {
				return field.Message != nil && field.Message.Desc.FullName() == reportResponseType
			},
			"ImportPath": func(ident protogen.GoIdent) string {
				parts := strings.Split(string(ident.GoImportPath), "/")
				return parts[len(parts)-1]
			},
		},
	},
	{
		Name:               "go_mock",
		Template:           mockTemplate,
		FileNameTemplate:   "mock/{{.}}_mock_gen.go",
		PbLabelTLangLabels: pkg.PbLabelToGoLabels,
		StringLblValue:     pkg.StringLblValue(false),
		ExtraFns:           map[string]any{},
	},
}

func GenerateClient(plugin *protogen.Plugin, file *protogen.File, toolName, localPrefix string) error {
	if len(file.Services) == 0 {
		return nil
	}

	for _, template := range clientTemplates {
		names := map[string]bool{}
		template.AddImport("github.com/smartcontractkit/cre-sdk-go/cre", "cre")
		template.ExtraFns["RegisterType"] = func(ident protogen.GoIdent) bool {
			_, ok := names[ident.GoName]
			if ok {
				return false
			}

			names[ident.GoName] = true
			return true
		}
		template.ExtraFns["FieldGoType"] = func(field *protogen.Field, currentPkg string) (string, error) {
			pkgName := ""
			if field.Message != nil {
				pkgName = field.Message.GoIdent.GoImportPath.String()
			}
			return fieldGoType(template, field, pkgName)
		}
		if err := template.GenerateFile(file, plugin, file, toolName, localPrefix); err != nil {
			return err
		}
	}

	return nil
}

func wrapMessage(input *protogen.Message) protogen.GoIdent {
	if input.Desc.FullName() == reportResponseType {
		return reportIdent
	}

	for _, field := range input.Fields {
		msg := field.Message
		if msg == nil {
			continue
		}

		if msg.Desc.FullName() == reportResponseType {
			ident := input.GoIdent
			ident.GoName = strings.Replace(ident.GoName, "Report", "CreReport", 1)
			return ident
		}
	}

	return input.GoIdent
}

func fieldGoType(t *pkg.TemplateGenerator, field *protogen.Field, currentPkg string) (string, error) {
	if field.Desc.IsMap() {
		return buildMapType(t, field, currentPkg)
	} else if field.Desc.IsList() {
		return buildListType(t, field, currentPkg)
	} else if field.Message != nil {
		return "*" + t.TypeName(wrapMessage(field.Message), currentPkg), nil
	} else if field.Enum != nil {
		return t.TypeName(field.GoIdent, currentPkg), nil
	}

	typ, err := scalarGoType(field.Desc.Kind())
	if err != nil {
		return "", fmt.Errorf("invalid type for field %s: %w", field.GoName, err)
	}

	if field.Desc.HasPresence() {
		return "*" + typ, nil
	}
	return typ, nil
}

func buildListType(t *pkg.TemplateGenerator, field *protogen.Field, currentPkg string) (string, error) {
	elemKind := field.Desc.Kind()
	var elem string
	if elemKind == protoreflect.MessageKind {
		elem = "*" + t.TypeName(wrapMessage(field.Message), currentPkg)
	} else if elemKind == protoreflect.EnumKind {
		elem = t.TypeName(field.GoIdent, currentPkg)
	} else {
		var err error
		elem, err = scalarGoType(elemKind)
		if err != nil {
			return "", err
		}
	}
	return "[]" + elem, nil
}

func buildMapType(t *pkg.TemplateGenerator, field *protogen.Field, currentPkg string) (string, error) {
	key, err := scalarGoType(field.Desc.MapKey().Kind())
	if err != nil {
		return "", fmt.Errorf("invalid map key type for field %s: %w", field.GoName, err)
	}

	valKind := field.Desc.MapValue().Kind()
	var val string
	if valKind == protoreflect.MessageKind || valKind == protoreflect.EnumKind {
		val = t.TypeName(field.Message.Fields[1].GoIdent, currentPkg)
	} else {
		val, err = scalarGoType(valKind)
		if err != nil {
			return "", fmt.Errorf("invalid map value type for field %s: %w", field.GoName, err)
		}
	}
	return fmt.Sprintf("map[%s]%s", key, val), nil
}

func scalarGoType(kind protoreflect.Kind) (string, error) {
	switch kind {
	case protoreflect.BoolKind:
		return "bool", nil
	case protoreflect.StringKind:
		return "string", nil
	case protoreflect.BytesKind:
		return "[]byte", nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "int32", nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "int64", nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "uint32", nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "uint64", nil
	case protoreflect.FloatKind:
		return "float32", nil
	case protoreflect.DoubleKind:
		return "float64", nil
	default:
		return "", fmt.Errorf("unsupported scalar type: %s", kind)
	}
}
