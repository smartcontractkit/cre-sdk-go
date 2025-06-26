module github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron

go 1.24.4

replace (
	github.com/smartcontractkit/cre-sdk-go => ../../..
	github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre => ../../../generator/protoc-gen-cre
	github.com/smartcontractkit/cre-sdk-go/generator/protos => ../../../generator/protos
)

require (
	github.com/smartcontractkit/cre-sdk-go v0.0.0-00010101000000-000000000000
	github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre v0.0.0-00010101000000-000000000000
	github.com/smartcontractkit/cre-sdk-go/generator/protos v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/go-viper/mapstructure/v2 v2.3.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/smartcontractkit/chainlink-common/pkg/values v0.0.0-20250626050139-95e779d9eac6 // indirect
)
