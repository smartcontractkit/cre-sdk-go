module github.com/smartcontractkit/cre-sdk-go/capabilities/scheduler/cron

go 1.24.4

replace (
	github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre => ../../../generator/protoc-gen-cre
	github.com/smartcontractkit/cre-sdk-go => ../../..
)

require (
	github.com/smartcontractkit/cre-sdk-go/generator/protoc-gen-cre v0.0.0-00010101000000-000000000000
	github.com/smartcontractkit/cre-sdk-go v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/smartcontractkit/chainlink-common v0.7.1-0.20250625225013-f52453b839ae // indirect
	github.com/smartcontractkit/chainlink-common/pkg/values v0.0.0-20250625225013-f52453b839ae // indirect
)
